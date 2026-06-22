# MVP Flow

Tailchase MVP turns one GitHub Actions run into local artifacts and a repair prompt:

```text
run ID -> collect failed logs -> normalize evidence -> bundle failure -> render prompt
```

## 1. Initialize

```bash
tailchase init
```

Creates:

```text
.tailchase/config.yml
.tailchase/goal.yml
```

Edit `goal.yml` before generating prompts. It defines the task goal, non-goals, preserved behavior, done conditions, expected paths, suspicious paths, and stop rules.

## 2. Collect

```bash
tailchase collect --run 123456789 --repo owner/name
tailchase collect-local --run 123456789 --kind go_test --file go-test.log
```

Collects failed GitHub Actions jobs by default, caps each job log using `max_log_lines_per_job`, and writes:

```text
.tailchase/runs/123456789/evidence/github-actions.log
.tailchase/runs/123456789/evidence/go-test.log
.tailchase/runs/123456789/run.yml
```

Repository resolution order:

1. `--repo owner/name`
2. `.tailchase/config.yml` field `github.repo`
3. `git remote origin`

## 3. Bundle

```bash
tailchase bundle --run 123456789
```

Reads the raw evidence log, extracts likely failure signals, checks the goal contract, and writes:

```text
.tailchase/runs/123456789/normalized-evidence.yml
.tailchase/runs/123456789/failure-bundle.yml
```

## 4. Prompt

```bash
tailchase prompt --run 123456789
tailchase prompt --run 123456789 --delta
tailchase export --run 123456789 --target codex
```

Reads `failure-bundle.yml`, renders a heuristic repair prompt by default, and writes:

```text
.tailchase/runs/123456789/repair-prompt.md
```

With `prompt_target: stdout`, the prompt is also printed for immediate copy/paste.

Use `--delta` after prior attempts exist to summarize repeated root errors, highlight new evidence, preserve the goal contract, and keep raw artifact links available.

Set `prompt.mode: model` with OpenAI-compatible provider settings to generate the prompt through a model. Model mode still writes `repair-prompt.md` and also records `.tailchase/runs/<run-id>/model-metadata.yml`.

Use `export` to write target-specific prompt files for Codex, Claude Code, or Copilot without live steering. Exports are stored under `.tailchase/runs/<run-id>/exports/`.
