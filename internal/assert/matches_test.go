package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatches_PatternMatches_Pass(t *testing.T) {
	result := Matches(`\d{3}`, "HTTP 200 OK")

	assert.True(t, result.IsPassed())
}

func TestMatches_PatternNoMatch_Fail(t *testing.T) {
	result := Matches(`\d{4}`, "HTTP 200 OK")

	assert.False(t, result.IsPassed())
}

func TestMatches_InvalidRegex_Error(t *testing.T) {
	result := Matches(`[invalid`, "text")

	assert.False(t, result.IsPassed())
	concrete := result.(*MatchesResult)
	assert.NotEmpty(t, concrete.Error)
}

func TestMatches_Format(t *testing.T) {
	result := &MatchesResult{
		BaseResult: BaseResult{Passed: false},
		Pattern:    `\d{4}`,
		Target:     "HTTP 200 OK",
	}

	output := result.Format()

	assert.Contains(t, output, "FAIL")
	assert.Contains(t, output, "Pattern:")
	assert.Contains(t, output, `\d{4}`)
	assert.Contains(t, output, "Target:")
	assert.Contains(t, output, "HTTP 200 OK")
}
