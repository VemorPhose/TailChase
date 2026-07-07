# Release Readiness

TailChase v0.1.28 has been released. This document tracks the post-release
state without treating future/manual distribution work as complete.

## Supported First Alpha Path

```bash
tailchase init
tailchase ci watch --export codex
tailchase prepare --run <run-id> --export codex
tailchase comment --run <run-id> --pr <number> --dry-run
tailchase cost report --run <run-id>
```

## Experimental Surfaces

These are available but should be presented as experimental until their UX and safety story settles:

- `tailchase mcp`
- `tailchase guard --agent ...`
- `tailchase run-loop`
- `tailchase tournament`
- `prompt.mode: model`

## Completed for v0.1.28

- pre-release branch merged into `main`
- `v0.1.28` release published
- release workflow passed
- release assets produced for Linux, macOS, and Windows
- `checksums.txt`, SBOM, checksum signature, and provenance produced
- `go install github.com/VemorPhose/TailChase/cmd/tailchase@v0.1.28` verified
- pkg.go.dev indexing requested or verified
- GitHub topics added
- GitHub Discussions enabled
- demo repository created
- `SECURITY.md`, `CONTRIBUTING.md`, and issue templates present
- local-first privacy docs reviewed

## Pending Manual Future Work

- Homebrew tap
- Homebrew install verification
- demo recording or GIF
- GitHub labels
- seed issues to open from [good-first-issues.md](good-first-issues.md)

## Deferred Future Work

- GitHub Action wrapper
- container image distribution
- Marketplace publishing
- broader launch and promotion campaign

## Distribution Roadmap

1. GitHub Releases remain the canonical channel.
2. `go install` remains the developer install path.
3. Add `VemorPhose/homebrew-tailchase` as future manual work.
4. Maintain the Homebrew formula manually for `v0.1.x`, then consider GoReleaser-managed releases and tap updates for `v0.2.x`.
5. Add a GitHub Action wrapper after the CLI flow is stable.
6. Consider a container image only after local CLI distribution is working.

See [docs/distribution.md](distribution.md) for channel-specific notes.

## Not This

TailChase is not:

- a coding agent
- a CI replacement
- an auto-merge bot
- a generic LLM gateway
- an observability platform

## Launch Positioning

Use:

```text
TailChase turns failed CI and local runtime evidence into compact, auditable repair context for coding agents.
```

Avoid:

```text
AI fixes your CI.
```
