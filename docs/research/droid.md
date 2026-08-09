# droid

## Identity
- **Binary:** `droid` — Factory AI's terminal coding agent ("Droid CLI"). Vendor: Factory AI (factory.ai). Closed-source; no public source repo for the CLI itself was found. Official plugin marketplace repo: `github.com/Factory-AI/factory-plugins` (https://docs.factory.ai/harness/plugins).
- **Install** (https://docs.factory.ai/droid-cli/quickstart):
  - macOS/Linux: `curl -fsSL https://app.factory.ai/cli | sh`
  - Homebrew: `brew install --cask droid`
  - Windows: `irm https://app.factory.ai/cli/windows | iex`
  - npm: `npm install -g droid`
- **Actively maintained:** yes — docs pages carry recent "updated" markers (e.g. CLI reference updated ~2 weeks before 2026-08-08 per search index), and the docs describe current-version behavior (Missions, plugins, exec mode).
- **Docs URLs actually read:**
  - https://docs.factory.ai/droid-cli/overview (canonical for /cli/getting-started/overview)
  - https://docs.factory.ai/harness/hooks (canonical for /cli/configuration/hooks-guide)
  - https://docs.factory.ai/harness/skills
  - https://docs.factory.ai/harness/agents-md
  - https://docs.factory.ai/harness/subagents (canonical for /cli/configuration/custom-droids)
  - https://docs.factory.ai/harness/mcp
  - https://docs.factory.ai/droid-cli/quickstart
  - https://docs.factory.ai/harness/custom-slash-commands
  - https://docs.factory.ai/droid-cli/settings
  - https://docs.factory.ai/harness/plugins
  - https://docs.factory.ai/guides/power-user/rules-conventions and /guides/power-user/memory-management — both redirect to /harness/agents-md (retired guide pages; see Rules section)

## Config dirs
- **Project dir:** `.factory/` in the repo root. Also folder-level `.factory/` in ancestor directories is honored for MCP (https://docs.factory.ai/harness/mcp) and for nested skills (https://docs.factory.ai/harness/skills).
- **Global/user dir:** `~/.factory/` (Windows: `%USERPROFILE%\.factory\`) — https://docs.factory.ai/droid-cli/settings.
- **Key contents:** `settings.json` + `settings.local.json` (local overrides merge on top at the same level; user and project both supported), `hooks.json`, `mcp.json`, `skills/`, `droids/`, `commands/` (https://docs.factory.ai/droid-cli/settings, /harness/hooks, /harness/mcp).
- **Compatibility dirs also read:** `.agents/` and `.agent/` at project and `~/` level for instructions and skills (https://docs.factory.ai/harness/agents-md, /harness/skills).
- **Discovery/precedence:**
  - Instructions: searched from cwd up to git root; each level checks the directory itself plus `.factory/`, `.agents/`, `.agent/`; personal dirs `~/.factory/`, `~/.agents/`, `~/.agent/`; project overrides personal (https://docs.factory.ai/harness/agents-md).
  - Skills precedence (high→low): folder-specific + project skills → project plugin skills → personal skills → user plugin skills → built-in; missions/org/CLI scopes can outrank; duplicates within one bucket are invalid (https://docs.factory.ai/harness/skills).
  - MCP: org-managed > project `.factory/mcp.json` / folder / user `~/.factory/mcp.json`; one definition per name wins; `droid mcp add` always writes user config (https://docs.factory.ai/harness/mcp).
  - Hooks: enterprise managed > project > user; legacy `.factory/hooks/hooks.json` still loads and is migrated to `.factory/hooks.json` on next save (https://docs.factory.ai/harness/hooks).
- **Legacy:** `.droid.yaml` is a deprecated project config surface (https://docs.factory.ai/droid-cli/settings).

## Instructions file
- **Filename:** `AGENTS.md` at repo root (recommended); compatible variants also read: `agents.md`, `Agents.md`, `CLAUDE.md`, `Claude.md` (https://docs.factory.ai/harness/agents-md).
- **Separate design file:** `DESIGN.md` / `Design.md` / `design.md` loaded separately as design guidelines (https://docs.factory.ai/harness/agents-md).
- **Nesting:** nested `AGENTS.md` files in subdirectories are discovered dynamically when Droid reads files under that tree; nested refines root (https://docs.factory.ai/harness/agents-md).
- **Heading conventions:** none enforced. Official template starts with `# Repository guide`; no required title line (https://docs.factory.ai/harness/agents-md).
- **Import/include syntax:** none documented — linking to other files is by prose reference only ("Link to the source of truth").
- **Size caps:** initial AGENTS-style load capped at 80,000 characters; dynamic Read-path discovery capped at 40,000 characters (https://docs.factory.ai/harness/agents-md).

## Skills
Supported. Dir-based, `SKILL.md` entry point (https://docs.factory.ai/harness/skills).
- **Dirs:** `<repo>/.factory/skills/<name>/SKILL.md` (project); `<repo>/<area>/.factory/skills/...` (folder-specific); `~/.factory/skills/<name>/SKILL.md` (personal); compatibility: `<repo>/.agents/skills/**/SKILL.md`, `<repo>/.agent/skills/**/SKILL.md`, `~/.agents/skills/**`, `~/.agent/skills/**`; plus mission, plugin (`skills/<name>/SKILL.md` in plugin), built-in. Search is recursive; first directory containing `SKILL.md` is the skill dir.
- **File layout:** entry must be named exactly `SKILL.md` ("Do not use `skill.mdx`"); optional supporting files (checklists, schemas, scripts) beside it, not auto-loaded.
- **Frontmatter keys** (YAML):
  - `name` (required) — identifier; **lowercase letters, numbers, hyphens only**.
  - `description` (required) — routing text; "what it does and when to use it".
  - `allowed-tools` (optional) — declared tool list; **metadata only, not a sandbox**, does not grant tools.
  - `enabled` (optional, default `true`) — `false` disables on-disk skill.
  - `user-invocable` (optional, default `true`) — `false` hides from `/skill-name` slash invocation.
  - `disable-model-invocation` (optional, default `false`) — `true` blocks automatic model invocation.
  - `license` (optional), `compatibility` (optional, e.g. `droid`), `version` (optional), `metadata` (optional map).
  - `tools` — **deprecated legacy**, replaced by `allowed-tools`.
- **Keys unique to droid** (vs. Claude Code's skill schema): `enabled`, `user-invocable`, `disable-model-invocation`, `compatibility`, `version`. Shared with Claude: `name`, `description`, `allowed-tools`, `license`, `metadata`.
- **Size limits:** none documented for SKILL.md (the 80k/40k char caps apply to AGENTS-style instruction files, not skills). Naming limit: lowercase-hyphen rule above; disable ledgers (`disabledSkills` in settings.json) store "sanitized skill names".
- **Slash surface:** enabled + user-invocable skills appear as `/skill-name`; a custom slash command with the same name keeps the slash binding (https://docs.factory.ai/harness/skills).

## Hooks
Supported. Full reference: https://docs.factory.ai/harness/hooks (cross-checked structure against plugin hooks at https://docs.factory.ai/harness/plugins).
- **Config file + format:** JSON. `~/.factory/hooks.json` (user), `.factory/hooks.json` (project), org-managed via enterprise settings; legacy `.factory/hooks/hooks.json` auto-migrated. Fallback: `hooks` key inside the matching `settings.json` when `hooks.json` is absent. Top-level shape: `{"hooks": {"<EventName>": [<matcher group>]}}` (the plugin variant omits the outer `hooks` wrapper — file is keyed directly by event, per the plugin example; note this inconsistency between the two official pages).
- **Event names:** `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Notification`, `Stop`, `SubagentStop`, `PreCompact`, `SessionStart`, `SessionEnd`. (Same naming as Claude Code.)
- **Matcher schema (per group):** `matcher` (optional; empty/omitted/`*` matches all; exact tool name or case-sensitive regex; tool names: `Execute`, `Read`, `Edit`, `Create`, `ApplyPatch`, `LS`, `Glob`, `Grep`, `Task`, `FetchUrl`, `WebSearch`; MCP tools match `mcp__<server>__<tool>`, `mcp__.*` for all), `commandRegex` (optional droid-specific extra regex against the actual shell command string for `Execute`), `hooks` (required array).
- **Command schema (per hook):** `type` (required, only `"command"`), `command` (required, shell command; JSON on stdin), `timeout` (optional seconds, default 60). Absolute paths required in practice; `$FACTORY_PROJECT_DIR` for project scripts; plugin hooks may use `${DROID_PLUGIN_ROOT}`/`${CLAUDE_PLUGIN_ROOT}`.
- **stdin JSON envelope:** `session_id`, `transcript_path`, `cwd`, `permission_mode` (`off` | `spec` | `auto-low` | `auto-medium` | `auto-high`), `hook_event_name`, optional `message_id`. Tool hooks add `tool_name`, `tool_input`; `PostToolUse` adds `tool_response`. Event-specific fields: `prompt`/`has_images` (UserPromptSubmit), `message`/`notification_type` (Notification; types `permission_prompt`, `idle_prompt`, `auth_success`, `elicitation_dialog`), `stop_hook_active`/`tool_execution_count`/`elapsed_time` (Stop), `task_name`/`task_result`/`task_error`/`stop_hook_active` (SubagentStop), `trigger` (`manual`|`auto`)/`custom_instructions`/`message_count`/`estimated_tokens` (PreCompact), `source` (`startup`|`resume`|`clear`|`compact`) (SessionStart), `reason` (`clear`|`logout`|`prompt_input_exit`|`other`)/`session_duration_ms`/`message_count` (SessionEnd). String/bool/number fields also exposed as env vars.
- **Output contract:** exit `0` = success (stdout becomes added context for UserPromptSubmit/SessionStart; otherwise shown in transcript); exit `2` = blocking — PreToolUse blocks the tool, PostToolUse/Stop feed stderr back to Droid, UserPromptSubmit blocks the prompt; other non-zero = non-blocking error. Optional JSON on stdout: `continue: false` + `stopReason`; `suppressOutput: true`; `hookSpecificOutput.permissionDecision` = `allow`|`deny`|`ask` with `permissionDecisionReason` and `updatedInput` (rewrite tool params) for PreToolUse; `decision: "block"` + `reason` for PostToolUse/UserPromptSubmit/Stop/SubagentStop; `hookSpecificOutput.additionalContext` for PostToolUse/UserPromptSubmit/SessionStart. SessionEnd cannot block.
- Global kill switch: `hooksDisabled` in settings.json; `showHookOutput` for debugging (https://docs.factory.ai/droid-cli/settings).

## Subagents
Supported as "custom droids" (https://docs.factory.ai/harness/subagents).
- **Dirs:** `<repo>/.factory/droids/<name>.md` (project) and `~/.factory/droids/<name>.md` (personal). Top-level scan only (no recursion stated). Project wins on name collision.
- **Format:** Markdown file, YAML frontmatter + non-empty system-prompt body.
- **Frontmatter keys:**
  - `name` (required) — must match `^[a-z0-9-_]+$` (lowercase, digits, `-`, `_`); drives `subagent_type` and filename.
  - `description` (optional, recommended) — >500 chars raises a validation warning.
  - `model` (optional, default `inherit`) — Factory model ID (e.g. `claude-sonnet-4-5-20250929`) or `custom:<byok-model-field>`.
  - `reasoningEffort` (optional) — `low`|`medium`|`high`; ignored when `model: inherit`.
  - `tools` (optional) — omit = all tools; category string (`read-only` = Read/LS/Grep/Glob, `edit` = Create/Edit/ApplyPatch, `execute` = Execute, `web` = WebSearch/FetchUrl, `mcp`) or array of case-sensitive tool IDs. `tools: all` literal is **rejected**; `TodoWrite` and `Skill` are always force-included; `ExitSpecMode` and `GenerateDroid` are forbidden (validation error).
  - `mcpServers` (optional) — array of server names from `mcp.json`; `[]` excludes all MCP; omitted = inherit parent.
- **Built-ins:** `worker` (all tools) and `explorer` (read-only). One-time **Import from Claude Code** flow reads `.claude/agents/` (project + `~/`) and translates name/description/body, model family, tool names (https://docs.factory.ai/harness/subagents).

## Commands
Supported (https://docs.factory.ai/harness/custom-slash-commands).
- **Dirs:** `<repo>/.factory/commands/` (workspace, wins on conflict) and `~/.factory/commands/` (personal). Recursive discovery; managed via `/commands`.
- **Format:** a file registers if it ends in `.md` **or** its first line starts with a `#!` shebang (executable command). Filenames slugged: lowercase, spaces/non-URL chars → `-`, extension dropped (`Code Review.md` → `/code-review`).
- **Markdown frontmatter:** only `description` (autocomplete summary) and `argument-hint` (e.g. `<branch-name>`). **No tool scoping** for commands.
- **Argument syntax:** `$ARGUMENTS` only; `$1`/`$2` positional placeholders **not supported** in Markdown commands (they are supported as real positional args in executable/shebang commands).
- **Executable commands:** run from cwd, inherit env; stdout+stderr capped at 64 KB and posted back to the transcript.
- Import flow copies `.md` files from `.agents/commands` and `.claude/commands` (repo root and `~`).

## Rules
**Not supported as a first-class auto-loaded surface.** The canonical instruction-surfaces table (https://docs.factory.ai/harness/agents-md) lists only `AGENTS.md`(+variants), `DESIGN.md`, `SKILL.md`, `.factory/commands/*`, and `settings.json` — no rules directory. Retired power-user guides (https://docs.factory.ai/guides/power-user/rules-conventions and /guides/power-user/memory-management — both now redirect to /harness/agents-md) described `.factory/rules/*.md` (and `~/.factory/rules/`) as a **convention** for organizing standards that AGENTS.md references and Droid "checks" during work, i.e. content pulled in via instructions, not a distinct loaded config surface. Treat rules content as belonging in `AGENTS.md` (or skills) for this target.

## MCP
Reference: https://docs.factory.ai/harness/mcp.
- **Config file:** `mcp.json`, JSON. Levels: user `~/.factory/mcp.json`; folder `.factory/mcp.json` in an ancestor dir; project `.factory/mcp.json` in repo root (commit-safe, no secrets). Org-managed servers/policy override all. Plugins can ship a root `mcp.json`.
- **Server-table shape:** top-level `"mcpServers": { "<name>": { ... } }`.
  - Common fields: `type` (`"stdio"|"http"|"sse"`, default `stdio`), `disabled` (bool), `disabledTools` (string[] of excluded tool names), `timeout` (ms per tool call), `connectTimeout` (ms handshake; defaults 10000 http/sse, 30000 stdio).
  - stdio: `command` (string), `args` (string[]), `env` (object).
  - http/sse: `url`, `headers` (object), `oauth` (overrides object, or `false` to disable OAuth).
- **Env handling:** `${NAME}` expansion from the shell environment applies **only** to stdio `env` values, http/sse `headers` values, and `oauth.clientId`/`oauth.clientSecret`. **Not** expanded in `command`, `args`, or `url`. No default-value syntax. OAuth tokens stored in system keyring, not the file. Tool approvals are fingerprinted to command+args or URL.
- CLI: `droid mcp add <name> <urlOrCommand...> --type stdio|http|sse [--env K=V] [--header "K: V"] [--no-oauth]`, `droid mcp list|remove|permissions`.

## Translation hazards
What a canonical `.agents` store must drop/rename/reformat to target droid:

1. **Root dir rename:** `.agents/` → `.factory/` for project config (`skills/`, `droids/`, `commands/`, `hooks.json`, `mcp.json`, `settings.json`). **Partial native compat shortcut:** droid reads `.agents/skills/**/SKILL.md` and `.agents/` instruction dirs natively (https://docs.factory.ai/harness/skills, /harness/agents-md), so symlinked skill/instruction trees work without copying; `droids/`, `commands/`, `hooks.json`, `mcp.json` do **not** have documented `.agents` compat and must live under `.factory/`.
2. **Instructions file:** canonical instructions → `AGENTS.md` at repo root. Droid also reads `CLAUDE.md` (so existing Claude files work), but `AGENTS.md` is canonical. No heading/title convention to normalize — but enforce the **80,000-char initial / 40,000-char dynamic caps**; an oversized canonical instructions file must be truncated/split (nested AGENTS.md or skills). No import syntax to translate.
3. **Skill names:** canonical names must be rewritten to lowercase letters/numbers/**hyphens only** — underscores valid in droid *subagent* names but **not** documented for skill names. Entry file must be `SKILL.md` exactly (no `.mdx`, no lowercase variant documented).
4. **Skill frontmatter:** drop canonical/Claude-only keys not in droid's schema (anything beyond `name`, `description`, `allowed-tools`, `enabled`, `user-invocable`, `disable-model-invocation`, `license`, `compatibility`, `version`, `metadata`). Never emit deprecated `tools`. Do not rely on `allowed-tools` as enforcement — it is display metadata in droid; a canonical store that treats it as a sandbox must lower that semantic or move the workflow to a droid with a tool policy. Droid-only keys (`enabled`, `user-invocable`, `disable-model-invocation`) have no Claude/Codex equivalent — strip them when syncing *from* droid to the canonical store, or the round-trip will drift.
5. **Hooks — event names:** droid uses the Claude Code event vocabulary (`PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Notification`, `Stop`, `SubagentStop`, `PreCompact`, `SessionStart`, `SessionEnd`). Codex-style events (e.g. its different lifecycle set) must be mapped or dropped; `SubagentStop` exists here, `PreCompact` exists here. There is no `SessionStart`/`SessionEnd` equivalent in some peers — drop or no-op.
6. **Hooks — format & fields:** config is **JSON at `.factory/hooks.json`** (not TOML, not `settings.json` normally). Wrap matcher groups under a top-level `"hooks"` key for the standalone file. Drop Claude-unsupported `type` values — only `"command"` exists (no `prompt`/`agent` hook types). Drop `commandRegex` when translating *away* from droid (droid-only field). Env var is `$FACTORY_PROJECT_DIR`, not `$CLAUDE_PROJECT_DIR` — rewrite command strings.
7. **Hooks — envelope:** stdin envelope is snake_case and Claude-shaped (`session_id`, `transcript_path`, `cwd`, `hook_event_name`, `tool_name`, `tool_input`, `tool_response`), so Claude envelopes pass through; **but** `permission_mode` enum differs (`off|spec|auto-low|auto-medium|auto-high` vs Claude's `default|plan|acceptEdits|bypassPermissions`) — any guard/remind logic keyed on permission mode must remap. Codex's PostToolUse envelope must be re-wrapped entirely.
8. **Subagents:** dir is `.factory/droids/` (not `agents/`); file is flat `<name>.md` (top-level scan). Names must match `^[a-z0-9-_]+$` — lowercase any CamelCase canonical names. Frontmatter: keep `name`, `description` (≤500 chars or warning), `model`, `reasoningEffort`, `tools`, `mcpServers`; drop other canonical keys (e.g. Claude `color`). **Tool-name mapping required:** Claude `Bash`→`Execute`, `Write`→`Create`, `MultiEdit`→`Edit`, `WebFetch`→`FetchUrl`; tool IDs are case-sensitive. Never emit literal `tools: all` (rejected) — omit the field. Never list `TodoWrite`, `Skill` (force-included), `ExitSpecMode`, `GenerateDroid` (validation error). `model` must be `inherit`, a Factory model ID, or `custom:<id>` — Claude aliases (`sonnet`/`opus`/`haiku`) are only translated by the interactive import wizard, not by the file loader.
9. **Commands:** dir `.factory/commands/`; only `description` and `argument-hint` frontmatter survive — drop `allowed-tools` or any tool-scoping keys. Rewrite `$1`/`$2` positional placeholders to `$ARGUMENTS` (unsupported in Markdown commands). Filenames are slugged to lowercase-hyphen — a canonical name with uppercase/underscores changes the resulting slash name. Non-`.md` files are ignored unless they start with a shebang.
10. **MCP:** JSON `mcp.json` with `mcpServers` table — a Codex-style TOML `config.toml` block must be reformatted; key mapping: `command`/`args`/`env` (stdio), `url`/`headers` (http/sse), plus `type`, `disabled`, `disabledTools`, `timeout`, `connectTimeout`. Env **values** support `${VAR}` expansion only in `env`/`headers`/`oauth` — canonical stores that inline values into `command`/`args`/`url` will not get expansion. Keep secrets out of project-level `.factory/mcp.json` (committed).
11. **Rules:** no rules surface — canonical `.agents/rules/**` content must be merged into `AGENTS.md` (mind the 80k cap), converted to skills, or dropped; do not emit a `.factory/rules/` dir expecting auto-loading.
12. **Legacy traps:** do not emit `.droid.yaml` (deprecated) or `.factory/hooks/hooks.json` (legacy path; next save migrates it to `.factory/hooks.json` and archives the old file — would cause sync churn).

### Critical Files for Implementation

- `.factory/skills/<name>/SKILL.md` — primary adopt/sync target; frontmatter schema (`name`, `description`, `allowed-tools`, droid-only `enabled`/`user-invocable`/`disable-model-invocation`) and lowercase-hyphen naming drive the skill normalizer (https://docs.factory.ai/harness/skills)
- `.factory/hooks.json` — hook pair endpoint; event-name/matcher/envelope schema determines how much of the existing Claude hook normalizer and `guard`/`remind` envelope parsing can be reused verbatim (https://docs.factory.ai/harness/hooks)
- `.factory/mcp.json` — MCP pair endpoint; JSON `mcpServers` table, comparable fields `command`/`args`/`url`/`type`, `${VAR}` env-name extraction rules (https://docs.factory.ai/harness/mcp)
- `AGENTS.md` — instructions pair endpoint; no heading convention (normalizer regex needs no droid branch), but 80k/40k char caps are a new constraint (https://docs.factory.ai/harness/agents-md)
- `.factory/droids/<name>.md` — subagent endpoint; frontmatter name regex `^[a-z0-9-_]+$`, tool-ID mapping (Bash→Execute etc.), forbidden-tool rules are the main translation-hazard surface (https://docs.factory.ai/harness/subagents)