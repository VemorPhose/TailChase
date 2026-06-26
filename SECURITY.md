# Security Policy

TailChase reads failure logs, repository paths, CI metadata, prompt files, and optional tokens. Please treat security reports with care.

## Supported Versions

Until the first stable release, only the latest commit on `main` and the latest published `v0.x` release receive security fixes.

## Reporting a Vulnerability

Use GitHub private vulnerability reporting if it is available for this repository. If private reporting is unavailable, open a minimal public issue that says you have a security report, but do not include exploit details, tokens, private logs, or affected private repository names.

Please include:

- affected TailChase version or commit
- affected command or artifact
- whether model mode, PR comments, wrappers, or run-loop automation were enabled
- a minimal reproduction with secrets removed

## Local-First Security Expectations

By default, TailChase stores artifacts locally under `.tailchase/` and uses the heuristic prompt writer. Network access is used only by commands that collect from remote services, post comments, or call a configured model provider.

See [docs/local-first-privacy.md](docs/local-first-privacy.md) for the data handling model.
