# Contributing

Thanks for helping make TailChase more useful and trustworthy.

## Development Setup

```bash
git clone https://github.com/VemorPhose/TailChase.git
cd TailChase
go test ./...
go install ./cmd/tailchase
```

Use the smallest change that solves the issue. TailChase handles local logs,
prompts, and repository metadata, so changes should stay auditable and
conservative by default.

## Before Opening a PR

Run:

```bash
go test ./... -count=1
go vet ./...
go test ./tests -run TestGolden -count=1
go test ./tests -run TestNoNetworkCoreFlow -count=1
```

For behavior that changes generated artifacts, update tests and docs in the same PR.

## Good First Issues

Good first issues should be small, testable, and avoid broad behavior changes.
Draft seeds live in [docs/good-first-issues.md](docs/good-first-issues.md).
They have not been opened as GitHub issues yet.

Useful areas include:

- improving one doc page with a verified command example
- adding a fixture for one collector edge case
- tightening one export format without changing the core bundle schema
- improving an error message that already has a test
- adding a small example to `docs/`

## Discussions vs Issues

Use GitHub Discussions for questions, ideas, and prompt-quality feedback. Use
Issues for reproducible bugs and focused, actionable tasks.

See [docs/community.md](docs/community.md) for the current community guidance.

## Pull Request Notes

Include:

- the user-visible behavior change
- the tests you ran
- any local-first or privacy impact
- whether generated artifact formats changed

Do not include raw private CI logs, secrets, customer repository names, or production stack traces in issues or PRs.
