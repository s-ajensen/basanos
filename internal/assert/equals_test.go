package assert

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEquals_IdenticalStrings_Pass(t *testing.T) {
	result := Equals("hello", "hello")

	assert.True(t, result.IsPassed())
}

func TestEquals_DifferentStrings_Fail(t *testing.T) {
	result := Equals("hello", "world")

	assert.False(t, result.IsPassed())
	concrete := result.(*Result)
	assert.Equal(t, "hello", concrete.Expected)
	assert.Equal(t, "world", concrete.Actual)
}

func TestEquals_GeneratesDiff(t *testing.T) {
	result := Equals("line1\nline2\nline3", "line1\nchanged\nline3")

	concrete := result.(*Result)
	assert.NotEmpty(t, concrete.Diff)
	assert.Contains(t, concrete.Diff, "-line2")
	assert.Contains(t, concrete.Diff, "+changed")
}

func TestResult_Format_Pass(t *testing.T) {
	result := Result{BaseResult: BaseResult{Passed: true}}

	output := result.Format()

	assert.Contains(t, output, "PASS")
}

func TestResult_Format_Fail(t *testing.T) {
	result := Result{
		BaseResult: BaseResult{Passed: false},
		Expected:   "hello",
		Actual:     "world",
		Diff:       "-hello\n+world\n",
	}

	output := result.Format()

	assert.Contains(t, output, "FAIL")
	assert.Contains(t, output, "Expected:")
	assert.Contains(t, output, "hello")
	assert.Contains(t, output, "Actual:")
	assert.Contains(t, output, "world")
	assert.Contains(t, output, "Diff:")
}

func TestEquals_DiffHasNoFileHeaders(t *testing.T) {
	result := Equals("hello", "world")

	concrete := result.(*Result)
	assert.NotContains(t, concrete.Diff, "--- expected")
	assert.NotContains(t, concrete.Diff, "+++ actual")
	assert.Contains(t, concrete.Diff, "@@ ")
	assert.Contains(t, concrete.Diff, "-hello")
	assert.Contains(t, concrete.Diff, "+world")
}

func TestResolveValue_ReturnsLiteralWhenFileDoesNotExist(t *testing.T) {
	result, err := ResolveValue("hello")

	assert.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestResolveValue_ReadsFileWhenExists(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(tempFile, []byte("file content"), 0644)

	result, err := ResolveValue(tempFile)

	assert.NoError(t, err)
	assert.Equal(t, "file content", result)
}
