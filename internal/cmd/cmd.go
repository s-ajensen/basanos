package cmd

import (
	"flag"
	"io"
	"strings"
	"time"

	"basanos/internal/executor"
	"basanos/internal/fs"
	"basanos/internal/runner"
	"basanos/internal/sink"
	"basanos/internal/sink/cli"
	"basanos/internal/tree"
)

type stringSlice []string

func (slice *stringSlice) String() string {
	return strings.Join(*slice, ",")
}

func (slice *stringSlice) Set(value string) error {
	*slice = append(*slice, value)
	return nil
}

type Config struct {
	SpecDir     string
	Outputs     []string
	Filter      string
	ShowHelp    bool
	ShowVersion bool
	Verbose     bool
}

type RunOptions struct {
	Config     *Config
	FileSystem fs.FileSystem
	Executor   executor.Executor
	Stdout     io.Writer
	OutputFS   fs.WritableFS
}

type RunResult struct {
	Success bool
	Passed  int
	Failed  int
	Error   error
}

func Run(opts RunOptions) RunResult {
	if opts.FileSystem == nil {
		return RunResult{Success: true}
	}
	specTree, err := tree.LoadSpecTree(opts.FileSystem, opts.Config.SpecDir)
	if err != nil {
		return RunResult{Error: err}
	}
	runID := time.Now().Format("2006-01-02_150405")
	var sinks []sink.Sink
	for _, output := range opts.Config.Outputs {
		sinks = append(sinks, createSink(output, opts, runID))
	}
	specRunner := runner.NewRunner(opts.Executor, sinks...)
	specRunner.Filter = opts.Config.Filter
	absSpecRootPath, err := opts.FileSystem.Abs(opts.Config.SpecDir)
	if err != nil {
		return RunResult{Error: err}
	}
	err = specRunner.RunWithID(runID, specTree, absSpecRootPath)
	return RunResult{
		Success: specRunner.Failed() == 0 && err == nil,
		Passed:  specRunner.Passed(),
		Failed:  specRunner.Failed(),
		Error:   err,
	}
}

var writerSinks = map[string]func(io.Writer) sink.Sink{
	"json":  sink.NewJsonStreamSink,
	"junit": sink.NewJunitSink,
}

func createSink(output string, opts RunOptions, runID string) sink.Sink {
	for prefix, factory := range writerSinks {
		if strings.HasPrefix(output, prefix) {
			return factory(opts.Stdout)
		}
	}
	if strings.HasPrefix(output, "cli") {
		return cli.NewReporter(opts.Stdout, opts.Config.Verbose, true)
	}
	if strings.HasPrefix(output, "files") {
		return createFileSink(output, opts, runID)
	}
	return nil
}

func createFileSink(output string, opts RunOptions, runID string) sink.Sink {
	path := extractFilesPath(output)
	writableFS := resolveWritableFS(opts.OutputFS, path)
	return sink.NewFileSink(writableFS, runID)
}

func extractFilesPath(output string) string {
	if _, after, found := strings.Cut(output, ":"); found {
		return after
	}
	return "runs"
}

func resolveWritableFS(outputFS fs.WritableFS, path string) fs.WritableFS {
	if outputFS != nil {
		return outputFS
	}
	return fs.NewOSWritableFS(path)
}

func ParseArgs(args []string) (*Config, error) {
	config := &Config{}

	var outputs stringSlice
	flags := flag.NewFlagSet("basanos", flag.ContinueOnError)
	flags.StringVar(&config.SpecDir, "s", "spec", "spec directory")
	flags.StringVar(&config.SpecDir, "spec", "spec", "spec directory")
	flags.Var(&outputs, "o", "output sink")
	flags.Var(&outputs, "output", "output sink")
	flags.StringVar(&config.Filter, "f", "", "filter pattern")
	flags.StringVar(&config.Filter, "filter", "", "filter pattern")
	flags.BoolVar(&config.ShowHelp, "h", false, "show help")
	flags.BoolVar(&config.ShowHelp, "help", false, "show help")
	flags.BoolVar(&config.ShowVersion, "v", false, "show version")
	flags.BoolVar(&config.ShowVersion, "version", false, "show version")
	flags.BoolVar(&config.Verbose, "verbose", false, "verbose output")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}

	if len(outputs) == 0 {
		config.Outputs = []string{"cli"}
	} else {
		config.Outputs = outputs
	}

	return config, nil
}
