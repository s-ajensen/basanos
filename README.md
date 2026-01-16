# Basanos

> **basanos** (βάσανος) — *noun, Ancient Greek*
>
> A touchstone; a dark stone used to test the purity of gold or silver by the streak left on it when rubbed with the metal.

---

An acceptance test framework for agentic orchestration. Specs are YAML. Execution is deterministic. Output is observable.

## Installation

```bash
# From source
git clone https://github.com/yourname/basanos
cd basanos
make build

# Binaries are in ./bin
./bin/basanos -h
```

## Quick Start

Create a spec directory with a `context.yaml`:

```yaml
# spec/context.yaml
name: "My Application"
description: "Acceptance tests for my application"

scenarios:
  - id: health_check
    name: "Health endpoint returns 200"
    run:
      command: curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health
      timeout: 5s
    assertions:
      - command: assert_equals 200 ${SCENARIO_OUTPUT}/_run/stdout
```

Run the specs:

```bash
basanos -s spec
```

Or try the included examples:

```bash
make build
basanos -s examples/spec
```

## Spec Structure

The directory tree is the spec tree:

```
spec/
  context.yaml           # Root context
  api/
    context.yaml         # Nested context
    users/
      context.yaml       # Deeper nesting
      expected.fixture   # Test fixtures
  ui/
    context.yaml
```

### context.yaml Schema

```yaml
name: "Context name"
description: "What this context tests"

# Environment variables (inherited and merged down the tree)
env:
  PORT: "8080"
  API_URL: "http://localhost:${PORT}"

# Failure handling: skip_children | continue | abort_run
on_failure: skip_children

# Lifecycle hooks
before:
  run: ./start-server.sh
  timeout: 10s

after:
  run: ./stop-server.sh
  timeout: 5s

before_each:
  run: ./reset-database.sh
  timeout: 3s

after_each:
  run: ./cleanup.sh
  timeout: 2s

# Test scenarios
scenarios:
  - id: unique_id
    name: "Human readable name"
    
    # Leaf scenarios can have their own before/after
    before:
      run: ./setup-test-data.sh
      timeout: 5s
    
    run:
      command: curl -s http://localhost:${PORT}/api/endpoint
      timeout: 30s
    
    assertions:
      - command: assert_equals expected.fixture ${SCENARIO_OUTPUT}/_run/stdout
      - command: assert_equals 0 ${SCENARIO_OUTPUT}/_run/exit_code
    
    after:
      run: ./cleanup-test-data.sh
      timeout: 5s

  # Scenarios can nest into groups
  - id: group_id
    name: "Grouped scenarios"
    
    # Groups can have before_each/after_each (run for each descendant leaf)
    before_each:
      run: ./reset-state.sh
      timeout: 3s
    
    scenarios:
      - id: nested_scenario
        run:
          command: echo "nested"
          timeout: 5s
        assertions:
          - command: assert_contains "nested" ${SCENARIO_OUTPUT}/_run/stdout
```

### Lifecycle Hooks

| Hook | Applies To | When It Runs |
|------|------------|--------------|
| `before` | Contexts, groups, leaves | Once when entering this node |
| `after` | Contexts, groups, leaves | Once when exiting this node |
| `before_each` | Contexts, groups only | Before each descendant leaf |
| `after_each` | Contexts, groups only | After each descendant leaf |

### Lifecycle Execution Order

For a leaf scenario, hooks execute in this order:

1. Ancestor `before` hooks (root to leaf)
2. Ancestor `before_each` hooks (root to leaf)
3. Scenario `before` hook
4. Scenario `run` command
5. Scenario `assertions`
6. Scenario `after` hook
7. Ancestor `after_each` hooks (leaf to root)
8. Ancestor `after` hooks (leaf to root)

### Variables

| Variable | Scope | Description |
|----------|-------|-------------|
| `${SPEC_ROOT}` | All | Root of spec directory |
| `${CONTEXT_OUTPUT}` | Context hooks | Output directory for current context |
| `${SCENARIO_OUTPUT}` | Scenario | Output directory for current scenario |
| Custom `env` vars | Inherited | Merged down the tree, child overrides parent |

## CLI Usage

```bash
# Run all specs (default output: files)
basanos

# Specify spec directory
basanos -s ./my-specs

# Output formats
basanos -o files              # Write to runs/ directory (default)
basanos -o files:./output     # Write to custom directory
basanos -o json               # NDJSON to stdout

# Multiple outputs
basanos -o files -o json

# Filter by path pattern
basanos -f "api/*"
basanos -f "api/users/login"

# Help
basanos -h
basanos -v
```

## Output

### File Sink

The `files` sink writes structured output to disk:

```
runs/
  2026-01-15_143022/
    api/
      _before/
        stdout
        stderr
        exit_code
      users/
        login/
          _before_each/
            api/
              stdout, stderr, exit_code
          _run/
            stdout, stderr, exit_code
          _assertions/
            0_assert_equals/
              stdout, stderr, exit_code
        _after/
          stdout, stderr, exit_code
```

### JSON Stream Sink

The `json` sink emits NDJSON events to stdout:

```json
{"event":"run_start","run_id":"2026-01-15_143022","timestamp":"..."}
{"event":"context_enter","run_id":"...","path":"api","name":"API Tests"}
{"event":"scenario_enter","run_id":"...","path":"api/login","name":"Login works"}
{"event":"hook_start","run_id":"...","path":"api/login","hook":"run"}
{"event":"output","run_id":"...","stream":"stdout","data":"..."}
{"event":"hook_end","run_id":"...","path":"api/login","hook":"run","exit_code":0}
{"event":"assertion_start","run_id":"...","path":"api/login","index":0,"command":"assert_equals ..."}
{"event":"assertion_end","run_id":"...","path":"api/login","index":0,"exit_code":0}
{"event":"scenario_exit","run_id":"...","path":"api/login","status":"pass"}
{"event":"context_exit","run_id":"...","path":"api"}
{"event":"run_end","run_id":"...","status":"pass","passed":5,"failed":0}
```

### Event Schema

Generate the JSON Schema for event types:

```bash
make schema
# Output: schema/events.json
```

## Assertion Executables

Standalone binaries for use in specs:

```bash
# Equality
assert_equals expected.txt actual.txt
assert_equals "expected string" actual.txt

# Containment
assert_contains needle.txt haystack.txt
assert_contains "search term" haystack.txt

# Pattern matching
assert_matches "regex.*pattern" target.txt

# Numeric comparisons
assert_gt 10 5      # 10 > 5
assert_gte 10 10    # 10 >= 10
assert_lt 5 10      # 5 < 10
assert_lte 10 10    # 10 <= 10
```

Each assertion:
- Exits 0 on pass, non-zero on fail
- Outputs human-readable comparison info
- Auto-detects file vs literal arguments

## Failure Modes

Configure via `on_failure`:

| Mode | Behavior |
|------|----------|
| `skip_children` | Skip remaining scenarios in context, continue siblings |
| `continue` | Log failure, continue executing |
| `abort_run` | Stop entire test run immediately |

## Design Principles

**Spec generation is agentic; execution is deterministic.**

The framework provides:
- Spec schema and directory conventions
- Event-emitting test runner
- Pluggable output sinks
- Assertion executables

Agents generate:
- Spec content (YAML files)
- Fixture files
- Helper scripts

No agent involvement at runtime. Execution is predictable and reproducible.

## Development

```bash
# Build all binaries
make build

# Run tests
make test

# Generate event schema
make schema

# Clean build artifacts
make clean
```

## License

MIT
