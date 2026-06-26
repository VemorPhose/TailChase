# Demo Plan

The launch demo should show the transformation from noisy failure evidence to compact repair context in under 90 seconds.

## Demo Repository

Recommended separate repository:

```text
VemorPhose/tailchase-demo
```

Use a tiny Go or Node project with one intentional CI failure on a branch such as `broken-refund-handler`.

## Script

```bash
git clone https://github.com/VemorPhose/tailchase-demo
cd tailchase-demo
tailchase init
git checkout broken-refund-handler
tailchase ci watch --export codex
cat .tailchase/runs/*/repair-prompt.md
```

Show:

- raw CI log size
- `failure-bundle.yml`
- `repair-prompt.md`
- `exports/codex-prompt.md`
- `report.md`

## Story

```text
Messy logs in.
Goal-aware repair context out.
Paste it into the coding agent you already use.
```
