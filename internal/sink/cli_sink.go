package sink

import (
	"fmt"
	"io"
	"strings"

	"basanos/internal/event"
)

type failure struct {
	path   string
	stdout string
	stderr string
}

type CLISink struct {
	writer        io.Writer
	failures      []failure
	currentStdout strings.Builder
	currentStderr strings.Builder
}

func NewCLISink(w io.Writer) Sink {
	return &CLISink{writer: w}
}

func (s *CLISink) Emit(e any) error {
	switch evt := e.(type) {
	case *event.ScenarioEnterEvent:
		s.currentStdout.Reset()
		s.currentStderr.Reset()
	case *event.OutputEvent:
		s.handleOutput(evt)
	case *event.ScenarioExitEvent:
		s.handleScenarioExit(evt)
	case *event.RunEndEvent:
		fmt.Fprintf(s.writer, "\n\n")
		s.printFailures()
		s.printSummary(evt.Passed, evt.Failed)
	}
	return nil
}

func (s *CLISink) handleOutput(evt *event.OutputEvent) {
	switch evt.Stream {
	case "stdout":
		s.currentStdout.WriteString(evt.Data)
	case "stderr":
		s.currentStderr.WriteString(evt.Data)
	}
}

func (s *CLISink) handleScenarioExit(evt *event.ScenarioExitEvent) {
	if evt.Status == "pass" {
		s.writer.Write([]byte("."))
		return
	}
	if evt.Status == "fail" {
		s.writer.Write([]byte("F"))
		s.failures = append(s.failures, failure{
			path:   evt.Path,
			stdout: s.currentStdout.String(),
			stderr: s.currentStderr.String(),
		})
	}
}

func (s *CLISink) printFailures() {
	if len(s.failures) == 0 {
		return
	}
	fmt.Fprintf(s.writer, "Failures:\n\n")
	for i, f := range s.failures {
		s.printFailure(i+1, f)
	}
	fmt.Fprintf(s.writer, "\n")
}

func (s *CLISink) printFailure(index int, f failure) {
	fmt.Fprintf(s.writer, "  %d) %s\n", index, f.path)
	s.printIndentedOutput("stdout", f.stdout)
	s.printIndentedOutput("stderr", f.stderr)
}

func (s *CLISink) printIndentedOutput(label, content string) {
	if content == "" {
		return
	}
	fmt.Fprintf(s.writer, "     %s:\n", label)
	for _, line := range strings.Split(strings.TrimSuffix(content, "\n"), "\n") {
		fmt.Fprintf(s.writer, "       %s\n", line)
	}
}

func (s *CLISink) printSummary(passed, failed int) {
	fmt.Fprintf(s.writer, "%d passed, %d failed\n", passed, failed)
}
