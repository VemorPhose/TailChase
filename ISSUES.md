# Tailchase Implementation Issues

This issue list is ordered bottom-up: dependencies before dependents, core logic before wrappers, and manual/local behavior before assisted or automatic steering.

## Production Baseline

Current production baseline includes:

- Go CLI
- `tailchase init`
- GitHub Actions failed-log collection and CI watch/push wrappers
- local `.tailchase/runs/<run-id>/` store
- versioned schemas, normalization, failure bundles, attempt memory, context budgets, safety findings, reports, and repair prompts
- local, JUnit-style, Docker Compose, Playwright, GitHub Actions, and GitLab evidence collectors
- heuristic prompts by default with optional OpenAI-compatible model prompt writing
- target exports, PR comment dry-run/posting, MCP resources, adapter capabilities, guard events, wrapper mode, assisted loop, and tournament evaluator
- local tests, black-box usage tests, smoke-test documentation, and GitHub Actions CI/CD gates

---

## 1. Stabilize Versioned Config, Goal, and Bundle Schemas

Labels: `core`, `schema`, `foundation`

Goal:
Define stable versioned schemas for config, goal contract, normalized evidence, failure bundle, and generated artifacts.

Depends on:
None.

Deliverables:
- Add explicit `version` fields to config, goal, normalized evidence, and failure bundle.
- Align current schemas with the shapes in `PLAN.md` where practical.
- Add migration-safe defaulting for missing version fields.
- Document schema fields and artifact compatibility expectations.

Acceptance criteria:
- Existing pre-version artifacts still load.
- New artifacts include version fields.
- Tests cover defaulting, validation, and backward-compatible reads.

---

## 2. Improve Local Run Store and Artifact Indexing

Labels: `core`, `storage`

Goal:
Make the local run store a stable foundation for attempts, evidence, prompts, and steering events.

Depends on:
Issue 1.

Deliverables:
- Add a run metadata file per run.
- Track artifact names, paths, source types, and created timestamps.
- Keep raw evidence paths visible and auditable.
- Add helper APIs for reading/writing known artifacts.

Acceptance criteria:
- `tailchase collect`, `bundle`, and `prompt` use shared store helpers.
- Tests verify expected `.tailchase/runs/<run-id>/` layout.
- Missing-artifact errors are clear and actionable.

---

## 3. Add Attempt Memory Data Model

Labels: `core`, `attempt-memory`

Goal:
Track repair attempts so Tailchase can compare repeated failures and generate delta context.

Depends on:
Issues 1, 2.

Deliverables:
- Add `attempt-history.yml`.
- Store attempt number, run ID, bundle path, prompt path, root candidates, and outcome.
- Add APIs to append and read attempt records.
- Keep the format local-first and human-readable.

Acceptance criteria:
- Multiple attempts can be recorded for one task.
- Attempt history survives across CLI invocations.
- Tests cover append/read/order behavior.

---

## 4. Add Repeated Failure Detection

Labels: `core`, `analysis`, `attempt-memory`

Goal:
Detect when the same root error appears across attempts.

Depends on:
Issue 3.

Deliverables:
- Normalize root-error fingerprints.
- Compare current root candidates with previous attempts.
- Mark `same_root_error_seen_before` in bundle context.
- Emit warnings when repeated failures are detected.

Acceptance criteria:
- Same error across two attempts is detected.
- Similar downstream symptoms do not override the root error.
- Tests cover exact and near-identical error messages.

---

## 5. Add Context Budget Manager

Labels: `core`, `budget`

Goal:
Track raw evidence size, included excerpt size, repeated blocks collapsed, and estimated prompt footprint.

Depends on:
Issues 1, 2, 4.

Deliverables:
- Add budget metadata to failure bundles.
- Count raw log bytes and included excerpt bytes.
- Collapse repeated log blocks before prompt generation.
- Expose budget summary in generated prompts.

Acceptance criteria:
- Large repeated logs produce compact bundle excerpts.
- Bundle records raw vs included sizes.
- Tests cover repeated-stack/log collapse behavior.

---

## 6. Add Delta Repair Prompt Mode

Labels: `prompt`, `attempt-memory`, `budget`

Goal:
Generate compact prompts that emphasize what changed since the previous attempt.

Depends on:
Issues 3, 4, 5.

Deliverables:
- Add `tailchase prompt --delta`.
- Include same-root-error and new-evidence summaries.
- Avoid resending full repeated context.
- Preserve goal, non-goals, stop condition, and raw artifact links.

Acceptance criteria:
- Delta prompt references prior attempts when available.
- Delta prompt falls back cleanly when no history exists.
- Tests verify repeated evidence is summarized, not duplicated.

---

## 7. Strengthen Goal Contract Checks

Labels: `core`, `safety`, `goal-contract`

Goal:
Use goal contract fields to detect drift risks before prompts or steering decisions.

Depends on:
Issue 1.

Deliverables:
- Support expected paths, suspicious paths, and stop rules.
- Warn on signals or edits touching suspicious paths.
- Prepare reusable checks for future guard mode.
- Document goal contract examples.

Acceptance criteria:
- Bundle includes goal-contract warnings.
- Suspicious path matches are tested.
- Missing or vague goal fields produce useful warnings.

---

## 8. Add Safety Engine

Labels: `core`, `safety`

Goal:
Centralize deterministic stop/warn decisions.

Depends on:
Issues 3, 4, 7.

Deliverables:
- Add safety checks for repeated root failure, goal drift, test weakening, dependency changes, and suspicious path edits.
- Emit structured safety findings.
- Keep manual mode as the default.

Acceptance criteria:
- Safety findings are written to bundle/report artifacts.
- Stop vs warn behavior follows config.
- Tests cover each safety rule.

---

## 9. Add Local Test Output Collectors

Labels: `collector`, `local-evidence`

Goal:
Collect local command/test output as evidence, starting with common test formats.

Depends on:
Issues 1, 2.

Deliverables:
- Add collectors for Go test output and generic shell command logs.
- Store raw local evidence under run evidence directories.
- Normalize local test failures into the shared signal model.

Acceptance criteria:
- Local failing Go test output produces failure signals.
- Raw command output is preserved.
- Tests use fixtures, not live shell failures where avoidable.

---

## 10. Add JUnit/Jest/Pytest Artifact Collectors

Labels: `collector`, `test-artifacts`

Goal:
Collect structured test reports from common ecosystems.

Depends on:
Issues 1, 2, 9.

Deliverables:
- Add configurable report path globs.
- Parse JUnit-style XML.
- Extract failing test names, files, messages, and stack traces.
- Preserve report paths in bundle sources.

Acceptance criteria:
- JUnit fixtures produce normalized signals.
- Missing report paths produce warnings, not crashes.
- Bundle links back to raw report files.

---

## 11. Add Docker Compose Log Collector

Labels: `collector`, `runtime-evidence`

Goal:
Collect runtime logs from Docker Compose services.

Depends on:
Issues 1, 2.

Deliverables:
- Add config for enabled services.
- Collect recent logs per service.
- Extract runtime exceptions, HTTP failures, missing env vars, and crash loops.
- Store service logs as raw evidence.

Acceptance criteria:
- Collector works against fixture logs in tests.
- Missing Docker/Compose produces clear errors.
- Runtime signals appear in normalized evidence.

---

## 12. Add Playwright Artifact Collector

Labels: `collector`, `browser-evidence`

Goal:
Collect browser test artifacts including console output, traces, screenshots, and videos where available.

Depends on:
Issues 1, 2.

Deliverables:
- Add configurable artifact directory.
- Index screenshots/traces/videos as evidence sources.
- Extract console errors and failed test names.
- Include artifact links in bundles and prompts.

Acceptance criteria:
- Fixture artifact directories are indexed.
- Console errors become normalized signals.
- Prompt references artifact paths without embedding large binaries.

---

## 13. Add Multi-Provider Model Interface

Labels: `model`, `prompt`

Goal:
Introduce a model abstraction without making models mandatory.

Depends on:
Issues 1, 5, 8.

Deliverables:
- Add provider interface for model-backed prompt writing.
- Support disabled-by-default model config.
- Add initial provider config shape for OpenAI-compatible endpoints.
- Keep heuristic prompt writer as fallback.

Acceptance criteria:
- Existing heuristic mode still works without API keys.
- Model mode validates required provider settings.
- Provider interface is unit-tested with fake clients.

---

## 14. Add Model-Backed Prompt Writer

Labels: `model`, `prompt`

Goal:
Use an LLM to write repair prompts from failure bundle, goal contract, attempt memory, and budget metadata.

Depends on:
Issues 5, 6, 8, 13.

Deliverables:
- Add `prompt.mode: model`.
- Build deterministic model input from bundle artifacts.
- Preserve raw evidence links and safety findings.
- Record generated prompt and model metadata.

Acceptance criteria:
- Model prompt writer can be tested with fake provider responses.
- Prompt generation fails safely when provider errors.
- Heuristic fallback remains available.

---

## 15. Add Target-Specific Prompt Export

Labels: `adapter`, `export`

Goal:
Export repair context for specific agent surfaces without live steering.

Depends on:
Issues 6, 14 optional.

Deliverables:
- Add `tailchase export --target codex`.
- Add `tailchase export --target claude-code`.
- Add `tailchase export --target copilot`.
- Write target-specific instruction/prompt files only.

Acceptance criteria:
- Export targets create documented files.
- Export does not modify unrelated project files.
- Tests verify target-specific content and paths.

---

## 16. Add GitHub PR Comment Mode

Labels: `adapter`, `github`

Goal:
Post repair context as a GitHub PR comment when explicitly requested.

Depends on:
Issues 6, 8.

Deliverables:
- Add `tailchase comment --pr <number>`.
- Use GitHub token from environment.
- Post compact prompt summary plus artifact references.
- Avoid posting raw full logs.

Acceptance criteria:
- Dry-run or fake GitHub client tests cover comment body generation.
- Missing token or PR number fails clearly.
- Command is opt-in only.

---

## 17. Add CI Provider Abstraction

Labels: `collector`, `architecture`

Goal:
Prepare collectors for GitLab CI, CircleCI, Buildkite, Jenkins, and other providers.

Depends on:
Issues 1, 2.

Deliverables:
- Define collector interface around evidence sources and normalized signals.
- Keep GitHub Actions as first implementation.
- Add provider metadata in evidence sources.
- Document how new collectors plug in.

Acceptance criteria:
- GitHub Actions collector behavior is unchanged.
- Collector interface is covered by tests.
- No new provider is required in this issue.

---

## 18. Add Additional Remote CI Collectors

Labels: `collector`, `ci`

Goal:
Add first non-GitHub CI collectors after the abstraction is stable.

Depends on:
Issue 17.

Deliverables:
- Add GitLab CI collector.
- Add one additional provider only if credentials/API shape is clear.
- Preserve raw logs and normalize provider-specific failures.

Acceptance criteria:
- Each provider has config validation.
- Tests use fake clients or fixtures.
- Missing credentials fail with clear messages.

---

## 19. Add MCP Resource/Tool Server

Labels: `adapter`, `mcp`

Goal:
Expose Tailchase bundle, goal, and repair instruction data through MCP.

Depends on:
Issues 6, 8, 15.

Deliverables:
- Add `tailchase mcp`.
- Expose resources for current goal, latest failure bundle, and next repair instruction.
- Expose tools for drift/check/budget queries if safe and deterministic.

Acceptance criteria:
- MCP server starts locally.
- Resource output matches latest local artifacts.
- Tests cover resource serialization.

---

## 20. Add Agent Adapter Capability Model

Labels: `adapter`, `steering`

Goal:
Represent what each agent integration can safely support.

Depends on:
Issues 8, 15, 19 optional.

Deliverables:
- Define capability levels: artifact, queued, checkpoint, hook/MCP, wrapper.
- Add adapter config and capability discovery.
- Document Codex, Claude Code, Copilot, Cursor/VS Code, and generic targets.

Acceptance criteria:
- Adapter capabilities are explicit and testable.
- Unsupported steering modes fail safely.
- File/stdout fallback is always available.

---

## 21. Add Run Guard Core

Labels: `guard`, `safety`, `steering`

Goal:
Monitor local agent work and produce drift, repeated-work, and stop warnings.

Depends on:
Issues 3, 5, 8, 20.

Deliverables:
- Track git diff, edited paths, command output, repeated commands, and known failures.
- Emit structured guard findings.
- Write `steering-events.yml`.
- Keep behavior advisory/manual by default.

Acceptance criteria:
- Guard detects suspicious path edits.
- Guard detects repeated command/error loops.
- Events are auditable in local artifacts.

---

## 22. Add Checkpoint Steering

Labels: `guard`, `adapter`, `steering`

Goal:
Send steering messages at safe boundaries where an adapter supports it.

Depends on:
Issues 20, 21.

Deliverables:
- Add checkpoint abstraction for command completion, file writes, permission prompts, and stop events.
- Deliver messages through supported adapter surfaces.
- Fall back to prompt files when live steering is unavailable.

Acceptance criteria:
- Fake adapter tests verify delivery and fallback.
- Steering is never attempted for unsupported capabilities.
- Events record what was sent and why.

---

## 23. Add Managed Agent Wrapper Mode

Labels: `guard`, `wrapper`, `experimental`

Goal:
Optionally run or supervise an agent command and steer at safe checkpoints.

Depends on:
Issues 20, 21, 22.

Deliverables:
- Add `tailchase guard --agent <target>`.
- Add wrapper config for command, working directory, and max attempts.
- Stop on repeated failure, drift, test weakening, or budget exhaustion.
- Keep wrapper mode opt-in.

Acceptance criteria:
- Wrapper works with a fake command in tests.
- Stop rules are enforced.
- No automatic code merge or commit behavior exists.

---

## 24. Add Assisted Repair Loop

Labels: `automation`, `loop`, `safety`

Goal:
Run a conservative collect/bundle/prompt/agent-attempt loop with strict stop rules.

Depends on:
Issues 6, 8, 21, 23.

Deliverables:
- Add `tailchase run-loop --agent <target> --max-attempts <n>`.
- Collect new evidence after each attempt.
- Compare attempts and generate delta context.
- Stop on repeat, drift, safety violation, or budget exhaustion.

Acceptance criteria:
- Loop works with fake agent and fake collector.
- Max attempts is enforced.
- Every prompt and decision is recorded.

---

## 25. Add Reporting and Metrics

Labels: `reporting`, `metrics`

Goal:
Generate local reports for evidence reduction, repeated failures, prompt size, safety findings, and repair outcomes.

Depends on:
Issues 3, 5, 8, 24 optional.

Deliverables:
- Add `report.md` per run.
- Add `tailchase cost report`.
- Track raw bytes vs included bytes and repeated context avoided.
- Summarize safety and attempt outcomes.

Acceptance criteria:
- Reports are deterministic and local.
- Metrics are computed from stored artifacts.
- Tests cover report generation from fixture runs.

---

## 26. Add Tournament Evaluator

Labels: `future`, `evaluation`

Goal:
Compare candidate branches from different agents using tests, diff risk, goal drift, repeated failures, and context cost.

Depends on:
Issues 8, 21, 25.

Deliverables:
- Add `tailchase tournament <branch-a> <branch-b>`.
- Compare test outcomes, changed paths, dependency changes, safety findings, and bundle quality.
- Produce a local evaluation report.

Acceptance criteria:
- Works on fixture branches in tests.
- Does not mutate candidate branches.
- Clearly reports evaluation criteria and winner rationale.
