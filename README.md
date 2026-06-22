# TailChase

Tailchase is a local-first CLI that turns failed GitHub Actions evidence into a structured failure bundle and a paste-ready repair prompt for a coding agent.

```text
GitHub Actions failure -> local evidence store -> failure bundle -> repair prompt
```

The MVP is intentionally narrow: GitHub Actions only, local YAML/Markdown artifacts only, no hosted service, no model API key, and no automatic agent steering.

## Setup

Prerequisites:

- Go matching `go.mod`
- Git
- A repository that uses GitHub Actions
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
0.1.6
```

If `$GOBIN` or `$GOPATH/bin` is not on your `PATH`, build a local binary instead:

```bash
go build -o /tmp/tailchase ./cmd/tailchase
/tmp/tailchase version
```

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
suspicious_paths:
  - .github/workflows
```

Collect failed logs, build the bundle, and render the prompt:

```bash
export GITHUB_TOKEN="<token-with-actions-read-access>"
tailchase collect --run <github-actions-run-id> --repo owner/name
tailchase bundle --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id>
tailchase prompt --run <github-actions-run-id> --delta
```

`--repo` can be omitted when `.tailchase/config.yml` has `github.repo` or `git remote origin` points at GitHub.

## Commands

- `tailchase init` creates `.tailchase/config.yml` and `.tailchase/goal.yml`.
- `tailchase collect --run <id> [--repo owner/name]` downloads failed GitHub Actions job logs into the local run store.
- `tailchase bundle --run <id>` extracts failure signals and writes `normalized-evidence.yml` plus `failure-bundle.yml`.
- `tailchase prompt --run <id>` writes `repair-prompt.md`; with `prompt_target: stdout`, it also prints the prompt.
- `tailchase prompt --run <id> --delta` writes a compact prompt focused on prior attempts, repeated root errors, new evidence, budgets, and artifact links.
- `tailchase version` prints the CLI version.

## Configuration

`.tailchase/config.yml` controls collection and prompt output:

```yaml
collectors:
  - github_actions
github:
  repo: owner/repo
failed_jobs_only: true
max_log_lines_per_job: 1200
prompt_target: stdout
prompt_size_limit: 12000
```

`prompt_target` may be `stdout` or `file`. `stdout` prints the prompt and writes the file; `file` only prints the written path.

Tailchase records each generated repair prompt in `attempt-history.yml`. Later bundles warn when the same root error appears again, helping separate repeated root failures from downstream noise.

Failure bundles also include a context budget with raw evidence bytes, included excerpt bytes, collapsed repeated log blocks, and an estimated prompt size.

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
      normalized-evidence.yml
      failure-bundle.yml
      repair-prompt.md
```

## Development

Use these checks before committing:

```bash
go test ./...
go vet ./...
go test -race ./...
go test -coverpkg=./... ./...
```

More detail:

- [MVP flow](docs/mvp-flow.md)
- [Schemas](docs/schemas.md)
- [Testing and development guide](docs/testing-and-development.md)
