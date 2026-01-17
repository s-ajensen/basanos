package sink

import (
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"basanos/internal/event"
)

type junitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Suites   []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
}

type pendingCase struct {
	name      string
	classname string
	startTime time.Time
}

type JunitSink struct {
	writer       io.Writer
	suites       map[string]*junitTestSuite
	suiteOrder   []string
	pendingCases map[string]*pendingCase
}

func NewJunitSink(writer io.Writer) Sink {
	return &JunitSink{
		writer:       writer,
		suites:       make(map[string]*junitTestSuite),
		pendingCases: make(map[string]*pendingCase),
	}
}

func (sink *JunitSink) Emit(incoming any) error {
	switch typed := incoming.(type) {
	case *event.ContextEnterEvent:
		sink.handleContextEnter(typed)
	case *event.ScenarioEnterEvent:
		sink.handleScenarioEnter(typed)
	case *event.ScenarioExitEvent:
		sink.handleScenarioExit(typed)
	case *event.RunEndEvent:
		return sink.handleRunEnd(typed)
	}
	return nil
}

func (sink *JunitSink) handleContextEnter(enter *event.ContextEnterEvent) {
	sink.suites[enter.Path] = &junitTestSuite{
		Name: enter.Path,
	}
	sink.suiteOrder = append(sink.suiteOrder, enter.Path)
}

func (sink *JunitSink) handleScenarioEnter(enter *event.ScenarioEnterEvent) {
	contextPath := filepath.Dir(enter.Path)
	sink.pendingCases[enter.Path] = &pendingCase{
		name:      enter.Name,
		classname: contextPath,
		startTime: enter.Timestamp,
	}
}

func (sink *JunitSink) handleScenarioExit(exit *event.ScenarioExitEvent) {
	pending := sink.pendingCases[exit.Path]
	suite := sink.findSuiteForPath(exit.Path)

	duration := exit.Timestamp.Sub(pending.startTime).Seconds()
	testCase := junitTestCase{
		Name:      pending.name,
		Classname: pending.classname,
		Time:      fmt.Sprintf("%.3f", duration),
	}

	if exit.Status == "fail" {
		testCase.Failure = &junitFailure{Message: "test failed"}
		suite.Failures++
	}
	suite.Cases = append(suite.Cases, testCase)
	suite.Tests++

	delete(sink.pendingCases, exit.Path)
}

func (sink *JunitSink) findSuiteForPath(scenarioPath string) *junitTestSuite {
	path := scenarioPath
	for path != "." && path != "" {
		path = filepath.Dir(path)
		if suite, exists := sink.suites[path]; exists {
			return suite
		}
	}
	return nil
}

func (sink *JunitSink) handleRunEnd(end *event.RunEndEvent) error {
	testsuites := junitTestSuites{
		Tests:    end.Passed + end.Failed,
		Failures: end.Failed,
	}

	for _, path := range sink.suiteOrder {
		testsuites.Suites = append(testsuites.Suites, *sink.suites[path])
	}

	output, err := xml.MarshalIndent(testsuites, "", "  ")
	if err != nil {
		return err
	}

	_, err = sink.writer.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = sink.writer.Write(output)
	return err
}
