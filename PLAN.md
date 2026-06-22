# Tailchase Product Brief

## One-line summary

Tailchase is a local-first repair-context and agent-run steering layer for coding agents.

It collects failed CI, runtime, browser, and local execution evidence; compresses that evidence into an auditable failure bundle; uses deterministic checks plus an optional LLM to generate high-signal repair context; and routes that context back into the user's chosen coding agent with the strongest steering surface available.

```text
Failure evidence + goal contract + attempt history
-> compact failure bundle
-> cost-aware repair context
-> prompt, adapter, or controlled agent steering
```

Tailchase does not replace Codex, Claude Code, Copilot, Cursor, or similar agents. It makes their repair loops cheaper, safer, and less repetitive.

## Core thesis

Coding agents waste time and tokens when they debug from incomplete, noisy, stale, or repeated context.

The common loop today is:

```text
agent works on branch
-> CI/runtime/test check fails
-> human searches logs and artifacts
-> human pastes long evidence back into agent
-> agent fixes a symptom or repeats work
-> new run fails again
-> more logs and more tokens are spent
```

Tailchase turns this into two related loops:

```text
Reactive repair loop:
failure happens -> collect evidence -> build bundle -> generate repair prompt -> send to next agent attempt

Proactive steering loop:
agent is running -> monitor diff/logs/commands/attempts -> detect drift or waste -> steer at a safe boundary
```

The broader product goal is not merely "AI explains CI failures." The goal is:

```text
Reduce repeated agent-debugging cycles, token waste, and goal drift by preserving high-signal repair context across attempts.
```

## Product pillars

### 1. Failure Bundle

Tailchase collects scattered failure evidence and turns it into a compact, inspectable bundle.

Evidence sources over time:

- GitHub Actions logs and artifacts
- GitLab CI, CircleCI, Buildkite, Jenkins, and other CI/CD logs
- local shell command output
- Docker Compose logs
- app boot logs
- Playwright screenshots, traces, videos, and console output
- JUnit/Jest/Pytest/Go test output
- browser/network errors
- migration logs
- repository diff and changed files

The bundle is a durable artifact, not just hidden model input.

### 2. Context Budget Manager

Tailchase should reduce expensive repeated context.

It tracks:

- raw log size versus included prompt size
- repeated stack traces collapsed
- repeated commands or test runs
- same root error across attempts
- evidence already shown to the agent
- delta evidence since the last attempt
- estimated token footprint of repair context

Instead of resending full logs, Tailchase should send compact deltas:

```text
Same root error as attempt 2: REFUND_WEBHOOK_SECRET is still undefined.
New evidence: CI env file was not changed. Do not reread full logs. Fix test/CI env setup.
```

### 3. Goal Contract

Tailchase keeps the agent anchored to the original task.

The goal contract contains:

- goal title and source
- non-goals
- must-preserve behavior
- done criteria
- expected paths
- suspicious paths
- stop/escalation rules

The contract powers repair prompts, drift warnings, stop decisions, and future run-guard behavior.

### 4. Agent Run Guard

Tailchase should eventually monitor agent execution and steer before the run wastes another cycle.

It watches:

- current git diff
- edited files and suspicious paths
- skipped or weakened tests
- repeated commands
- repeated errors
- dependency changes
- local logs and command output
- evidence reuse versus full-log rereading

It emits:

- steering hints
- stop warnings
- context-budget warnings
- drift warnings
- compact repair bundles
- adapter-specific steering messages

## Positioning

Tailchase is:

- local-first agent repair infrastructure
- failure-bundle centered
- model-agnostic
- agent-agnostic
- goal-aware
- cost-aware
- auditable by default
- manual-first, then assisted, then automatic

Tailchase is not:

- a coding agent
- an AI code reviewer
- a CI/CD platform
- a testing framework
- a GitHub/GitLab replacement
- a generic LLM gateway
- a general observability platform
- an auto-merge bot

Avoid this positioning:

```text
Tailchase explains failed CI logs with AI.
```

Use this positioning:

```text
Tailchase packages failed CI, runtime, and local evidence into portable repair context for whichever coding agent you already use.
```

## Target users

Primary users:

- developers using Codex, Claude Code, GitHub Copilot, Cursor, Gemini CLI, OpenHands, or similar agents
- teams shipping agent-created branches and PRs
- maintainers reviewing AI-generated contributions
- platform teams standardizing safe AI development workflows
- developers using weaker/cheaper models that need better steering context

Core user stories:

```text
As a developer, when an agent-created branch fails CI, I want Tailchase to fetch the failure evidence and generate the next precise repair context so I do not manually paste logs into the agent.
```

```text
As a developer, when an agent is drifting or repeating the same failed debugging loop, I want Tailchase to detect that and steer the agent before it burns more tokens.
```

```text
As a team lead, I want an audit trail of what evidence was collected, what repair context was sent, and why an automatic loop stopped.
```

## Data flow: failure to repair context

Initial CI-focused flow:

```text
1. Branch is pushed.
2. GitHub Actions runs and fails.
3. Tailchase collects failed-job logs and metadata.
4. Raw evidence is stored locally under .tailchase/runs/<run-id>/evidence/.
5. Evidence is normalized, cleaned, trimmed, and deduplicated.
6. Tailchase compiles failure-bundle.yml.
7. Goal contract and attempt history are attached.
8. Prompt writer generates repair-prompt.md.
9. Output adapter prints, writes, comments, exports, or steers depending on mode.
```

Short pipeline:

```text
CI/runtime/local failure
-> collectors
-> local evidence store
-> normalizer/extractor
-> failure bundle
-> goal + attempt memory
-> context budget manager
-> prompt writer
-> output or steering adapter
```

## Data flow: proactive agent steering

Future run-guard flow:

```text
1. Agent starts from a task goal and goal contract.
2. Tailchase watches git diff, command output, known failures, and local logs.
3. Tailchase detects drift, repeated work, oversized context, or unsafe changes.
4. Tailchase creates a short steering instruction or stop warning.
5. Adapter delivers it through the strongest available control surface.
6. Tailchase records whether the agent followed the steering signal.
```

This is adapter-dependent. Tailchase should not promise universal live injection into every running agent.

## Steering model

Tailchase should support multiple steering levels.

### Level 0: Artifact steering

Always available.

Tailchase writes files that agents or humans can read:

- `repair-prompt.md`
- `failure-bundle.yml`
- `AGENTS.md` snippets
- `CLAUDE.md` snippets
- `GEMINI.md` snippets
- `.github/copilot-instructions.md` snippets
- Cursor/VS Code instruction files where supported

### Level 1: Queued or next-turn steering

Tailchase sends context that the agent consumes after the current turn or task segment.

Useful for agents that do not expose reliable live steering but support follow-up prompts, session resume, prompt files, or comments.

### Level 2: Checkpoint / tool-boundary steering

Tailchase steers after safe boundaries such as:

- tool call completed
- terminal command finished
- file write completed
- permission request appeared
- test command failed
- suspicious diff detected

This is the practical form of mid-execution steering.

### Level 3: Hook / MCP steering

Tailchase exposes tools, resources, or hooks:

- `tailchase.current_goal`
- `tailchase.latest_failure_bundle`
- `tailchase.next_repair_instruction`
- `tailchase.check_drift`
- `tailchase.check_token_waste`
- `tailchase.should_continue`

Targets may include Claude Code hooks, Claude Agent SDK, VS Code/Copilot MCP, Cursor MCP, Codex-compatible MCP/tool surfaces, and generic MCP clients.

### Level 4: Managed agent wrapper

Tailchase runs or supervises the agent command, watches output and filesystem changes, and injects context where the agent supports it. If live injection is unavailable, it can stop, restart, or resume the agent with updated context.

This should be opt-in only.

## Steering adapter assumptions

Steering capability differs by agent. Tailchase must encode capabilities per adapter.

### GitHub Copilot / Copilot CLI / Copilot SDK

Public docs describe active steering: users can send input while Copilot is working, and the agent considers that input in the current task. The Copilot SDK also distinguishes immediate steering from queued messages. GitHub has also described Copilot adapting after the current tool call completes.

Tailchase implication:

- high-priority target for checkpoint steering
- good candidate for direct steering adapters
- adapter should prefer immediate/tool-boundary steering where APIs expose it

### OpenAI Codex

Codex should be treated as a first-class target, but integration surfaces should be validated per runtime.

Near-term support:

- prompt file export
- `AGENTS.md` / project instruction export
- GitHub Action prompt handoff
- stdout/manual prompt handoff

Advanced support:

- turn/steer or session APIs only when stable and accessible
- MCP/tool-based access where available
- wrapper-based checkpoint steering where reliable

### Claude Code

Claude Code should be treated as a first-class target, but Tailchase should not assume that typing into every running CLI session creates reliable interstitial steering.

Reliable surfaces to target:

- `CLAUDE.md` and project instructions
- hooks such as `PreToolUse`, `PostToolUse`, `Stop`, and `Notification`
- Claude Agent SDK streaming input and interruption modes
- MCP tools/resources
- wrapper checkpoints and explicit human-in-the-loop stops

Tailchase implication:

- use Claude Code hooks for deterministic guardrails
- use Agent SDK or wrapper mode for stronger control
- use artifact/queued steering when live CLI steering is unavailable

### Cursor, VS Code, Gemini CLI, OpenHands, and others

Initial support should be file/prompt/MCP based:

- repair prompt files
- workspace instruction files
- MCP resources/tools
- generic stdin/stdout adapters
- optional wrapper mode later

## Model strategy

MVP:

- no model required
- deterministic extraction and template prompt generation
- useful without any API key

First production-ready version:

- LLM is the main writer of analysis and repair prompts
- deterministic code controls evidence collection, safety, token limits, and stop rules
- user chooses provider and model

Supported provider strategy:

- native OpenAI adapter
- native Anthropic adapter
- native Google Gemini adapter
- Ollama/local model adapter
- OpenAI-compatible base URL adapter
- optional OpenRouter/LiteLLM-compatible adapter

This allows future support for DeepSeek, Kimi/Moonshot, Qwen, local models, and other providers without one-off integrations for every model family.

Principle:

```text
The model writes and reasons.
Deterministic checks collect, constrain, budget, compare, and protect.
```

## Core components

| Component | Function |
|---|---|
| CLI | User control surface for init, collect, bundle, prompt, export, guard, and future loop commands |
| Config loader | Reads project, collector, model, prompt, safety, budget, and adapter settings |
| Goal contract | Defines goal, non-goals, done criteria, expected paths, suspicious paths, and stop rules |
| Collectors | Fetch remote CI evidence and local runtime/test/browser evidence |
| Local run store | Saves evidence, bundles, prompts, attempts, and steering events per run |
| Normalizer/extractor | Cleans logs, extracts error signals, collapses repeated stack traces |
| Failure bundle compiler | Creates portable structured bundle with root candidates, sources, artifacts, and evidence links |
| Attempt memory | Tracks previous attempts, errors, prompts, diffs, and outcomes |
| Context budget manager | Reduces repeated context and estimates token footprint |
| Prompt writer | Generates repair prompt via heuristic template or configured LLM |
| Steering adapters | Deliver context through files, comments, MCP, hooks, SDKs, or wrappers |
| Run guard | Monitors active agent work and emits drift/waste/stop/steering signals |
| Safety engine | Stops or warns on repeated failure, goal drift, test weakening, dependency risk, or low confidence |

## Failure bundle shape

```yaml
version: 1
run:
  id: gha-123456
  branch: agent/refund-support
  commit: abc123
  created_at: "2026-06-22T10:00:00Z"

goal:
  title: "Add refund support for cancelled orders"
  source: "github_issue_482"

sources:
  - type: github_actions
    job: integration-tests
    status: failed
    log_path: .tailchase/runs/gha-123456/evidence/integration-tests.log
    url: https://github.com/org/repo/actions/runs/123456

root_error_candidates:
  - message: "REFUND_WEBHOOK_SECRET is undefined"
    source: github_actions
    confidence: high
    evidence_path: .tailchase/runs/gha-123456/evidence/integration-tests.log

likely_downstream_errors:
  - message: "POST /api/refunds returned HTTP 500"
    reason: "API failed before UI reached confirmation state"

attempt_context:
  previous_attempts: 2
  same_root_error_seen_before: true
  new_evidence_since_last_attempt:
    - "CI env file unchanged"

budget:
  raw_log_bytes: 420000
  included_excerpt_bytes: 18000
  repeated_blocks_collapsed: 14
```

## Goal contract shape

```yaml
version: 1

goal:
  title: "Add refund support for cancelled orders"
  source: "github_issue_482"

non_goals:
  - "Do not modify payment capture retry behavior"
  - "Do not change subscription renewal logic"

must_preserve:
  - "Existing checkout behavior"
  - "Existing webhook idempotency behavior"

done_when:
  - "Refund API works for cancelled orders"
  - "Duplicate refund webhook delivery is safe"
  - "Focused refund tests pass"

expected_paths:
  - "src/refunds/**"
  - "src/webhooks/**"
  - "tests/refunds/**"

suspicious_paths:
  - "src/payments/capture/**"
  - "src/subscriptions/renewal/**"

stop_rules:
  same_root_error_twice: true
  test_weakening: true
  dependency_change: warn
  suspicious_path_edit: warn
```

## Configuration shape

MVP config:

```yaml
version: 1

project:
  name: payments-service
  default_branch: main

collectors:
  github_actions:
    enabled: true
    failed_jobs_only: true
    max_log_lines_per_job: 500

prompt:
  mode: heuristic
  target: generic
  include_goal: true
  include_non_goals: true
  include_commands_to_run: true
  max_prompt_tokens: 3000

models:
  enabled: false

safety:
  stop_on_same_failure_twice: false
  warn_on_goal_drift: true
  warn_on_test_weakening: true
```

Production-ready config direction:

```yaml
version: 1

collectors:
  github_actions:
    enabled: true
  gitlab_ci:
    enabled: true
  local_shell:
    enabled: true
  docker_compose:
    enabled: true
    services: [api, worker, postgres]
  playwright:
    enabled: true
    artifact_dir: ./test-results
  junit:
    enabled: true
    paths: ["./reports/**/*.xml"]

prompt:
  mode: model
  target: codex # generic | codex | claude-code | copilot | cursor | vscode
  max_prompt_tokens: 6000
  include_delta_from_previous_attempt: true

models:
  enabled: true
  provider: openai-compatible # openai | anthropic | gemini | ollama | openai-compatible
  model: gpt-5.5
  api_key_env: OPENAI_API_KEY
  base_url_env: OPENAI_BASE_URL

budget:
  track_estimates: true
  max_prompt_tokens: 6000
  max_attempts_per_task: 3
  stop_on_same_root_error_twice: true

steering:
  mode: assisted # manual | assisted | automatic
  adapter: copilot-cli
  prefer_interstitial: true
  fallback: prompt-file

safety:
  stop_on_goal_drift: true
  stop_on_test_weakening: true
  dependency_change: warn
  protected_paths:
    - .github/workflows/**
    - src/payments/capture/**
```

## Local artifact layout

```text
.tailchase/
  config.yml
  goal.yml
  runs/
    <run-id>/
      evidence/
        github-actions-integration-tests.log
        docker-api.log
        playwright-console.log
        screenshot.png
      normalized-evidence.yml
      failure-bundle.yml
      repair-prompt.md
      attempt-history.yml
      steering-events.yml
      report.md
```

MVP may only produce:

```text
.tailchase/
  config.yml
  goal.yml
  runs/
    <run-id>/
      evidence/github-actions.log
      failure-bundle.yml
      repair-prompt.md
```

## CLI direction

MVP commands:

```bash
tailchase init
tailchase collect --run <github-actions-run-id>
tailchase bundle
tailchase prompt
```

Near-term commands:

```bash
tailchase prompt --delta
tailchase export --target codex
tailchase export --target claude-code
tailchase export --target copilot
tailchase compare-attempts
tailchase comment --pr <number>
```

Future commands:

```bash
tailchase guard --agent copilot-cli
tailchase guard --agent claude-code
tailchase mcp
tailchase run-loop --agent codex --max-attempts 3
tailchase cost report
tailchase tournament <branch-a> <branch-b>
```

## Example repair prompt

```text
Repair context for coding agent

Original goal:
Add refund support for cancelled orders.

Non-goals:
- Do not modify payment capture retry behavior.
- Do not change subscription renewal logic.

Current failure summary:
GitHub Actions failed in integration-tests. The first high-confidence root candidate is:
REFUND_WEBHOOK_SECRET is undefined.

Evidence:
- refund-webhook.integration.test.ts failed.
- POST /api/refunds returned HTTP 500.
- The HTTP 500 appears downstream of missing env setup.

Attempt memory:
This root error appeared in the previous attempt too. The CI env setup has not changed since then.

Required next actions:
1. Add REFUND_WEBHOOK_SECRET to test/CI environment setup.
2. Re-run the focused refund webhook integration test.
3. Confirm duplicate webhook delivery is tested.
4. Do not edit payment capture retry code.

Commands to run:
- pnpm test refund-webhook.integration.test.ts
- pnpm test checkout-refund.smoke.test.ts

Stop condition:
If POST /api/refunds still returns HTTP 500 after fixing test env setup, inspect API logs before changing business logic.
```

## MVP scope

The MVP remains intentionally small.

Build:

1. Go CLI.
2. `tailchase init`.
3. GitHub Actions failed-log collector.
4. Local `.tailchase/runs/<run-id>/` evidence store.
5. Basic log normalization and extraction.
6. Failure bundle generation.
7. Goal contract file.
8. Heuristic/template repair prompt generation.
9. stdout + markdown prompt output.

Do not build in MVP:

- model API integration
- direct agent steering
- PR comments
- broad CI support
- Docker/Playwright/JUnit collectors
- hosted dashboard
- auto-fix or patch generation
- automatic loop
- MCP server
- exact token billing

MVP success question:

```text
Can Tailchase save one manual log-reading / re-prompting turn after a failed agent-created GitHub Actions run?
```

## First production-ready version

Add:

- model-backed prompt writing
- multi-provider model interface
- attempt memory
- context budget manager
- repeated-failure detection
- delta repair prompts
- GitHub PR comment mode
- Docker Compose collector
- Playwright/JUnit/test artifact collectors
- target-specific prompt exports
- initial agent adapters

The first production-ready version should still be manual/assisted by default. Automatic loops require stronger stop rules and adapter maturity.

## Full vision roadmap

### Phase 0: MVP

```text
GitHub Actions failed logs -> failure bundle -> heuristic repair prompt
```

### Phase 1: Cost-aware repair context

- attempt memory
- same-root-error detection
- delta bundles
- context-size estimates
- repeated-log collapse
- `tailchase prompt --delta`

### Phase 2: Model-backed analysis

- AI prompt writer
- OpenAI, Anthropic, Gemini, Ollama, OpenAI-compatible providers
- model-generated diagnosis from bundle + goal + attempt memory
- deterministic safety checks remain authoritative

### Phase 3: Broad evidence collection

- Docker Compose logs
- Playwright artifacts
- JUnit/Jest/Pytest/Go test output
- GitLab/CircleCI/Buildkite/Jenkins collectors
- local app logs and shell logs

### Phase 4: Agent adapter layer

- Codex prompt/export adapter
- Copilot CLI/SDK steering adapter where available
- Claude Code artifact/hook/SDK adapter
- Cursor/VS Code/MCP adapter
- generic prompt-file/stdout adapter

### Phase 5: Agent Run Guard

- monitor git diff, commands, logs, and attempts
- emit drift and context-budget warnings
- steer at supported checkpoints
- use hooks/MCP/SDKs where available
- wrapper mode for agents without live steering

### Phase 6: Safe automatic loop

- collect
- bundle
- generate repair context
- inject into agent
- monitor next attempt
- stop on repeat, drift, test weakening, dependency risk, or budget exhaustion

### Future: Tournament evaluator

Compare candidate branches from different agents by:

- test pass/fail
- smoke behavior
- diff risk
- goal drift
- dependency changes
- repeated failure patterns
- context cost
- final repair prompt quality

## Tech stack

Implementation language:

- Go

Recommended stack:

- Go CLI
- Cobra for command routing
- `gopkg.in/yaml.v3` for config, goal, and bundle files
- `github.com/google/go-github` for GitHub API integration
- `log/slog` for logging
- `text/template` for heuristic prompt templates
- local filesystem storage
- Go test for test suite
- GitHub Actions for Tailchase's own CI

Possible future dependencies:

- MCP server library when adapter layer is ready
- SQLite only if local run history outgrows flat files
- provider-specific SDKs for model integrations, or plain HTTP adapters where practical

Avoid early:

- web dashboard
- database
- background services
- microservices
- Kubernetes dependency
- hosted backend
- mandatory model dependency

## Suggested Go repository structure

```text
tailchase/
  cmd/
    tailchase/
      main.go
  internal/
    project/      # config, goal, run directories
    collect/      # GitHub Actions collector first; more collectors later
    bundle/       # normalization, extraction, bundle writing
    prompt/       # heuristic and future model prompt writers
    steering/     # future adapters; empty or omitted in MVP
    guard/        # future run guard; empty or omitted in MVP
  templates/
    repair_prompt.md.tmpl
  examples/
  docs/
  .github/workflows/
  go.mod
  README.md
```

For the MVP, keep only:

```text
internal/project
internal/collect
internal/bundle
internal/prompt
```

## Safety principles

1. Never auto-merge code.
2. Never hide raw failure evidence.
3. Always save the bundle and prompt that were sent to the agent.
4. Keep manual mode as the default.
5. Treat automatic steering as opt-in.
6. Stop on repeated root failure.
7. Stop or warn on goal drift.
8. Stop or warn on test weakening.
9. Warn on suspicious dependency changes.
10. Prefer compact deltas over repeated full logs.
11. Use deterministic checks for safety boundaries.
12. Make every steering action auditable.

## Metrics

Product metrics:

- manual re-prompting turns avoided
- repeated agent attempts avoided
- raw log bytes reduced to repair-context bytes
- estimated tokens avoided
- repeated stack traces collapsed
- same-root-error loops detected
- goal drift detections
- test-weakening detections
- first repair success rate after Tailchase prompt
- successful steering events by adapter
- percentage of runs escalated to human

Engineering metrics:

- collector reliability
- extraction precision/recall on known failures
- prompt size
- bundle size
- false positive drift warnings
- adapter success/failure rate
- model cost per generated prompt

## Competitive stance

Crowded areas:

- AI CI failure explanation
- AI PR comments on failed checks
- single-platform auto-fix bots
- platform-native GitHub/GitLab/Bitbucket repair flows
- coding agents that directly fix CI failures

Tailchase should differentiate through the combination of:

- local-first artifacts
- portable failure bundle
- broad remote + local evidence collection
- goal/non-goal preservation
- attempt memory
- context-budget reduction
- multi-model prompt writing
- multi-agent context routing
- adapter-dependent steering
- conservative stop rules

## Key risks

### Risk: The product looks like another CI failure explainer

Mitigation:

- lead with repair-context routing and cost reduction
- make failure bundle and attempt memory first-class
- avoid PR-comment-only positioning

### Risk: Steering support differs too much across agents

Mitigation:

- define capability levels per adapter
- support files/stdout everywhere
- use hooks/MCP/SDKs where available
- treat true live injection as experimental

### Risk: Model analysis is wrong

Mitigation:

- preserve raw evidence
- show confidence levels
- keep prompts editable
- deterministic checks control stops and warnings

### Risk: Auto-loop wastes tokens or makes unsafe changes

Mitigation:

- manual-first default
- strict budgets
- repeated-failure stop
- goal-drift stop
- test-weakening stop
- audit log

## Final product direction

Build Tailchase first as:

```text
failed GitHub Actions logs + goal contract -> local failure bundle -> heuristic repair prompt
```

Then expand into:

```text
local-first, model-backed, multi-agent repair context and run steering
```

The winning product is not the fastest CI fixer. It is the most reliable way to reduce wasted coding-agent repair cycles by giving agents the right evidence, at the right time, through the right steering surface.
