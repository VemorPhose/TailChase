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
safety:
  mode: manual
  stop_on:
    - test_weakening
    - suspicious_path_edit
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
expected_paths:
  - internal/app
suspicious_paths:
  - .github/workflows
stop_rules:
  - Stop before weakening tests.
  - Stop before changing behavior outside the task.
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

## `run.yml`

Indexes local artifacts for one run.

```yaml
version: 1
id: "12345"
created_at: "2026-06-22T10:00:00Z"
artifacts:
  - name: github_actions_log
    type: github_actions
    path: .tailchase/runs/12345/evidence/github-actions.log
    created_at: "2026-06-22T10:00:00Z"
```

## `attempt-history.yml`

Records repair attempts for a run.

```yaml
version: 1
attempts:
  - number: 1
    run_id: "12345"
    bundle_path: .tailchase/runs/12345/failure-bundle.yml
    prompt_path: .tailchase/runs/12345/repair-prompt.md
    root_error_candidates:
      - "undefined: Handler"
    outcome: failed
    created_at: "2026-06-22T10:00:00Z"
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
  expected_paths:
    - internal/app
  suspicious_paths:
    - .github/workflows
  stop_rules:
    - Stop before weakening tests.
attempt_context:
  same_root_error_seen_before: true
  matching_attempt_numbers:
    - 1
budget:
  raw_evidence_bytes: 24576
  included_excerpt_bytes: 1024
  repeated_blocks_collapsed: 3
  estimated_prompt_bytes: 4096
safety_findings:
  - rule: goal_drift
    decision: warn
    message: failure signal "cmd/main.go" is outside expected_paths
    path: cmd/main.go
root_error_candidates:
  - type: file_error
    source: github_actions
    message: "undefined: Handler"
    confidence: high
artifacts:
  - name: github_actions_log
    path: .tailchase/runs/12345/evidence/github-actions.log
```
