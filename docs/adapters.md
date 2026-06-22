# Adapter Capabilities

Tailchase adapter capabilities are explicit and conservative.

Capability levels:

- `artifact`: write local files or stdout for manual use
- `queued`: prepare messages for later delivery
- `checkpoint`: deliver messages at safe command/file boundaries
- `hook_mcp`: expose context through a local hook or MCP surface
- `wrapper`: supervise an agent process

Current targets:

- `codex`: `artifact`, `hook_mcp`
- `claude-code`: `artifact`, `hook_mcp`
- `copilot`: `artifact`
- `cursor-vscode`: `artifact`
- `generic`: `artifact`

Unsupported modes fail with an error and point back to the `artifact` fallback. Live steering is added only by later guard/wrapper issues.
