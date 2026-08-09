# mastracode

## Identity

- **Binary / npm package**: `mastracode` (`bin.mastracode` → `dist/cli.js` in `mastracode/tui/package.json`, version `0.33.0-alpha.6` — pre-1.0 alpha, actively developed). Vendor: **Mastra** (mastra-ai), Apache-2.0.
- **Install**: `npm install -g mastracode` or `npx mastracode`; requires Node.js ≥ 22.13.0 per docs, but `package.json` engines says `>=22.19.0` (docs/source mismatch, both official).
- **Repo**: lives in the `mastra-ai/mastra` monorepo under `mastracode/` (packages: `tui` = the published CLI, `sdk` = `@mastra/code-sdk` runtime, plus Factory web product).
- **Docs URLs read**:
  - https://code.mastra.ai/configuration (dedicated docs site; mirrors the MDX below)
  - https://raw.githubusercontent.com/mastra-ai/mastra/main/docs/src/mastra-code/index.mdx
  - https://raw.githubusercontent.com/mastra-ai/mastra/main/docs/src/mastra-code/configuration.mdx
  - https://raw.githubusercontent.com/mastra-ai/mastra/main/docs/src/mastra-code/customization.mdx
  - https://github.com/mastra-ai/mastra/blob/main/mastracode/README.md (contributor guide)
  - Source-of-truth files read (all raw.githubusercontent.com/mastra-ai/mastra/main): `mastracode/tui/package.json`, `mastracode/sdk/src/constants.ts`, `mastracode/sdk/src/hooks/{config,types,executor}.ts`, `mastracode/sdk/src/mcp/config.ts`, `mastracode/sdk/src/utils/{slash-command-loader,project}.ts`, `mastracode/sdk/src/agents/{instructions.ts,prompts/agent-instructions.ts,__tests__/build-skill-paths.test.ts}`, `mastracode/sdk/src/onboarding/settings.ts`, `packages/core/src/workspace/skills/{schemas,workspace-skills,local-skill-source}.ts`

## Config dirs

- **Project config dir**: `.mastracode/` at project root (constant `DEFAULT_CONFIG_DIR = '.mastracode'`, `sdk/src/constants.ts`). Contains `mcp.json`, `hooks.json`, `commands/`, `skills/`, `database.json`, `AGENTS.md`/`CLAUDE.md`. The dir name is a parameter (`configDirName`) in the SDK, but the shipped CLI uses `.mastracode`.
- **Global/user config dir**: `~/.mastracode/` (`hooks.json`, `mcp.json`, `commands/`, `skills/`, `database.json`, instructions) — configuration.mdx + `hooks/config.ts`, `mcp/config.ts`.
- **App data dir (separate!)**: platform data dir holds `settings.json`, `auth.json`, `mastra.db`, vector DB (`sdk/src/utils/project.ts#getAppDataDir`): macOS `~/Library/Application Support/mastracode`, Linux `$XDG_DATA_HOME/mastracode` or `~/.local/share/mastracode`, Windows `%APPDATA%/mastracode`; override `MASTRA_APP_DATA_DIR`. Docs say "global `settings.json`" without a path — source places it in the app data dir, NOT `~/.mastracode/`.
- **Extra instructions-only locations**: `~/.config/mastracode/` and `~/.config/claude/` are scanned only for instruction files (`agent-instructions.ts`).
- **Precedence**: project overrides global by name for MCP servers and slash commands; hooks **append** (global runs first, then project); instructions are first-match-wins per scope with both global and project loaded (global first).

## Instructions file

- Filenames: `AGENTS.md` **or** `CLAUDE.md` — `AGENTS.md` wins when both exist at the same location (configuration.mdx §Agent instructions; `agent-instructions.ts`: `INSTRUCTION_FILES = ['AGENTS.md', 'CLAUDE.md']`).
- Project lookup order (first match wins): project root → `.claude/` → `.mastracode/`. Global lookup order: `~/.claude/` → `~/.mastracode/` → `~/.config/claude/` → `~/.config/mastracode/`.
- **Heading convention inside the file**: none required — the file body is injected verbatim. When composing the system prompt, mastracode wraps each file as `<!-- Global|Project instructions from <path> -->` under a `# Agent Instructions` heading (`formatAgentInstructions`). No import/include syntax (no `@file` includes) is documented or present in the loader.

## Skills

**Supported.** Directory per skill containing `SKILL.md` (Agent Skills spec layout; parsed with gray-matter, validated in `packages/core/src/workspace/skills/schemas.ts`).

- **Scan paths, highest→lowest** (configuration.mdx §Skills; confirmed by `build-skill-paths.test.ts`): `.mastracode/skills/` → `.claude/skills/` → `.agents/skills/` (project), then `~/.mastracode/skills/` → `~/.claude/skills/` → `~/.agents/skills/` (global). Symlinked skill dirs are resolved (project symlinks must stay inside the project root; global symlinks may point anywhere).
- **Exact frontmatter keys** (schema-validated; unknown keys are dropped when constructing `SkillMetadata`):
  - `name` (required): 1–64 chars, lowercase `[a-z0-9-]` only, no leading/trailing/consecutive hyphens, **must match the directory name**.
  - `description` (required): 1–1024 chars, non-whitespace.
  - `license` (optional string).
  - `compatibility` (optional, any value; spec suggests ≤500 chars).
  - `user-invocable` (optional boolean, default true) — **Mastra-specific**: `false` hides the skill from `/skills`, `/skill/<name>` autocomplete and direct activation; agent can still auto-load it.
  - `metadata` (optional record, arbitrary values) — **`metadata.goal: true` is a Mastra Code extension** exposing the skill as `/goal/<skill-name>`.
- **Not supported**: `allowed-tools` (and other Claude keys like `disable-model-invocation`) appear nowhere in the schema or loader — silently ignored.
- **Size limits** (recommendations from the Agent Skills spec, emitted as warnings): ≤5000 instruction tokens, ≤500 lines; name ≤64, description ≤1024 chars.
- Invocation: `/skills` lists, `/skill/<name>` activates.

## Hooks

**Supported.** JSON file, event-name-keyed (configuration.mdx §Hooks; `sdk/src/hooks/{config,types,executor}.ts`).

- **Config files**: `~/.mastracode/hooks.json` (global, runs first) and `.mastracode/hooks.json` (project, appended). Same event's hooks execute in order.
- **Format**: top-level keys are event names mapping **directly** to arrays of hook definitions (no Claude-style `{"hooks": {Event: [{matcher, hooks: []}]}}` nesting):
  ```json
  { "PreToolUse": [ { "type": "command", "command": "node validate.js", "matcher": { "tool_name": "execute_command" }, "timeout": 5000, "description": "..." } ] }
  ```
- **Event names** (exact): `PreToolUse`, `PostToolUse`, `Stop`, `UserPromptSubmit`, `SessionStart`, `SessionEnd`, `Notification`, `AgentStart`, `AgentEnd`, `PermissionRequest`, `PermissionResult`, `Interrupt`, `SubagentStart`, `SubagentEnd`. Blocking: only `PreToolUse`, `Stop`, `UserPromptSubmit`.
- **Hook schema**: `type` (only `"command"`), `command` (string, required; run via `/bin/sh -c`, `cmd /c` on Windows), `matcher` (optional object; only key is `tool_name`, a **regex** matched against the tool name, PreToolUse/PostToolUse only), `timeout` (ms, default 10000, SIGKILL on expiry), `description` (display only).
- **stdin JSON envelope** (snake_case): base `session_id`, `cwd`, `hook_event_name`, optional `run_id` (present during a run). Per event: tool events add `tool_name`, `tool_input`, `tool_output?`, `tool_error?`; `UserPromptSubmit` adds `user_message`; `Stop` adds `stop_reason` (`complete|aborted|error`), `assistant_message?`; `Notification` adds `reason` (`agent_done|ask_question|tool_approval|plan_approval|sandbox_access`), `message?`; `AgentEnd` adds `stop_reason` (`...|suspended`); `PermissionRequest/Result` add `permission_kind` (`tool_approval|sandbox_access|plan_approval`), `tool_call_id`, `tool_name`, `tool_input?`, plus `decision` (`approved|declined|dismissed|auto_approved|auto_declined`) on Result; `Interrupt` adds `reason` (`user_interrupt|goal_judge_interrupt|process_sigint`); `SubagentStart/End` add `tool_call_id`, `agent_type`, `task`/`result`, `is_error`, `duration_ms`, `model_id?`, `forked?`. Process env gets `MASTRA_HOOK_EVENT=<event>`.
- **Output contract**: stdout may be one JSON object `{ "decision": "allow"|"block", "reason": string, "additionalContext": string }`. **Exit code 2 on a blocking event blocks** (reason from stdout `reason`, else stderr); exit 0 = ok; other non-zero = warning only. `additionalContext` is concatenated across hooks and injected.

## Subagents

**Not file-based.** No `.mastracode/agents/` directory exists anywhere in docs or source. Subagents are (a) built-in Explore/Plan/Execute, (b) overridden programmatically via `createMastraCode({ subagents: [{ id, name, description, instructions }] })` (customization.mdx §Custom subagents), (c) tuned via `/subagents` which sets per-agent-type model defaults persisted in `settings.json` → `models.subagentModels` (index.mdx slash-command table; `onboarding/settings.ts`). There is no markdown/frontmatter subagent format to target.

## Commands

**Supported** as markdown prompt templates (`sdk/src/utils/slash-command-loader.ts`; configuration.mdx §Custom slash commands).

- **Scan dirs, highest→lowest**: `.mastracode/commands/` → `.claude/commands/` → `.opencode/command/` (project), then `~/.mastracode/commands/` → `~/.claude/commands/` → `~/.opencode/command/` (global). Same-name commands: later (higher-priority) source wins. Note opencode uses singular `command/`.
- **Format**: `.md` files, recursive. Optional YAML frontmatter keys: `name` (else derived from path), `description`, `namespace`, `goal` (`goal: true` exposes it as `/goal/<name>`). Body is the prompt template.
- **Naming**: subdirectories namespace with colons — `git/commit.md` → `/git:commit`.
- **Template variables**: `$ARGUMENTS`, positional `$1`, `$2`, `@filename` (file-content injection), `` !command `` (shell output injection).

## Rules

**Not supported.** No rules directory or `.mdc`-style mechanism exists in the CLI docs or SDK source. (The monorepo's `mastracode/factory` server product has a "rules" feature, but that is the hosted Factory backend, not the `mastracode` CLI.) Closest equivalents: `AGENTS.md`/`CLAUDE.md` instructions and skills.

## MCP

- **Config files & precedence** (highest→lowest, merged by server name; source `sdk/src/mcp/config.ts` header): `.mastracode/mcp.json` (project) > `.mcp.json` (project root, Claude-Code-compatible) > `~/.mastracode/mcp.json` (global) > `.claude/settings.local.json` (top-level `mcpServers` key; lowest). **Doc/source mismatch**: the docs table omits project-root `.mcp.json`; the source reads it.
- **Format**: JSON `{ "mcpServers": { "<name>": { ... } } }`.
- **Server shapes**:
  - stdio: `{ "command": string (required), "args": string[]?, "env": {KEY: value}? }`
  - HTTP (Streamable HTTP or SSE): `{ "url": string (required), "headers": {H: value}?, "oauth": { "redirectUrl"? | "callbackPort"? (mutually exclusive; callbackPort synthesizes `http://localhost:<port>/callback`), "clientName"?, "scopes"?: string[], "clientId"?, "clientSecret"? }? }`
  - Both `command` and `url`, or neither → server skipped with a reason (reported at startup and in `/mcp`).
- **Env handling**: `${VAR}`, `${VAR:-default}`, and bare `$VAR` are expanded from `process.env` inside stdio `env` values, HTTP `header` values, and `url` (matching Claude Code's `.mcp.json` behavior). No TOML support; Codex `config.toml` MCP blocks must be converted to this JSON shape.
- MCP tools are namespaced `serverName_toolName`; `/mcp disable|enable <name|all> [--global]` persists disable state.

## Translation hazards

What a canonical `.agents` store must drop/rename/reformat to target mastracode:

1. **Skills — drop unsupported frontmatter keys**: only `name`, `description`, `license`, `compatibility`, `user-invocable`, `metadata` survive; everything else (Claude's `allowed-tools`, `disable-model-invocation`, `context`, `agent`, `model`, etc.) is silently discarded. Conversely `user-invocable` and `metadata.goal` are mastracode-only and must be dropped when exporting back out.
2. **Skills — naming rules enforced**: `name` must be 1–64 chars, `[a-z0-9-]`, no leading/trailing/double hyphens, and **must equal the directory name**; `description` ≤1024 chars. A canonical store with looser names (uppercase, underscores) will fail validation. Layout must be `<skillsdir>/<name>/SKILL.md`.
3. **Skills — multi-root layout**: mastracode natively scans `.agents/skills/` (project and `~/.agents/skills/`) at lower priority than `.mastracode/skills/` — a canonical `.agents/skills` tree works as-is, but a same-named skill in `.mastracode/skills` or `.claude/skills` shadows it (tie-break: local > managed > external; same-type duplicates are an error).
4. **Hooks — file shape differs from Claude Code**: top-level event name → array of hooks directly. Claude's nested `[{"matcher": "<string>", "hooks": [{"type","command"}]}]` must be flattened, and Claude's string `matcher` must become `matcher: { tool_name: "<regex>" }` (regex, not glob; tool names differ too, e.g. `execute_command` not `Bash`).
5. **Hooks — event-name mismatches**: mastracode has no `PreCompact`, `Setup`, `TeammateIdle`, etc.; its extras (`AgentStart`, `AgentEnd`, `PermissionRequest`, `PermissionResult`, `Interrupt`) have no Claude equivalent. Codex hooks (if any) don't share this envelope. Unmappable events must be dropped.
6. **Hooks — stdin envelope is snake_case and minimal**: `session_id`/`cwd`/`hook_event_name`/`run_id`/`tool_name`/`tool_input`/... — no `transcript_path`, no `hook_event_name`-adjacent Claude fields, no Codex patch/`file_path` envelope. agentlink's `guard`/`remind` would need a mastracode envelope reader (paths live inside `tool_input`, shape varies per tool, e.g. `execute_command` → `{command}`).
7. **Hooks — output contract**: blocking via **exit code 2** (not Claude's JSON-only `permissionDecision`); stdout JSON keys are `decision: allow|block`, `reason`, `additionalContext` — not Claude's `hookSpecificOutput`/`continue`/`suppressOutput`. Default timeout 10s vs Claude's 60s.
8. **Commands — frontmatter subset**: keep `name`/`description`; `namespace` is rarely needed (path-derived); `goal` is mastracode-only (drop on export). Drop Claude command keys (`allowed-tools`, `argument-hint`, `model`). Template variables `$ARGUMENTS`/`$1`/`@file`/`!cmd` are Claude-compatible; colon-namespacing (`git:commit`) matches Claude.
9. **MCP — JSON only, `mcpServers` table**: Codex TOML `[mcp_servers.x]` must be converted; server entries limited to `command/args/env` or `url/headers/oauth` — no `type`/`transport` field (inferred from presence of `command` vs `url`; both = skipped). `env`/header/url values may use `${VAR}`/`${VAR:-default}`/`$VAR`. OAuth: `callbackPort` XOR `redirectUrl`.
10. **Instructions**: filename must be `AGENTS.md` or `CLAUDE.md` (AGENTS.md wins); no heading requirement and no include/import syntax to preserve. Only ONE project file and ONE global file are loaded (first match wins) — a canonical store that composes multiple instruction files must inline them.
11. **Subagents/rules**: no file-based target exists — canonical `.agents/agents/*` subagent files and any rules directory have nowhere to land; they'd have to be folded into instructions/skills or dropped.
12. **Config dir naming**: project dir is `.mastracode/` (not `.mastra/`, not `.mastra-code/`); global mirror `~/.mastracode/`; `settings.json`/auth/db live in a separate platform app-data dir (`~/Library/Application Support/mastracode` on macOS) that is machine-local state, not syncable config.