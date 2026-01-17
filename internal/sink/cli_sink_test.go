package sink

import (
	"bytes"
	"testing"
	"time"

	"basanos/internal/event"

	"github.com/stretchr/testify/assert"
)

func TestCLISink_PrintsDotOnPassingScenario(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/login", "pass", timestamp))

	assert.Equal(t, ".", buffer.String())
}

func TestCLISink_PrintsFOnFailingScenario(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/login", "fail", timestamp))

	assert.Equal(t, "F", buffer.String())
}

func TestCLISink_PrintsSummaryOnRunEnd(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewRunEndEvent("run-1", "fail", 3, 1, timestamp))

	assert.Equal(t, "\n\n3 passed, 1 failed\n", buffer.String())
}

func TestCLISink_PrintsFailuresBeforeSummary(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/health", "pass", timestamp))
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/login", "fail", timestamp))
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/status", "pass", timestamp))
	sink.Emit(event.NewRunEndEvent("run-1", "fail", 2, 1, timestamp))

	expected := `.F.

Failures:

  1) basic_http/login

2 passed, 1 failed
`
	assert.Equal(t, expected, buffer.String())
}

func TestCLISink_DisplaysStdoutForFailedScenario(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewScenarioEnterEvent("run-1", "basic_http/health", "Health Check", timestamp))
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/health", "pass", timestamp))
	sink.Emit(event.NewScenarioEnterEvent("run-1", "basic_http/login", "Login", timestamp))
	sink.Emit(event.NewOutputEvent("run-1", "stdout", "Login failed\n"))
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/login", "fail", timestamp))
	sink.Emit(event.NewRunEndEvent("run-1", "fail", 1, 1, timestamp))

	expected := `.F

Failures:

  1) basic_http/login
     stdout:
       Login failed

1 passed, 1 failed
`
	assert.Equal(t, expected, buffer.String())
}

func TestCLISink_DisplaysStderrForFailedScenario(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewCLISink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	sink.Emit(event.NewScenarioEnterEvent("run-1", "basic_http/error", "Error Test", timestamp))
	sink.Emit(event.NewOutputEvent("run-1", "stdout", "Attempting request\n"))
	sink.Emit(event.NewOutputEvent("run-1", "stderr", "Connection refused\n"))
	sink.Emit(event.NewScenarioExitEvent("run-1", "basic_http/error", "fail", timestamp))
	sink.Emit(event.NewRunEndEvent("run-1", "fail", 0, 1, timestamp))

	expected := `F

Failures:

  1) basic_http/error
     stdout:
       Attempting request
     stderr:
       Connection refused

0 passed, 1 failed
`
	assert.Equal(t, expected, buffer.String())
}
