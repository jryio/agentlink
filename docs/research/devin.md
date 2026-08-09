# devin

## Identity

- **Binary:** `devin` (invoked as `devin`, `devin -- <prompt>`, `devin -p "..."`; `man devin` available). Product was renamed "Devin for Terminal" → "Devin CLI" in May 2026; binary name and config paths unchanged. (https://docs.devin.ai/cli/changelog/stable.md, https://docs.devin.ai/cli/reference/commands.md)
- **Vendor:** Cognition AI (Devin). Docs base: https://docs.devin.ai (CLI section at https://docs.devin.ai/cli). Product page: https://devin.ai/cli.
- **Install:** `curl -fsSL https://cli.devin.ai/install.sh | bash` (macOS/Linux/WSL); `brew install --cask devin-cli`; Windows PowerShell `irm https://static.devin.ai/cli/setup.ps1 | iex`; also bundled with Devin Desktop. (https://docs.devin.ai/cli)
- **Actively maintained:** yes — stable changelog shows v3000.3.27 dated 2026-08-01 (one week before research date). Versioning scheme jumped from `v2026.x` to `v3000.x` ("Local 3.x") mid-2026. (https://docs.devin.ai/cli/changelog/stable.md)
- **Note:** there is also a *cloud* Devin (app.devin.ai) with its own repo-level config (`AGENTS.md`, blueprints, Knowledge/Playbooks/Secrets). This brief covers the **local CLI** only; cloud features (Knowledge, Playbooks, Secrets) are explicitly NOT available in the CLI. (https://docs.devin.ai/cli)

URLs actually read: see `sources_read`.

## Config dirs

- **Project config dir:** `.devin/` at project root. Project root is detected by walking up from cwd looking for `.git` or `.jj`. Nested `.devin/` dirs (monorepo) take precedence over ancestor ones. (https://docs.devin.ai/cli/reference/configuration/global-vs-local.md)
  - `.devin/config.json` — committed team settings (JSONC: JSON **with comment support**). Only `permissions`, `read_config_from`, `hooks` allowed here. (https://docs.devin.ai/cli/extensibility/configuration.md)
  - `.devin/config.local.json` — personal, auto-gitignored overrides (via `.git/info/exclude`).
  - `.devin/mcp_config.json` / `.devin/mcp_config.local.json` — MCP servers (dedicated files since v3000.3; `mcpServers` in config.json is auto-migrated). (https://docs.devin.ai/cli/extensibility/mcp/configuration.md)
  - `.devin/hooks.v1.json` — standalone hooks file.
  - `.devin/skills/<name>/SKILL.md`, `.devin/agents/`, `.devin/rules/*.md`, `.devin/global_rules.md` — see sections below. (https://docs.devin.ai/cli/extensibility/index.md)
- **User/global dir:** `~/.config/devin/` (XDG) on Linux/macOS; `%APPDATA%\devin\` on Windows. Contains `config.json`, `mcp_config.json`, `AGENTS.md`, `skills/`, `agents/`. Additionally `~/.devin/rules/*.md` and `~/.devin/global_rules.md` are read for global rules. (https://docs.devin.ai/cli/extensibility/rules.md)
- **Precedence (high→low):** 1) Organization/team settings (enterprise, never overridable) → 2) session grants (in-memory) → 3) `.devin/config.local.json` → 4) `.devin/config.json` → 5) `~/.config/devin/config.json`. Permissions merge across levels (higher-level deny wins); MCP servers merge by name (higher level wins same-name); hooks are collected from ALL sources and all run. Files with `.local.` in the name are auto-excluded from git. (https://docs.devin.ai/cli/reference/configuration/global-vs-local.md)
- **Foreign-config import (on by default):** `read_config_from` keys `agents_standard`, `cursor`, `windsurf`, `claude`, `copilot`, `opencode`, `vscode`, `zed` — imports rules/skills/hooks/MCP from `.cursor/`, `.windsurf/`, `.claude/`, `.github/skills/`, `.mcp.json`, `opencode.json`, `.vscode/mcp.json`, `.zed/settings.json`, etc. (https://docs.devin.ai/cli/reference/configuration/read-config-from.md)

## Instructions file

- **Filenames (all treated identically, all always-on):** `AGENTS.md` (recommended), `AGENTS.local.md` (personal, gitignored), `AGENT.md`, `.windsurfrules`, `CLAUDE.md`. (https://docs.devin.ai/cli/extensibility/rules.md)
- **Location:** project root AND any subdirectory — root files load at session start; subdirectory files are discovered lazily when the agent touches files in that directory. Global: `~/.config/devin/AGENTS.md` (or `AGENT.md`); also reads `~/.claude/CLAUDE.md` globally. (https://docs.devin.ai/cli/extensibility/rules.md)
- **Heading/title conventions:** none documented — docs example uses `# Project Rules` but no heading is required or parsed. Plain markdown body.
- **Import/include syntax:** none documented (no `@file` includes). Evidence: the full rules reference (https://docs.devin.ai/cli/extensibility/rules.md) describes no include mechanism.
- Plugins can also ship an always-on `AGENTS.md` plus `rules/*.md` with `trigger` frontmatter. (https://docs.devin.ai/cli/changelog/stable.md)

## Skills

**Supported — yes.** (https://docs.devin.ai/cli/extensibility/skills/overview.md, https://docs.devin.ai/cli/extensibility/skills/creating-skills.md)

- **Dirs (project):** `.devin/skills/<name>/SKILL.md` (preferred), `.windsurf/skills/<name>/SKILL.md`, **`.agents/skills/<name>/SKILL.md`** (docs: "We support the `.agents` skills standards").
- **Dirs (global):** `~/.config/devin/skills/<name>/SKILL.md` (`%APPDATA%\devin\skills\` on Windows), `~/.agents/skills/<name>/SKILL.md`, `~/.codeium/<channel>/skills/` (channel = windsurf|windsurf-next|windsurf-insiders).
- **File layout:** directory per skill; the **directory name is the identifier** used for `/name` invocation; single `SKILL.md` with YAML frontmatter + markdown prompt body. Imported (not migrated) Claude skills from `.claude/skills/**/SKILL.md` and Copilot skills from `.github/skills/**/SKILL.md` / `~/.copilot/skills/**/SKILL.md` are also read in place.
- **Exact frontmatter keys:**

| Key | Type | Default | Meaning |
|---|---|---|---|
| `name` | string | directory name | Display name / identifier override |
| `description` | string | none | Shown in slash-command completions; used by agent for auto-invocation |
| `argument-hint` | string | none | Hint after command name, e.g. `"[file] [options]"` |
| `model` | string | current model | Model override; values = `--model` flag values (`opus`, `sonnet`, `swe`, `codex`, …) |
| `subagent` | boolean | `false` | **Devin-unique (experimental):** run skill as a subagent with default `subagent_general` profile |
| `agent` | string | none | **Devin-unique (experimental):** run skill as a subagent using a named custom profile; takes precedence over `subagent` |
| `allowed-tools` | list | all tools | Tool allowlist. Tool names are **lowercase**: `read`, `edit`, `grep`, `glob`, `exec` (+ `write`, `webfetch`, etc.); MCP tools as `mcp__<server>__<tool>` |
| `permissions` | object | inherit | **Devin-unique:** per-skill `allow`/`deny`/`ask` permission scopes, e.g. `Read(src/**)`, `Exec(npm run)`; additive, cannot widen higher-level denies |
| `triggers` | list | `[user, model]` | **Devin-unique:** invocation modes; `[user]` disables autonomous agent invocation |

- **Unique keys vs Claude/agents.md:** `subagent`, `agent`, `permissions`, `triggers`, `argument-hint`, `model`. (`name`, `description`, `allowed-tools` overlap with Claude.)
- **Size/naming limits:** none documented. Invocation-name collisions across sources get a location prefix: `/agents:foo`, `/claude:foo` (changelog v3000.2.17).

## Hooks

**Supported — yes, explicitly "Claude Code compatible" format.** (https://docs.devin.ai/cli/extensibility/hooks/overview.md, https://docs.devin.ai/cli/extensibility/hooks/lifecycle-hooks.md)

- **Config files (project, discovered in cwd + ancestors up to repo root):**
  - `.devin/hooks.v1.json` — standalone; the **whole file is the hooks object** (no `"hooks"` wrapper key).
  - `"hooks"` key inside `.devin/config.json`, `.devin/config.local.json`.
  - Claude-format sources read by default: `.claude/settings.json`, `.claude/settings.local.json` (`"hooks"` key).
- **Config files (user):** `"hooks"` key in `~/.config/devin/config.json`; also `~/.claude.json`, `~/.claude/settings.json`, `~/.claude/settings.local.json`. Format: JSON.
- **Event names:** `PreToolUse`, `PostToolUse`, `PermissionRequest`, `UserPromptSubmit`, `Stop`, `PostCompaction`, `SessionStart`, `SessionEnd`. (Note: `PostCompaction` fires *after* compaction; there is **no** `PreCompact`, `Notification`, or `SubagentStop` event.)
- **Schema:** `{ "<Event>": [ { "matcher": "<regex>", "hooks": [ { "type": "command"|"prompt", "command": "...", "prompt": "...", "timeout": <seconds> } ] } ] }`. `matcher` is a **regex** matched against the event's `tool_name` (NOT a glob — `mcp__github__.*`, not `mcp__github__*`); empty/omitted matches all. Only meaningful for `PreToolUse`/`PostToolUse`/`PermissionRequest`. `type: "prompt"` evaluates an LLM prompt.
- **stdin JSON envelope fields:** always `hook_event_name`, `session_id` (stable per session), `prompt_id` (per-turn; absent for `SessionStart`). Per event:
  - `PreToolUse`/`PermissionRequest`: `tool_name`, `tool_input` (e.g. `{"command": "...", "shell_id": "main"}` for `exec`)
  - `PostToolUse`: `tool_name`, `tool_input`, `tool_response` = `{success: bool, output: string, error: string|null}`
  - `UserPromptSubmit`: `prompt`
  - `Stop`: `stop_hook_active`
  - `PostCompaction`: `summary` (nullable)
  - `SessionStart`: `source`; `SessionEnd`: `reason`
  - Env: `DEVIN_PROJECT_DIR` is set to the project root.
- **Output contract:** stdout JSON object — top-level `decision` (`"approve"`|`"block"`) + optional `reason`; or `hookSpecificOutput: { hookEventName, additionalContext }` to inject context (`UserPromptSubmit`, `SessionStart`, `PostToolUse`); or `hookSpecificOutput: { hookEventName: "PreToolUse", updatedInput: {...} }` to merge-rewrite tool args. Exit codes: `0` success, `2` block, anything else = non-blocking error. Claude-compat nuance: for exit-2 blocks, the reason is read from **stderr** (changelog v3000.3.22).
- Matchable tool names include `read`, `write`, `edit`, `apply_patch`, `notebook_read`, `notebook_edit`, `grep`, `glob`, `exec`, `get_output`, `write_to_process`, `kill_shell`, `webfetch`, `todo_write`, `exit_plan_mode`, `skill`, `run_subagent`, `read_subagent`, `request_scope`, `mcp_*`, plus `mcp__<server>__<tool>`.

## Subagents

**Supported — yes (custom profiles marked experimental).** (https://docs.devin.ai/cli/subagents.md)

- **Dirs:** project: `.devin/agents/` or `.agents/agents/`; global: `~/.config/devin/agents/` (`%APPDATA%\devin\agents\`).
- **File format (two layouts):** flat `agents/<name>.md` (Claude Code convention) OR directory `agents/<name>/AGENT.md` (also accepts `AGENTS.md`, `agent.md`, `agents.md`; precedence `AGENT.md` > `AGENTS.md` > `agent.md` > `agents.md`). YAML frontmatter + markdown system-prompt body. `.claude/agents/*.md` files are also imported (their `tools` key is accepted as an alias for `allowed-tools`).
- **Frontmatter keys:** `name` (default: file/dir name; must not collide with built-ins `subagent_explore`/`subagent_general`), `description` (shown to parent agent when choosing a profile), `model` (default: "default subagent model" = SWE-1.6 via router, NOT the parent's model), `allowed-tools` (list of lowercase tool names; `ask_user_question` can never be granted), `max-nesting` (int, **Devin-unique**, opts into spawning nested subagents).
- Skills can reference a profile via their `agent:` key.

## Commands

**No dedicated slash-command directory.** Skills ARE the slash commands: every skill is invoked as `/<skill-name>` (when `triggers` includes `user`). (https://docs.devin.ai/cli/extensibility/skills/overview.md)

- Claude Code commands are imported **as skills** from `.claude/commands/**/*.md` (https://docs.devin.ai/cli/reference/configuration/read-config-from.md).
- MCP server prompts become slash commands `/mcp__<server>__<prompt>` with positional argument mapping (https://docs.devin.ai/cli/extensibility/mcp/overview.md).
- Plugin skills appear as `/<plugin>:<skill>`; duplicate skill names across sources as `/<source>:<skill>`.
- Legacy Windsurf workflows (`.windsurf/workflows/`) are NOT imported as skills; `devin migrate workflows` converts them into skills (https://docs.devin.ai/cli/reference/commands.md).
- Built-in slash commands (`/hooks`, `/mcp`, `/model`, `/mode`, `/plan`, …) are listed in the commands reference.

## Rules

**Supported — a rules mechanism distinct from `AGENTS.md`.** (https://docs.devin.ai/cli/extensibility/rules.md)

- **Dirs:** `.devin/rules/*.md` (project; preferred over `.windsurf/`), `.devin/global_rules.md` (single always-on file), plus home-dir `~/.devin/rules/*.md` and `~/.devin/global_rules.md`. Also reads `.windsurf/rules/*.md`, `.windsurf/global_rules.md`, and `.cursor/rules/*.md|*.mdc`.
- **Format:** one rule per `.md` file with YAML frontmatter. Windsurf-style keys: `description`, `trigger` with values `always_on` | `manual` | `model_decision` | `agent` | `glob`. Cursor-style keys: `description`, `globs`, `alwaysApply` (activation: alwaysApply→always-on; globs→glob-activated; description-only→agent-decided; none→manual).
- Root-level rules load at session start; subdirectory rules load lazily. If both `.devin/global_rules.md` and `.windsurf/global_rules.md` exist, only `.devin` wins; `rules/*.md` from both dirs are merged.

## MCP

**Supported.** (https://docs.devin.ai/cli/extensibility/mcp/overview.md, https://docs.devin.ai/cli/extensibility/mcp/configuration.md)

- **Config files (since v3000.3, "Local 3.6"):** `.devin/mcp_config.json` (project, committed), `.devin/mcp_config.local.json` (project, gitignored), `~/.config/devin/mcp_config.json` (user; `%APPDATA%\devin\mcp_config.json` on Windows). Pre-v3000.3 they lived in the `mcpServers` key of `config.json`; auto-migrated on startup. Format: JSON(C).
- **Server-table shape:** top-level `{ "mcpServers": { "<name>": {…} } }` — same key name as Claude's `.mcp.json`.
  - stdio: `command` (req), `args`, `env` (object), `disabled` (bool).
  - remote: `url` (req), `transport` (`"http"` default w/ automatic SSE fallback on 4xx, or `"sse"`), `headers` (object), `oauthClientId`, `oauthClientSecret`, `oauthResource` (RFC 8707 override; `""` omits), `disabled`.
- **Env/secret handling:** values support `${env:VAR}` and `${file:/path}` interpolation (documented for OAuth fields; recommended pattern is committed `mcp_config.json` with placeholders + secrets in gitignored `mcp_config.local.json`). OAuth tokens are stored locally per-CLI via `devin mcp login` — not shared with Windsurf/Claude Code.
- **CLI management:** `devin mcp add|list|get|remove|login|logout|enable|disable` with `-s local|project|user` scope (default `local`).
- **Tool namespace:** `mcp__<server>__<tool>` for permissions and hook matchers.
- **Imports (read in place, convertible):** Claude `.mcp.json` + all `.claude*` settings files; Cursor `.cursor/mcp.json`; VS Code `.vscode/mcp.json` (key `servers`); Zed `.zed/settings.json` (key `context_servers`); OpenCode `opencode.json` (key differences auto-converted: `environment`→`env`, `enabled`→inverted `disabled`, string/array commands); Windsurf `~/.codeium/<channel>/mcp_config.json`.

## Translation hazards

What a canonical `.agents` store must drop/rename/reformat to target Devin CLI:

1. **Skills — target path works natively, frontmatter must be filtered.** Devin reads `.agents/skills/<name>/SKILL.md` directly, so no rename needed for the tree. But:
   - Drop unsupported canonical keys or map them: keep `name`, `description`, `allowed-tools`; pass through Devin-only keys (`argument-hint`, `model`, `subagent`, `agent`, `permissions`, `triggers`) only when targeting Devin. Claude-only keys (`license`, `metadata`, etc.) are undocumented for Devin — treat as inert/droppable.
   - **`allowed-tools` values must be renamed to lowercase Devin tool names:** Claude `Bash`→`exec`, `Read`→`read`, `Edit`→`edit`, `Write`→`write`, `Grep`→`grep`, `Glob`→`glob`, `WebFetch`→`webfetch`, `NotebookEdit`→`notebook_edit`. MCP tool syntax `mcp__server__tool` is identical.
   - Identifier = directory name; `name:` frontmatter overrides it (same as Claude). No documented size or charset limits.
2. **Hooks — same event vocabulary as Claude, but not identical:**
   - Devin events: `PreToolUse`, `PostToolUse`, `PermissionRequest`, `UserPromptSubmit`, `Stop`, `PostCompaction`, `SessionStart`, `SessionEnd`. A canonical store must **drop or remap** Claude's `Notification`, `SubagentStop`, `PreCompact` (closest Devin analog is `PostCompaction`, but semantics differ — fires after, receives `summary`).
   - **File layout change:** canonical hooks must be written to `.devin/hooks.v1.json` as the *bare* hooks object (no `"hooks"` wrapper) OR nested under `"hooks"` in `.devin/config.json`. (`.claude/settings.json` also works via import, but that's the Claude path.)
   - **Matcher is regex, not Claude's glob-ish/string matching** — `mcp__github__*` must become `mcp__github__.*`; exact tool matches need `^exec$` anchors or they substring-match (`"exec"` matches any tool name containing "exec").
   - **Tool names in matchers/envelope are Devin's lowercase names** (`exec`, not `Bash`).
   - Envelope deltas vs Claude: adds `prompt_id`; `tool_response` shape is `{success, output, error}`; `DEVIN_PROJECT_DIR` env var (Claude uses `CLAUDE_PROJECT_DIR`). agentlink's hook-input normalizer must accept `hook_event_name` (shared), `session_id`, `prompt_id`, `tool_input.command`, `prompt`, `stop_hook_active`, `summary`, `source`, `reason`.
   - Exit-2 block reason comes from **stderr** (Claude convention, confirmed in changelog); JSON stdout `decision`/`hookSpecificOutput.additionalContext`/`updatedInput` contract matches Claude's schema names.
   - Windsurf-format hooks (`.windsurf/hooks.json`, `trajectory_id`/`execution_id` envelope fields) exist as a legacy third format — `devin migrate hooks` converts to `.devin/hooks.v1.json`.
3. **Instructions file — heading-agnostic, multiple aliases:** Devin treats `AGENTS.md`, `AGENT.md`, `AGENTS.local.md`, `.windsurfrules`, `CLAUDE.md` identically with no required `# <Agent> instructions` heading. agentlink's `instructions` normalizer regex (Claude|Codex headings) needs no Devin branch — but a canonical store should emit plain `AGENTS.md` with no heading assumption, and can emit `AGENTS.local.md` for personal overlays. No `@import` include syntax — canonical includes must be inlined.
4. **Subagents — dir name collision with the canonical store:** Devin reads `.agents/agents/` (nested `agents` inside `.agents`!) and `.devin/agents/`. Flat `<name>.md` or `<name>/AGENT.md`. Frontmatter: drop Claude's `tools` (only accepted in `.claude/agents/` import path; canonical files must use `allowed-tools`); `max-nesting` is Devin-only; no `model` default inheritance from parent (unpinned = SWE-1.6). Names colliding with `subagent_explore`/`subagent_general` are skipped with a warning.
5. **Slash commands — no separate commands dir:** canonical `commands/*.md` must be materialized as **skills** (`.agents/skills/<name>/SKILL.md` or `.devin/skills/`), not a `commands/` folder. Windsurf-style `workflows/` are not read.
6. **Rules — frontmatter dialect split:** `.devin/rules/*.md` uses Windsurf `trigger` values (`always_on|manual|model_decision|agent|glob`); Cursor `alwaysApply`/`globs` only works in `.cursor/rules/`. A canonical rule with glob activation must pick one dialect per target dir. `AGENTS.md` rules are always-on only.
7. **MCP — JSON, not TOML; dedicated file:** target is `.devin/mcp_config.json` with `mcpServers` map (Claude-compatible shape `command/args/env/url` — a near drop-in from `.mcp.json`). agentlink's Codex comparison (TOML `config.toml`) needs a JSON emitter for Devin. Env comparison can keep key-names-only semantics; note Devin's `${env:VAR}`/`${file:/path}` interpolation and OAuth-only fields (`oauthClientId/Secret/Resource`, `headers`, `transport`, `disabled`) that Codex/Claude lack. Server names feed `mcp__<name>__<tool>` namespaces everywhere.
8. **Config format:** JSONC (comments allowed) — not strict JSON, not TOML/YAML. Project-level config keys are restricted to `permissions`, `read_config_from`, `hooks` (+ `mcp_config.json`); model/theme/etc. are user-only — a canonical store cannot express per-project model pinning for Devin.
9. **Permissions dialect (if canonical store carries them):** PascalCase scope syntax `Read(**)` / `Exec(git)` / `Write(src/**)` with `allow`/`deny`/`ask` lists — distinct from Claude's permission rule strings and from hook matchers (lowercase, regex).
10. **Platform path fork:** user dir is `~/.config/devin/` on POSIX but `%APPDATA%\devin\` on Windows — sync/adopt logic must branch per-OS.

## Verified corrections (fact-checker pass)

- was: "Files with `.local.` in the name are auto-excluded from git" (and "`AGENTS.local.md` (personal, gitignored)") → now: only `.devin/config.local.json` / `.devin/mcp_config.local.json` are auto-excluded (via `.git/info/exclude`); `AGENTS.local.md` is NOT auto-gitignored — docs instruct "Add it to your `.gitignore` so it stays local" (https://docs.devin.ai/cli/extensibility/rules.md, https://docs.devin.ai/cli/reference/configuration/global-vs-local.md)
- was: "allowed-tools ... Tool names are lowercase: `read`, `edit`, `grep`, `glob`, `exec` (+ `write`, `webfetch`, etc.)" → now: the skills reference lists exactly five available tool names for `allowed-tools` — `read`, `edit`, `grep`, `glob`, `exec` — plus MCP tools as `mcp__<server>__<tool>`; `write`/`webfetch`/other built-ins are documented only as hook-matcher tool names, not in the skills allowed-tools list (https://docs.devin.ai/cli/extensibility/skills/creating-skills.md, https://docs.devin.ai/cli/extensibility/hooks/lifecycle-hooks.md)
- was: "`.devin/skills/<name>/SKILL.md` (preferred)" → now: skills docs state no preference among project skill dirs — the overview table lists `.agents/skills/<name>/SKILL.md` first; "preferred" language exists only for `.devin/` over `.windsurf/` in the rules context (https://docs.devin.ai/cli/extensibility/skills/overview.md, https://docs.devin.ai/cli/extensibility/rules.md)
