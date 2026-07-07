# GitHub Actions

TailChase can collect failed GitHub Actions job logs and turn them into a local repair bundle.

## Token

Set one of:

```bash
export GITHUB_TOKEN="<token>"
export GH_TOKEN="<token>"
```

For public repositories, the token mainly avoids low API limits. For private
repositories, grant repository metadata, contents read, and actions read. Add
issues or pull request write access only if you plan to post PR comments.

## Watch Current Branch

```bash
tailchase ci watch --export codex
```

This waits for the matching workflow run on the current branch. If the run fails, TailChase collects failed-job logs and prepares artifacts.

## Push and Watch

```bash
tailchase ci push --export codex
```

Pass git flags after `--`:

```bash
tailchase ci push -- --set-upstream origin main
```

## Known Run

```bash
tailchase collect --run <run-id> --repo owner/name
tailchase prepare --run <run-id> --export codex
```

`--repo` can be omitted when `.tailchase/config.yml` has `github.repo` or the Git remote points at GitHub.

## PR Comments

PR comments are opt-in. Preview first:

```bash
tailchase comment --run <run-id> --pr <number> --dry-run
```

Comments include compact repair context and artifact references, not raw full logs.
