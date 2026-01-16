package runner

import (
	"testing"

	"basanos/internal/event"
	"basanos/internal/executor"
	"basanos/internal/spec"
	"basanos/internal/tree"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ExecutedCommand struct {
	Command string
	Timeout string
	Env     map[string]string
}

type FakeExecutor struct {
	Commands         []ExecutedCommand
	Stdout           string
	Stderr           string
	DefaultExitCode  int
	ExitCodes        map[string]int
	TimeoutCommands  map[string]bool
	TimeoutExitCodes map[string]int
}

func (f *FakeExecutor) Execute(command string, timeout string, env map[string]string) (stdout, stderr string, exitCode int, err error) {
	f.Commands = append(f.Commands, ExecutedCommand{Command: command, Timeout: timeout, Env: env})
	if f.TimeoutCommands != nil && f.TimeoutCommands[command] {
		exitCode = -1
		if f.TimeoutExitCodes != nil {
			if code, ok := f.TimeoutExitCodes[command]; ok {
				exitCode = code
			}
		}
		return "", "", exitCode, executor.ErrTimeout
	}
	exitCode = f.DefaultExitCode
	if f.ExitCodes != nil {
		if code, ok := f.ExitCodes[command]; ok {
			exitCode = code
		}
	}
	return f.Stdout, f.Stderr, exitCode, nil
}

type SpySink struct {
	Events []any
}

func (s *SpySink) Emit(e any) error {
	s.Events = append(s.Events, e)
	return nil
}

func newSpecTree(name string) *tree.SpecTree {
	return &tree.SpecTree{
		Path: name,
		Context: &spec.Context{
			Name: name,
			Scenarios: []spec.Scenario{
				{
					ID:   "scenario",
					Name: "Test scenario",
					Run: &spec.RunBlock{
						Command: "test_command",
						Timeout: "10s",
					},
				},
			},
		},
	}
}

func withBeforeHook(t *tree.SpecTree, cmd string) *tree.SpecTree {
	t.Context.Before = &spec.Hook{Run: cmd, Timeout: "5s"}
	return t
}

func withAfterHook(t *tree.SpecTree, cmd string) *tree.SpecTree {
	t.Context.After = &spec.Hook{Run: cmd, Timeout: "5s"}
	return t
}

func withBeforeEachHook(t *tree.SpecTree, cmd string) *tree.SpecTree {
	t.Context.BeforeEach = &spec.Hook{Run: cmd, Timeout: "2s"}
	return t
}

func withAfterEachHook(t *tree.SpecTree, cmd string) *tree.SpecTree {
	t.Context.AfterEach = &spec.Hook{Run: cmd, Timeout: "2s"}
	return t
}

func withAssertions(t *tree.SpecTree, commands ...string) *tree.SpecTree {
	var assertions []spec.Assertion
	for _, cmd := range commands {
		assertions = append(assertions, spec.Assertion{Command: cmd, Timeout: "1s"})
	}
	t.Context.Scenarios[0].Assertions = assertions
	return t
}

func withTwoScenarios(t *tree.SpecTree) *tree.SpecTree {
	t.Context.Scenarios = []spec.Scenario{
		{ID: "scenario1", Name: "First", Run: &spec.RunBlock{Command: "cmd1", Timeout: "5s"}},
		{ID: "scenario2", Name: "Second", Run: &spec.RunBlock{Command: "cmd2", Timeout: "5s"}},
	}
	return t
}

func withChildContext(t *tree.SpecTree, name string) *tree.SpecTree {
	child := &tree.SpecTree{
		Path: t.Path + "/" + name,
		Context: &spec.Context{
			Name: name,
			Scenarios: []spec.Scenario{
				{
					ID:   "child_scenario",
					Name: "Child scenario",
					Run:  &spec.RunBlock{Command: "child_command", Timeout: "10s"},
				},
			},
		},
	}
	t.Children = append(t.Children, child)
	return t
}

func withNestedScenario(t *tree.SpecTree) *tree.SpecTree {
	t.Context.Scenarios = []spec.Scenario{
		{
			ID:   "group",
			Name: "Scenario Group",
			Scenarios: []spec.Scenario{
				{ID: "leaf1", Name: "First", Run: &spec.RunBlock{Command: "cmd1", Timeout: "5s"}},
				{ID: "leaf2", Name: "Second", Run: &spec.RunBlock{Command: "cmd2", Timeout: "5s"}},
			},
		},
	}
	return t
}

func withScenarioCommand(t *tree.SpecTree, cmd, timeout string) *tree.SpecTree {
	t.Context.Scenarios[0].Run = &spec.RunBlock{Command: cmd, Timeout: timeout}
	return t
}

func runSpec(t *testing.T, specTree *tree.SpecTree) (*FakeExecutor, *SpySink) {
	executor := &FakeExecutor{}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	err := runner.Run(specTree)
	require.NoError(t, err)

	return executor, sink
}

func runSpecWithOutput(t *testing.T, specTree *tree.SpecTree, stdout, stderr string) (*FakeExecutor, *SpySink) {
	executor := &FakeExecutor{Stdout: stdout, Stderr: stderr}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	err := runner.Run(specTree)
	require.NoError(t, err)

	return executor, sink
}

func findEvents[T any](events []any) []T {
	var result []T
	for _, e := range events {
		if typed, ok := e.(T); ok {
			result = append(result, typed)
		}
	}
	return result
}

func TestRunner_ExecutesScenarioRunCommand(t *testing.T) {
	specTree := withScenarioCommand(newSpecTree("basic"), "curl http://localhost/", "30s")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "curl http://localhost/", executor.Commands[0].Command)
}

func TestRunner_ExecutesScenarioRunWithTimeout(t *testing.T) {
	specTree := withScenarioCommand(newSpecTree("basic"), "curl http://localhost/", "45s")

	executor, _ := runSpec(t, specTree)

	assert.Equal(t, "45s", executor.Commands[0].Timeout)
}

func TestRunner_EmitsScenarioEnterEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ScenarioEnterEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic/scenario", events[0].Path)
}

func TestRunner_EmitsScenarioExitEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ScenarioExitEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic/scenario", events[0].Path)
}

func TestRunner_EmitsScenarioRunStartEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ScenarioRunStartEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic/scenario", events[0].Path)
}

func TestRunner_EmitsScenarioRunEndEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ScenarioRunEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic/scenario", events[0].Path)
}

func TestRunner_EmitsStdoutOutputEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpecWithOutput(t, specTree, "hello\n", "")

	events := findEvents[*event.OutputEvent](sink.Events)
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "stdout", events[0].Stream)
	assert.Equal(t, "hello\n", events[0].Data)
}

func TestRunner_EmitsStderrOutputEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpecWithOutput(t, specTree, "", "warning\n")

	events := findEvents[*event.OutputEvent](sink.Events)
	require.GreaterOrEqual(t, len(events), 1)
	assert.Equal(t, "stderr", events[0].Stream)
	assert.Equal(t, "warning\n", events[0].Data)
}

func TestRunner_EmitsBothStdoutAndStderrOutputEvents(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpecWithOutput(t, specTree, "out\n", "err\n")

	events := findEvents[*event.OutputEvent](sink.Events)
	assert.Len(t, events, 2)
}

func TestRunner_BeforeHook_ExecutesBeforeScenario(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 2)
	assert.Equal(t, "setup.sh", executor.Commands[0].Command)
	assert.Equal(t, "test_command", executor.Commands[1].Command)
}

func TestRunner_BeforeHook_ExecutesWithTimeout(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")

	executor, _ := runSpec(t, specTree)

	assert.Equal(t, "5s", executor.Commands[0].Timeout)
}

func TestRunner_BeforeHook_EmitsHookStartEvent(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookStartEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "_before", events[0].Hook)
	assert.Equal(t, "basic", events[0].Path)
}

func TestRunner_BeforeHook_EmitsHookEndEvent(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "_before", events[0].Hook)
	assert.Equal(t, "basic", events[0].Path)
}

func TestRunner_AfterHook_ExecutesAfterScenario(t *testing.T) {
	specTree := withAfterHook(newSpecTree("basic"), "cleanup.sh")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 2)
	assert.Equal(t, "test_command", executor.Commands[0].Command)
	assert.Equal(t, "cleanup.sh", executor.Commands[1].Command)
}

func TestRunner_AfterHook_ExecutesWithTimeout(t *testing.T) {
	specTree := withAfterHook(newSpecTree("basic"), "cleanup.sh")

	executor, _ := runSpec(t, specTree)

	assert.Equal(t, "5s", executor.Commands[1].Timeout)
}

func TestRunner_AfterHook_EmitsHookStartEvent(t *testing.T) {
	specTree := withAfterHook(newSpecTree("basic"), "cleanup.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookStartEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "_after", events[0].Hook)
	assert.Equal(t, "basic", events[0].Path)
}

func TestRunner_AfterHook_EmitsHookEndEvent(t *testing.T) {
	specTree := withAfterHook(newSpecTree("basic"), "cleanup.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "_after", events[0].Hook)
	assert.Equal(t, "basic", events[0].Path)
}

func TestRunner_BeforeEachHook_ExecutesBeforeEachScenario(t *testing.T) {
	specTree := withBeforeEachHook(withTwoScenarios(newSpecTree("basic")), "reset.sh")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 4)
	assert.Equal(t, "reset.sh", executor.Commands[0].Command)
	assert.Equal(t, "cmd1", executor.Commands[1].Command)
	assert.Equal(t, "reset.sh", executor.Commands[2].Command)
	assert.Equal(t, "cmd2", executor.Commands[3].Command)
}

func TestRunner_BeforeEachHook_ExecutesWithTimeout(t *testing.T) {
	specTree := withBeforeEachHook(withTwoScenarios(newSpecTree("basic")), "reset.sh")

	executor, _ := runSpec(t, specTree)

	assert.Equal(t, "2s", executor.Commands[0].Timeout)
	assert.Equal(t, "2s", executor.Commands[2].Timeout)
}

func TestRunner_BeforeEachHook_EmitsHookStartEventForEachScenario(t *testing.T) {
	specTree := withBeforeEachHook(withTwoScenarios(newSpecTree("basic")), "reset.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookStartEvent](sink.Events)
	require.Len(t, events, 2)
	assert.Equal(t, "_before_each", events[0].Hook)
	assert.Equal(t, "_before_each", events[1].Hook)
}

func TestRunner_BeforeEachHook_EmitsHookEndEventForEachScenario(t *testing.T) {
	specTree := withBeforeEachHook(withTwoScenarios(newSpecTree("basic")), "reset.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookEndEvent](sink.Events)
	require.Len(t, events, 2)
	assert.Equal(t, "_before_each", events[0].Hook)
	assert.Equal(t, "_before_each", events[1].Hook)
}

func TestRunner_AfterEachHook_ExecutesAfterEachScenario(t *testing.T) {
	specTree := withAfterEachHook(withTwoScenarios(newSpecTree("basic")), "cleanup.sh")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 4)
	assert.Equal(t, "cmd1", executor.Commands[0].Command)
	assert.Equal(t, "cleanup.sh", executor.Commands[1].Command)
	assert.Equal(t, "cmd2", executor.Commands[2].Command)
	assert.Equal(t, "cleanup.sh", executor.Commands[3].Command)
}

func TestRunner_Assertions_ExecutesAssertionCommands(t *testing.T) {
	specTree := withAssertions(newSpecTree("basic"), "assert_equals 0 exit_code", "assert_contains expected.txt stdout")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 3)
	assert.Equal(t, "test_command", executor.Commands[0].Command)
	assert.Equal(t, "assert_equals 0 exit_code", executor.Commands[1].Command)
	assert.Equal(t, "assert_contains expected.txt stdout", executor.Commands[2].Command)
}

func TestRunner_EmitsContextEnterEvent(t *testing.T) {
	specTree := newSpecTree("basic_http")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ContextEnterEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic_http", events[0].Path)
	assert.Equal(t, "basic_http", events[0].Name)
}

func TestRunner_EmitsContextExitEvent(t *testing.T) {
	specTree := newSpecTree("basic_http")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.ContextExitEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic_http", events[0].Path)
}

func TestRunner_ExecutesChildContexts(t *testing.T) {
	specTree := withChildContext(newSpecTree("root"), "child")

	executor, _ := runSpec(t, specTree)

	assert.Len(t, executor.Commands, 2)
}

func TestRunner_ExecutesNestedScenarios(t *testing.T) {
	specTree := withNestedScenario(newSpecTree("basic"))

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 2)
	assert.Equal(t, "cmd1", executor.Commands[0].Command)
	assert.Equal(t, "cmd2", executor.Commands[1].Command)
}

func runSpecWithID(t *testing.T, runID string, specTree *tree.SpecTree) (*FakeExecutor, *SpySink) {
	executor := &FakeExecutor{}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	err := runner.RunWithID(runID, specTree)
	require.NoError(t, err)

	return executor, sink
}

func TestRunner_EmitsRunStartEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpecWithID(t, "2026-01-15_143022", specTree)

	events := findEvents[*event.RunStartEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "2026-01-15_143022", events[0].RunID)
}

func TestRunner_EmitsRunEndEvent(t *testing.T) {
	specTree := newSpecTree("basic")

	_, sink := runSpecWithID(t, "2026-01-15_143022", specTree)

	events := findEvents[*event.RunEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "2026-01-15_143022", events[0].RunID)
	assert.Equal(t, "pass", events[0].Status)
	assert.Equal(t, 1, events[0].Passed)
	assert.Equal(t, 0, events[0].Failed)
}

func TestRunner_RunWithID_CountsMultiplePassingScenarios(t *testing.T) {
	specTree := withTwoScenarios(newSpecTree("basic"))

	_, sink := runSpecWithID(t, "run-1", specTree)

	events := findEvents[*event.RunEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, 2, events[0].Passed)
}

func TestRunner_RunWithID_CountsFailedScenario(t *testing.T) {
	specTree := newSpecTree("basic")
	executor := &FakeExecutor{ExitCodes: map[string]int{"test_command": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.RunWithID("run-1", specTree)

	events := findEvents[*event.RunEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, 0, events[0].Passed)
	assert.Equal(t, 1, events[0].Failed)
}

func TestRunner_RunWithID_StatusFailWhenScenarioFails(t *testing.T) {
	specTree := newSpecTree("basic")
	executor := &FakeExecutor{ExitCodes: map[string]int{"test_command": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.RunWithID("run-1", specTree)

	events := findEvents[*event.RunEndEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "fail", events[0].Status)
}

func TestRunner_ScenarioFailsWhenAssertionFails(t *testing.T) {
	specTree := withAssertions(newSpecTree("basic"), "assert_equals 0 exit_code")
	executor := &FakeExecutor{ExitCodes: map[string]int{
		"test_command":              0,
		"assert_equals 0 exit_code": 1,
	}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	events := findEvents[*event.ScenarioExitEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "fail", events[0].Status)
}

func withGroupBeforeEach(t *tree.SpecTree, cmd string) *tree.SpecTree {
	t.Context.Scenarios = []spec.Scenario{
		{
			ID:         "group",
			Name:       "Scenario Group",
			BeforeEach: &spec.Hook{Run: cmd, Timeout: "2s"},
			Scenarios: []spec.Scenario{
				{ID: "leaf", Name: "Leaf Scenario", Run: &spec.RunBlock{Command: "leaf_cmd", Timeout: "5s"}},
			},
		},
	}
	return t
}

func TestRunner_GroupLevelBeforeEach_ExecutesForNestedScenarios(t *testing.T) {
	specTree := withGroupBeforeEach(newSpecTree("basic"), "group_setup.sh")

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 2)
	assert.Equal(t, "group_setup.sh", executor.Commands[0].Command)
	assert.Equal(t, "leaf_cmd", executor.Commands[1].Command)
}

func TestRunner_AncestorBeforeEach_AllRun(t *testing.T) {
	specTree := newSpecTree("basic")
	specTree.Context.BeforeEach = &spec.Hook{Run: "context_setup.sh", Timeout: "2s"}
	specTree.Context.Scenarios = []spec.Scenario{
		{
			ID:         "group",
			Name:       "Scenario Group",
			BeforeEach: &spec.Hook{Run: "group_setup.sh", Timeout: "2s"},
			Scenarios: []spec.Scenario{
				{ID: "leaf", Name: "Leaf Scenario", Run: &spec.RunBlock{Command: "leaf_cmd", Timeout: "5s"}},
			},
		},
	}

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 3)
	assert.Equal(t, "context_setup.sh", executor.Commands[0].Command)
	assert.Equal(t, "group_setup.sh", executor.Commands[1].Command)
	assert.Equal(t, "leaf_cmd", executor.Commands[2].Command)
}

func TestRunner_AbortRun_StopsAfterFailure(t *testing.T) {
	specTree := withTwoScenarios(newSpecTree("basic"))
	specTree.Context.OnFailure = "abort_run"
	executor := &FakeExecutor{ExitCodes: map[string]int{"cmd1": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	assert.Len(t, executor.Commands, 1)
}

func TestRunner_AbortRun_StopsChildContexts(t *testing.T) {
	specTree := withChildContext(newSpecTree("parent"), "child")
	specTree.Context.OnFailure = "abort_run"
	executor := &FakeExecutor{ExitCodes: map[string]int{"test_command": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	assert.Len(t, executor.Commands, 1)
}

func TestRunner_SkipChildren_SkipsRemainingScenarios(t *testing.T) {
	specTree := withTwoScenarios(newSpecTree("basic"))
	specTree.Context.OnFailure = "skip_children"
	executor := &FakeExecutor{ExitCodes: map[string]int{"cmd1": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	assert.Len(t, executor.Commands, 1)
}

func TestRunner_SkipChildren_ContinuesSiblingContexts(t *testing.T) {
	specTree := newSpecTree("root")
	withChildContext(specTree, "first_child")
	withChildContext(specTree, "second_child")
	specTree.Children[0].Context.OnFailure = "skip_children"
	specTree.Children[0].Context.Scenarios[0].Run.Command = "fail_cmd"
	specTree.Children[1].Context.Scenarios[0].Run.Command = "sibling_cmd"
	executor := &FakeExecutor{ExitCodes: map[string]int{"fail_cmd": 1}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	require.Len(t, executor.Commands, 3)
	assert.Equal(t, "test_command", executor.Commands[0].Command)
	assert.Equal(t, "fail_cmd", executor.Commands[1].Command)
	assert.Equal(t, "sibling_cmd", executor.Commands[2].Command)
}

func withEnv(t *tree.SpecTree, env map[string]string) *tree.SpecTree {
	t.Context.Env = env
	return t
}

func TestRunner_PassesEnvToExecutor(t *testing.T) {
	env := map[string]string{"PORT": "8080", "HOST": "localhost"}
	specTree := withEnv(newSpecTree("basic"), env)

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "8080", executor.Commands[0].Env["PORT"])
	assert.Equal(t, "localhost", executor.Commands[0].Env["HOST"])
}

func TestRunner_MergesScenarioEnvWithContext(t *testing.T) {
	specTree := newSpecTree("basic")
	specTree.Context.Env = map[string]string{"PORT": "8080", "HOST": "localhost"}
	specTree.Context.Scenarios = []spec.Scenario{
		{
			ID:   "group",
			Name: "Scenario Group",
			Env:  map[string]string{"PORT": "9090", "DEBUG": "true"},
			Scenarios: []spec.Scenario{
				{ID: "leaf", Name: "Leaf Scenario", Run: &spec.RunBlock{Command: "leaf_cmd", Timeout: "5s"}},
			},
		},
	}

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "9090", executor.Commands[0].Env["PORT"])
	assert.Equal(t, "localhost", executor.Commands[0].Env["HOST"])
	assert.Equal(t, "true", executor.Commands[0].Env["DEBUG"])
}

func TestRunner_SubstitutesEnvVarsInCommand(t *testing.T) {
	specTree := withEnv(newSpecTree("basic"), map[string]string{"MY_VAR": "hello"})
	specTree.Context.Scenarios[0].Run.Command = "echo ${MY_VAR}"

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "echo hello", executor.Commands[0].Command)
}

func TestRunner_SubstitutesSpecRoot(t *testing.T) {
	specTree := newSpecTree("spec")
	specTree.Context.Scenarios[0].Run.Command = "cat ${SPEC_ROOT}/fixture.txt"

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "cat spec/fixture.txt", executor.Commands[0].Command)
}

func TestRunner_SubstitutesContextOutput(t *testing.T) {
	specTree := newSpecTree("basic_http")
	specTree.Context.Scenarios[0].Run.Command = "cat ${CONTEXT_OUTPUT}/before/stdout"

	executor, _ := runSpecWithID(t, "test-run", specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "cat runs/test-run/basic_http/before/stdout", executor.Commands[0].Command)
}

func TestRunner_SubstitutesScenarioOutput(t *testing.T) {
	specTree := newSpecTree("basic_http")
	specTree.Context.Scenarios[0].ID = "login"
	specTree.Context.Scenarios[0].Run.Command = "cat ${SCENARIO_OUTPUT}/stdout"

	executor, _ := runSpecWithID(t, "test-run", specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "cat runs/test-run/basic_http/login/stdout", executor.Commands[0].Command)
}

func TestRunner_EmitsTimeoutEvent(t *testing.T) {
	specTree := withScenarioCommand(newSpecTree("basic"), "slow_command", "30s")
	executor := &FakeExecutor{TimeoutCommands: map[string]bool{"slow_command": true}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	events := findEvents[*event.TimeoutEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "basic/scenario", events[0].Path)
	assert.Equal(t, "run", events[0].Phase)
	assert.Equal(t, "30s", events[0].Limit)
}

func TestRunner_ScenarioFailsOnTimeout(t *testing.T) {
	specTree := withScenarioCommand(newSpecTree("basic"), "slow_command", "30s")
	executor := &FakeExecutor{TimeoutCommands: map[string]bool{"slow_command": true}}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	events := findEvents[*event.ScenarioExitEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "fail", events[0].Status)
}

func TestRunner_ScenarioFailsOnTimeout_EvenWithZeroExitCode(t *testing.T) {
	specTree := withScenarioCommand(newSpecTree("basic"), "slow_command", "30s")
	executor := &FakeExecutor{
		TimeoutCommands:  map[string]bool{"slow_command": true},
		TimeoutExitCodes: map[string]int{"slow_command": 0},
	}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.Run(specTree)

	events := findEvents[*event.ScenarioExitEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "fail", events[0].Status)
}

func TestRunner_AllEventsIncludeRunID(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")
	executor := &FakeExecutor{Stdout: "output\n"}
	sink := &SpySink{}
	runner := NewRunner(executor, sink)

	runner.RunWithID("test-run-123", specTree)

	contextEnterEvents := findEvents[*event.ContextEnterEvent](sink.Events)
	require.Len(t, contextEnterEvents, 1)
	assert.Equal(t, "test-run-123", contextEnterEvents[0].RunID)

	scenarioEnterEvents := findEvents[*event.ScenarioEnterEvent](sink.Events)
	require.Len(t, scenarioEnterEvents, 1)
	assert.Equal(t, "test-run-123", scenarioEnterEvents[0].RunID)

	outputEvents := findEvents[*event.OutputEvent](sink.Events)
	require.GreaterOrEqual(t, len(outputEvents), 1)
	assert.Equal(t, "test-run-123", outputEvents[0].RunID)

	hookStartEvents := findEvents[*event.HookStartEvent](sink.Events)
	require.Len(t, hookStartEvents, 1)
	assert.Equal(t, "test-run-123", hookStartEvents[0].RunID)
}

func TestRunner_HookNamesArePrefixedWithUnderscore(t *testing.T) {
	specTree := withBeforeHook(newSpecTree("basic"), "setup.sh")

	_, sink := runSpec(t, specTree)

	events := findEvents[*event.HookStartEvent](sink.Events)
	require.Len(t, events, 1)
	assert.Equal(t, "_before", events[0].Hook)
}

func TestRunner_FilterByExactPath(t *testing.T) {
	specTree := &tree.SpecTree{
		Path: "spec",
		Context: &spec.Context{
			Name: "spec",
			Scenarios: []spec.Scenario{
				{ID: "login", Name: "Login", Run: &spec.RunBlock{Command: "login_cmd", Timeout: "5s"}},
				{ID: "logout", Name: "Logout", Run: &spec.RunBlock{Command: "logout_cmd", Timeout: "5s"}},
			},
		},
	}
	fakeExecutor := &FakeExecutor{}
	sink := &SpySink{}
	runner := NewRunner(fakeExecutor, sink)
	runner.Filter = "spec/login"

	runner.Run(specTree)

	require.Len(t, fakeExecutor.Commands, 1)
	assert.Equal(t, "login_cmd", fakeExecutor.Commands[0].Command)
}

func TestRunner_FilterByGlobPattern(t *testing.T) {
	specTree := &tree.SpecTree{
		Path: "spec",
		Context: &spec.Context{
			Name: "spec",
		},
		Children: []*tree.SpecTree{
			{
				Path: "spec/api",
				Context: &spec.Context{
					Name: "api",
					Scenarios: []spec.Scenario{
						{ID: "login", Name: "Login", Run: &spec.RunBlock{Command: "api_login_cmd", Timeout: "5s"}},
						{ID: "logout", Name: "Logout", Run: &spec.RunBlock{Command: "api_logout_cmd", Timeout: "5s"}},
					},
				},
			},
			{
				Path: "spec/ui",
				Context: &spec.Context{
					Name: "ui",
					Scenarios: []spec.Scenario{
						{ID: "home", Name: "Home", Run: &spec.RunBlock{Command: "ui_home_cmd", Timeout: "5s"}},
					},
				},
			},
		},
	}
	fakeExecutor := &FakeExecutor{}
	sink := &SpySink{}
	runner := NewRunner(fakeExecutor, sink)
	runner.Filter = "spec/api/*"

	runner.Run(specTree)

	require.Len(t, fakeExecutor.Commands, 2)
	assert.Equal(t, "api_login_cmd", fakeExecutor.Commands[0].Command)
	assert.Equal(t, "api_logout_cmd", fakeExecutor.Commands[1].Command)
}

func TestRunner_ScenarioOutputNoDoubleSlash(t *testing.T) {
	specTree := &tree.SpecTree{
		Path: "/tmp/test",
		Context: &spec.Context{
			Name: "test",
			Scenarios: []spec.Scenario{
				{
					ID:   "scenario",
					Name: "Test scenario",
					Run:  &spec.RunBlock{Command: "test_cmd", Timeout: "5s"},
				},
			},
		},
	}

	executor, _ := runSpecWithID(t, "run-1", specTree)

	require.Len(t, executor.Commands, 1)
	scenarioOutput := executor.Commands[0].Env["SCENARIO_OUTPUT"]
	assert.NotContains(t, scenarioOutput, "//")
}

func TestRunner_ChildContext_InheritsParentEnv(t *testing.T) {
	specTree := &tree.SpecTree{
		Path: "parent",
		Context: &spec.Context{
			Name: "parent",
			Env:  map[string]string{"MY_VAR": "from_parent"},
		},
		Children: []*tree.SpecTree{
			{
				Path: "parent/child",
				Context: &spec.Context{
					Name: "child",
					Scenarios: []spec.Scenario{
						{
							ID:   "scenario",
							Name: "Child scenario",
							Run:  &spec.RunBlock{Command: "echo ${MY_VAR}", Timeout: "5s"},
						},
					},
				},
			},
		},
	}

	executor, _ := runSpec(t, specTree)

	require.Len(t, executor.Commands, 1)
	assert.Equal(t, "echo from_parent", executor.Commands[0].Command)
}
