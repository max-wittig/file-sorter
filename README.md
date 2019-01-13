# File-sorter

Sorts files into directories, based on their file extension or modification date.

This is a re-implementation in Go of the previous [file_sorter](https://github.com/max-wittig/file_sorter), written in Python.
Mostly written to play a bit with the Go programming language.

## build instructions

1. Get dependencies
   ```
   make deps
   ```

1. Build for linux
   ```
   make build-linux
   ```

## usage

```
NAME:
   file-sorter - Sorts files into directories, based on their file extension or modification date

USAGE:
   file-sorter [global options] command [command options] [arguments...]

VERSION:
   0.0.3

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   -c value, --criteria value  Sort criteria of the files (ext|mod). Default: ext (default: "ext")
   --help, -h                  show help
   --version, -v               print the version
```
