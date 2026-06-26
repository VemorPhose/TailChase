# Quickstart

This is the supported first alpha path.

## 1. Initialize TailChase

Run this inside the repository whose failure you want to inspect:

```bash
tailchase init
```

Edit `.tailchase/goal.yml` so the generated repair prompt has the real goal, non-goals, expected paths, and stop rules.

## 2. Push and Wait for CI

```bash
git push
tailchase ci watch --export codex
```

If the current branch's GitHub Actions run fails, TailChase writes:

```text
.tailchase/runs/<run-id>/failure-bundle.yml
.tailchase/runs/<run-id>/repair-prompt.md
.tailchase/runs/<run-id>/exports/codex-prompt.md
.tailchase/runs/<run-id>/report.md
```

## 3. Use the Repair Context

Open the generated export for your coding agent. The file links back to the local failure bundle and raw evidence artifacts, so you can inspect exactly what the agent was given.

For an already-known failed run:

```bash
tailchase collect --run <run-id> --repo owner/name
tailchase prepare --run <run-id> --export claude-code
```

For repeated repair attempts:

```bash
tailchase prompt --run <run-id> --delta
tailchase cost report --run <run-id>
```
