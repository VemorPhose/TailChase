# Demo Plan

The demo repository exists, but the README recording or GIF is still future
work. The eventual recording should show the transformation from noisy failure
evidence to compact repair context in 60-90 seconds.

## Demo Repository

[VemorPhose/tailchase-demo](https://github.com/VemorPhose/tailchase-demo)

Use the demo repository to show one intentional CI failure and the generated
TailChase repair artifacts.

## Intended Flow

```bash
git clone https://github.com/VemorPhose/tailchase-demo
cd tailchase-demo
tailchase init
git checkout broken-refund-handler
tailchase ci watch --export codex
```

Once a failed CI run exists, users can also run:

```bash
tailchase collect --run <run-id> --repo VemorPhose/tailchase-demo
tailchase prepare --run <run-id> --export codex
cat .tailchase/runs/<run-id>/repair-prompt.md
```

Expected artifacts:

```text
.tailchase/runs/<run-id>/failure-bundle.yml
.tailchase/runs/<run-id>/repair-prompt.md
.tailchase/runs/<run-id>/exports/codex-prompt.md
.tailchase/runs/<run-id>/report.md
```

Recording coming soon. Do not add a dead GIF link.

## Story

```text
Messy logs in.
Goal-aware repair context out.
Paste it into the coding agent you already use.
```
