# Testing and Development

This guide covers the Tailchase MVP as built today:

```text
init -> collect -> bundle -> prompt
```

The deterministic local smoke test exercises everything after GitHub log collection. The live collector test exercises GitHub Actions access with a real run ID.

## Prerequisites

- Go installed
- A shell in the repository root
- For live collection only: a GitHub Actions run ID and repository
- For private repositories or higher rate limits: `GITHUB_TOKEN` or `GH_TOKEN`

## Standard Checks

Run these before committing:

```bash
go test ./...
go vet ./...
go build -o /tmp/tailchase ./cmd/tailchase
/tmp/tailchase version
```

Expected version output:

```text
0.1.0
```

## Test Layout

Most tests live in the top-level `tests/` directory and exercise exported package behavior. The collector keeps one package-local white-box test in `internal/collect` because it uses a fake GitHub Actions client without making the production collector interface public just for tests.

## Local MVP Smoke Test

This smoke test does not call GitHub. It creates a temporary project, injects a sample GitHub Actions evidence log, then verifies bundle and prompt generation.

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
.tailchase/runs/12345/evidence/github-actions.log
.tailchase/runs/12345/normalized-evidence.yml
.tailchase/runs/12345/failure-bundle.yml
.tailchase/runs/12345/repair-prompt.md
```

Useful checks:

```bash
grep -n "undefined: Handler" .tailchase/runs/12345/failure-bundle.yml
grep -n "Fix the failing CI compile error" .tailchase/runs/12345/repair-prompt.md
grep -n "go test ./..." .tailchase/runs/12345/repair-prompt.md
```

## Live GitHub Actions Collector Test

From a real project with GitHub Actions:

```bash
tailchase init
```

Edit `.tailchase/goal.yml` so the prompt has the actual task goal.

Then run:

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase bundle --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id>
```

Expected collector behavior:

- failed jobs are written into `.tailchase/runs/<run-id>/evidence/github-actions.log`
- successful jobs are skipped when `failed_jobs_only: true`
- each job log is capped by `max_log_lines_per_job`
- missing credentials produce a warning, not an immediate local validation error

## Prompt Output Modes

`prompt_target` in `.tailchase/config.yml` controls command output:

```yaml
prompt_target: stdout
```

This prints the full prompt to stdout and writes `repair-prompt.md`.

```yaml
prompt_target: file
```

This writes `repair-prompt.md` and prints only the file path.

## Debugging Notes

- If `bundle` fails, check that `.tailchase/runs/<run-id>/evidence/github-actions.log` exists.
- If `prompt` fails, check that `.tailchase/runs/<run-id>/failure-bundle.yml` exists.
- If `collect` cannot find the repository, pass `--repo owner/name` or set `github.repo` in `.tailchase/config.yml`.
- If GitHub log downloads fail for a private repository, set `GITHUB_TOKEN` or `GH_TOKEN`.
- If a prompt feels unanchored, update `.tailchase/goal.yml`; the MVP intentionally depends on that local contract.
