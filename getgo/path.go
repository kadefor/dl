// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !plan9
// +build !plan9

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	bashConfig = ".bash_profile"
	zshConfig  = ".zshrc"
)

var errExitCleanly error = errors.New("exit cleanly sentinel value")

func prompt(ctx context.Context, query, defaultAnswer string, interactive bool) (string, error) {
	if !interactive {
		return defaultAnswer, nil
	}

	fmt.Printf("%s [%s]: ", query, defaultAnswer)

	type result struct {
		answer string
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		s := bufio.NewScanner(os.Stdin)
		if !s.Scan() {
			ch <- result{"", s.Err()}
			return
		}
		answer := s.Text()
		if answer == "" {
			answer = defaultAnswer
		}
		ch <- result{answer, nil}
	}()

	select {
	case r := <-ch:
		return r.answer, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func setupGOPATH(ctx context.Context, interactive bool) error {
	answer, err := prompt(ctx, "Would you like us to setup your GOPATH? Y/n", "Y", interactive)
	if err != nil {
		return err
	}

	if strings.ToLower(answer) != "y" {
		fmt.Println("Exiting and not setting up GOPATH.")
		return errExitCleanly
	}

	log.Println("Setting up GOPATH")
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// set $GOPATH
		gopath = filepath.Join(home, "go")
		if err := persistEnvVar("GOPATH", gopath); err != nil {
			return err
		}
		log.Println("GOPATH has been set up!")
	} else {
		log.Printf("GOPATH is already set to %s", gopath)
	}

	defaultGoRoot, err := versionRoot("go")
	if err != nil {
		return err
	}
	if err := appendToPATH(filepath.Join(defaultGoRoot, "bin")); err != nil {
		return err
	}

	if err := appendToPATH(filepath.Join(gopath, "bin")); err != nil {
		return err
	}

	return persistEnvChangesForSession()
}

// appendToPATH adds the given path to the PATH environment variable and
// persists it for future sessions.
func appendToPATH(value string) error {
	if isInPATH(value) {
		return nil
	}
	return persistEnvVar("PATH", pathVar+envSeparator+value)
}

func isInPATH(dir string) bool {
	p := os.Getenv("PATH")

	paths := strings.Split(p, envSeparator)
	for _, d := range paths {
		if d == dir {
			return true
		}
	}

	return false
}

func getHomeDir() (string, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return homedir, nil
}

func checkStringExistsFile(filename, value string) (bool, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == value {
			return true, nil
		}
	}

	return false, scanner.Err()
}

func appendToFile(filename, value string) error {
	log.Printf("Adding %q to %s", value, filename)

	ok, err := checkStringExistsFile(filename, value)
	if err != nil {
		return err
	}
	if ok {
		// Nothing to do.
		return nil
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(lineEnding + value + lineEnding)
	return err
}

func isShell(name string) bool {
	return strings.Contains(currentShell(), name)
}

// persistEnvVarWindows sets an environment variable in the Windows
// registry.
func persistEnvVarWindows(name, value string) error {
	err := run("powershell", "-command", fmt.Sprintf(`[Environment]::SetEnvironmentVariable("%s", "%s", "User")`, name, value))
	return err
}

func persistEnvVar(name, value string) error {
	if runtime.GOOS == "windows" {
		if err := persistEnvVarWindows(name, value); err != nil {
			return err
		}

		if isShell("cmd.exe") || isShell("powershell.exe") {
			return os.Setenv(strings.ToUpper(name), value)
		}
		// User is in bash, zsh, etc.
		// Also set the environment variable in their shell config.
	}

	rc, err := shellConfigFile()
	if err != nil {
		return err
	}

	line := fmt.Sprintf("export %s=%s", strings.ToUpper(name), value)
	if err := appendToFile(rc, line); err != nil {
		return err
	}

	return os.Setenv(strings.ToUpper(name), value)
}

func shellConfigFile() (string, error) {
	home, err := getHomeDir()
	if err != nil {
		return "", err
	}

	switch {
	case isShell("bash"):
		return filepath.Join(home, bashConfig), nil
	case isShell("zsh"):
		return filepath.Join(home, zshConfig), nil
	default:
		return "", fmt.Errorf("%q is not a supported shell", currentShell())
	}
}
