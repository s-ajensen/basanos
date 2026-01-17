package sink

import (
	"bytes"
	"encoding/xml"
	"testing"
	"time"

	"basanos/internal/event"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJunitSink_WritesValidXML(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewJunitSink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	runID := "2026-01-15_143022"

	sink.Emit(event.NewRunStartEvent(runID, timestamp))
	sink.Emit(event.NewContextEnterEvent(runID, "api", "API Tests", timestamp))
	sink.Emit(event.NewScenarioEnterEvent(runID, "api/health_check", "Health Check", timestamp))
	sink.Emit(event.NewScenarioExitEvent(runID, "api/health_check", "pass", timestamp.Add(100*time.Millisecond)))
	sink.Emit(event.NewContextExitEvent(runID, "api", timestamp.Add(100*time.Millisecond)))
	sink.Emit(event.NewRunEndEvent(runID, "pass", 1, 0, timestamp.Add(100*time.Millisecond)))

	output := buffer.String()

	var testsuites struct {
		XMLName  xml.Name `xml:"testsuites"`
		Tests    int      `xml:"tests,attr"`
		Failures int      `xml:"failures,attr"`
		Suites   []struct {
			Name     string `xml:"name,attr"`
			Tests    int    `xml:"tests,attr"`
			Failures int    `xml:"failures,attr"`
			Cases    []struct {
				Name      string `xml:"name,attr"`
				Classname string `xml:"classname,attr"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}

	err := xml.Unmarshal([]byte(output), &testsuites)
	require.NoError(t, err, "Output should be valid XML: %s", output)

	assert.Equal(t, 1, testsuites.Tests)
	assert.Equal(t, 0, testsuites.Failures)
	require.Len(t, testsuites.Suites, 1)

	suite := testsuites.Suites[0]
	assert.Equal(t, "api", suite.Name)
	assert.Equal(t, 1, suite.Tests)
	assert.Equal(t, 0, suite.Failures)
	require.Len(t, suite.Cases, 1)

	testcase := suite.Cases[0]
	assert.Equal(t, "Health Check", testcase.Name)
	assert.Equal(t, "api", testcase.Classname)
}

func TestJunitSink_IncludesFailureElement(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewJunitSink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	runID := "2026-01-15_143022"

	sink.Emit(event.NewRunStartEvent(runID, timestamp))
	sink.Emit(event.NewContextEnterEvent(runID, "api", "API Tests", timestamp))
	sink.Emit(event.NewScenarioEnterEvent(runID, "api/login", "Login", timestamp))
	sink.Emit(event.NewScenarioExitEvent(runID, "api/login", "fail", timestamp.Add(200*time.Millisecond)))
	sink.Emit(event.NewContextExitEvent(runID, "api", timestamp.Add(200*time.Millisecond)))
	sink.Emit(event.NewRunEndEvent(runID, "fail", 0, 1, timestamp.Add(200*time.Millisecond)))

	output := buffer.String()

	var testsuites struct {
		XMLName xml.Name `xml:"testsuites"`
		Suites  []struct {
			Cases []struct {
				Name    string `xml:"name,attr"`
				Failure *struct {
					Message string `xml:"message,attr"`
				} `xml:"failure"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}

	err := xml.Unmarshal([]byte(output), &testsuites)
	require.NoError(t, err, "Output should be valid XML: %s", output)

	require.Len(t, testsuites.Suites, 1)
	require.Len(t, testsuites.Suites[0].Cases, 1)

	testcase := testsuites.Suites[0].Cases[0]
	assert.Equal(t, "Login", testcase.Name)
	assert.NotNil(t, testcase.Failure, "Failed testcase should have a <failure> element")
}

func TestJunitSink_NestedScenarioGroups_FindsCorrectSuite(t *testing.T) {
	buffer := &bytes.Buffer{}
	sink := NewJunitSink(buffer)

	timestamp := time.Date(2026, 1, 15, 14, 30, 22, 0, time.UTC)
	runID := "2026-01-15_143022"

	sink.Emit(event.NewRunStartEvent(runID, timestamp))
	sink.Emit(event.NewContextEnterEvent(runID, "spec/assertions", "Assertions", timestamp))
	sink.Emit(event.NewScenarioEnterEvent(runID, "spec/assertions/contains/substring_found", "Substring found", timestamp))
	sink.Emit(event.NewScenarioExitEvent(runID, "spec/assertions/contains/substring_found", "pass", timestamp.Add(100*time.Millisecond)))
	sink.Emit(event.NewContextExitEvent(runID, "spec/assertions", timestamp.Add(100*time.Millisecond)))
	sink.Emit(event.NewRunEndEvent(runID, "pass", 1, 0, timestamp.Add(100*time.Millisecond)))

	output := buffer.String()

	var testsuites struct {
		XMLName xml.Name `xml:"testsuites"`
		Suites  []struct {
			Name  string `xml:"name,attr"`
			Tests int    `xml:"tests,attr"`
			Cases []struct {
				Name string `xml:"name,attr"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}

	err := xml.Unmarshal([]byte(output), &testsuites)
	require.NoError(t, err, "Output should be valid XML: %s", output)

	require.Len(t, testsuites.Suites, 1)
	suite := testsuites.Suites[0]
	assert.Equal(t, "spec/assertions", suite.Name)
	assert.Equal(t, 1, suite.Tests)
	require.Len(t, suite.Cases, 1)
	assert.Equal(t, "Substring found", suite.Cases[0].Name)
}
