# Distribution Plan

TailChase should ship as a developer CLI first. Server-style deployment can wait.

## GitHub Releases

GitHub Releases are the canonical release channel.

Release assets should use these names:

```text
tailchase_Darwin_arm64.tar.gz
tailchase_Darwin_x86_64.tar.gz
tailchase_Linux_arm64.tar.gz
tailchase_Linux_x86_64.tar.gz
tailchase_Windows_x86_64.zip
checksums.txt
checksums.txt.sig
checksums.txt.sigstore.json
sbom.spdx.json
```

The release workflow also generates SLSA provenance for tagged builds.

## Go Install

Developer install path:

```bash
go install github.com/VemorPhose/TailChase/cmd/tailchase@latest
```

After tagging, verify:

```bash
go install github.com/VemorPhose/TailChase/cmd/tailchase@v<version>
```

Then confirm pkg.go.dev has indexed the module.

## Homebrew

Planned tap:

```text
VemorPhose/homebrew-tailchase
```

Planned install command:

```bash
brew tap VemorPhose/tailchase
brew install tailchase
```

For `v0.1.x`, maintain the formula manually after a release exists. For `v0.2.x`, consider GoReleaser-managed release and tap updates.

## GitHub Action Wrapper

Do not publish a Marketplace action before the CLI release is stable.

The first wrapper should do one thing:

```text
On failed CI, collect the current run logs and upload TailChase artifacts.
```

It should not auto-comment or auto-run agents by default.

Future usage shape:

```yaml
- uses: VemorPhose/tailchase-action@v0
  with:
    mode: prepare
    export: codex
```

## Container Image

A container image is useful for CI environments, but lower priority than GitHub Releases, `go install`, and Homebrew.

Future usage shape:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/vemorphose/tailchase:0.1.0 prepare --run <run-id>
```
