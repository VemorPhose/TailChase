# Tailchase MVP Flow

Tailchase MVP is intentionally small:

```text
GitHub Actions run ID
-> collect failed job logs
-> store raw evidence locally
-> extract failure signals
-> write failure-bundle.yml
-> render repair-prompt.md
```

## 1. Initialize

```bash
tailchase init
```

This creates `.tailchase/config.yml` and `.tailchase/goal.yml`.

Update `goal.yml` before generating prompts. The repair prompt is anchored to:

- `goal`
- `non_goals`
- `must_preserve`
- `done_conditions`
- `suspicious_paths`

## 2. Collect

```bash
tailchase collect --run 123456789 --repo owner/name
```

Tailchase fetches GitHub Actions jobs for the run, keeps failed jobs by default, downloads each failed job log, caps each job log according to config, and writes:

```text
.tailchase/runs/123456789/evidence/github-actions.log
```

## 3. Bundle

```bash
tailchase bundle --run 123456789
```

This writes:

```text
.tailchase/runs/123456789/normalized-evidence.yml
.tailchase/runs/123456789/failure-bundle.yml
```

## 4. Prompt

```bash
tailchase prompt --run 123456789
```

This renders a paste-ready repair prompt to stdout and writes:

```text
.tailchase/runs/123456789/repair-prompt.md
```
