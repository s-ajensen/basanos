package assert

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func passingAssert(a, b string) AssertResult {
	return &Result{BaseResult: BaseResult{Passed: true}, Expected: a, Actual: b}
}

func failingAssert(a, b string) AssertResult {
	return &Result{BaseResult: BaseResult{Passed: false}, Expected: a, Actual: b}
}

func protocolInput(expected, actual string) string {
	var buf strings.Builder
	buf.WriteString("basanos:1\n")
	buf.WriteString(intToStr(len(expected)) + "\n")
	buf.WriteString(expected)
	buf.WriteString(intToStr(len(actual)) + "\n")
	buf.WriteString(actual)
	return buf.String()
}

func intToStr(n int) string {
	return strings.TrimSpace(strings.Replace(string(rune(n+'0')), "\x00", "", -1))
}

func TestRunCLI_StdinMode_PassingAssertion(t *testing.T) {
	stdin := strings.NewReader(protocolInput("hello", "hello"))
	stdout := &bytes.Buffer{}

	exitCode := RunCLI([]string{}, stdin, stdout, ResolveBothValues, passingAssert)

	assert.Equal(t, 0, exitCode)
}

func TestRunCLI_StdinMode_FailingAssertion(t *testing.T) {
	stdin := strings.NewReader(protocolInput("expected", "actual"))
	stdout := &bytes.Buffer{}

	exitCode := RunCLI([]string{}, stdin, stdout, ResolveBothValues, failingAssert)

	assert.Equal(t, 1, exitCode)
}

func TestRunCLI_ArgsMode_UsesResolver(t *testing.T) {
	var resolvedFirst, resolvedSecond string
	trackingResolver := func(args []string) (string, string, error) {
		resolvedFirst = "resolved:" + args[0]
		resolvedSecond = "resolved:" + args[1]
		return resolvedFirst, resolvedSecond, nil
	}
	var assertedFirst, assertedSecond string
	trackingAssert := func(first, second string) AssertResult {
		assertedFirst = first
		assertedSecond = second
		return &Result{BaseResult: BaseResult{Passed: true}, Expected: first, Actual: second}
	}
	stdout := &bytes.Buffer{}

	RunCLI([]string{"arg1", "arg2"}, nil, stdout, trackingResolver, trackingAssert)

	assert.Equal(t, "resolved:arg1", assertedFirst)
	assert.Equal(t, "resolved:arg2", assertedSecond)
}

func TestResolveBothValues_ValidArgs(t *testing.T) {
	first, second, err := ResolveBothValues([]string{"hello", "world"})

	assert.NoError(t, err)
	assert.Equal(t, "hello", first)
	assert.Equal(t, "world", second)
}

func TestResolveBothValues_WrongArgCount(t *testing.T) {
	_, _, err := ResolveBothValues([]string{"only_one"})

	assert.Error(t, err)
}

func TestResolveLiterals_ValidArgs(t *testing.T) {
	first, second, err := ResolveLiterals([]string{"10", "20"})

	assert.NoError(t, err)
	assert.Equal(t, "10", first)
	assert.Equal(t, "20", second)
}

func TestResolveLiterals_WrongArgCount(t *testing.T) {
	_, _, err := ResolveLiterals([]string{"only_one"})

	assert.Error(t, err)
}
