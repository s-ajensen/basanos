package main

import (
	"fmt"
	"os"

	"basanos/internal/cmd"
	"basanos/internal/executor"
	"basanos/internal/fs"
)

var version = "dev"

func main() {
	config, err := cmd.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if config.ShowHelp {
		printHelp()
		return
	}

	if config.ShowVersion {
		fmt.Println(version)
		return
	}

	opts := cmd.RunOptions{
		Config:     config,
		FileSystem: fs.OSFileSystem{},
		Executor:   executor.NewShellExecutor(),
		Stdout:     os.Stdout,
	}

	if err := cmd.Run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`basanos - acceptance test framework

Usage: basanos [options]

Options:
  -s, --spec DIR      Spec directory (default: spec)
  -o, --output SINK   Output sink (default: cli)
                      Can be specified multiple times
                      Formats: cli, json, files, files:PATH, junit
  -f, --filter PAT    Filter specs by path pattern
  -h, --help          Show this help
  -v, --version       Show version`)
}
