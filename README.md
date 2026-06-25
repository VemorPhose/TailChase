# TailChase

Tailchase is a local-first CLI that turns failed CI, local test, runtime, and browser evidence into structured repair context for a coding agent.

```text
Failure evidence -> local evidence store -> failure bundle -> repair prompt
```

Tailchase is intentionally conservative: it stores auditable local artifacts, uses deterministic collectors and safety checks, keeps heuristic prompts available without model credentials, and makes model writing, PR comments, guard mode, wrappers, and assisted loops opt-in.

## Setup

Prerequisites:

- Go matching `go.mod`
- Git
- A repository that uses GitHub Actions if you want remote CI collection
- `GITHUB_TOKEN` or `GH_TOKEN` for private repositories or higher GitHub API limits

Install from this repository:

```bash
git clone https://github.com/VemorPhose/TailChase.git
cd TailChase
go test ./...
go install ./cmd/tailchase
tailchase version
```

Expected version:

```text
0.1.28
```

If `$GOBIN` or `$GOPATH/bin` is not on your `PATH`, build a local binary instead:

```bash
go build -o /tmp/tailchase ./cmd/tailchase
/tmp/tailchase version
```

## GitHub Token Setup

Tailchase uses one token for GitHub Actions logs, CI watching, and optional PR comments.

1. Create a GitHub token from GitHub settings.
2. For public repositories, a classic token with `repo` is the simplest option. For a fine-grained token, allow repository metadata, contents read, actions read, and issues or pull request write access if you want PR comments.
3. Add it to your shell:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

`GH_TOKEN` also works. Add the export line to `~/.zshrc` if you want it available in every new terminal.

First-time checklist inside your project:

```bash
tailchase init
git push
tailchase ci watch --export codex
```

If CI fails, Tailchase creates `.tailchase/runs/<run-id>/failure-bundle.yml`, `repair-prompt.md`, and `report.md` locally. If CI passes, it tells you no repair bundle is needed.

## Quick Start

Run Tailchase from the repository whose CI failure you want to inspect:

```bash
tailchase init
```

Edit `.tailchase/goal.yml` so the generated prompt is anchored to the real task:

```yaml
goal: Fix the failing GitHub Actions run for the current branch.
non_goals:
  - Do not weaken, skip, or delete tests.
must_preserve:
  - Existing public behavior unless the task explicitly changes it.
done_conditions:
  - Relevant tests pass locally.
  - GitHub Actions passes for the branch.
expected_paths:
  - internal/app
suspicious_paths:
  - .github/workflows
stop_rules:
  - Stop before weakening, skipping, or deleting tests.
  - Stop before changing behavior outside the original task.
```

Collect failed logs, build the bundle, and render the prompt:

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase prepare --run <github-actions-run-id> --export codex
tailchase comment --run <github-actions-run-id> --pr <number> --dry-run
```

`--repo` can be omitted when `.tailchase/config.yml` has `github.repo` or `git remote origin` points at GitHub.
After pushing a branch, you can avoid opening GitHub and let Tailchase wait for CI:

```bash
tailchase ci watch --export codex
```

Or push and wait in one command:

```bash
tailchase ci push --export codex
```

For local evidence, capture output to a file and run `tailchase collect-local --run <id> --kind go_test --file go-test.log` or `--kind shell`.
For GitLab CI, set `GITLAB_TOKEN` and run `tailchase collect-gitlab --run <pipeline-id> --project group/name`.
For JUnit-style reports from Jest, Pytest, or other test runners, use `tailchase collect-reports --run <id> --glob "reports/*.xml"`.
For Docker Compose runtime logs, use `tailchase collect-compose --run <id> --service api` or pass `--file api.log` for captured logs.
For browser test artifacts, use `tailchase collect-playwright --run <id> --dir playwright-report`.

## Commands

- `tailchase init` creates `.tailchase/config.yml` and `.tailchase/goal.yml`.
- `tailchase collect --run <id> [--repo owner/name]` downloads failed GitHub Actions job logs into the local run store.
- `tailchase collect-gitlab --run <pipeline-id> --project group/name [--base-url <url>]` downloads failed GitLab CI job traces.
- `tailchase collect-local --run <id> --kind go_test|shell --file <path>` imports captured local output into the run store.
- `tailchase collect-reports --run <id> [--glob <pattern>]` imports JUnit-style XML reports.
- `tailchase collect-compose --run <id> --service <name> [--file <path>]` imports Docker Compose service logs.
- `tailchase collect-playwright --run <id> --dir <path>` imports Playwright console logs, traces, screenshots, and videos.
- `tailchase prepare --run <id> [--delta] [--export codex]` runs `bundle`, `prompt`, optional exports, and `cost report`.
- `tailchase ci watch [--export codex]` waits for the current branch's GitHub Actions run, then prepares artifacts if CI fails.
- `tailchase ci push [git push args...] [--export codex]` runs `git push`, waits for GitHub Actions, then prepares artifacts if CI fails. Put `--` before git flags, for example `tailchase ci push -- --set-upstream origin main`.
- `tailchase bundle --run <id>` extracts failure signals and writes `normalized-evidence.yml` plus `failure-bundle.yml`.
- `tailchase prompt --run <id>` writes `repair-prompt.md`; with `prompt_target: stdout`, it also prints the prompt.
- `tailchase prompt --run <id> --delta` writes a compact prompt focused on prior attempts, repeated root errors, new evidence, budgets, and artifact links.
- `tailchase export --run <id> --target codex|claude-code|copilot` writes target-specific prompt files under the run's `exports/` directory.
- `tailchase comment --run <id> --pr <number> [--repo owner/name] [--dry-run]` previews or posts compact GitHub PR repair context.
- `tailchase mcp --run <id>` starts a local stdio MCP server exposing the goal, failure bundle, repair prompt, budget summary, and safety findings.
- `tailchase adapters [--target codex]` lists supported agent adapter capabilities and artifact fallback behavior.
- `tailchase guard --run <id> [--command-log commands.log]` records advisory guard findings in `steering-events.yml`.
- `tailchase guard --run <id> --agent <target> --agent-command "<cmd>" --max-attempts <n>` runs an opt-in managed wrapper.
- `tailchase steer --run <id> --target <target> --message <text>` records checkpoint steering or writes a fallback prompt file.
- `tailchase run-loop --run <id> --agent <target> --agent-command "<cmd>" --max-attempts <n>` runs a conservative assisted repair loop.
- `tailchase cost report --run <id>` writes `report.md` with evidence reduction, prompt size, safety, and attempt metrics.
- `tailchase tournament <branch-a> <branch-b> [--test-command "go test ./..."]` compares candidate repair branches without changing the current worktree.
- `tailchase version` prints the CLI version.

## Configuration

`.tailchase/config.yml` controls collection and prompt output:

```yaml
collectors:
  - github_actions
github:
  repo: owner/repo
gitlab:
  project: group/project
  base_url: https://gitlab.com
failed_jobs_only: true
max_log_lines_per_job: 1200
prompt_target: stdout
prompt_size_limit: 12000
prompt:
  mode: heuristic
model:
  provider: openai_compatible
  base_url: ""
  model: ""
  api_key_env: OPENAI_API_KEY
report_globs:
  - reports/*.xml
compose:
  services:
    - api
  tail_lines: 300
playwright:
  artifact_dir: playwright-report
adapters:
  - target: codex
    capability: artifact
safety:
  mode: manual
  stop_on:
    - test_weakening
    - suspicious_path_edit
```

`prompt_target` may be `stdout` or `file`. `stdout` prints the prompt and writes the file; `file` only prints the written path.
`prompt.mode` defaults to `heuristic` and needs no model credentials. Set `prompt.mode: model`, `model.base_url`, `model.model`, and `model.api_key_env` to use an OpenAI-compatible `/chat/completions` endpoint; generated model prompts also write `model-metadata.yml`.
Safety mode is advisory/manual in this version. `safety.stop_on` controls which structured findings are marked `stop` instead of `warn`.

Tailchase records each generated repair prompt in `attempt-history.yml`. Later bundles warn when the same root error appears again, helping separate repeated root failures from downstream noise.

Failure bundles also include a context budget with raw evidence bytes, included excerpt bytes, collapsed repeated log blocks, and an estimated prompt size.

PR comments are opt-in. Use `--dry-run` to preview the compact body locally; posting requires `GITHUB_TOKEN` or `GH_TOKEN` and never includes raw full logs.

Collector extension notes live in [docs/collectors.md](docs/collectors.md).
Adapter capability notes live in [docs/adapters.md](docs/adapters.md).

## Artifacts

Tailchase writes all artifacts under the inspected project:

```text
.tailchase/
  config.yml
  goal.yml
  runs/
    <run-id>/
      run.yml
      attempt-history.yml
      evidence/
        github-actions.log
        gitlab-ci.log
        go-test.log
        shell-command.log
        test-reports/
          01-junit.xml
        compose/
          api.log
        playwright/
          console.log
          screenshot.png
          trace.zip
      normalized-evidence.yml
      failure-bundle.yml
      repair-prompt.md
      model-metadata.yml
      report.md
      steering-events.yml
      run-loop-decisions.yml
      steering/
        <timestamp>-stop_event.md
      exports/
        codex-prompt.md
        claude-code-prompt.md
        copilot-instructions.md
  tournaments/
    <branch-a>-vs-<branch-b>.md
```

## Development

Use these checks before committing:

```bash
go test ./...
go vet ./...
go test -race ./...
go test -coverpkg=./... ./...
```

CI/CD runs the repository gates on GitHub Actions:

- metadata and release-ref detection
- module download, verification, and `go mod tidy` drift checks
- `gofmt`, workflow YAML validation, and `go vet ./...`
- `go test ./...` on Linux, macOS, and Windows
- `go test -race ./...`
- `go test -coverpkg=./... ./...` with coverage artifacts
- CLI build and local no-network core smoke test
- cross-platform release builds for Linux, macOS, and Windows
- tag-based GitHub releases with checksums for `v*` tags

The workflow lives at `.github/workflows/ci.yml` and uses GitHub Actions marketplace actions for checkout, Go setup/cache, artifact upload/download, and release publishing.

More detail:

- [Core flow](docs/core-flow.md)
- [Schemas](docs/schemas.md)
- [Testing and development guide](docs/testing-and-development.md)
