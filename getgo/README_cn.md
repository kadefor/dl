!!! 这只是一个实验性的工具 !!!

```shell
getgo - A command-line installer for Go

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
```

1. 显示当前Go版本信息, 如果没有找到go命令, 则下载安装最新版本

```shell
$ getgo
go1.16: already downloaded in /Users/kadefor/sdk/go1.16
go1.16: you may need to run `getgo setup` to set up the environment, just once

$ getgo setup   # 只需要配置一次, 以后就不需要了
Would you like us to setup your GOPATH? Y/n [Y]:
Setting up GOPATH
GOPATH is already set to /Users/kadefor/go

One more thing! Run `source /Users/kadefor/.bash_profile` to persist the
new environment variables to your current session, or open a
new shell prompt.

$ source ~/.bash_profile

$ getgo
go version go1.16 darwin/amd64 (/Users/kadefor/sdk/go)
```

```shell
$ getgo
```

2. 列出已安装或所有的稳定版本, `*`号开头为当前版本, `+`号开头为已安装

```shell
$ getgo list
* go1.16
+ go1.15
+ go1.11

$ getgo list all 
* go1.16
  go1.15.8
  go1.15.7
  go1.15.6
  go1.15.5
  go1.15.4
  go1.15.3
  go1.15.2
  go1.15.1
+ go1.15
  ...
```

3. 切换到指定版本, 如果已安装直接切换, 如果未安装则安装后切换; 其中版本号:

* `up, update, latest`: 代表最新版本, 升级并切换
* `1.14, go1.14`: 特定版本, 可以省略前面的`go`
* `tip, gotip, tip 23108`: 开发版本, 源码构建, GFW

```shell
$ getgo 1.13
Downloaded   0.0% (     3267 / 121212387 bytes) ...
Downloaded  12.6% ( 15269776 / 121212387 bytes) ...
Downloaded  24.2% ( 29359904 / 121212387 bytes) ...
Downloaded  86.6% (104971488 / 121212387 bytes) ...
Downloaded  98.6% (119540009 / 121212387 bytes) ...
Downloaded 100.0% (121212387 / 121212387 bytes)
Unpacking /Users/kadefor/sdk/go1.13/go1.13.darwin-amd64.tar.gz ...
Success. You may now run 'go1.13'
go1.13: already set default

$ getgo 1.13
go1.13: already downloaded in /Users/kadefor/sdk/go1.13
go1.13: already set default

$ getgo
go version go1.13 darwin/amd64 (/Users/kadefor/sdk/go)

$ go version
go version go1.13 darwin/amd64

$ getgo up
go version go1.16 darwin/amd64 (/Users/kadefor/sdk/go)

$ go version
go version go1.16 darwin/amd64
```

4. 删除指定版本

```shell
$ getgo list
+ go1.16
+ go1.15
* go1.13
+ go1.11

$ getgo remove 1.13
go1.13: can't remove default version

$ getgo remove 1.15
go1.15: removed

$ getgo list
+ go1.16
* go1.13
+ go1.11
```