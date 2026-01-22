package runner

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"basanos/internal/assert"
)

type CapturedOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func resolveAssertionArgs(command string, captured CapturedOutput, env map[string]string) (first, second string, err error) {
	expanded := os.Expand(command, func(key string) string {
		return env[key]
	})

	_, args := parseCommandArgs(expanded)
	if len(args) < 2 {
		return "", "", fmt.Errorf("assertion command must have executable and 2 args")
	}

	first = resolveArg(args[0], captured, env)
	second = resolveArg(args[1], captured, env)

	return first, second, nil
}

func runOutputVar() string {
	return "RUN_OUTPUT"
}

func runOutput(env map[string]string) string {
	return env[runOutputVar()]
}

func stdoutPath(runOutput string) string {
	return runOutput + "/stdout"
}

func stderrPath(runOutput string) string {
	return runOutput + "/stderr"
}

func exitCodePath(runOutput string) string {
	return runOutput + "/exit_code"
}

func usesResources(unexpanded_command string, env map[string]string) bool {
	runOutput := fmt.Sprintf("${%s}", runOutputVar())
	return strings.Contains(unexpanded_command, exitCodePath(runOutput)) ||
		strings.Contains(unexpanded_command, stdoutPath(runOutput)) ||
		strings.Contains(unexpanded_command, stderrPath(runOutput))
}

func resolveArg(arg string, captured CapturedOutput, env map[string]string) string {
	runOutput := runOutput(env)

	capturedValues := map[string]string{
		stdoutPath(runOutput):   captured.Stdout,
		stderrPath(runOutput):   captured.Stderr,
		exitCodePath(runOutput): strconv.Itoa(captured.ExitCode),
	}

	if value, ok := capturedValues[arg]; ok {
		return value
	}
	return resolveFileOrLiteral(arg)
}

func resolveFileOrLiteral(arg string) string {
	value, err := assert.ResolveValue(arg)
	if err != nil {
		return arg
	}
	return value
}

func parseCommandArgs(command string) (executable string, args []string) {
	var result []string
	var current strings.Builder
	inDoubleQuote := false
	inSingleQuote := false
	escaped := false
	hasContent := false

	for _, char := range command {
		if escaped {
			current.WriteRune(char)
			escaped = false
			continue
		}

		if char == '\\' && inDoubleQuote {
			escaped = true
			continue
		}

		if char == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			hasContent = true
			continue
		}

		if char == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			hasContent = true
			continue
		}

		if char == ' ' && !inDoubleQuote && !inSingleQuote {
			if current.Len() > 0 || hasContent {
				result = append(result, current.String())
				current.Reset()
				hasContent = false
			}
			continue
		}

		current.WriteRune(char)
	}

	if current.Len() > 0 || hasContent {
		result = append(result, current.String())
	}

	if len(result) == 0 {
		return "", nil
	}

	return result[0], result[1:]
}
