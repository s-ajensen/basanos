package assert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContains_NeedleFound_Pass(t *testing.T) {
	result := Contains("200 OK", "HTTP/1.1 200 OK\nContent-Type: text/html")

	assert.True(t, result.IsPassed())
}

func TestContains_NeedleNotFound_Fail(t *testing.T) {
	result := Contains("404", "HTTP/1.1 200 OK")

	assert.False(t, result.IsPassed())
	concrete := result.(*ContainsResult)
	assert.Equal(t, "404", concrete.Needle)
	assert.Contains(t, concrete.Haystack, "200 OK")
}

func TestContains_Format_ShowsContext(t *testing.T) {
	result := &ContainsResult{
		BaseResult: BaseResult{Passed: false},
		Needle:     "404",
		Haystack:   "HTTP/1.1 200 OK\nContent-Type: text/html",
	}

	output := result.Format()

	assert.Contains(t, output, "FAIL")
	assert.Contains(t, output, "Looking for:")
	assert.Contains(t, output, "404")
	assert.Contains(t, output, "In:")
	assert.Contains(t, output, "200 OK")
}
