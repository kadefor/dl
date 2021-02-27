// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9
// +build !plan9

// The getgo command installs Go to the user's system.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	goversion "golang.org/dl/internal/version"
)

const (
	stableVersionURL = "https://golang.google.cn/dl/?mode=json&include=all"
)

var usage = `getgo - A command-line installer for Go

Usage:
    getgo (VERSION|list [all]|setup|[status]|remove VERSION)

Commands:
    [status]           # Display current info, install latest if not found
    list [all]         # List installed; "all" - list all stable versions
    setup [-s]         # Set environment variables, interactive mode? [WIP]
    remove VERSION     # Remove specific version
    VERSION            # Set default, install specific version if not exist
                         eg: up, latest, tip, go1.16, 1.15

Examples:
    getgo              # Display current info, install latest if not found
    getgo list         # List installed
    getgo list all     # List all stable
    getgo remove 1.15  # Remove 1.15
    getgo setup        # Set environment variables, interactive mode [WIP]
    getgo setup -s     # Set environment variables, noninteractive mode [WIP]

    getgo up           # Set default, install latest if not exist
    getgo latest       # Set default, install latest if not exist
    getgo 1.15         # Set default, install 1.15 if not exist
    getgo tip          # Set default, install tip/master if not exist [GFW]
    getgo tip 23102    # Set default, install CL#23102 if not exist [GFW]

`

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "status")
	}

	switch arg := os.Args[1]; arg {
	case "status":
		statusCmd()
	case "list":
		listCmd()
	case "remove":
		removeCmd()
	case "setup":
		setupCmd()
	default:
		if strings.HasPrefix(arg, "-") {
			log.Fatal(usage)
		}
		installCmd()
	}
}

func runOut(cmd string, arg ...string) (string, error) {
	out, err := exec.Command(cmd, arg...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func run(cmd string, arg ...string) error {
	env := os.Environ()
	env = append(env, "GO111MODULE=on")
	env = append(env, "GOPROXY=https://goproxy.cn,direct")

	c := exec.Command(cmd, arg...)
	c.Env = env
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err
}

func versionRoot(version string) (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(homedir, "sdk", version), nil
}

type Version struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	Sha256   string `json:"sha256"`
	Size     int    `json:"size"`
	Kind     string `json:"kind"`
}

type Release struct {
	Version  string    `json:"version"`
	Stable   bool      `json:"stable"`
	Versions []Version `json:"files"`
}

func statusCmd() {
	gobin := findGo()
	if gobin == "" {
		var version string
		var err error
		gobin, version, err = bootstrap()
		if err != nil {
			log.Fatal(err)
		}
		err = setDefaultVersion(version)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s: you may need to run `getgo setup` to set up the environment, just once", version)
		os.Exit(0)
	}

	version, err := runOut(gobin, "version")
	if err != nil {
		log.Fatal(err)
	}

	goroot, err := runOut(gobin, "env", "GOROOT")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s (%s)", version, goroot)
}

func listCmd() {
	base, err := versionRoot("")
	dirs, err := filepath.Glob(filepath.Join(base, "go?*"))
	if err != nil {
		log.Fatal(err)
	}

	installed := map[string]bool{}
	for _, p := range dirs {
		v := filepath.Base(p)
		installed[v] = true
	}

	var currentVersion string
	gobin := findGo()
	if gobin != "" {
		s, err := runOut(gobin, "tool", "dist", "version")
		if err != nil {
			log.Fatal(err)
		}
		currentVersion = s
	}

	requireAll := len(os.Args) == 3 && (os.Args[2] == "all" || os.Args[2] == "-a")

	versions, err := listVersions()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range versions {
		if !isValidArchive(v) {
			continue
		}

		if installed[v.Version] {
			if currentVersion == v.Version {
				log.Println("*", v.Version)
			} else {
				log.Println("+", v.Version)
			}
		} else if requireAll {
			log.Println(" ", v.Version)
		}
	}
}

func removeCmd() {
	if len(os.Args) != 3 {
		log.Fatal(usage)
	}

	version := os.Args[2]
	if !strings.HasPrefix(version, "go") {
		version = "go" + version
	}

	if isDefault(version) {
		log.Fatalf("%s: can't remove default version", version)
	}

	goroot, err := versionRoot(version)
	if err != nil {
		log.Fatal(err)
	}

	err = os.RemoveAll(goroot)
	if err != nil {
		log.Fatalf("%s: remove failed: %v", version, err)
	}

	log.Fatalf("%s: removed", version)
}

func setupCmd() {
	interactive := true
	if len(os.Args) == 3 && os.Args[2] == "-s" {
		interactive = false
	}
	err := setupGOPATH(context.TODO(), interactive)
	if err != nil {
		log.Fatal(err)
	}
}

func installCmd() {
	if len(os.Args) == 2 {
		os.Args = append(os.Args, "latest")
	}

	version := strings.ToLower(os.Args[1])

	var CL string
	switch version {
	case "up", "latest", "update":
		versions, err := listVersions()
		if err != nil {
			log.Fatalf("update: %v", err)
		}
		version = versions[0].Version
	case "tip", "gotip":
		if len(os.Args) == 3 {
			CL = os.Args[2]
		}
	}

	if !strings.HasPrefix(version, "go") {
		version = "go" + version
	}

	var needSetup bool
	var err error

	gobin := findGo()
	if gobin == "" {
		gobin, _, err = bootstrap()
		if err != nil {
			log.Fatal(err)
		}
		needSetup = true
	}

	err = run(gobin, "get", "golang.org/dl/"+version)
	if err != nil {
		log.Fatalf("%s: %v", version, err)
	}

	if CL != "" {
		err = run(filepath.Join(gobinPath(gobin), version), "download", CL)
	} else {
		err = run(filepath.Join(gobinPath(gobin), version), "download")
	}
	if err != nil {
		log.Fatalf("%s: %v", version, err)
	}

	err = setDefaultVersion(version)
	if err != nil {
		log.Fatalf("%s: %v", version, err)
	}
	log.Printf("%s: already set default", version)
	if needSetup {
		log.Printf("%s: you may need to run `getgo setup` to set up the environment, just once", version)
	}
}

func listVersions() ([]Version, error) {
	res, err := http.Get(stableVersionURL)
	if err != nil {
		return nil, err
	}
	if v := res.StatusCode; v != 200 {
		return nil, fmt.Errorf("http request failed: %d %s", v, stableVersionURL)
	}

	var releases []Release
	err = json.NewDecoder(res.Body).Decode(&releases)
	if err != nil {
		return nil, fmt.Errorf("version json parse failed: %v", err)
	}

	var versions []Version
	for _, release := range releases {
		for _, v := range release.Versions {
			if release.Stable && isValidArchive(v) {
				versions = append(versions, v)
			}
		}
	}

	return versions, nil
}

func setDefaultVersion(version string) error {
	goroot, err := versionRoot(version)
	if err != nil {
		return err
	}
	defaultGoRoot, err := versionRoot("go")
	if err != nil {
		return err
	}

	_ = os.Remove(defaultGoRoot)

	err = os.Symlink(goroot, defaultGoRoot)
	if err != nil {
		return err
	}
	return nil
}

func findGo() string {
	gobin, _ := exec.LookPath("go")
	return gobin
}

func gobinPath(gobin string) string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if gobin != "" {
		binPath, _ := runOut(gobin, "env", "GOBIN")
		if binPath == "" {
			goPath, _ := runOut(gobin, "env", "GOPATH")
			if goPath == "" {
				goPath = filepath.Join(homedir, "go")
			}
			binPath = filepath.Join(goPath, "bin")
		}
		return binPath
	}
	binPath := filepath.Join(homedir, "go", "bin")
	return binPath
}

func bootstrap() (gobin string, latestVersion string, err error) {
	versions, err := listVersions()
	if err != nil {
		return "", "", err
	}

	latestVersion = versions[0].Version
	goroot, err := versionRoot(latestVersion)
	if err != nil {
		return "", "", fmt.Errorf("bootstrap %s: %v", latestVersion, err)
	}
	if err := goversion.Install(goroot, latestVersion); err != nil {
		return "", "", fmt.Errorf("bootstrap %s: download failed: %v", latestVersion, err)
	}

	gobin = filepath.Join(goroot, "bin", "go")
	return gobin, latestVersion, nil
}

func isValidArchive(v Version) bool {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goos == "linux" && goarch == "arm" {
		goarch = "armv6l"
	}

	if v.OS == goos && v.Arch == goarch && v.Kind == "archive" && v.Sha256 != "" {
		return true
	}
	return false
}

func isDefault(version string) bool {
	goroot, err := versionRoot(version)
	if err != nil {
		log.Fatal(err)
	}

	golink, err := versionRoot("go")
	if err != nil {
		log.Fatal(err)
	}

	defaultGoRoot, err := os.Readlink(golink)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		log.Fatal(err)
	}
	if goroot == defaultGoRoot {
		return true
	}
	return false
}
