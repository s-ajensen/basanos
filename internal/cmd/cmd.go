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
	"basanos/internal/tree"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Config struct {
	SpecDir     string
	Outputs     []string
	Filter      string
	ShowHelp    bool
	ShowVersion bool
}

type RunOptions struct {
	Config     *Config
	FileSystem fs.FileSystem
	Executor   executor.Executor
	Stdout     io.Writer
	OutputFS   sink.WritableFS
}

func Run(opts RunOptions) error {
	if opts.FileSystem == nil {
		return nil
	}
	specTree, err := tree.LoadSpecTree(opts.FileSystem, opts.Config.SpecDir)
	if err != nil {
		return err
	}
	runID := time.Now().Format("2006-01-02_150405")
	var sinks []sink.Sink
	for _, output := range opts.Config.Outputs {
		sinks = append(sinks, createSink(output, opts, runID))
	}
	r := runner.NewRunner(opts.Executor, sinks...)
	r.Filter = opts.Config.Filter
	return r.RunWithID(runID, specTree)
}

var writerSinks = map[string]func(io.Writer) sink.Sink{
	"json":  sink.NewJsonStreamSink,
	"junit": sink.NewJunitSink,
	"cli":   sink.NewCLISink,
}

func createSink(output string, opts RunOptions, runID string) sink.Sink {
	for prefix, factory := range writerSinks {
		if strings.HasPrefix(output, prefix) {
			return factory(opts.Stdout)
		}
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

func resolveWritableFS(outputFS sink.WritableFS, path string) sink.WritableFS {
	if outputFS != nil {
		return outputFS
	}
	return fs.NewOSWritableFS(path)
}

func ParseArgs(args []string) (*Config, error) {
	config := &Config{}

	var outputs stringSlice
	fs := flag.NewFlagSet("basanos", flag.ContinueOnError)
	fs.StringVar(&config.SpecDir, "s", "spec", "spec directory")
	fs.StringVar(&config.SpecDir, "spec", "spec", "spec directory")
	fs.Var(&outputs, "o", "output sink")
	fs.Var(&outputs, "output", "output sink")
	fs.StringVar(&config.Filter, "f", "", "filter pattern")
	fs.StringVar(&config.Filter, "filter", "", "filter pattern")
	fs.BoolVar(&config.ShowHelp, "h", false, "show help")
	fs.BoolVar(&config.ShowHelp, "help", false, "show help")
	fs.BoolVar(&config.ShowVersion, "v", false, "show version")
	fs.BoolVar(&config.ShowVersion, "version", false, "show version")
	fs.Parse(args)

	if len(outputs) == 0 {
		config.Outputs = []string{"cli"}
	} else {
		config.Outputs = outputs
	}

	return config, nil
}
