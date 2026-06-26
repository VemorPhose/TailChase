# Launch Plan

Use this after the first tagged release is installable and the demo is ready.

## Pre-Launch Checklist

- publish `v0.1.0-alpha.1` or the next version that matches `tailchase version`
- verify GitHub Releases, `go install`, and local binary installs
- add a README demo GIF or terminal recording
- create the demo repository described in [docs/demo.md](demo.md)
- add GitHub topics
- enable Discussions if maintainers are ready to answer questions
- label at least five good first issues from [docs/good-first-issues.md](good-first-issues.md)
- pin the positioning: TailChase is not an agent and not a CI replacement

## Suggested GitHub Topics

```text
ai-coding
coding-agents
github-actions
ci-cd
developer-tools
cli
golang
llm
mcp
local-first
devops
test-automation
agent-tools
```

## Launch Copy

Short:

```text
TailChase turns failed CI and local runtime evidence into compact, auditable repair context for coding agents.
```

Outcome:

```text
Stop pasting failed CI logs into coding agents by hand.
```

Avoid:

```text
AI fixes your CI.
```

## Channels

Start with GitHub Releases and the demo repository. Then share with developer-tool communities that can give concrete artifact feedback.

Good first channels:

- GitHub project README and release notes
- Show HN with a technical, feedback-seeking post
- `r/golang`, `r/devops`, or agent-specific communities when the post is relevant
- Product Hunt only after the install path and demo are polished
- Go and DevOps newsletters with a before/after demo

## Feedback Questions

Ask for specific critique:

- Is `failure-bundle.yml` useful and readable?
- Is the repair prompt too verbose or too compact?
- Which export target should be improved first?
- Is the local-first privacy story clear?
