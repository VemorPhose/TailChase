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

Edit `goal.yml` before generating prompts. It defines the task goal, non-goals, preserved behavior, done conditions, and suspicious paths.

## 2. Collect

```bash
tailchase collect --run 123456789 --repo owner/name
```

Collects failed GitHub Actions jobs by default, caps each job log using `max_log_lines_per_job`, and writes:

```text
.tailchase/runs/123456789/evidence/github-actions.log
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
```

Reads `failure-bundle.yml`, renders a heuristic repair prompt, and writes:

```text
.tailchase/runs/123456789/repair-prompt.md
```

With `prompt_target: stdout`, the prompt is also printed for immediate copy/paste.

Use `--delta` after prior attempts exist to summarize repeated root errors, highlight new evidence, preserve the goal contract, and keep raw artifact links available.
