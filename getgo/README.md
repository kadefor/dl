!!! experiment !!!

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