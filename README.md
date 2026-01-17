# Basanos

[![CI](https://github.com/s-ajensen/basanos/actions/workflows/ci.yml/badge.svg)](https://github.com/s-ajensen/basanos/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/s-ajensen/basanos)](https://goreportcard.com/report/github.com/s-ajensen/basanos)

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
make install

# Now available system-wide
basanos -h
```

By default, binaries install to `/usr/local/bin` (may require `sudo`). To install elsewhere:

```bash
# User-local install (no sudo needed)
make install PREFIX=~/.local

# Custom location
make install PREFIX=/opt/basanos
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
      - command: assert_equals "200" ${RUN_OUTPUT}/stdout
```

Run the specs:

```bash
basanos -s spec
```

## Self-Test Suite

Basanos tests itself. Run the acceptance specs for the framework:

```bash
# Run the full test suite
basanos -s spec

# With verbose output
basanos -s spec --verbose
```

This builds the binaries, runs 90+ scenarios covering CLI, assertions, runner lifecycle, output sinks, and validation.

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
      - command: assert_equals expected.fixture ${RUN_OUTPUT}/stdout
      - command: assert_equals 0 ${RUN_OUTPUT}/exit_code
    
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
          - command: assert_contains "nested" ${RUN_OUTPUT}/stdout
```

### Lifecycle Hooks

| Hook | Applies To | When It Runs |
|------|------------|--------------|
| `before` | Contexts, groups, leaves | Once when entering this node |
| `after` | Contexts, groups, leaves | Once when exiting this node (after all children) |
| `before_each` | Contexts, groups only | Before each descendant leaf |
| `after_each` | Contexts, groups only | After each descendant leaf |

### Lifecycle Execution Order

For a leaf scenario, hooks execute in this order:

1. Ancestor `before` hooks (root to leaf, once per context on entry)
2. Ancestor `before_each` hooks (root to leaf)
3. Scenario `before` hook
4. Scenario `run` command
5. Scenario `assertions`
6. Scenario `after` hook
7. Ancestor `after_each` hooks (leaf to root)
8. Ancestor `after` hooks run when exiting each context (after all children complete)

### Variables

| Variable | Scope | Description |
|----------|-------|-------------|
| `${SPEC_ROOT}` | All | Root of spec directory |
| `${CONTEXT_OUTPUT}` | Context hooks | Output directory for current context |
| `${SCENARIO_OUTPUT}` | Scenario | Output directory for current scenario |
| `${RUN_OUTPUT}` | Scenario | Shorthand for `${SCENARIO_OUTPUT}/_run` |
| Custom `env` vars | Inherited | Merged down the tree, child overrides parent |

## CLI Usage

```bash
# Run specs (default output: cli)
basanos -s ./spec

# Output formats
basanos -o cli                # Pretty terminal output (default)
basanos -o json               # NDJSON to stdout
basanos -o files              # Write to runs/ directory
basanos -o files:./output     # Write to custom directory
basanos -o junit              # JUnit XML to stdout

# Multiple outputs
basanos -o cli -o files
basanos -o json -o junit

# Filter by path pattern
basanos -f "api/*"
basanos -f "*/*/*/login"

# Verbose mode (show context/scenario names)
basanos --verbose

# Help and version
basanos -h
basanos -v
```

## Output

### CLI Reporter

The default `cli` output shows pass/fail indicators with a summary:

```
My Application
  Health check
    [32m✓[0m Returns 200 for valid request
    [31m✗[0m Handles missing auth header

2 passed, 1 failed
```

Use `--verbose` to see full context and scenario names with indentation.

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
            stdout
            stderr
            exit_code
          _before/
            stdout
            stderr
            exit_code
          _run/
            stdout
            stderr
            exit_code
          _assertions/
            0/
              stdout
              stderr
              exit_code
            1/
              stdout
              stderr
              exit_code
          _after/
            stdout
            stderr
            exit_code
      _after/
        stdout
        stderr
        exit_code
```

### JSON Stream Sink

The `json` sink emits NDJSON events to stdout:

```json
{"event":"run_start","run_id":"2026-01-15_143022","timestamp":"..."}
{"event":"context_enter","run_id":"...","path":"api","name":"API Tests","timestamp":"..."}
{"event":"scenario_enter","run_id":"...","path":"api/login","name":"Login works","timestamp":"..."}
{"event":"hook_start","run_id":"...","path":"api/login","hook":"_before_each"}
{"event":"output","run_id":"...","stream":"stdout","data":"..."}
{"event":"hook_end","run_id":"...","path":"api/login","hook":"_before_each","exit_code":0}
{"event":"run_start","run_id":"...","path":"api/login"}
{"event":"output","run_id":"...","stream":"stdout","data":"..."}
{"event":"run_end","run_id":"...","path":"api/login","exit_code":0}
{"event":"assertion_start","run_id":"...","path":"api/login","index":0,"command":"assert_equals ..."}
{"event":"assertion_end","run_id":"...","path":"api/login","index":0,"exit_code":0}
{"event":"scenario_exit","run_id":"...","path":"api/login","status":"pass","timestamp":"..."}
{"event":"context_exit","run_id":"...","path":"api","timestamp":"..."}
{"event":"run_end","run_id":"...","status":"pass","passed":5,"failed":0,"timestamp":"..."}
```

### JUnit Sink

The `junit` sink outputs JUnit XML format for CI integration.

## Assertion Executables

Standalone binaries for use in specs. All assertions auto-detect whether arguments are file paths or literal values.

```bash
# Equality (both args can be files or literals)
assert_equals expected.txt actual.txt
assert_equals "expected string" actual.txt

# Containment
assert_contains needle.txt haystack.txt
assert_contains "search term" haystack.txt

# Pattern matching (regex)
assert_matches pattern.txt target.txt
assert_matches "regex.*pattern" target.txt

# Numeric comparisons (both args can be files or literals)
assert_gt value.txt threshold.txt    # value > threshold
assert_gte 10 10                     # 10 >= 10
assert_lt count.txt max.txt          # count < max
assert_lte 5 10                      # 5 <= 10
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

## Agent Workflow

Basanos includes an agent and skill for AI-assisted spec generation:

```
agents/
  spec-writer.md    # Agent: translates requirements into specs
skills/
  basanos.md        # Skill: spec format reference
```

### Using the Spec Writer Agent

The `spec-writer` agent helps you translate requirements into comprehensive test specifications. It probes for edge cases, error conditions, and completeness—then generates valid `context.yaml` files and fixtures.

To use it in your AI tool (Claude, Cursor, etc.):

1. **Load the agent** — Point your tool at `agents/spec-writer.md` or copy its contents as a system prompt
2. **Describe what you're testing** — Provide requirements, user stories, OpenAPI specs, or just describe the behavior
3. **Iterate** — The agent will ask clarifying questions, draft specs, and refine based on feedback

The agent will automatically load the `basanos` skill to ensure it generates valid specs.

### Using the Skill Directly

If you're writing specs yourself (or using a different agent), load the `basanos` skill for the complete format reference:

- `skills/basanos.md` — Schema, lifecycle hooks, variables, assertions, and common patterns

This is useful when you want the spec format reference without the full spec-writer workflow.

### IDE Integration

Most AI-enabled editors can load agents and skills from your project:

- **Claude Code / OpenCode** — Agents in `agents/` and skills in `skills/` are auto-discovered
- **Cursor** — Add to your `.cursorrules` or reference directly in chat
- **Other tools** — Copy the content into your system prompt or context

## Development

```bash
# Build all binaries
make build

# Run unit tests
make test

# Run acceptance tests (basanos testing itself)
basanos -s spec

# Install to system (default: /usr/local/bin)
make install
make install PREFIX=~/.local

# Uninstall
make uninstall
make uninstall PREFIX=~/.local

# Clean build artifacts
make clean
```

## License

MIT
