# qodercli

## Identity — binary, vendor, install, docs URLs (list every URL you actually read)

- **Binary**: `qodercli` (verify with `qodercli --version`). npm package: `@qoder-ai/qodercli` (Node.js ≥ 20 for npm installs). Source: https://docs.qoder.com/cli/installation.md
- **Vendor**: Qoder (Alibaba). Actively maintained — release notes show v1.1.17 dated 2026-08-07, with near-daily releases. Source: https://docs.qoder.com/release-notes/qoder-cli.md
- **Install**: `curl -fsSL https://qoder.com/install | bash` (macOS/Linux); `irm https://qoder.com/install.ps1 | iex` (Windows PowerShell); `npm install -g @qoder-ai/qodercli` (`@latest` stable, `@beta` native preview). Windows arm64 unsupported. Source: https://docs.qoder.com/cli/installation.md
- **Docs base**: https://docs.qoder.com (CLI section under `/cli/*`; every page also servable as `.md`; full index at https://docs.qoder.com/llms.txt)
- **Pages actually read** (all official):
  - https://docs.qoder.com/ (→ /product-series/what-is-qoder)
  - https://docs.qoder.com/llms.txt
  - https://docs.qoder.com/cli/overview.md
  - https://docs.qoder.com/cli/installation.md
  - https://docs.qoder.com/cli/config-scope.md
  - https://docs.qoder.com/cli/settings.md
  - https://docs.qoder.com/cli/settings-reference.md
  - https://docs.qoder.com/cli/memory.md
  - https://docs.qoder.com/cli/Skills.md
  - https://docs.qoder.com/cli/subagent.md
  - https://docs.qoder.com/cli/commands.md
  - https://docs.qoder.com/cli/hooks.md
  - https://docs.qoder.com/cli/hooks-reference.md
  - https://docs.qoder.com/cli/mcp-servers.md
  - https://docs.qoder.com/cli/mcp-reference.md
  - https://docs.qoder.com/cli/built-ins.md
  - https://docs.qoder.com/cli/builtins-reference.md
  - https://docs.qoder.com/cli/plugins-reference.md
  - https://docs.qoder.com/cli/troubleshoot-loading.md
  - https://docs.qoder.com/release-notes/qoder-cli.md
  - https://docs.qoder.com/extensions/skills.md

## Config dirs — project dir, global/user dir, discovery/precedence rules

- **Project dir**: `<project>/.qoder/` — always at project root. Contains `settings.json` (committable), `settings.local.json` (gitignored), `rules/`, `skills/`, `agents/`, `commands/`, `worktrees/`, `scheduled_tasks.json`. Additionally `<project>/AGENTS.md`, `<project>/AGENTS.local.md`, and `<project>/.mcp.json` live at the project **root**, outside `.qoder/`. Sources: https://docs.qoder.com/cli/config-scope.md, https://docs.qoder.com/cli/memory.md, https://docs.qoder.com/cli/mcp-servers.md
- **Global/user dir**: `~/.qoder/` — overridable via env var `QODER_CONFIG_DIR`. Holds `settings.json`, `AGENTS.md`, `rules/`, `skills/`, `agents/`, `commands/`, auth state, plugins, auto-memory (`~/.qoder/memory/`, `~/.qoder/projects/<project>/memory/`). Sources: https://docs.qoder.com/cli/installation.md, https://docs.qoder.com/cli/settings-reference.md, https://docs.qoder.com/cli/memory.md
- **Settings precedence** (low→high): built-in defaults < `~/.qoder/settings.json` < `<project>/.qoder/settings.json` < `<project>/.qoder/settings.local.json` < `--settings` CLI flag. Deep merge (objects field-wise; some arrays union-merged; scalars overridden). JSON format with `//` comments allowed; env-var references resolved at runtime. Source: https://docs.qoder.com/cli/settings.md
- **Folder Trust gate**: project/local settings (and project-level memory, agents, hooks, MCP) load only when the workspace is trusted; `security.folderTrust.enabled` defaults `true`. Untrusted → user-level config only. Sources: https://docs.qoder.com/cli/settings.md, https://docs.qoder.com/cli/troubleshoot-loading.md

## Instructions file — filename(s), location, heading/title conventions, import/include syntax if any

- **Filename**: `AGENTS.md` (default). Configurable via `context.fileName` (string or string array) in settings.json. Source: https://docs.qoder.com/cli/memory.md, https://docs.qoder.com/cli/settings-reference.md
- **Locations**: `~/.qoder/AGENTS.md` (user, cross-project), `<project>/AGENTS.md` (committable), `<project>/AGENTS.local.md` (private, not committed). Source: https://docs.qoder.com/cli/memory.md
- **Discovery**: upward search from cwd for `AGENTS.md` + `AGENTS.local.md` + `.qoder/rules/**/*.md`, stopping at the directory containing `.git` (`context.memoryBoundaryMarkers`, default `['.git']`); cap `context.discoveryMaxDirs` (default 200); `agentsMdExcludes` glob exclusion. Subdirectory memory is **not** preloaded — loaded on demand after the agent reads a file under that subdirectory. Source: https://docs.qoder.com/cli/memory.md, https://docs.qoder.com/cli/settings-reference.md
- **Heading/title conventions**: none documented. `/init` generates `AGENTS.md`; examples use plain topic headings (e.g. `# Development`). No branded title line analogous to `# Claude Code instructions`. Source: https://docs.qoder.com/cli/memory.md, https://docs.qoder.com/cli/commands.md
- **Import syntax**: `@path/to/file` inside AGENTS.md, resolved relative to the containing AGENTS.md; supports relative, absolute, and `~/` paths; `@...` inside inline code/code blocks is not an import; project-level memory may only import files inside the project boundary by default (external imports need approval/security settings); recursive expansion with a depth limit. `context.importFormat` = `tree`/`flat` controls expansion rendering. Source: https://docs.qoder.com/cli/memory.md, https://docs.qoder.com/cli/settings-reference.md

## Skills — supported? dir, file layout, EXACT frontmatter keys with meaning, which keys are unique to this tool, size/naming limits

- **Supported**: yes. Source: https://docs.qoder.com/cli/Skills.md
- **Dirs + layout**: `~/.qoder/skills/{skill-name}/SKILL.md` (user) and `.qoder/skills/{skill-name}/SKILL.md` (project). One directory per skill; `SKILL.md` required; optional sibling files (`REFERENCE.md`, `EXAMPLES.md`, `scripts/`, `templates/`). Skill dirs are scanned **recursively** (v1.1.9) and symlinks are supported (release notes). A `.agents/skills` compatibility source exists and is **enabled by default since v1.1.11** (toggle in `/settings`; exact settings.json key not documented). Sources: https://docs.qoder.com/cli/Skills.md, https://docs.qoder.com/release-notes/qoder-cli.md
- **Frontmatter keys** (only two documented; anything else is undocumented):

  | Key | Required | Meaning | Limits |
  |---|---|---|---|
  | `name` | Yes | Unique skill identifier | Lowercase letters, numbers, hyphens only; max 64 chars |
  | `description` | Yes | What it does + when to use; drives model auto-selection | Max 1024 chars |

  Source: https://docs.qoder.com/cli/Skills.md
- **Unique keys**: none beyond the Claude-compatible `name`/`description` pair — conversely, there is **no** `allowed-tools`, `license`, or `metadata` support documented. "Conditional Skill … activated only when the file path matches" is mentioned in troubleshooting, but the frontmatter key for it is undocumented. Source: https://docs.qoder.com/cli/troubleshoot-loading.md
- **Precedence**: built-in < plugin < project-level < user-level — **user-level wins name conflicts** (docs contradiction: the IDE-oriented page https://docs.qoder.com/extensions/skills.md says project-level takes priority; the two CLI pages say user-level overrides). Sources: https://docs.qoder.com/cli/Skills.md, https://docs.qoder.com/cli/troubleshoot-loading.md
- Skills are also manually invocable as `/{skill-name}`; internally converted to a special command type. Source: https://docs.qoder.com/cli/Skills.md

## Hooks — supported? config file + format, event names, matcher schema, command schema, stdin JSON envelope fields, output contract

- **Supported**: yes, JSON-only config. Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/hooks-reference.md
- **Config file + format**: top-level `"hooks"` object in `settings.json` at all three tiers (`~/.qoder/settings.json`, `.qoder/settings.json`, `.qoder/settings.local.json`); all three sources are **merged** (same-event hooks do not override). Plugins ship hooks via `hooks/hooks.json` in the same format. Shape: `hooks.<EventName> = [ { matcher?, async?, hooks: [ <entry> ] } ]`. Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/hooks-reference.md
- **Event names** (union of both pages; † = only in hooks-reference.md): `SessionStart`, `SessionEnd`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PermissionRequest`, `PermissionDenied`, `Stop`, `StopFailure`, `SubagentStart`, `SubagentStop`, `PreCompact`, `PostCompact`, `Notification`, `InstructionsLoaded`, `ConfigChange`, `CwdChanged`, `FileChanged`, `WorktreeCreate`, `WorktreeRemove`, `Elicitation`, `ElicitationResult`, `TaskCreated`†, `TaskCompleted`†, `TeammateIdle`†, `Setup`†. Matcher target varies per event: tool name (Pre/PostToolUse etc.), `source` (SessionStart: startup/resume/clear/compact/new; ConfigChange: user_settings/project_settings/local_settings/policy_settings/skills/agents), `reason` (SessionEnd), `trigger` (Pre/PostCompact: manual/auto), `error_type` (StopFailure), agent type (SubagentStart/Stop), `notification_type`, `load_reason`, file basename (FileChanged), `mcp_server_name` (Elicitation*). Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/hooks-reference.md
- **Matcher schema**: omitted or `"*"` = match all; exact string; `|`-separated alternation (`"Write|Edit"`); regex (`"mcp__.*"`). Per-entry finer filter `if`: `"ToolName"` or `"ToolName(arg_pattern)"` where arg_pattern is a **glob** matched against the tool's primary argument (Bash `command`, file tools' `file_path`). Source: https://docs.qoder.com/cli/hooks.md
- **Command schema** (entry fields by `type`):
  - `command`: `command` (req), `timeout` (s, default 600), `shell` (`bash`|`powershell`), `env` (object), `if`, `async`, `asyncRewake`, `rewakeMessage`, `rewakeSummary` (≤300 chars), `once`, `statusMessage`, `args` (argv array → exec form, no shell). Group level adds `matcher`, `async`.
  - `http`: `url` (req), `headers` (values support `${ENV_VAR}` interpolation), `allowedEnvVars` (whitelist), `timeout`, plus common fields. POSTs input JSON, expects HookOutput JSON back.
  - `prompt`: `prompt` (req), `model`, `timeout` (default 30); isolated single-turn LLM returns `{ok, reason}`; `ok=false` blocks.
  - `agent`: `prompt` (req, `$ARGUMENTS` placeholder = hook input JSON), `tools`, `maxTurns` (default 50), `model`, `timeout` (default 60); sub-agent must call `StructuredOutput` with `{ok, reason?}`.
  - hooks-reference.md additionally lists an entry-level `name` field. Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/hooks-reference.md
- **stdin JSON envelope** — common fields: `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `permission_mode` (when provided), `agent_id` (when provided), `agent_type` (when provided). Tool events add `tool_name`, `tool_input`, `tool_use_id`, plus `tool_response` (PostToolUse) or `error`/`error_type`/`is_interrupt` (PostToolUseFailure); MCP tools add `mcp_context` (`server_name`, `tool_name`, connection info) and `original_request_name`. Event-specific extras: `prompt`, `source`, `model`, `reason`, `stop_hook_active`, `last_assistant_message`, `compact_summary`, `file_path`, `old_cwd`/`new_cwd`, `worktree_path`, `requested_schema`, etc. Hook subprocess env: `QODER_PROJECT_DIR`, `QODER_PLUGIN_ROOT`, `QODER_PLUGIN_DATA`. Source: https://docs.qoder.com/cli/hooks.md
- **Output contract**: exit `0` = success (stdout parsed as JSON if valid); exit `2` = blocking, stderr fed back to the Agent/user (only for block-capable events); any other code = non-blocking error, stderr logged. Stdout JSON fields: `continue` (false stops), `stopReason`, `suppressOutput`, `systemMessage` (user-only, not model context), `decision` (`allow`|`deny`; deny ≡ exit 2), `reason`, `hookSpecificOutput` — which **must include `hookEventName`** or the whole output is rejected. Event-specific subfields: PreToolUse → `permissionDecision` (`allow`/`deny`/`ask`), `permissionDecisionReason`, `updatedInput`, `additionalContext`; PostToolUse → `updatedToolOutput`, `updatedMCPToolOutput`, `additionalContext`; PermissionRequest → `decision` object with `behavior: "allow"|"deny"` (+`updatedInput`/`updatedPermissions` or `message`/`interrupt`); Stop/SubagentStop → `clearContext`; Elicitation* → `action`/`content`; WorktreeCreate → path on stdout or `hookSpecificOutput.worktreePath`. Plain-text stdout is injected as context only for `SessionStart`/`UserPromptSubmit`. Source: https://docs.qoder.com/cli/hooks.md
- Subagent frontmatter `hooks` scope events `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `Stop`, `SubagentStart`, `SubagentStop`, `Notification` to that subagent; inside a subagent `Stop` is remapped to `SubagentStop`. Source: https://docs.qoder.com/cli/subagent.md

## Subagents — dir + file format + frontmatter

- **Dirs**: `.qoder/agents/*.md` (project), `~/.qoder/agents/*.md` (user), plus `--agents '<json>'` flag (session-only, highest priority). Flat `.md` files; **filename does not define the name** — frontmatter `name` does. Priority (low→high): Built-in < User < Project < Plugin < Flag — note **project overrides user** (opposite of skills/commands). Shadowed entries marked in `qodercli agents list`. Source: https://docs.qoder.com/cli/subagent.md
- **File format**: Markdown with YAML frontmatter; body = system prompt. Unknown frontmatter fields are ignored. Full field reference (req = required):

  | Field | Values/notes |
  |---|---|
  | `name` (req) | non-empty string |
  | `description` (req) | drives selection |
  | `tools` | string or string[]; supports `*`, `mcp__server__tool`, `mcp__*`, `Agent(name1, name2)` |
  | `disallowedTools` | denylist applied after allowlist |
  | `mcpServers` | name array, inline server object, or mix (inline uses **snake_case** `include_tools`/`exclude_tools`) |
  | `model` | `inherit` (default), `auto`, `lite`, `efficient`, `performance`, or any name/alias |
  | `effort` | `low`/`medium`/`high`/`xhigh`/`max` or positive int |
  | `permissionMode` | `default`/`acceptEdits`/`bypassPermissions`/`dontAsk`/`auto`/`plan` (camelCase; `yolo` tolerated as alias) |
  | `maxTurns` / `timeoutMins` | positive ints (turn/runtime caps) |
  | `temperature` | number |
  | `initialPrompt` | used only when run via `--agent` as session agent |
  | `skills` | restrict usable skills |
  | `memory` | `user`/`project`/`local` (auto-memory scope) |
  | `background` | bool — launch in background by default |
  | `isolation` | `worktree` — run in separate git worktree |
  | `kind` | only `local` currently |
  | `hooks` | subagent-scoped hook config (no string shorthand) |
  | `color` | red/blue/green/yellow/purple/orange/pink/cyan |

  Source: https://docs.qoder.com/cli/subagent.md
- `settings.json` → `agents.overrides.<name>` can override (enabled, tools, runConfig.maxTurns/maxTimeMinutes, modelConfig, mcpServers) but **cannot create** agents. Plugin-supplied subagents get `hooks`, `mcpServers`, `permissionMode` stripped by safety policy. Sources: https://docs.qoder.com/cli/subagent.md, https://docs.qoder.com/cli/settings-reference.md

## Commands — slash-command dir + format

- **Dirs**: `.qoder/commands/<name>.md` (project), `~/.qoder/commands/<name>.md` (user). Only **Prompt-type** commands are extensible (TUI-type built-ins are not). Source: https://docs.qoder.com/cli/commands.md
- **Format**: Markdown + YAML frontmatter. `description` (required, multiline via YAML `|`); `name` (optional, **display-only** — invocation name always derives from the file path). Body = prompt submitted to the conversation. Source: https://docs.qoder.com/cli/commands.md
- **Naming**: lowercase + hyphens recommended; subdirectories namespace with `:` (`commands/git/commit.md` → `/git:commit`); name segments preserved as-is. If a directory contains `SKILL.md`, the whole directory registers as a single command and sibling `.md` files are ignored. Source: https://docs.qoder.com/cli/commands.md
- **Precedence**: **user-level overrides project-level** for same-name commands (same inversion as skills). Reload via `/commands`. Source: https://docs.qoder.com/cli/commands.md

## Rules — rules dir + format (if distinct from instructions)

- **Dirs**: `<project>/.qoder/rules/**/*.md` (committable, discovered upward at any workspace depth) and `~/.qoder/rules/**/*.md` (user). Plain Markdown files with optional YAML frontmatter; frontmatter is compatible with Qoder Desktop rules. Source: https://docs.qoder.com/cli/memory.md
- **Frontmatter keys**:

  | Key | Values | Meaning |
  |---|---|---|
  | `trigger` | `always_on` / `manual` / `model_decision` / `glob` | activation mode; takes precedence over `alwaysApply` |
  | `alwaysApply` | `true`/`false` | compat alias: true ≡ `trigger: always_on`, false ≡ `trigger: manual` |
  | `description` | string | required (non-empty) for `model_decision` |
  | `glob` | glob or glob[] | required for `trigger: glob` |
  | `paths` | glob or glob[] | equivalent to `trigger: glob` + `glob` |

  No frontmatter ⇒ always active. Globs are gitignore-style, matched relative to the directory containing `.qoder/` (project rules) or the current project root (user rules); `glob`/`paths` are routing metadata, not injected into context. Loaded rules are file-watched live. Source: https://docs.qoder.com/cli/memory.md

## MCP — config file, format, server-table shape, env handling

- **Config files** (all JSON): `mcpServers` top-level key in `~/.qoder/settings.json` (user), `<project>/.qoder/settings.json` (project), `<project>/.qoder/settings.local.json` (local — default scope of `qodercli mcp add`); `<project>/.mcp.json` (project-shared, requires top-level `mcpServers` key); plugin `.mcp.json` or `mcp.json` (dotfile wins if both); CLI `--mcp-config <path>` / `--settings`. Same-name override order (later wins): user → project settings.json → project `.mcp.json` → local → CLI arg. Sources: https://docs.qoder.com/cli/mcp-servers.md, https://docs.qoder.com/cli/mcp-reference.md, https://docs.qoder.com/cli/plugins-reference.md
- **Server-table shape**: `mcpServers: { "<name>": { ... } }`. Per transport:
  - `stdio` (default): `command` (string), `args` (string[]), `env` (object), `cwd` (string)
  - `sse`: `url`, `type: "sse"`, `headers` (object)
  - `http` / `streamable-http`: `url`, `type: "http"`, `headers`
  - `ws`: `tcp` (object with host/port), `type: "ws"`
  - `sdk`: in-process built-in

  Common optional fields: `type`, `timeout` (**milliseconds**), `description`, `trust` (skip tool confirmation), `includeTools` / `excludeTools` (camelCase string[]), `disabled`, `alwaysAllow` (string[]), `oauth` (`enabled`, `clientId`, `clientSecret`, `authorizationUrl`, `tokenUrl`, `scopes`, `callbackPort`). Source: https://docs.qoder.com/cli/mcp-reference.md
- **Env handling**: `env` is a plain `{KEY: "value"}` object passed to the stdio subprocess. `${ENV_VAR}` interpolation is documented for hook `http.headers` gated by `allowedEnvVars`; no equivalent interpolation is documented for MCP `env`/`headers`. Allow/exclude via `mcp.allowed` / `mcp.excluded`; lazy loading via `mcp.lazyLoad` or `QODER_MCP_LAZY=1`. Sources: https://docs.qoder.com/cli/mcp-reference.md, https://docs.qoder.com/cli/hooks.md
- **Approval gate**: project-level servers require individual approval unless `mcp.enableAllProjectMcpServers: true` or listed in `mcp.enabledProjectMcpServers`. Tool names: `mcp__<server>__<tool>`. Sources: https://docs.qoder.com/cli/mcp-reference.md, https://docs.qoder.com/cli/mcp-servers.md

## Translation hazards — every concrete thing a canonical `.agents` store must drop/rename/reformat to target this agent

**Config dirs / layout**
- Project config dir is `.qoder/` (not `.claude/`/`.codex/`); user dir `~/.qoder/`, relocatable via `QODER_CONFIG_DIR`. Instructions (canonical `AGENTS.md`), `.mcp.json`, and `AGENTS.local.md` live at the **project root**, not under `.qoder/`. Sources: https://docs.qoder.com/cli/config-scope.md, https://docs.qoder.com/cli/memory.md
- Project-level everything is gated by Folder Trust — freshly synced project config silently won't load until the directory is trusted. Source: https://docs.qoder.com/cli/settings.md
- `.agents/skills` is a native compatibility source, enabled by default since v1.1.11 — a canonical `.agents/skills` tree can be consumed directly with no symlink, for skills only. All other canonical dirs (`agents/`, `commands/`, `rules/`) must be materialized under `.qoder/`. Source: https://docs.qoder.com/release-notes/qoder-cli.md

**Instructions**
- Filename is `AGENTS.md` — already matches the canonical name; do not rename to `CLAUDE.md`. Optional private tier `AGENTS.local.md` has no Claude/Codex counterpart. Source: https://docs.qoder.com/cli/memory.md
- No branded title/heading convention → agentlink's instructions normalizer needs no Qoder heading pattern; any `# …` heading is user content, do not rewrite.
- Import syntax `@path` is Claude-compatible, but project-level imports are confined to the project boundary — canonical files importing outside the tree need approval or fail. Upward discovery stops at `.git`; home-dir fallback chains aren't followed past the boundary. Source: https://docs.qoder.com/cli/memory.md

**Skills**
- Frontmatter: keep only `name` + `description`; **drop** Claude keys (`allowed-tools`, `license`, `metadata`, etc.) — undocumented/unsupported. Enforce `name` = `[a-z0-9-]+`, ≤64 chars; `description` ≤1024 chars. Source: https://docs.qoder.com/cli/Skills.md
- Layout `.qoder/skills/<name>/SKILL.md` matches Claude's dir-per-skill shape 1:1 (tree copy works).
- **Precedence inversion**: user-level skills beat project-level (docs contradict: CLI pages say user wins, IDE page says project wins). A canonical project-level sync can be silently shadowed by a same-name user skill; consider name-mangling or user-level install. Sources: https://docs.qoder.com/cli/Skills.md, https://docs.qoder.com/cli/troubleshoot-loading.md, https://docs.qoder.com/extensions/skills.md

**Hooks**
- Config is embedded JSON under `"hooks"` in `settings.json` — no standalone project hooks file (except plugin `hooks/hooks.json`); merging canonical hooks means editing settings.json without clobbering other keys. Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/plugins-reference.md
- Event names: Claude's core set (`PreToolUse`, `PostToolUse`, `SessionStart`, `SessionEnd`, `UserPromptSubmit`, `Stop`, `SubagentStop`, `SubagentStart`, `PreCompact`, `Notification`) maps 1:1. Qoder-only events (`PostToolUseFailure`, `PermissionRequest`, `PermissionDenied`, `StopFailure`, `PostCompact`, `InstructionsLoaded`, `ConfigChange`, `CwdChanged`, `FileChanged`, `WorktreeCreate`, `WorktreeRemove`, `Elicitation`, `ElicitationResult`, `TaskCreated`, `TaskCompleted`, `TeammateIdle`, `Setup`) must be dropped when translating canonical→other agents. Sources: https://docs.qoder.com/cli/hooks.md, https://docs.qoder.com/cli/hooks-reference.md
- stdin envelope is Claude-shaped (`session_id`, `transcript_path`, `cwd`, `hook_event_name`, `permission_mode`, `tool_name`, `tool_input`, `tool_response`) — agentlink's `guard`/`remind` readers for `file_path` inside `tool_input` work unchanged (Qoder file tools use `tool_input.file_path`; Bash uses `tool_input.command`). Qoder extras (`agent_id`, `agent_type`, `mcp_context`, `original_request_name`, `is_interrupt`) can be ignored. Source: https://docs.qoder.com/cli/hooks.md
- Output contract: exit 0/2/other semantics match Claude, and `hookSpecificOutput` requires `hookEventName` (same as Claude). Qoder extras to drop/rename when retargeting: entry fields `if`, `async`, `asyncRewake`, `rewakeMessage`, `rewakeSummary`, `once`, `statusMessage`, `shell`, `args` (exec form), group `async`; hook types `prompt` and `agent` (no Claude/Codex equivalent — drop or rewrite as `command`); `updatedToolOutput`/`updatedMCPToolOutput` (PostToolUse), `clearContext`, `worktreePath`. Source: https://docs.qoder.com/cli/hooks.md
- Env vars in hook commands: rename `CLAUDE_PROJECT_DIR` → `QODER_PROJECT_DIR`; `QODER_PLUGIN_ROOT`/`QODER_PLUGIN_DATA` exist only for plugin hooks. Under PowerShell the CLI pre-substitutes `${...}`; under bash the shell expands them. Source: https://docs.qoder.com/cli/hooks.md
- Subagent-frontmatter `hooks` remap `Stop` → `SubagentStop` inside subagents — semantics differ from a top-level `Stop` hook. Source: https://docs.qoder.com/cli/subagent.md

**Subagents**
- Map canonical agent frontmatter to Qoder's schema: keep `name`, `description`, `tools`, `disallowedTools`, `model`; rename Claude-isms to Qoder keys (`maxTurns`, `timeoutMins`, `permissionMode`, `mcpServers`, `isolation: worktree`, `effort`, `skills`, `initialPrompt`, `background`, `memory`, `temperature`, `color`, `kind: local`, `hooks`). Unknown keys are ignored (safe but silently inert). Filename is irrelevant — `name` frontmatter is authoritative, so canonical stores keyed by filename must write a matching `name`. Source: https://docs.qoder.com/cli/subagent.md
- `permissionMode` uses camelCase (`acceptEdits`, `bypassPermissions`, `dontAsk`) while settings.json `general.defaultPermissionMode` uses snake_case (`accept_edits`, `bypass_permissions`, `dont_ask`) — translate per surface, not globally. Sources: https://docs.qoder.com/cli/subagent.md, https://docs.qoder.com/cli/settings-reference.md
- Inline `mcpServers` in agent frontmatter uses snake_case `include_tools`/`exclude_tools`, but settings.json MCP uses camelCase `includeTools`/`excludeTools` — case-flip required depending on target surface. Sources: https://docs.qoder.com/cli/subagent.md, https://docs.qoder.com/cli/mcp-reference.md
- Priority is Built-in < User < Project < Plugin < Flag (project beats user) — the opposite of skills/commands; don't apply one shadowing rule globally. Source: https://docs.qoder.com/cli/subagent.md

**Commands**
- `.qoder/commands/**/*.md`; frontmatter `description` required, `name` is display-only (invocation = file path) — drop any reliance on `name` for identity; drop Claude-only keys like `allowed-tools`, `argument-hint`, `model` (undocumented). Subdir namespacing uses `:` (`/git:commit`), same as Claude. **User overrides project** for same-name commands. A `SKILL.md` inside a commands directory suppresses its sibling `.md` files — never colocate skills and commands. Source: https://docs.qoder.com/cli/commands.md

**Rules**
- Target `.qoder/rules/**/*.md` + `~/.qoder/rules/**/*.md`. Map canonical rule frontmatter to `trigger` (`always_on`/`manual`/`model_decision`/`glob`) + `description` + `glob`; `alwaysApply` accepted as compat alias (Cursor-style `alwaysApply: true` works directly). Cursor's `globs` key must be renamed to `glob` (or `paths`); `model_decision` without non-empty `description`, and `glob` trigger without `glob`, silently disable the rule body. Source: https://docs.qoder.com/cli/memory.md

**MCP**
- Project-shared file is `.mcp.json` at project root with top-level `mcpServers` — identical shape to Claude; agentlink's MCP comparison maps directly (compare `command`/`args`/`url`/`type` + env key names; `env` is a plain object, values pass through). Also duplicated inside settings.json as `mcpServers` — syncing one without the other creates override-order surprises (local `settings.local.json` beats project `.mcp.json`). Sources: https://docs.qoder.com/cli/mcp-servers.md, https://docs.qoder.com/cli/mcp-reference.md
- Transport enum differs from Claude: `stdio` (default, omit `type`), `sse`, `http`/`streamable-http`, `ws` (uses a `tcp` object, not `url`), `sdk` (in-process). `timeout` is **milliseconds**. Extra keys `trust`, `includeTools`, `excludeTools`, `disabled`, `alwaysAllow`, `oauth` are Qoder-specific — drop when retargeting to Claude/Codex. Source: https://docs.qoder.com/cli/mcp-reference.md
- New project-level servers trigger a per-server approval prompt (`mcp.enableAllProjectMcpServers` / `mcp.enabledProjectMcpServers` to pre-approve) — canonical syncs into `.mcp.json` are not zero-touch. Source: https://docs.qoder.com/cli/mcp-reference.md

### Critical Files for Implementation

- `internal/config/schema.go` — add `qoder` endpoint kind: dirs `.qoder/` + `~/.qoder/`, root-level `AGENTS.md`/`AGENTS.local.md`/`.mcp.json`, per-artifact precedence (user-wins for skills/commands, project-wins for agents/settings).
- `internal/link/normalize.go` — skill normalizer: strip all frontmatter except `name`/`description` for qoder, enforce 64-char `[a-z0-9-]` name and 1024-char description; instructions normalizer needs no heading rewrite for qoder.
- `internal/link/mcp.go` — MCP compare: qoder project endpoint is `.mcp.json` (same shape as Claude's) plus `mcpServers` inside `.qoder/settings.json`; handle `type` enum (`http`/`sse`/`ws`/`sdk`) and millisecond `timeout`.
- `internal/hookinput/input.go` — hook envelope reader: qoder is Claude-shaped (`tool_input.file_path`, `hook_event_name`); accept Qoder extras (`agent_id`, `mcp_context`); exit-2/output contract parity.
- `internal/adopt/adopt.go` — adopt mapping: `.qoder/skills` → `.agents/skills` (optionally skipped: qoder reads `.agents/skills` natively since v1.1.11), `.qoder/{agents,commands,rules}` → `.agents/qoder/*`; root `AGENTS.md`/`.mcp.json` stay in place.

## Verified corrections (fact-checker pass)

- MCP env handling — was: 'no equivalent interpolation is documented for MCP env/headers' and 'env is a plain object, values pass through' → now: MCP configuration DOES support ${VAR} environment-variable expansion: settings.md states 'Environment variables can be referenced in values and will be resolved and replaced at runtime' for settings.json (which hosts mcpServers), and release notes v1.0.17 record a fix for 'MCP ${VAR} environment variable expansion' (https://docs.qoder.com/cli/settings.md, https://docs.qoder.com/release-notes/qoder-cli.md)
