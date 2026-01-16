package assert

import (
	"fmt"
	"io"
)

type AssertResult interface {
	Format() string
	IsPassed() bool
}

type BaseResult struct {
	Passed bool
}

func (b BaseResult) IsPassed() bool {
	return b.Passed
}

type AssertFunc func(first, second string) AssertResult

func RunCLI(args []string, stdin io.Reader, stdout io.Writer,
	resolveArgs func([]string) (string, string, error),
	assertFn AssertFunc) int {
	var first, second string
	var err error

	if len(args) == 0 {
		first, second, err = ParseProtocol(stdin)
	} else {
		first, second, err = resolveArgs(args)
	}

	if err != nil {
		fmt.Fprintln(stdout, err.Error())
		return 1
	}

	result := assertFn(first, second)
	fmt.Fprint(stdout, result.Format())

	if result.IsPassed() {
		return 0
	}
	return 1
}

func requireTwoArgs(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("expected 2 arguments, got %d", len(args))
	}
	return nil
}

func ResolveBothValues(args []string) (string, string, error) {
	if err := requireTwoArgs(args); err != nil {
		return "", "", err
	}
	first, err := ResolveValue(args[0])
	if err != nil {
		return "", "", err
	}
	second, err := ResolveValue(args[1])
	if err != nil {
		return "", "", err
	}
	return first, second, nil
}

func ResolveLiterals(args []string) (string, string, error) {
	if err := requireTwoArgs(args); err != nil {
		return "", "", err
	}
	return args[0], args[1], nil
}

func ResolveLiteralAndValue(args []string) (string, string, error) {
	if err := requireTwoArgs(args); err != nil {
		return "", "", err
	}
	second, err := ResolveValue(args[1])
	if err != nil {
		return "", "", err
	}
	return args[0], second, nil
}
