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
0.1.26
```

## CI

GitHub Actions runs the repository gates in `.github/workflows/ci.yml` on pushes to `main`, pull requests, and manual dispatch:

- checkout with `actions/checkout`
- Go setup/cache with `actions/setup-go`
- `go test ./...`
- `go vet ./...`
- `go test -race ./...`
- `go test -coverpkg=./... ./...`
- CLI build and version check
- local no-network CLI smoke test
- coverage and binary upload with `actions/upload-artifact`

## Test Layout

Most tests live in `tests/` and exercise exported package behavior. Collector fake-client tests stay in `internal/collect` so provider interfaces do not need test-only public wrappers.

Current layout:

```text
tests/
  adapter_test.go
  bundle_test.go
  cli_test.go
  comment_test.go
  export_test.go
  guard_test.go
  github_test.go
  helpers_test.go
  loop_test.go
  mcp_test.go
  project_test.go
  prompt_test.go
  model_test.go
  report_test.go
  steering_test.go
  tournament_test.go
  wrapper_test.go
internal/collect/
  github_actions_test.go
  gitlab_ci_test.go
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
expected_paths:
  - internal/app
suspicious_paths:
  - .github/workflows
stop_rules:
  - Stop before weakening tests.
  - Stop before changing behavior outside the failing compile error.
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
.github/workflows/ci.yml:2: unexpected workflow change
--- FAIL: TestHandler
panic: missing required environment variable API_TOKEN
--- tailchase-end-job id=11 ---
LOG

/tmp/tailchase bundle --run 12345
/tmp/tailchase prompt --run 12345
/tmp/tailchase prompt --run 12345 --delta
/tmp/tailchase export --run 12345 --target codex
/tmp/tailchase export --run 12345 --target claude-code
/tmp/tailchase export --run 12345 --target copilot
/tmp/tailchase comment --run 12345 --pr 7 --dry-run
/tmp/tailchase mcp --run 12345 --list-resources
/tmp/tailchase adapters --target codex
printf '$ go test ./...\n$ go test ./...\n$ go test ./...\ninternal/app/app.go:42: undefined: Handler\n' > commands.log
/tmp/tailchase guard --run 12345 --command-log commands.log
/tmp/tailchase steer --run 12345 --target copilot --checkpoint stop_event --message "Stop and ask for help."
/tmp/tailchase guard --run 12345 --agent codex --agent-command "false" --max-attempts 1
/tmp/tailchase run-loop --run 12345 --agent codex --agent-command "false" --max-attempts 1
/tmp/tailchase cost report --run 12345
```

Expected artifacts:

```text
.tailchase/config.yml
.tailchase/goal.yml
.tailchase/runs/12345/run.yml
.tailchase/runs/12345/attempt-history.yml
.tailchase/runs/12345/evidence/github-actions.log
.tailchase/runs/12345/normalized-evidence.yml
.tailchase/runs/12345/failure-bundle.yml
.tailchase/runs/12345/repair-prompt.md
.tailchase/runs/12345/report.md
.tailchase/runs/12345/steering-events.yml
.tailchase/runs/12345/run-loop-decisions.yml
.tailchase/runs/12345/steering/<timestamp>-stop_event.md
.tailchase/runs/12345/exports/codex-prompt.md
.tailchase/runs/12345/exports/claude-code-prompt.md
.tailchase/runs/12345/exports/copilot-instructions.md
```

Quick assertions:

```bash
grep -n "undefined: Handler" .tailchase/runs/12345/failure-bundle.yml
grep -n "safety_findings" .tailchase/runs/12345/failure-bundle.yml
grep -n "Fix the failing CI compile error" .tailchase/runs/12345/repair-prompt.md
grep -n "go test ./..." .tailchase/runs/12345/repair-prompt.md
grep -n "Delta Repair Prompt" .tailchase/runs/12345/repair-prompt.md
grep -n "Codex Repair Context" .tailchase/runs/12345/exports/codex-prompt.md
grep -n "Claude Code Repair Context" .tailchase/runs/12345/exports/claude-code-prompt.md
grep -n "GitHub Copilot Repair Context" .tailchase/runs/12345/exports/copilot-instructions.md
/tmp/tailchase comment --run 12345 --pr 7 --dry-run | grep -n "Raw logs are intentionally omitted"
/tmp/tailchase mcp --run 12345 --list-resources | grep -n "Next repair instruction"
/tmp/tailchase adapters --target codex | grep -n "hook_mcp"
grep -n "known_failure_repeated" .tailchase/runs/12345/steering-events.yml
grep -n "Stop and ask for help" .tailchase/runs/12345/steering/*-stop_event.md
grep -n "managed_agent_wrapper" .tailchase/runs/12345/steering-events.yml
grep -n "assisted_repair_loop" .tailchase/runs/12345/steering-events.yml
grep -n "max attempts reached" .tailchase/runs/12345/run-loop-decisions.yml
grep -n "Evidence Reduction" .tailchase/runs/12345/report.md
```

## Local Evidence Smoke Test

```bash
go test ./... > go-test.log 2>&1 || true
/tmp/tailchase collect-local --run 12345 --kind go_test --file go-test.log
/tmp/tailchase bundle --run 12345
grep -n "local_go_test" .tailchase/runs/12345/normalized-evidence.yml
```

## GitLab CI Smoke Test

Use a real GitLab pipeline ID and project path:

```bash
export GITLAB_TOKEN="<token-with-ci-read-access>"
/tmp/tailchase collect-gitlab --run <pipeline-id> --project group/project
/tmp/tailchase bundle --run <pipeline-id>
grep -n "gitlab_ci" .tailchase/runs/<pipeline-id>/normalized-evidence.yml
```

## Tournament Smoke Test

Run from a repository with two local candidate branches:

```bash
/tmp/tailchase tournament candidate-a candidate-b --test-command "go test ./..."
grep -n "Evaluation Criteria" .tailchase/tournaments/candidate-a-vs-candidate-b.md
```

## Test Report Smoke Test

```bash
mkdir -p reports
cat > reports/junit.xml <<'XML'
<testsuite name="unit">
  <testcase classname="pkg.HandlerTest" name="TestHandler" file="internal/app/handler_test.go">
    <failure message="expected 200 got 500">handler_test.go:12 expected 200 got 500</failure>
  </testcase>
</testsuite>
XML

/tmp/tailchase collect-reports --run 12345 --glob "reports/*.xml"
/tmp/tailchase bundle --run 12345
grep -n "junit_report" .tailchase/runs/12345/normalized-evidence.yml
```

## Docker Compose Log Smoke Test

```bash
cat > api.log <<'LOG'
api | GET /health HTTP 500
api | container exited with code 1
LOG

/tmp/tailchase collect-compose --run 12345 --service api --file api.log
/tmp/tailchase bundle --run 12345
grep -n "docker_compose" .tailchase/runs/12345/normalized-evidence.yml
```

## Playwright Artifact Smoke Test

```bash
mkdir -p playwright-report
printf 'console.error: failed to render checkout\n' > playwright-report/console.log
printf 'png bytes' > playwright-report/checkout.png
printf 'zip bytes' > playwright-report/trace.zip

/tmp/tailchase collect-playwright --run 12345 --dir playwright-report
/tmp/tailchase bundle --run 12345
/tmp/tailchase prompt --run 12345
grep -n "playwright" .tailchase/runs/12345/normalized-evidence.yml
grep -n "checkout.png" .tailchase/runs/12345/repair-prompt.md
```

## Optional Model Prompt Smoke Test

Heuristic prompt mode is the default and needs no credentials. To test model-backed prompt writing, configure an OpenAI-compatible endpoint and API key after the local smoke test has produced `failure-bundle.yml`:

```yaml
prompt:
  mode: model
model:
  provider: openai_compatible
  base_url: https://api.openai.com/v1
  model: <model-name>
  api_key_env: OPENAI_API_KEY
```

Then run:

```bash
export OPENAI_API_KEY="<token>"
/tmp/tailchase prompt --run 12345
test -f .tailchase/runs/12345/model-metadata.yml
grep -n "prompt_mode: model" .tailchase/runs/12345/model-metadata.yml
```

## Live Collector Test

Use a real GitHub Actions run ID:

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase init
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase bundle --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id> --delta
tailchase comment --run <github-actions-run-id> --pr <number> --dry-run
tailchase comment --run <github-actions-run-id> --pr <number> --repo owner/name
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
- PR comment posting fails: set `GITHUB_TOKEN` or `GH_TOKEN`, pass `--repo owner/name` if repository discovery fails, and preview first with `--dry-run`.
- Prompt is too generic: improve `.tailchase/goal.yml`.
