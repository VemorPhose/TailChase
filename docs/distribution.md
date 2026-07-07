# Distribution Plan

TailChase should ship as a developer CLI first. Server-style deployment can wait.

## GitHub Releases

GitHub Releases are the canonical release channel. The current released tag is
`v0.1.28`.

Release assets for `v0.1.28` were produced with these names:

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

The release workflow also produced checksums, an SBOM, a checksum signature, and
SLSA provenance for the tagged build.

## Go Install

Verified install path:

```bash
go install github.com/VemorPhose/TailChase/cmd/tailchase@v0.1.28
tailchase version
```

Expected version:

```text
0.1.28
```

pkg.go.dev indexing has been requested or verified for the released module.

## Homebrew

Homebrew is planned but not complete.

Planned future tap:

```text
VemorPhose/homebrew-tailchase
```

Planned future install shape:

```bash
brew tap VemorPhose/tailchase
brew install tailchase
```

Do not present Homebrew as a working install path until the tap exists and
install verification has passed.

For `v0.1.x`, maintain the formula manually once the tap is created. For
`v0.2.x`, consider GoReleaser-managed release and tap updates.

## GitHub Action Wrapper

The GitHub Action wrapper is future work. Do not publish a Marketplace action
before the CLI flow is stable.

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

A container image is future work. It is useful for CI environments, but lower
priority than GitHub Releases, `go install`, and Homebrew.

Future usage shape:

```bash
docker run --rm -v "$PWD:/repo" ghcr.io/vemorphose/tailchase:0.1.0 prepare --run <run-id>
```
