# Release Readiness

TailChase is in alpha. The first public release should prove the current loop before adding more collectors or adapters.

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

## Release Checklist

- README quickstart verified from a clean checkout
- `go install github.com/VemorPhose/TailChase/cmd/tailchase@latest` verified after tagging
- pkg.go.dev indexes the tagged module
- Linux, macOS, and Windows release artifacts built
- `checksums.txt`, signature, SBOM, and SLSA provenance attached
- `CHANGELOG.md` has an entry matching the CLI version
- demo recording or GIF added to README
- `SECURITY.md`, `CONTRIBUTING.md`, and issue templates present
- local-first privacy docs reviewed

## Distribution Roadmap

1. GitHub Releases remain the canonical channel.
2. `go install` remains the developer install path.
3. Add `VemorPhose/homebrew-tailchase` after the first tagged release.
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
