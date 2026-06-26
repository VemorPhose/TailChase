<div align="center">

# TailChase

**Compact, auditable repair context for failed coding-agent runs**

[![Go Version](https://img.shields.io/github/go-mod/go-version/VemorPhose/TailChase?style=flat-square&color=00add8)](go.mod)
[![Release](https://img.shields.io/github/v/release/VemorPhose/TailChase?style=flat-square&color=2f80ed)](https://github.com/VemorPhose/TailChase/releases)
[![License](https://img.shields.io/badge/License-MIT-grey.svg?style=flat-square)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-go%20test%20.%2F...-2ea44f?style=flat-square)](#development)
[![Local First](https://img.shields.io/badge/local--first-artifacts-6f42c1?style=flat-square)](#artifacts)

[Why TailChase?](#why-tailchase) - [Installation](#installation) - [Quick Start](#quick-start) - [Workflows](#workflows) - [Evidence Sources](#evidence-sources) - [Commands](#commands) - [Docs](#documentation)

</div>

---

## What is TailChase?

TailChase turns failed CI and local runtime evidence into compact, auditable repair context for coding agents.

```text
Messy CI logs in -> local failure bundle -> goal-aware repair context out
```

It is built for the moment after "CI failed" but before "try random edits." TailChase collects noisy evidence, trims it into durable local artifacts, checks for risky repair patterns, and exports focused prompts for Codex, Claude Code, Copilot, or any workflow that can read markdown.

Stop pasting failed logs into agents by hand.

---

## Why TailChase?

- **Local-first artifacts:** Evidence, bundles, reports, prompts, and steering history stay under `.tailchase/` in the project being inspected.
- **Deterministic by default:** The built-in heuristic prompt writer needs no model credentials and produces auditable files.
- **Multiple evidence streams:** Import GitHub Actions, GitLab CI, captured shell output, Go test logs, JUnit XML, Docker Compose logs, and Playwright artifacts.
- **Agent-ready exports:** Generate target-specific prompt files for Codex, Claude Code, and Copilot.
- **Repair guardrails:** Surface stop/warn findings for suspicious path edits, repeated root errors, and test weakening patterns.
- **Opt-in automation:** PR comments, model-written prompts, advisory guard mode, wrappers, and assisted repair loops are explicit choices.

> [!IMPORTANT]
> TailChase is intentionally conservative. It helps an agent understand a failure; it does not silently weaken tests, post comments, call a model provider, or run managed repair loops unless you ask it to.

---

## Installation

| Method | Command | Notes |
| :-- | :-- | :-- |
| **Go install** | `go install github.com/VemorPhose/TailChase/cmd/tailchase@latest` | Requires Go matching `go.mod`. |
| **Homebrew** | `brew tap VemorPhose/tailchase && brew install tailchase` | Planned after the first tagged release. |
| **From source** | `git clone https://github.com/VemorPhose/TailChase.git` | Best for development and local testing. |
| **Local binary** | `go build -o /tmp/tailchase ./cmd/tailchase` | Useful when Go bin paths are not on `PATH`. |

Build and verify from this repository:

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

---

## GitHub Token Setup

TailChase uses one token for GitHub Actions logs, CI watching, and optional PR comments.

1. Create a GitHub token from GitHub settings.
2. For public repositories, a classic token with `repo` is the simplest option. For a fine-grained token, allow repository metadata, contents read, actions read, and issues or pull request write access if you want PR comments.
3. Add it to your shell:

```bash
export GITHUB_TOKEN="ghp_your_token_here"
```

`GH_TOKEN` also works. Add the export line to `~/.zshrc` if you want it available in every new terminal.

---

## Quick Start

Run TailChase from the repository whose failure you want to inspect:

```bash
tailchase init
git push
tailchase ci watch --export codex
```

If CI fails, TailChase creates a local run directory with `failure-bundle.yml`, `repair-prompt.md`, and `report.md`. If CI passes, it tells you no repair bundle is needed.

### Anchor the repair goal

Edit `.tailchase/goal.yml` so generated prompts stay aligned with the real task:

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

### Collect, prepare, and preview

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase prepare --run <github-actions-run-id> --export codex
tailchase comment --run <github-actions-run-id> --pr <number> --dry-run
```

`--repo` can be omitted when `.tailchase/config.yml` has `github.repo` or `git remote origin` points at GitHub.

---

## Workflows

| Workflow | Command |
| :-- | :-- |
| **I just pushed and CI failed** | `tailchase ci watch --export codex` |
| **I already know the failed CI run** | `tailchase collect --run <id> --repo owner/name` then `tailchase prepare --run <id> --export claude-code` |
| **I want to reduce repeated agent loops** | `tailchase prompt --run <id> --delta` then `tailchase cost report --run <id>` |
| **I want to preview a PR repair comment** | `tailchase comment --run <id> --pr <number> --dry-run` |

For `tailchase ci push`, put `--` before git flags:

```bash
tailchase ci push -- --set-upstream origin main
```

### Alpha support boundary

The first public alpha is focused on:

```bash
tailchase init
tailchase ci watch --export codex
tailchase prepare --run <run-id> --export codex
tailchase comment --run <run-id> --pr <number> --dry-run
tailchase cost report --run <run-id>
```

These surfaces are available but experimental:

- `tailchase mcp`
- `tailchase guard --agent ...`
- `tailchase run-loop`
- `tailchase tournament`
- `prompt.mode: model`

---

## Evidence Sources

TailChase can collect more than remote CI logs.

| Source | Command |
| :-- | :-- |
| **GitHub Actions** | `tailchase collect --run <id> --repo owner/name` |
| **GitLab CI** | `tailchase collect-gitlab --run <pipeline-id> --project group/name` |
| **Local shell or test output** | `tailchase collect-local --run <id> --kind go_test --file go-test.log` |
| **JUnit XML reports** | `tailchase collect-reports --run <id> --glob "reports/*.xml"` |
| **Docker Compose logs** | `tailchase collect-compose --run <id> --service api` |
| **Playwright artifacts** | `tailchase collect-playwright --run <id> --dir playwright-report` |

> [!NOTE]
> GitLab collection uses `GITLAB_TOKEN`. GitHub collection uses `GITHUB_TOKEN` or `GH_TOKEN`.

---

## Commands

| Command | What it does |
| :-- | :-- |
| `tailchase init` | Creates `.tailchase/config.yml` and `.tailchase/goal.yml`. |
| `tailchase collect --run <id> [--repo owner/name]` | Downloads failed GitHub Actions job logs. |
| `tailchase collect-gitlab --run <pipeline-id> --project group/name [--base-url <url>]` | Downloads failed GitLab CI job traces. |
| `tailchase collect-local --run <id> --kind go_test\|shell --file <path>` | Imports captured local command output. |
| `tailchase collect-reports --run <id> [--glob <pattern>]` | Imports JUnit-style XML reports. |
| `tailchase collect-compose --run <id> --service <name> [--file <path>]` | Imports Docker Compose service logs. |
| `tailchase collect-playwright --run <id> --dir <path>` | Imports Playwright console logs, traces, screenshots, and videos. |
| `tailchase prepare --run <id> [--delta] [--export codex]` | Runs bundle, prompt, optional exports, and cost report. |
| `tailchase ci watch [--export codex]` | Waits for CI on the current branch, then prepares artifacts if CI fails. |
| `tailchase ci push [git push args...] [--export codex]` | Runs `git push`, waits for CI, then prepares artifacts if CI fails. |
| `tailchase bundle --run <id>` | Writes `normalized-evidence.yml` and `failure-bundle.yml`. |
| `tailchase prompt --run <id> [--delta]` | Writes `repair-prompt.md`, or a compact delta prompt. |
| `tailchase export --run <id> --target codex\|claude-code\|copilot` | Writes target-specific prompt files. |
| `tailchase comment --run <id> --pr <number> [--repo owner/name] [--dry-run]` | Previews or posts compact GitHub PR repair context. |
| `tailchase mcp --run <id>` | Starts a local stdio MCP server exposing repair resources. |
| `tailchase adapters [--target codex]` | Lists supported agent adapter capabilities. |
| `tailchase guard --run <id> [--command-log commands.log]` | Records advisory guard findings. |
| `tailchase guard --run <id> --agent <target> --agent-command "<cmd>" --max-attempts <n>` | Runs an opt-in managed wrapper. |
| `tailchase steer --run <id> --target <target> --message <text>` | Records checkpoint steering or writes a fallback prompt file. |
| `tailchase run-loop --run <id> --agent <target> --agent-command "<cmd>" --max-attempts <n>` | Runs a conservative assisted repair loop. |
| `tailchase cost report --run <id>` | Writes `report.md` with evidence reduction, prompt size, safety, and attempt metrics. |
| `tailchase tournament <branch-a> <branch-b> [--test-command "go test ./..."]` | Compares candidate repair branches without changing the current worktree. |
| `tailchase version` | Prints the CLI version. |

---

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

TailChase records each generated repair prompt in `attempt-history.yml`. Later bundles warn when the same root error appears again, helping separate repeated root failures from downstream noise.

Failure bundles also include a context budget with raw evidence bytes, included excerpt bytes, collapsed repeated log blocks, and an estimated prompt size.

PR comments are opt-in. Use `--dry-run` to preview the compact body locally; posting requires `GITHUB_TOKEN` or `GH_TOKEN` and never includes raw full logs.

---

## Artifacts

TailChase writes all artifacts under the inspected project:

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

---

## Documentation

- [Quickstart](docs/quickstart.md)
- [GitHub Actions guide](docs/github-actions.md)
- [Agent exports](docs/agent-exports.md)
- [Model mode](docs/model-mode.md)
- [Local-first privacy](docs/local-first-privacy.md)
- [Demo plan](docs/demo.md)
- [Distribution plan](docs/distribution.md)
- [Release readiness](docs/release-readiness.md)
- [Launch plan](docs/launch-plan.md)
- [Good first issue seeds](docs/good-first-issues.md)
- [Roadmap](docs/roadmap.md)
- [Collector extension notes](docs/collectors.md)
- [Adapter capability notes](docs/adapters.md)
- [Core flow](docs/core-flow.md)
- [Schemas](docs/schemas.md)
- [Testing and development guide](docs/testing-and-development.md)
- [Contributing](CONTRIBUTING.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

---

## Development

Use these checks before committing:

```bash
go test ./...
go vet ./...
go test -race ./...
go test -coverpkg=./... ./...
```

GitHub Actions runs repository gates across split workflows:

- metadata and module integrity
- module download, verification, and `go mod tidy` drift checks
- `gofmt`, workflow YAML validation, and `go vet ./...`
- `go test ./... -count=1` on Linux, macOS, and Windows
- golden artifact and no-network core-flow tests
- `go test -race ./...`
- `go test -coverpkg=./... ./...` with coverage artifacts
- CLI build and local no-network core smoke test
- tag-only cross-platform release builds for Linux, macOS, and Windows
- release checksums, checksum signatures, SBOM, and SLSA provenance
- scheduled nightly fixture replay and optional live API/model smokes

Workflow details live in `.github/workflows/` and [docs/testing-and-development.md](docs/testing-and-development.md).
