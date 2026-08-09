# crush

## Identity
- **Binary**: `crush` — "Glamourous agentic coding for all 💘", written in Go, license FSL-1.1-MIT.
- **Vendor**: Charm (charmbracelet). Actively maintained: v0.88.1 latest, frequent releases through v0.82–v0.88 series (https://github.com/charmbracelet/crush/releases).
- **Install**: `brew install charmbracelet/tap/crush`; `npm install -g @charmland/crush`; `winget install charmbracelet.crush`; `go install github.com/charmbracelet/crush@latest`; apt/yum via `repo.charm.sh`; Nix NUR `nur.repos.charmbracelet.crush` (https://github.com/charmbracelet/crush).
- **No dedicated docs site** — documentation lives in the repo README and `docs/` folder.
- **URLs actually read**:
  - https://github.com/charmbracelet/crush (README, full)
  - https://raw.githubusercontent.com/charmbracelet/crush/main/docs/config/README.md
  - https://raw.githubusercontent.com/charmbracelet/crush/main/docs/hooks/README.md
  - https://raw.githubusercontent.com/charmbracelet/crush/main/docs/hooks/FUTURE.md
  - https://raw.githubusercontent.com/charmbracelet/crush/main/schema.json
  - https://raw.githubusercontent.com/charmbracelet/crush/main/internal/skills/skills.go
  - https://raw.githubusercontent.com/charmbracelet/crush/main/internal/config/config.go
  - https://raw.githubusercontent.com/charmbracelet/crush/main/internal/config/load.go
  - https://raw.githubusercontent.com/charmbracelet/crush/main/internal/agent/prompt/prompt.go
  - https://raw.githubusercontent.com/charmbracelet/crush/main/.agents/skills/shell-builtins/SKILL.md
  - https://github.com/charmbracelet/crush/releases

## Config dirs
- **Crush has no folder-based config dir.** Config is Bash files (`crushrc`) executed at startup via Crush's embedded POSIX shell; legacy JSON (`crush.json`) is deprecated but still supported (https://raw.githubusercontent.com/charmbracelet/crush/main/docs/config/README.md).
- **Discovery order** (lower number wins; everything found is merged, project overrides global, `crushrc` overrides JSON in the same directory):
  1. `./.crushrc` (legacy: `./.crush.json`)
  2. `./crushrc` (legacy: `./crush.json`)
  3. `$XDG_CONFIG_HOME/crush/crushrc` i.e. `~/.config/crush/crushrc` (legacy: `~/.config/crush/crush.json`); Windows: `%USERPROFILE%\.config\crush\crushrc`
- `.crush/` in a project is **not** a config dir — it's the data directory (default `options.data_directory = ".crush"`, holds `./.crush/logs/crush.log` and workspace state `crush.json`) (https://raw.githubusercontent.com/charmbracelet/crush/main/schema.json, README "Logging").
- State (not config) also lives in `$XDG_DATA_HOME/crush` (`~/.local/share/crush`) / `%LOCALAPPDATA%\crush`; "Crush does not discover or execute a `crushrc` from those locations" (docs/config/README.md).
- Env overrides: `CRUSH_GLOBAL_CONFIG`, `CRUSH_GLOBAL_DATA` (README).

## Instructions file
- **No single canonical filename** — Crush loads a fixed default list of project context files, always prepended to `options.context_paths` (https://raw.githubusercontent.com/charmbracelet/crush/main/internal/config/config.go, `defaultContextPaths`):
  `.github/copilot-instructions.md`, `.cursorrules`, `.cursor/rules/`, `CLAUDE.md`, `CLAUDE.local.md`, `GEMINI.md`, `gemini.md`, `crush.md`, `crush.local.md`, `Crush.md`, `Crush.local.md`, `CRUSH.md`, `CRUSH.local.md`, `AGENTS.md`, `agents.md`, `Agents.md`.
- `crush init` generates **`AGENTS.md`** by default; rename via `option initialize-as` (crushrc) / `options.initialize_as` (JSON) (README "Initialization").
- Global instruction files: `~/.config/crush/CRUSH.md` and `~/.config/AGENTS.md` (defaults of `options.global_context_paths`; configurable via `option global-context-path`) (README "Global context files"; load.go `setDefaults`).
- **Heading conventions**: none. File contents are loaded verbatim into the system prompt (`ContextFile{Path, Content}` in prompt.go). No heading regex, no title requirement.
- **Import/include syntax**: none in instruction files (no `@file` expansion). A path pointing at a directory is walked recursively loading files (README says "all Markdown files", but `processContextPath` in prompt.go loads **all** files regardless of extension — doc/code mismatch). `source` works only inside `crushrc`, not in markdown.

## Skills
- **Supported** — implements the [Agent Skills](https://agentskills.io) open standard (README "Agent Skills"; package doc in internal/skills/skills.go).
- **Layout**: `<skills-dir>/<skill-name>/SKILL.md` (folder per skill; extra resource files allowed alongside).
- **Discovery paths** (load.go `GlobalSkillsDirs`/`ProjectSkillsDir`; README):
  - Global: `$CRUSH_SKILLS_DIR` (exclusive if set), `$XDG_CONFIG_HOME/crush/skills`, `$XDG_CONFIG_HOME/agents/skills`, `~/.agents/skills`, `~/.claude/skills`; on Windows additionally `%LOCALAPPDATA%\crush\skills`, `%LOCALAPPDATA%\agents\skills`.
  - Project (checked at cwd **and** git worktree root; cwd wins): `.agents/skills`, `.crush/skills`, `.claude/skills`, `.cursor/skills`.
  - Extra dirs: `option skill-path <dir>` (crushrc) / `options.skills_paths` (JSON).
  - Note: **`.agents/skills` is natively read** — a canonical `.agents` store needs no symlink for skills.
- **Exact frontmatter keys** (YAML struct in internal/skills/skills.go):
  | Key | Type | Meaning |
  |---|---|---|
  | `name` | string, required | Skill ID; ≤64 chars; must match regex `^[a-zA-Z0-9]+(-[a-zA-Z0-9]+)*$` (no leading/trailing/consecutive hyphens); **must equal the containing directory name** (case-insensitive) |
  | `description` | string, required | When-to-use blurb shown to the model; ≤1024 chars |
  | `user-invocable` | bool | Expose in command palette (ctrl+p) as `user:<name>` / `project:<name>` |
  | `disable-model-invocation` | bool | Hide from model's skill list; user-invocation still works |
  | `license` | string, optional | License (Agent Skills standard key) |
  | `compatibility` | string, optional | ≤500 chars (Agent Skills standard key) |
  | `metadata` | map[string]string, optional | Arbitrary string→string metadata |
- **Crush-unique keys**: `user-invocable`, `disable-model-invocation` (invocation control; not in the base Agent Skills spec). `license`/`compatibility`/`metadata` come from the open standard.
- **Silently ignored keys**: anything else — notably Claude's `allowed-tools` — because parsing is non-strict `yaml.Unmarshal` and the struct has no such field (confirmed parsed-but-not-enforced in https://github.com/charmbracelet/crush/discussions/2352).
- Disable by name: `option disable-skill <name>` / `options.disabled_skills`. User skill with same name overrides a builtin (prompt.go).

## Hooks
- **Supported, one event only: `PreToolUse`** ("Crush currently supports just one hook"; https://raw.githubusercontent.com/charmbracelet/crush/main/docs/hooks/README.md). Event names are case-insensitive and snake-caseable (`pretooluse`, `pre_tool_use`, `PRE_TOOL_USE` all work). Planned-but-not-implemented events (`UserPromptSubmit`, `context_files`, sub-agent opt-in) are documented in docs/hooks/FUTURE.md.
- **Config file + format** (doc contradiction — both work):
  - Canonical (docs/config/README.md): `crushrc` builtin — `hook add <event> --command <cmd> [--name n] [--matcher regex] [--timeout sec]`.
  - Deprecated JSON (docs/hooks/README.md still teaches this): `"hooks": {"PreToolUse": [{...}]}` in `crush.json`/`.crush.json`, global or project.
- **Entry schema** (schema.json `HookConfig`): `name` (optional display name), `matcher` (optional regex tested against tool name, e.g. `^bash$`, `^mcp_`; omit = all tools), `command` (required, shell command — inline string or script path), `timeout` (optional seconds, default 30). Flat shape — no nested `hooks: [{type: "command", ...}]` array like Claude Code.
- **Stdin JSON envelope**: `{"event", "session_id", "cwd", "tool_name", "tool_input"}` — `tool_input` is the raw per-tool JSON. Also env vars: `CRUSH=1`, `AGENT=crush`, `AI_AGENT=crush`, `CRUSH_EVENT`, `CRUSH_TOOL_NAME`, `CRUSH_SESSION_ID`, `CRUSH_CWD`, `CRUSH_PROJECT_DIR`, `CRUSH_TOOL_INPUT_COMMAND` (bash only), `CRUSH_TOOL_INPUT_FILE_PATH` (file tools only).
- **Output contract**: exit `0` → stdout parsed as JSON envelope `{"version"?, "decision": "allow"|"deny"|null, "halt"?, "reason"?, "context": string|string[], "updated_input"?: object}`; exit `2` → block tool, stderr = deny reason; exit `49` → halt entire turn, stderr = reason; any other code → non-blocking, logged, tool proceeds. `decision: "allow"` pre-approves and skips the permission prompt. `updated_input` is a **shallow-merge patch** (deliberate divergence from Claude Code's full-replacement). Multiple hooks run in parallel, aggregate in config order: deny > allow > null; `halt` sticky; `reason`/`context` concatenated; `updated_input` merged in order.
- Hooks fire **only on the top-level agent's** tool calls; sub-agent inner loops are exempt (the outer `agent` call itself is hooked). Relative `command` paths resolve against CWD, not the config file. Execution via embedded POSIX shell (mvdan.cc/sh); shebang'd scripts dispatch to the named interpreter.
- Advertised as "Claude Code-compatible" for config shape/stdin/exit codes, with the `updated_input` shallow-merge caveat (docs/hooks/README.md "Claude Code compatibility").

## Subagents
**Not supported as user config.** Crush has exactly two builtin agents hardcoded in source: `coder` (main) and `task` (sub-agent invoked via the `agent` tool) — constants `AgentCoder`/`AgentTask` in internal/config/config.go; prompt templates in `internal/agent/templates/task.md.tpl`. The config schema has **no** `agents` key (schema.json Config properties: `models, providers, mcp, lsp, options, permissions, tools, hooks, env` only) and there is no agents directory. Early crush versions had a JSON `agents` key; it was removed. Hooks docs confirm sub-agents exist only internally (`agent` task tool, `agentic_fetch`).

## Commands
**Not supported as folder-based config.** No `commands` key in schema.json, no `.crush/commands/` dir. The TUI has builtin slash commands; the custom-invocation mechanism is **user-invocable skills**: `user-invocable: true` in SKILL.md frontmatter surfaces the skill in the command palette (ctrl+p) as `user:<name>` (global dirs) or `project:<name>` (project dirs) (README "User-Invocable Skills").

## Rules
**Not a distinct feature.** There is no rules dir/format of Crush's own; `.cursorrules` and `.cursor/rules/` are simply entries in the default context-paths list, loaded verbatim as prompt context (internal/config/config.go `defaultContextPaths`; prompt.go). Any directory path added via `option context-path` behaves the same (recursive walk). `.crushignore` (gitignore syntax) exists but controls file exclusion, not rules.

## MCP
- **Config**: crushrc `mcp add <name> --type stdio|sse|http [--command c] [--args a]... [--env k v]... [--url u] [--header k v]... [--timeout s] [--disabled] [--disabled-tools t]... [--enabled-tools t]... [--oauth ...]`; `mcp remove <name>` (docs/config/README.md). Legacy JSON: top-level **`mcp`** object (NOT `mcpServers`) mapping name → server (README "MCPs", schema.json).
- **Server-table shape** (schema.json `MCPConfig`; `type` is required): `{type: "stdio"|"sse"|"http" (default stdio), command, args[], env{}, url, headers{}, disabled, disabled_tools[], enabled_tools[], timeout (default 10s), oauth, oauth_client_id, oauth_client_secret, oauth_callback_port}`.
- **Env handling**: `env` map for stdio servers; in JSON, selected string fields (API keys, URLs, MCP/LSP commands/args, headers) undergo `$VAR`/`$(cmd)` shell expansion at load time; in crushrc everything is just Bash. A header/env value that expands to empty is dropped from the request. Top-level `"env": {...}` sets process env at startup (README "Environment Variables"; internal/config/config.go `MCPConfig` comments).
- Transports: `stdio`, `http`, `sse` (README features list).

## Translation hazards
1. **No folder-based config dir**: project config is root-level files (`crushrc`/`.crushrc`/`crush.json`/`.crush.json`), not `.crush/*`. `.crush/` is machine state (logs, workspace JSON) — an adopt/symlink mapping of `.crush/* → .agents/crush/*` would suck in state. Only `.crush/skills` under it is user config.
2. **Config format is executable Bash**: canonical stores must emit `mcp add`/`hook add`/`option`/`permissions` command lines (or deprecated JSON). There is no declarative YAML/TOML target. `crushrc` and JSON merge with precedence project > global, crushrc > JSON per directory.
3. **MCP table name**: `mcp`, not Claude's `mcpServers`. Per-server `type` is **required** (enum stdio/sse/http; Claude uses no `type` field). Env values may contain `$VAR`/`$(cmd)` expansions — canonical env-key-only comparison still works, but emitting values needs care. Crush-only server keys to drop when targeting other agents: `disabled_tools`, `enabled_tools`, `timeout`, `oauth*`, `disabled`.
4. **Skills — mostly compatible**: `.agents/skills/<name>/SKILL.md` is natively discovered (as are `.claude/skills`, `.cursor/skills`). Required normalizations:
   - `name` must match the parent directory name (case-insensitive) — rename dirs or names on mismatch.
   - `name` charset: alphanumeric + single hyphens only, ≤64 chars — slugs with underscores/spaces/dots are rejected.
   - `description` ≤1024 chars, `compatibility` ≤500 chars — truncate or fail.
   - Drop/ignore Claude-only frontmatter keys (`allowed-tools`, etc.) — silently ignored by Crush, so they need not be removed for Crush, but they have **no effect** (permissions must go to `permissions allow` instead).
   - `metadata` must be flat `map[string]string`.
   - Crush-unique keys `user-invocable`/`disable-model-invocation` should be dropped when translating away from Crush (other tools won't know them; Claude treats unknown keys leniently).
5. **Hooks — event gap**: only `PreToolUse` exists. `PostToolUse`, `SessionStart`, `UserPromptSubmit`, `Notification`, `Stop`, etc. must be dropped. Event names are case/snake tolerant, easing mapping.
6. **Hook entry shape**: flat `{name?, matcher?, command, timeout?}` per event — Claude's nested `hooks: [{type: "command", command}]` inside a matcher group must be flattened. Timeout default 30s (Claude: 60s).
7. **Hook envelope**: stdin uses `event`/`session_id`/`cwd`/`tool_name`/`tool_input` (Claude-aligned snake_case, but Claude calls the event field `hook_event_name` — rename needed from Claude envelopes; agentlink's guard must accept `event`). Output is flat `decision`/`updated_input`/`context`/`halt`/`reason` — NOT Claude's `hookSpecificOutput.permissionDecision` nesting; `updated_input` is snake_case and a **shallow merge** (Claude: `updatedInput`, full replace). Exit code **49** (halt turn) is crush-specific; exit 2 semantics match Claude.
8. **Subagents**: no user-defined subagent files — drop `.claude/agents/*.md` equivalents entirely; there is nothing to translate to.
9. **Slash commands**: no commands dir — translate canonical commands to user-invocable skills (folder + SKILL.md with `user-invocable: true`) or drop.
10. **Instructions**: `AGENTS.md` is natively read (zero-copy from a canonical store). No heading convention — do not rely on the Claude/Codex title-regex normalizer; crush adds/needs no heading. No `@import` syntax — flatten any includes before writing. Also natively reads `CLAUDE.md`, `CRUSH.md`, `.cursorrules`, `.cursor/rules/` — beware double-loading if a canonical store also writes those names.
11. **Rules**: `.cursor/rules/*.mdc` would be loaded verbatim including Cursor frontmatter (`alwaysApply`, `globs`) as raw prompt text — strip frontmatter if routing cursor-style rules into crush context.
12. **Global instruction files differ per tool**: crush reads `~/.config/crush/CRUSH.md` and `~/.config/AGENTS.md` (not `~/.claude/CLAUDE.md`).
13. **Ignore file**: `.crushignore` (gitignore syntax) is crush-specific.
14. **Windows divergence**: config at `%USERPROFILE%\.config\crush\crushrc`, extra skill dirs under `%LOCALAPPDATA%`, data at `%LOCALAPPDATA%\crush`.

## Verified corrections (fact-checker pass)

- was: "A header/env value that expands to empty is dropped from the request" → now: Only header values expanding to empty are dropped (ResolvedHeaders skips empty); MCP/LSP env values expanding to empty are KEPT as KEY= — resolveEnvs has no empty-drop and LSPConfig.ResolvedEnv's comment states empties are kept 'matching MCPConfig.ResolvedEnv'; docs promise the drop for headers only (https://raw.githubusercontent.com/charmbracelet/crush/main/internal/config/config.go; https://raw.githubusercontent.com/charmbracelet/crush/main/docs/config/README.md)
