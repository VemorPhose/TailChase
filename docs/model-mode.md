# Model Mode

TailChase defaults to deterministic heuristic prompt generation. No model
provider is called unless you explicitly configure model mode.

## Configuration

```yaml
prompt:
  mode: model
model:
  provider: openai_compatible
  base_url: "https://api.example.com/v1"
  model: "example-model"
  api_key_env: OPENAI_API_KEY
```

Then run:

```bash
tailchase prompt --run <run-id>
```

## What Gets Sent

Model mode sends structured repair context derived from the failure bundle. It
may contain file paths, redacted log excerpts, stack traces, test names, CI
metadata, and goal-contract text.

Inspect these local files before enabling model mode:

```text
.tailchase/runs/<run-id>/failure-bundle.yml
.tailchase/runs/<run-id>/repair-prompt.md
```

## Metadata

When model mode writes a prompt, TailChase also writes:

```text
.tailchase/runs/<run-id>/model-metadata.yml
```

Use this to audit which provider configuration produced the prompt.
