package cmd

import (
	"bytes"
	"strings"
	"testing"

	fakeexec "basanos/internal/testutil/executor"
	memfs "basanos/internal/testutil/fs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs_DefaultValues(t *testing.T) {
	config, err := ParseArgs([]string{})

	require.NoError(t, err)
	assert.Equal(t, "spec", config.SpecDir)
	assert.Equal(t, []string{"cli"}, config.Outputs)
	assert.Equal(t, "", config.Filter)
	assert.False(t, config.ShowHelp)
	assert.False(t, config.ShowVersion)
}

func TestParseArgs_SpecFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"short form", []string{"-s", "./my-specs"}, "./my-specs"},
		{"long form", []string{"--spec", "./other"}, "./other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.SpecDir)
		})
	}
}

func TestParseArgs_OutputFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{"single output", []string{"-o", "json"}, []string{"json"}},
		{"long form", []string{"--output", "files:./custom"}, []string{"files:./custom"}},
		{"multiple outputs", []string{"-o", "json", "-o", "files"}, []string{"json", "files"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.Outputs)
		})
	}
}

func TestParseArgs_FilterFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"short form", []string{"-f", "auth/*"}, "auth/*"},
		{"long form", []string{"--filter", "basic_http/login"}, "basic_http/login"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.Filter)
		})
	}
}

func TestParseArgs_HelpFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"short form", []string{"-h"}, true},
		{"long form", []string{"--help"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.ShowHelp)
		})
	}
}

func TestParseArgs_VersionFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"short form", []string{"-v"}, true},
		{"long form", []string{"--version"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseArgs(tt.args)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, config.ShowVersion)
		})
	}
}

func TestRun_ReturnsNil(t *testing.T) {
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"files"}},
		FileSystem: nil,
		Executor:   nil,
	}

	err := Run(opts)

	assert.NoError(t, err)
}

func TestRun_ExecutesSpec(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: test
    name: "Test scenario"
    run:
      command: "echo hello"
      timeout: "10s"
`))

	fakeExec := &fakeexec.FakeExecutor{}
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"files"}},
		FileSystem: memFS,
		Executor:   fakeExec,
	}

	err := Run(opts)

	require.NoError(t, err)
	require.Len(t, fakeExec.Commands, 1)
	assert.Equal(t, "echo hello", fakeExec.Commands[0].Command)
}

func TestRun_UsesJsonSink(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: test
    name: "Test scenario"
    run:
      command: "echo hello"
      timeout: "10s"
`))

	var buf bytes.Buffer
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"json"}},
		FileSystem: memFS,
		Executor:   &fakeexec.FakeExecutor{},
		Stdout:     &buf,
	}

	err := Run(opts)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "run_start")
}

func TestRun_CreatesFileSink(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: test
    name: "Test scenario"
    run:
      command: "echo hello"
      timeout: "10s"
`))

	outputFS := memfs.NewMemoryFS()
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"files"}},
		FileSystem: memFS,
		Executor:   &fakeexec.FakeExecutor{},
		OutputFS:   outputFS,
	}

	err := Run(opts)

	require.NoError(t, err)
	files := outputFS.AllFiles()
	require.NotEmpty(t, files, "FileSink should have written output files")

	var foundRunStdout bool
	for _, f := range files {
		if strings.HasSuffix(f, "/spec/test/_run/stdout") {
			foundRunStdout = true
			break
		}
	}
	assert.True(t, foundRunStdout, "Expected to find _run/stdout file, got: %v", files)
}

func TestRun_UsesJunitSink(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: test
    name: "Test scenario"
    run:
      command: "echo hello"
      timeout: "10s"
`))

	var buf bytes.Buffer
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"junit"}},
		FileSystem: memFS,
		Executor:   &fakeexec.FakeExecutor{},
		Stdout:     &buf,
	}

	err := Run(opts)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "<testsuites")
}

func TestRun_UsesCLISink(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: test
    name: "Test scenario"
    run:
      command: "echo hello"
      timeout: "10s"
`))

	var buf bytes.Buffer
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"cli"}},
		FileSystem: memFS,
		Executor:   &fakeexec.FakeExecutor{},
		Stdout:     &buf,
	}

	err := Run(opts)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), ".")
	assert.Contains(t, buf.String(), "passed")
}

func TestRun_UsesFilter(t *testing.T) {
	memFS := memfs.NewMemoryFS()
	memFS.AddDir("spec")
	memFS.AddFile("spec/context.yaml", []byte(`name: "Test"
scenarios:
  - id: first
    name: "First scenario"
    run:
      command: "echo first"
      timeout: "10s"
  - id: second
    name: "Second scenario"
    run:
      command: "echo second"
      timeout: "10s"
`))

	fakeExec := &fakeexec.FakeExecutor{}
	opts := RunOptions{
		Config:     &Config{SpecDir: "spec", Outputs: []string{"files"}, Filter: "spec/first"},
		FileSystem: memFS,
		Executor:   fakeExec,
	}

	err := Run(opts)

	require.NoError(t, err)
	require.Len(t, fakeExec.Commands, 1, "Filter should limit execution to one scenario")
	assert.Equal(t, "echo first", fakeExec.Commands[0].Command)
}
