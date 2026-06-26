# Local-First Privacy

TailChase is local-first by default. Its core job is to preserve auditable local artifacts instead of hiding evidence in a model call.

## Stays Local by Default

These files are written under the inspected repository:

```text
.tailchase/config.yml
.tailchase/goal.yml
.tailchase/runs/<run-id>/evidence/
.tailchase/runs/<run-id>/normalized-evidence.yml
.tailchase/runs/<run-id>/failure-bundle.yml
.tailchase/runs/<run-id>/repair-prompt.md
.tailchase/runs/<run-id>/report.md
.tailchase/runs/<run-id>/exports/
```

The default heuristic prompt mode does not call a model provider.

## Sent to GitHub

GitHub API calls happen only when you run commands such as:

```bash
tailchase collect --run <run-id> --repo owner/name
tailchase ci watch
tailchase ci push
tailchase comment --run <run-id> --pr <number>
```

Collection reads workflow metadata and logs. Commenting writes a compact PR comment only when `--dry-run` is not used.

## Sent to GitLab

GitLab API calls happen only when you run:

```bash
tailchase collect-gitlab --run <pipeline-id> --project group/name
```

## Sent to a Model Provider

Nothing is sent to a model provider unless `prompt.mode: model` is configured. Model mode may send goal text, file paths, CI metadata, log excerpts, stack traces, and extracted failure signals.

## Secrets and Logs

TailChase keeps raw evidence files unchanged under `.tailchase/runs/<run-id>/evidence/` so the local audit trail remains inspectable.

Generated signals and excerpts redact common assignment-shaped secrets before they appear in `normalized-evidence.yml`, `failure-bundle.yml`, prompts, exports, reports, or comments. Current redaction covers keys such as `token`, `access_token`, `auth_token`, `api_key`, `secret`, `password`, `passwd`, and `authorization` when they appear as `key=value` or `key: value`.

TailChase does not claim full secret scanning. Logs may still contain sensitive values in unusual formats, in raw evidence files, screenshots, traces, or binary artifacts. Review generated artifacts before sharing them outside your machine or before enabling model mode.

Recommended habits:

- keep `.tailchase/` out of public commits unless the artifacts are intentionally sanitized
- run PR comments with `--dry-run` first
- inspect `failure-bundle.yml` before sharing exports
- avoid collecting logs from production systems unless they are already redacted
