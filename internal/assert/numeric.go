package assert

import (
	"fmt"
	"strconv"
	"strings"
)

type NumericResult struct {
	BaseResult
	Left     string
	Right    string
	LeftVal  float64
	RightVal float64
	Error    string
	Op       string
}

func parseNumeric(left, right string) (float64, float64, error) {
	leftVal, err := strconv.ParseFloat(left, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid number: %s", left)
	}
	rightVal, err := strconv.ParseFloat(right, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid number: %s", right)
	}
	return leftVal, rightVal, nil
}

func numericCompare(left, right, op string, compare func(l, r float64) bool) *NumericResult {
	result := &NumericResult{
		Left:  left,
		Right: right,
		Op:    op,
	}

	leftVal, rightVal, err := parseNumeric(left, right)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.LeftVal = leftVal
	result.RightVal = rightVal
	result.Passed = compare(leftVal, rightVal)
	return result
}

func GreaterThan(left, right string) AssertResult {
	return numericCompare(left, right, ">", func(l, r float64) bool { return l > r })
}

func GreaterThanOrEqual(left, right string) AssertResult {
	return numericCompare(left, right, ">=", func(l, r float64) bool { return l >= r })
}

func LessThan(left, right string) AssertResult {
	return numericCompare(left, right, "<", func(l, r float64) bool { return l < r })
}

func LessThanOrEqual(left, right string) AssertResult {
	return numericCompare(left, right, "<=", func(l, r float64) bool { return l <= r })
}

func (result *NumericResult) Format() string {
	if result.Passed {
		return fmt.Sprintf("PASS: %s %s %s\n", result.Left, result.Op, result.Right)
	}
	return result.formatFailure()
}

func (result *NumericResult) formatFailure() string {
	var output strings.Builder
	output.WriteString(result.failureHeader())
	output.WriteString(fmt.Sprintf("\nLeft:  %s\n", result.Left))
	output.WriteString(fmt.Sprintf("Right: %s\n", result.Right))
	return output.String()
}

func (result *NumericResult) failureHeader() string {
	if result.Error != "" {
		return "FAIL: invalid numeric comparison\n──────────────────────────────────\nError:\n  " + result.Error + "\n"
	}
	return fmt.Sprintf("FAIL: %s %s %s is false\n──────────────────────────────────\n", result.Left, result.Op, result.Right)
}
