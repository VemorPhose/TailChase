# Testing and Development

Use this guide to verify the MVP during development.

## Checks

Run from the Tailchase repository root:

```bash
go test ./...
go vet ./...
go test -race ./...
go test -coverpkg=./... ./...
go build -o /tmp/tailchase ./cmd/tailchase
/tmp/tailchase version
```

Expected version:

```text
0.1.2
```

## Test Layout

Most tests live in `tests/` and exercise exported package behavior. The collector keeps one package-local white-box test in `internal/collect` so the fake GitHub Actions client does not force a test-only public interface.

Current layout:

```text
tests/
  bundle_test.go
  cli_test.go
  github_test.go
  helpers_test.go
  project_test.go
  prompt_test.go
internal/collect/
  github_actions_test.go
```

## Local Smoke Test

This verifies `init -> bundle -> prompt` without calling GitHub.

```bash
go build -o /tmp/tailchase ./cmd/tailchase

SMOKE_DIR="$(mktemp -d)"
cd "$SMOKE_DIR"

/tmp/tailchase init

cat > .tailchase/goal.yml <<'YAML'
goal: Fix the failing CI compile error.
non_goals:
  - Do not weaken tests.
  - Do not broaden the change beyond the failing compile error.
must_preserve:
  - Existing public behavior.
done_conditions:
  - go test ./... passes
suspicious_paths:
  - .github/workflows
YAML

mkdir -p .tailchase/runs/12345/evidence
cat > .tailchase/runs/12345/evidence/github-actions.log <<'LOG'
# Tailchase GitHub Actions evidence
repository: owner/repo
run_id: 12345
collected_at: 2026-06-21T00:00:00Z
failed_jobs_only: true

--- tailchase-job id=11 name="unit tests" status="completed" conclusion="failure" html_url="https://github.com/owner/repo/actions/runs/12345/job/11" ---
internal/app/app.go:42:10: undefined: Handler
--- FAIL: TestHandler
panic: missing required environment variable API_TOKEN
--- tailchase-end-job id=11 ---
LOG

/tmp/tailchase bundle --run 12345
/tmp/tailchase prompt --run 12345
```

Expected artifacts:

```text
.tailchase/config.yml
.tailchase/goal.yml
.tailchase/runs/12345/run.yml
.tailchase/runs/12345/evidence/github-actions.log
.tailchase/runs/12345/normalized-evidence.yml
.tailchase/runs/12345/failure-bundle.yml
.tailchase/runs/12345/repair-prompt.md
```

Quick assertions:

```bash
grep -n "undefined: Handler" .tailchase/runs/12345/failure-bundle.yml
grep -n "Fix the failing CI compile error" .tailchase/runs/12345/repair-prompt.md
grep -n "go test ./..." .tailchase/runs/12345/repair-prompt.md
```

## Live Collector Test

Use a real GitHub Actions run ID:

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase init
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase bundle --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id>
```

Expected behavior:

- failed jobs are written to `.tailchase/runs/<run-id>/evidence/github-actions.log`
- successful jobs are skipped when `failed_jobs_only: true`
- job logs are capped by `max_log_lines_per_job`
- missing credentials produce a warning; private repositories still require a token

## Troubleshooting

- `bundle` fails: confirm `.tailchase/runs/<run-id>/evidence/github-actions.log` exists.
- `prompt` fails: confirm `.tailchase/runs/<run-id>/failure-bundle.yml` exists.
- `collect` cannot find the repository: pass `--repo owner/name` or set `github.repo`.
- GitHub log download fails: set `GITHUB_TOKEN` or `GH_TOKEN`.
- Prompt is too generic: improve `.tailchase/goal.yml`.
