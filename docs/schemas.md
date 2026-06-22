# Tailchase Schemas

Tailchase stores local YAML artifacts with explicit schema versions. Version `1` is the current schema for all MVP files.

## Compatibility

- Missing `version` means version `1` for MVP compatibility.
- Unsupported nonzero versions fail during load.
- Raw evidence files are not versioned; generated YAML artifacts are.

## `.tailchase/config.yml`

Controls collection and prompt output.

```yaml
version: 1
collectors:
  - github_actions
github:
  repo: owner/repo
failed_jobs_only: true
max_log_lines_per_job: 1200
prompt_target: stdout
prompt_size_limit: 12000
```

## `.tailchase/goal.yml`

Anchors prompts and checks to the original task.

```yaml
version: 1
goal: Fix the failing GitHub Actions run.
non_goals:
  - Do not weaken tests.
must_preserve:
  - Existing public behavior.
done_conditions:
  - Relevant tests pass locally.
suspicious_paths:
  - .github/workflows
```

## `normalized-evidence.yml`

Stores extracted signals from raw evidence.

```yaml
version: 1
generated_at: "2026-06-22T10:00:00Z"
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
sources:
  - source: github_actions
    path: .tailchase/runs/12345/evidence/github-actions.log
signals:
  - type: file_error
    source: github_actions
    job: unit tests
    message: "undefined: Handler"
    file: internal/app/app.go
    line: 42
    confidence: high
```

## `failure-bundle.yml`

Stores the portable repair context consumed by prompt generation.

```yaml
version: 1
generated_at: "2026-06-22T10:00:00Z"
run:
  source: github_actions
  repository: owner/repo
  run_id: "12345"
goal:
  goal: Fix CI
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    confidence: high
artifacts:
  - name: github_actions_log
    path: .tailchase/runs/12345/evidence/github-actions.log
```
