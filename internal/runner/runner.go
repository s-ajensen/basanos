package runner

import (
	"errors"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"basanos/internal/assert"
	eventpkg "basanos/internal/event"
	"basanos/internal/executor"
	sinkpkg "basanos/internal/sink"
	"basanos/internal/spec"
	"basanos/internal/tree"
)

func substituteVars(command string, env map[string]string) string {
	return os.Expand(command, func(key string) string {
		if value, ok := env[key]; ok {
			return value
		}
		return "${" + key + "}"
	})
}

type runContext struct {
	runID           string
	beforeEachHooks []*spec.Hook
	afterEachHooks  []*spec.Hook
	onFailure       string
	env             map[string]string
	specRoot        string
	outputRoot      string
}

type Runner struct {
	executor executor.Executor
	sinks    []sinkpkg.Sink
	passed   int
	failed   int
	aborted  bool
	runID    string
	Filter   string
}

func NewRunner(exec executor.Executor, sinks ...sinkpkg.Sink) *Runner {
	return &Runner{
		executor: exec,
		sinks:    sinks,
	}
}

func (runner *Runner) Passed() int {
	return runner.passed
}

func (runner *Runner) Failed() int {
	return runner.failed
}

func (runner *Runner) emit(event any) {
	for _, sink := range runner.sinks {
		sink.Emit(event)
	}
}

func (runner *Runner) emitOutput(stream, data string) {
	if data != "" {
		runner.emit(eventpkg.NewOutputEvent(runner.runID, stream, data))
	}
}

func (runner *Runner) exec(command, timeout string, env map[string]string) (int, bool) {
	_, _, exitCode, timedOut := runner.execCapture(command, timeout, env)
	return exitCode, timedOut
}

func (runner *Runner) execCapture(command, timeout string, env map[string]string) (string, string, int, bool) {
	expandedCommand := substituteVars(command, env)
	stdout, stderr, exitCode, err := runner.executor.Execute(expandedCommand, timeout, env)
	runner.emitOutput("stdout", stdout)
	runner.emitOutput("stderr", stderr)
	return stdout, stderr, exitCode, errors.Is(err, executor.ErrTimeout)
}

func (runner *Runner) runHook(path, hookName string, hook *spec.Hook, env map[string]string) {
	if hook == nil {
		return
	}
	runner.emit(eventpkg.NewHookStartEvent(runner.runID, path, "_"+hookName, ""))
	exitCode, _ := runner.exec(hook.Run, hook.Timeout, env)
	runner.emit(eventpkg.NewHookEndEvent(runner.runID, path, "_"+hookName, "", exitCode))
}

func (runner *Runner) runHooks(path, hookName string, hooks []*spec.Hook, env map[string]string) {
	for _, hook := range hooks {
		runner.runHook(path, hookName, hook, env)
	}
}

func reversed(hooks []*spec.Hook) []*spec.Hook {
	result := slices.Clone(hooks)
	slices.Reverse(result)
	return result
}

func extractExecutable(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}
	return parts[0]
}

func (runner *Runner) runAssertions(path string, assertions []spec.Assertion, env map[string]string, captured CapturedOutput) bool {
	allPassed := true
	for index, assertion := range assertions {
		executable := extractExecutable(assertion.Command)

		first, second, err := resolveAssertionArgs(assertion.Command, captured, env)
		if err != nil {
			runner.emit(eventpkg.NewAssertionStartEvent(runner.runID, path, index, assertion.Command))
			runner.emit(eventpkg.NewAssertionEndEvent(runner.runID, path, index, 1))
			allPassed = false
			continue
		}

		protocol := assert.BuildProtocol(first, second)

		runner.emit(eventpkg.NewAssertionStartEvent(runner.runID, path, index, assertion.Command))
		stdout, stderr, exitCode, _ := runner.executor.ExecuteWithStdin(executable, assertion.Timeout, env, protocol)
		runner.emitOutput("stdout", stdout)
		runner.emitOutput("stderr", stderr)
		runner.emit(eventpkg.NewAssertionEndEvent(runner.runID, path, index, exitCode))

		if exitCode != 0 {
			allPassed = false
		}
	}
	return allPassed
}

func (runner *Runner) runScenario(scenarioPath string, scenario spec.Scenario, ctx runContext) bool {
	scenarioOutput := path.Join(ctx.outputRoot, scenarioPath)
	scenarioEnv := mergeEnv(ctx.env, map[string]string{
		"SCENARIO_OUTPUT": scenarioOutput,
		"RUN_OUTPUT":      path.Join(scenarioOutput, "_run"),
	})

	runner.emit(eventpkg.NewScenarioEnterEvent(runner.runID, scenarioPath, scenario.Name, time.Now()))

	runner.runHooks(scenarioPath, "before_each", ctx.beforeEachHooks, scenarioEnv)
	runner.runHook(scenarioPath, "before", scenario.Before, scenarioEnv)

	runner.emit(eventpkg.NewScenarioRunStartEvent(runner.runID, scenarioPath))
	stdout, stderr, exitCode, timedOut := runner.execCapture(scenario.Run.Command, scenario.Run.Timeout, scenarioEnv)
	if timedOut {
		runner.emit(eventpkg.NewTimeoutEvent(runner.runID, scenarioPath, "run", scenario.Run.Timeout))
	}
	runner.emit(eventpkg.NewScenarioRunEndEvent(runner.runID, scenarioPath, exitCode))

	captured := CapturedOutput{Stdout: stdout, Stderr: stderr, ExitCode: exitCode}
	assertionsPassed := runner.runAssertions(scenarioPath, scenario.Assertions, scenarioEnv, captured)
	passed := assertionsPassed && !timedOut

	status := "fail"
	if passed {
		status = "pass"
	}
	runner.emit(eventpkg.NewScenarioExitEvent(runner.runID, scenarioPath, status, time.Now()))

	if passed {
		runner.passed++
	} else {
		runner.failed++
	}

	runner.runHook(scenarioPath, "after", scenario.After, scenarioEnv)
	runner.runHooks(scenarioPath, "after_each", reversed(ctx.afterEachHooks), scenarioEnv)

	return passed
}

func (runner *Runner) shouldStopAfterFailure(passed bool, onFailure string) bool {
	if passed {
		return false
	}
	if onFailure == "abort_run" {
		runner.aborted = true
		return true
	}
	return onFailure == "skip_children"
}

func (runner *Runner) matchesFilter(scenarioPath string) bool {
	if runner.Filter == "" {
		return true
	}
	matched, err := path.Match(runner.Filter, scenarioPath)
	if err != nil {
		return scenarioPath == runner.Filter
	}
	return matched
}

func (runner *Runner) executeLeaf(path string, scenario spec.Scenario, ctx runContext) bool {
	if scenario.Run == nil {
		return false
	}
	if !runner.matchesFilter(path) {
		return false
	}
	passed := runner.runScenario(path, scenario, ctx)
	return runner.shouldStopAfterFailure(passed, ctx.onFailure)
}

func (runner *Runner) runScenarios(basePath string, scenarios []spec.Scenario, ctx runContext) {
	for _, scenario := range scenarios {
		if runner.aborted {
			return
		}
		path := basePath + "/" + scenario.ID

		if runner.executeLeaf(path, scenario, ctx) {
			return
		}
		runner.runChildScenarios(path, scenario, ctx)
	}
}

func mergeEnv(parent, child map[string]string) map[string]string {
	result := make(map[string]string)
	for key, value := range parent {
		result[key] = value
	}
	for key, value := range child {
		result[key] = value
	}
	return result
}

func (runner *Runner) runChildScenarios(path string, scenario spec.Scenario, ctx runContext) {
	if len(scenario.Scenarios) == 0 {
		return
	}
	childCtx := runContext{
		beforeEachHooks: append(ctx.beforeEachHooks, scenario.BeforeEach),
		afterEachHooks:  append(ctx.afterEachHooks, scenario.AfterEach),
		onFailure:       ctx.onFailure,
		env:             mergeEnv(ctx.env, scenario.Env),
		specRoot:        ctx.specRoot,
		outputRoot:      ctx.outputRoot,
	}
	runner.runScenarios(path, scenario.Scenarios, childCtx)
}

func (runner *Runner) runTree(specTree *tree.SpecTree, specRoot string, outputRoot string, parentEnv map[string]string) error {
	if runner.aborted {
		return nil
	}

	contextOutput := outputRoot + "/" + specTree.Path
	env := mergeEnv(parentEnv, mergeEnv(specTree.Context.Env, map[string]string{
		"SPEC_ROOT":      specRoot,
		"CONTEXT_OUTPUT": contextOutput,
	}))

	runner.emit(eventpkg.NewContextEnterEvent(runner.runID, specTree.Path, specTree.Context.Name, time.Now()))

	runner.runHook(specTree.Path, "before", specTree.Context.Before, env)

	ctx := runContext{
		runID:           runner.runID,
		beforeEachHooks: []*spec.Hook{specTree.Context.BeforeEach},
		afterEachHooks:  []*spec.Hook{specTree.Context.AfterEach},
		onFailure:       specTree.Context.OnFailure,
		env:             env,
		specRoot:        specRoot,
		outputRoot:      outputRoot,
	}
	runner.runScenarios(specTree.Path, specTree.Context.Scenarios, ctx)

	for _, child := range specTree.Children {
		runner.runTree(child, ctx.specRoot, ctx.outputRoot, env)
	}

	runner.runHook(specTree.Path, "after", specTree.Context.After, env)

	runner.emit(eventpkg.NewContextExitEvent(runner.runID, specTree.Path, time.Now()))

	return nil
}

func (runner *Runner) Run(specTree *tree.SpecTree) error {
	return runner.runTree(specTree, specTree.Path, "", nil)
}

func (runner *Runner) RunWithID(runID string, specTree *tree.SpecTree) error {
	runner.runID = runID
	runner.emit(eventpkg.NewRunStartEvent(runID, time.Now()))
	runner.passed = 0
	runner.failed = 0
	runner.aborted = false

	outputRoot := "runs/" + runID
	err := runner.runTree(specTree, specTree.Path, outputRoot, nil)

	status := "pass"
	if runner.failed > 0 {
		status = "fail"
	}

	runner.emit(eventpkg.NewRunEndEvent(runID, status, runner.passed, runner.failed, time.Now()))

	return err
}
