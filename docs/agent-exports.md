# Agent Exports

TailChase writes portable markdown for the coding agent you already use.

## Supported Targets

```bash
tailchase export --run <run-id> --target codex
tailchase export --run <run-id> --target claude-code
tailchase export --run <run-id> --target copilot
```

`tailchase prepare` can generate exports in one pass:

```bash
tailchase prepare --run <run-id> --export codex --export claude-code
```

## Output Files

```text
.tailchase/runs/<run-id>/exports/codex-prompt.md
.tailchase/runs/<run-id>/exports/claude-code-prompt.md
.tailchase/runs/<run-id>/exports/copilot-instructions.md
```

Each export includes:

- the repair prompt
- local artifact paths
- source evidence paths
- safety findings when present

## Current Stability

Codex, Claude Code, and Copilot exports are alpha formats. They are designed to be readable markdown first, with adapter-specific polish improving over time.
