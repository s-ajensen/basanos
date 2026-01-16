package assert

import (
	"os"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

type Result struct {
	BaseResult
	Expected string
	Actual   string
	Diff     string
}

func Equals(expected, actual string) AssertResult {
	return &Result{
		BaseResult: BaseResult{Passed: expected == actual},
		Expected:   expected,
		Actual:     actual,
		Diff:       generateDiff(expected, actual),
	}
}

func (result *Result) Format() string {
	var output strings.Builder

	if result.Passed {
		output.WriteString("PASS: values are equal\n")
	} else {
		output.WriteString("FAIL: values differ\n")
		output.WriteString("──────────────────────────────────\n")
		output.WriteString("Expected:\n")
		output.WriteString("  " + strings.ReplaceAll(result.Expected, "\n", "\n  ") + "\n")
		output.WriteString("\nActual:\n")
		output.WriteString("  " + strings.ReplaceAll(result.Actual, "\n", "\n  ") + "\n")
		if result.Diff != "" {
			output.WriteString("\nDiff:\n")
			output.WriteString("  " + strings.ReplaceAll(result.Diff, "\n", "\n  ") + "\n")
		}
	}

	return output.String()
}

func ResolveValue(arg string) (string, error) {
	if _, err := os.Stat(arg); err == nil {
		content, err := os.ReadFile(arg)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}
	return arg, nil
}

func generateDiff(expected, actual string) string {
	if expected == actual {
		return ""
	}

	diff := difflib.UnifiedDiff{
		A:       difflib.SplitLines(expected),
		B:       difflib.SplitLines(actual),
		Context: 3,
	}

	result, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return ""
	}

	return result
}
