# Good First Issue Seeds

These are draft issue seeds. Labels and issues have not been created yet.
Do not claim issue numbers exist until maintainers create them manually.

## Maintainer Checklist

Before opening these issues:

- create the suggested labels below
- choose which issue seeds are still relevant
- open each issue manually
- add the correct labels
- update this file only after issue numbers exist

## 1. Add a Screenshot or GIF Placeholder to the README

Add a README section that links to the demo recording once it exists. Keep the
section hidden or text-only until a real asset is available.

## 2. Add a Fixture for a GitHub Annotation Error

Create a small test fixture with `::error file=...::...` syntax and assert that
TailChase extracts the file, line, and message.

## 3. Improve One Export Instruction

Pick one export target and tighten its instruction copy without changing the
generated artifact structure.

## 4. Add a Local Privacy Example

Add a short example to [docs/local-first-privacy.md](local-first-privacy.md)
showing which commands stay fully local and which commands call remote APIs.

## 5. Add a GitLab CI Fixture Edge Case

Add a fixture for a failed GitLab job trace that includes a file/line error and one downstream symptom.

## Suggested Labels

| Label | Color | Use |
| :-- | :-- | :-- |
| `good first issue` | `#7057ff` | Small, well-scoped starter tasks. |
| `docs` | `#0075ca` | Documentation-only work. |
| `fixtures` | `#cfd3d7` | Test fixture and sample artifact work. |
| `agent exports` | `#5319e7` | Codex, Claude Code, Copilot export polish. |
| `redaction` | `#d73a4a` | Secret handling and generated-artifact privacy work. |
| `collector edge case` | `#fbca04` | Provider parsing and evidence extraction edge cases. |
| `prompt quality` | `#0e8a16` | Repair prompt clarity, size, and usefulness. |

Manual future commands:

```bash
gh label create "good first issue" --color 7057ff --repo VemorPhose/TailChase
gh label create docs --color 0075ca --repo VemorPhose/TailChase
gh label create fixtures --color cfd3d7 --repo VemorPhose/TailChase
gh label create "agent exports" --color 5319e7 --repo VemorPhose/TailChase
gh label create redaction --color d73a4a --repo VemorPhose/TailChase
gh label create "collector edge case" --color fbca04 --repo VemorPhose/TailChase
gh label create "prompt quality" --color 0e8a16 --repo VemorPhose/TailChase
```

These commands are examples only. Run them manually after confirming repository
permissions and label naming.
