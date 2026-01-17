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

func (s *JunitSink) Emit(evt any) error {
	switch e := evt.(type) {
	case *event.ContextEnterEvent:
		s.handleContextEnter(e)
	case *event.ScenarioEnterEvent:
		s.handleScenarioEnter(e)
	case *event.ScenarioExitEvent:
		s.handleScenarioExit(e)
	case *event.RunEndEvent:
		return s.handleRunEnd(e)
	}
	return nil
}

func (s *JunitSink) handleContextEnter(e *event.ContextEnterEvent) {
	s.suites[e.Path] = &junitTestSuite{
		Name: e.Path,
	}
	s.suiteOrder = append(s.suiteOrder, e.Path)
}

func (s *JunitSink) handleScenarioEnter(e *event.ScenarioEnterEvent) {
	contextPath := filepath.Dir(e.Path)
	s.pendingCases[e.Path] = &pendingCase{
		name:      e.Name,
		classname: contextPath,
		startTime: e.Timestamp,
	}
}

func (s *JunitSink) handleScenarioExit(e *event.ScenarioExitEvent) {
	pending := s.pendingCases[e.Path]
	contextPath := filepath.Dir(e.Path)
	suite := s.suites[contextPath]

	duration := e.Timestamp.Sub(pending.startTime).Seconds()
	testCase := junitTestCase{
		Name:      pending.name,
		Classname: pending.classname,
		Time:      fmt.Sprintf("%.3f", duration),
	}

	if e.Status == "fail" {
		testCase.Failure = &junitFailure{Message: "test failed"}
		suite.Failures++
	}
	suite.Cases = append(suite.Cases, testCase)
	suite.Tests++

	delete(s.pendingCases, e.Path)
}

func (s *JunitSink) handleRunEnd(e *event.RunEndEvent) error {
	testsuites := junitTestSuites{
		Tests:    e.Passed + e.Failed,
		Failures: e.Failed,
	}

	for _, path := range s.suiteOrder {
		testsuites.Suites = append(testsuites.Suites, *s.suites[path])
	}

	output, err := xml.MarshalIndent(testsuites, "", "  ")
	if err != nil {
		return err
	}

	_, err = s.writer.Write([]byte(xml.Header))
	if err != nil {
		return err
	}
	_, err = s.writer.Write(output)
	return err
}
