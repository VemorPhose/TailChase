# TailChase

Tailchase is an open-source, local-first failure-feedback sidecar for coding agents.

It turns failed GitHub Actions evidence into a compact local failure bundle and a heuristic repair prompt that can be pasted into an agent.

```text
GitHub Actions failure -> local evidence store -> failure bundle -> repair prompt
```

## MVP Commands

```bash
go run ./cmd/tailchase init
go run ./cmd/tailchase collect --run <github-actions-run-id> --repo owner/name
go run ./cmd/tailchase bundle --run <github-actions-run-id>
go run ./cmd/tailchase prompt --run <github-actions-run-id>
```

`collect` also accepts the repository from `.tailchase/config.yml` or `git remote origin` when `--repo` is omitted.

For private repositories, set `GITHUB_TOKEN` or `GH_TOKEN`.

## Local Artifacts

Tailchase writes artifacts under the project being inspected:

```text
.tailchase/
  config.yml
  goal.yml
  runs/
    <run-id>/
      evidence/
        github-actions.log
      normalized-evidence.yml
      failure-bundle.yml
      repair-prompt.md
```

`prompt` prints the repair prompt to stdout and writes the same content to `repair-prompt.md`.

## Development

```bash
go test ./...
go run ./cmd/tailchase version
```

For a full MVP verification checklist, including a local smoke test and live GitHub Actions collection test, see [docs/testing-and-development.md](docs/testing-and-development.md).
