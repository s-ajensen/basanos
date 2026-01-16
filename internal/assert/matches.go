package assert

import (
	"regexp"
	"strings"
)

type MatchesResult struct {
	BaseResult
	Pattern string
	Target  string
	Error   string
}

func Matches(pattern, target string) AssertResult {
	result := &MatchesResult{
		Pattern: pattern,
		Target:  target,
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Passed = regex.MatchString(target)
	return result
}

func (result *MatchesResult) Format() string {
	if result.Passed {
		return "PASS: pattern matches target\n"
	}
	return result.formatFailure()
}

func (result *MatchesResult) formatFailure() string {
	var output strings.Builder
	output.WriteString(result.failureHeader())
	output.WriteString("\nPattern:\n")
	output.WriteString("  " + result.Pattern + "\n")
	output.WriteString("\nTarget:\n")
	output.WriteString("  " + strings.ReplaceAll(result.Target, "\n", "\n  ") + "\n")
	return output.String()
}

func (result *MatchesResult) failureHeader() string {
	if result.Error != "" {
		return "FAIL: invalid regex pattern\n──────────────────────────────────\nError:\n  " + result.Error + "\n"
	}
	return "FAIL: pattern does not match target\n──────────────────────────────────\n"
}
