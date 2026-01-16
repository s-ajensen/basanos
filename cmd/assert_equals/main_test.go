package main

import (
	"bytes"
	"strings"
	"testing"

	"basanos/internal/assert"
)

func TestRun_StdinMode_EqualValues_ExitsZero(t *testing.T) {
	stdin := strings.NewReader("basanos:1\n5\nhello5\nhello")
	stdout := &bytes.Buffer{}

	exitCode := assert.RunCLI([]string{}, stdin, stdout, assert.ResolveBothValues, assert.Equals)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_StdinMode_DifferentValues_ExitsOne(t *testing.T) {
	stdin := strings.NewReader("basanos:1\n5\nhello5\nworld")
	stdout := &bytes.Buffer{}

	exitCode := assert.RunCLI([]string{}, stdin, stdout, assert.ResolveBothValues, assert.Equals)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "FAIL") {
		t.Errorf("expected output to contain FAIL, got %q", stdout.String())
	}
}

func TestRun_TwoArgsMode_BackwardCompatibility(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}

	exitCode := assert.RunCLI([]string{"hello", "hello"}, stdin, stdout, assert.ResolveBothValues, assert.Equals)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "PASS") {
		t.Errorf("expected output to contain PASS, got %q", stdout.String())
	}
}
