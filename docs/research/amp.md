# amp

## Identity
- **Binary:** `amp` (npm package `@ampcode/cli`) — https://ampcode.com/manual/get-started.md
- **Vendor:** Sourcegraph (bug-bounty scope covers "assets… owned by Amp or Sourcegraph"; auth terms link to sourcegraph.com) — https://ampcode.com/security
- **Install:** `curl -fsSL https://ampcode.com/install.sh | bash` (macOS/Linux/WSL), `brew install ampcode/tap/ampcode`, `powershell -c "irm https://ampcode.com/install.ps1 | iex"` (Windows), npm `@ampcode/cli` "(not recommended)"; updates via `amp update` — https://ampcode.com/manual/get-started.md
- **Docs URLs actually read:** https://ampcode.com/manual (index), https://ampcode.com/manual/get-started.md, https://ampcode.com/manual/usage.md (contains the AGENTS.md section; the `/manual/AGENTS.md` ToC link 404s and `/manual/AGENTS` requires auth), https://ampcode.com/manual/agent-skills.md, https://ampcode.com/manual/tools.md, https://ampcode.com/manual/configuration.md, https://ampcode.com/manual/mcp.md, https://ampcode.com/manual/plugins.md, https://ampcode.com/manual/plugin-api.md, https://ampcode.com/manual/subagents.md, https://ampcode.com/manual/define-a-custom-subagent.md, https://ampcode.com/manual/cli.md, https://ampcode.com/manual/why-amp.md, https://ampcode.com/security, https://ampcode.com/news/agent-skills, https://ampcode.com/news/slashing-custom-commands
- Actively maintained: manual is current (features dated Jan 2026+, e.g. https://ampcode.com/news/slashing-custom-commands).

## Config dirs
- **Project/workspace dir:** `.amp/` — holds `settings.json` or `settings.jsonc` and `plugins/`. Amp uses "the nearest `.amp/settings.json`… searched upward from your current working directory to the repository root" — https://ampcode.com/manual/configuration.md
- **Global/user dir:** `~/.config/amp/` on macOS/Linux, `%USERPROFILE%\.config\amp\` on Windows — holds `settings.json(.jsonc)`, `plugins/`, `AGENTS.md`, `skills/`, `checks/`, `keymap` — https://ampcode.com/manual/configuration.md, https://ampcode.com/manual/plugins.md, https://ampcode.com/manual/tools.md
- **Other state dirs:** OAuth tokens `~/.amp/oauth/` (https://ampcode.com/manual/mcp.md); credentials `~/.local/share/amp/secrets.json` (https://ampcode.com/security).
- **Skill dirs (global):** `~/.config/agents/skills/`, `~/.agents/skills/`, `~/.config/amp/skills/` — https://ampcode.com/manual/agent-skills.md
- **Enterprise managed settings (override all):** macOS `/Library/Application Support/ampcode/managed-settings.json`, Linux `/etc/ampcode/managed-settings.json`, Windows `%ProgramData%\ampcode\managed-settings.json` — https://ampcode.com/manual/configuration.md
- **Precedence:** workspace settings override user settings; managed settings override both. Exception: keymaps in *user* settings override workspace entries — https://ampcode.com/manual/configuration.md, https://ampcode.com/manual/cli.md
- Settings files are JSON/JSONC; **all keys use the `amp.` prefix** — https://ampcode.com/manual/configuration.md

## Instructions file
- **Filename:** `AGENTS.md`. Fallback: if no `AGENTS.md` in a directory but `AGENT.md` (no S) or `CLAUDE.md` exists, that file is included — https://ampcode.com/manual/usage.md
- **Discovery:** `AGENTS.md` in cwd (or editor workspace roots) *and* parent directories up to `$HOME` are always included; subtree `AGENTS.md` files load when the agent reads a file in that subtree; global files `~/.config/amp/AGENTS.md` and `~/.config/AGENTS.md` always included; system-wide `/etc/ampcode/AGENTS.md`, `/Library/Application Support/ampcode/AGENTS.md`, `%ProgramData%\ampcode\AGENTS.md` — https://ampcode.com/manual/usage.md
- **Heading/title conventions:** none documented — plain Markdown body; no required H1.
- **Import/include syntax:** `@`-mentions inside the file, e.g. `See @doc/style.md and @specs/**/*.md`. Relative paths resolve relative to the containing agent file; absolute paths and `@~/…` supported; glob patterns supported; `@`-mentions inside code blocks are ignored — https://ampcode.com/manual/usage.md
- **Conditional inclusion:** mentioned files may carry YAML frontmatter `globs: ['**/*.ts']`; the file is included only if Amp has read a matching file. Globs implicitly prefixed `**/` unless starting `../` or `./` — https://ampcode.com/manual/usage.md

## Skills
- **Supported.** Skills are directories containing `SKILL.md` plus optional resources — https://ampcode.com/manual/agent-skills.md
- **Dirs & precedence (first match on frontmatter `name` wins):** 1. `~/.config/agents/skills/` 2. `~/.agents/skills/` 3. `~/.config/amp/skills/` 4. `.agents/skills/` in project *and searched parent directories* 5. `.claude/skills/` in those dirs 6. `~/.claude/skills/` 7. `~/.claude/plugins/cache/` 8. dirs in `amp.skills.path` (colon-separated) 9. built-in skills 10. personal skills repo 11. workspace skills repo. Claude locations skippable via `amp.skills.disableClaudeCodeSkills` — https://ampcode.com/manual/agent-skills.md
- **File layout:** one top-level directory per skill with `SKILL.md` directly inside; **the directory name and the `name` in SKILL.md must match**. Sibling `scripts/`, `references/`, etc. allowed; agent accesses them via paths relative to the skill directory — https://ampcode.com/manual/agent-skills.md
- **Frontmatter keys (YAML):**
  - `name` (required) — identifier; always visible to the model; must equal directory name
  - `description` (required in practice) — always visible; drives when the model loads the skill
  - `mcpServers` — object of MCP server definitions; if present, a sibling `mcp.json` is ignored
  - Only these three keys are documented. Unique-to-Amp: `mcpServers` (Claude does not document this key). No documented `allowed-tools`, `license`, or `metadata` support (Claude keys are not mentioned anywhere in Amp's docs).
- **Size/naming limits:** no size limits documented. Only naming rule: dir name == frontmatter `name`. Body of `SKILL.md` is loaded only when the skill is invoked (lazy loading) — https://ampcode.com/manual/agent-skills.md
- Skill-bundled MCP: sibling `mcp.json` (flat map, see MCP section) or `mcpServers` frontmatter; tools from skill servers stay hidden until the skill loads — https://ampcode.com/manual/agent-skills.md

## Hooks
- **Claude-style shell-command hooks: not supported.** There is no `hooks` key in Amp settings and no stdin-JSON-hook mechanism anywhere in the manual (https://ampcode.com/manual/configuration.md lists every setting; none is hook-related). The equivalent is the **TypeScript plugin event system** — https://ampcode.com/manual/plugins.md
- **Config file + format:** plugins are `.ts`/`.js` files (single file or directory with `index.ts`/`index.js`) in `.amp/plugins/` (project) or `~/.config/amp/plugins/` (user), plus personal/workspace plugin repos — https://ampcode.com/manual/plugins.md
- **Event names (exact):** `session.start`, `tool.call`, `tool.result`, `agent.start`, `agent.end`. No `session.end` exists. Registered via `amp.on('<event>', handler)` — https://ampcode.com/manual/plugin-api.md (`PluginEventMap`)
- **Matcher schema:** none. No glob/tool matchers; the handler inspects `event.tool` / `event.input` in code (helpers `amp.helpers.shellCommandFromToolCall(event)`, `amp.helpers.filesModifiedByToolCall(event)`) — https://ampcode.com/manual/plugin-api.md
- **"Command" schema / stdin envelope:** N/A — handlers are in-process JS functions receiving typed event objects, not shell commands reading JSON on stdin. Event payloads:
  - `session.start`: `{ thread: { id } }`
  - `tool.call`: `{ toolUseID, tool, input: Record<string,unknown>, thread: { id } }`
  - `tool.result`: `{ toolUseID, tool, input, status: 'done'|'error'|'cancelled', error?, output?, thread: { id } }`
  - `agent.start`: `{ thread: { id }, message: string, id: ThreadMessageID }`
  - `agent.end`: `{ thread: { id }, message, id, status, messages: ThreadMessage[] }`
  — https://ampcode.com/manual/plugin-api.md
- **Output contract (handler return values, not stdout/exit codes):**
  - `tool.call` → `{action:'allow'}` | `{action:'reject-and-continue', message}` | `{action:'modify', input}` | `{action:'synthesize', result:{output, exitCode?}}` | `{action:'error', message}`
  - `tool.result` → replacement `{status, output?, error?}` or `undefined` to keep original
  - `agent.start` → `{ message?: { content, display? } }` (append context to user message)
  - `agent.end` → `{action:'continue', userMessage}` (starts a follow-up turn) or void
  - `session.start` → void
  — https://ampcode.com/manual/plugin-api.md
- Non-interactive caveat: `amp -x` needs `--plugin-ready-timeout` or `agent.start`/`agent.end` events may be skipped — https://ampcode.com/manual/cli.md

## Subagents
- **No file-based subagent directory/format.** Built-in subagents are spawned automatically by the main agent (mostly in `medium` mode) — https://ampcode.com/manual/subagents.md
- **Custom subagents exist only as plugin code:** `amp.createAgent({ name, model, instructions, tools, reasoningEffort })` exposed to the agent via `amp.registerTool(...)`; nothing Markdown/frontmatter-based — https://ampcode.com/manual/define-a-custom-subagent.md
- Fixed built-in named subagent-like tools: Oracle, Librarian, Painter — https://ampcode.com/manual/tools.md

## Commands
- **Removed as of 2026-01-29.** "Custom commands are gone. Use skills instead." — https://ampcode.com/news/slashing-custom-commands
- Former format (for migration reference only): `.agents/commands/<name>.md` (plain Markdown, no frontmatter; filename = command name) and global `~/.config/amp/commands/`; executable scripts with `#!` shebang were also supported. Official migration: `.agents/commands/code-review.md` → `.agents/skills/code-review/SKILL.md` with added `name`/`description` frontmatter; executables move to a `scripts/` subdir of the skill — https://ampcode.com/news/slashing-custom-commands
- Runtime commands now exist only as **command-palette commands** registered by plugins via `amp.registerCommand(name, {title, category, description, availability?}, handler)` — not prompt templates — https://ampcode.com/manual/plugins.md

## Rules
- **No dedicated rules directory/format** (nothing like `.cursor/rules`). The granular-guidance mechanism is the `globs:` YAML frontmatter key on files `@`-mentioned from `AGENTS.md` (see Instructions) — https://ampcode.com/manual/usage.md
- Closest distinct concept — **review checks**: `.agents/checks/<name>.md`, single Markdown files (not directories) with frontmatter `name` (required), `description`, `severity-default` (`low|medium|high|critical`), `tools` (array of tool names). Locations: `.agents/checks/` anywhere in the tree (scoped to that subtree), global `$HOME/.config/amp/checks/` or `$HOME/.config/agents/checks/`; closer project checks override same-named parent/global checks. Checks only run during `amp review` code review, each in its own subagent — https://ampcode.com/manual/tools.md

## MCP
- **Config file:** `amp.mcpServers` key in `.amp/settings.json` (workspace) or `~/.config/amp/settings.json` (user); also `--mcp-config '<json>'` CLI flag, `amp mcp add` CLI, or skill-level `mcp.json` / SKILL.md `mcpServers` — https://ampcode.com/manual/mcp.md
- **Format/shape:** JSON object mapping server name → server block. Local: `{command: string, args?: string[], env?: object}`. Remote: `{url: string, headers?: object}`. Common: `includeTools?: string[]` (names/globs filtering exposed tools — Amp-specific field). Skill `mcp.json` is the **flat** name→server map with **no** top-level `mcpServers` wrapper (unlike Claude's `.mcp.json`); in settings files the wrapper is the `amp.mcpServers` key — https://ampcode.com/manual/agent-skills.md, https://ampcode.com/manual/mcp.md
- **Env handling:** `env` object for local servers; in settings files, `${VAR}` interpolation in strings (e.g. `"url": "${SRC_ENDPOINT}/.api/mcp/v1"`, header `"Authorization": "token ${SRC_ACCESS_TOKEN}"`); OAuth automatic for many remote servers, tokens in `~/.amp/oauth/` — https://ampcode.com/manual/mcp.md
- **Precedence (high→low):** CLI `--mcp-config` → workspace `amp.mcpServers` → user `amp.mcpServers` → skills. Workspace-defined servers require explicit `amp mcp approve` before running — https://ampcode.com/manual/mcp.md

## Translation hazards
Canonical `.agents` store → Amp requires these drops/renames/reformats:

1. **Hooks cannot be translated as data.** No JSON/TOML hook config, no stdin envelope, no shell-command hooks, no matchers. Claude `PreToolUse`/`PostToolUse`/`SessionStart`/`UserPromptSubmit`/`Stop` map only conceptually to `tool.call`/`tool.result`/`session.start`/`agent.start`/`agent.end`, and the target artifact is a **TypeScript plugin** in `.amp/plugins/` returning typed objects (`{action:'allow'|'reject-and-continue'|'modify'|'synthesize'|'error'}`), not exit codes or stdout JSON. A canonical hook store must drop hooks for Amp or codegen a plugin. `Notification`, `SubagentStop`, `PreCompact`, `SessionEnd` have no Amp counterpart (no `session.end` exists). — https://ampcode.com/manual/plugin-api.md
2. **Slash commands must not be emitted** — the feature was removed (2026-01-29). Convert each canonical command to a skill: `.agents/skills/<name>/SKILL.md` with `name`+`description` frontmatter; executable commands become scripts under the skill's `scripts/` subdir referenced from SKILL.md (convention `{baseDir}/scripts/...`). — https://ampcode.com/news/slashing-custom-commands
3. **File-based subagents unsupported.** No `.claude/agents/*.md` equivalent; canonical subagent definitions (name/description/tools frontmatter) can only become plugin code via `amp.createAgent`. Drop or generate a plugin. — https://ampcode.com/manual/define-a-custom-subagent.md
4. **Skill frontmatter:** keep only `name`, `description`, `mcpServers`. Drop Claude-only keys (`allowed-tools`, `license`, `metadata`, etc.) — undocumented in Amp; no documented size limits. **Hard naming rule:** skill directory name must equal frontmatter `name` (mismatch breaks repo-installed skills). `SKILL.md` must sit directly inside the skill dir. — https://ampcode.com/manual/agent-skills.md
5. **Instructions filename:** Amp natively wants `AGENTS.md` (matches the agents.md standard — no rename needed from a canonical store) and reads `CLAUDE.md` only as a fallback when no `AGENTS.md`/`AGENT.md` exists in that directory. No heading convention to rewrite (unlike Claude/Codex title normalization). Include syntax is `@path` mentions with glob support — a canonical store using Claude `@import` syntax is compatible, but `globs:` conditional-inclusion frontmatter is Amp-specific and safe to emit only for Amp. — https://ampcode.com/manual/usage.md
6. **MCP shape mismatch:** (a) settings files key is `amp.mcpServers` (dotted prefix, inside `settings.json`, not a standalone `.mcp.json`); (b) skill `mcp.json` is a flat name→server map — do **not** wrap in a top-level `mcpServers` key as Claude's `.mcp.json` requires; (c) drop/rename Claude-only fields (`type: "stdio"` transport field is undocumented in Amp — transport is inferred from `command` vs `url`); (d) `includeTools` is Amp-specific, strip for other targets; (e) env interpolation is `${VAR}` in settings strings. — https://ampcode.com/manual/mcp.md, https://ampcode.com/manual/agent-skills.md
7. **Two canonical `.agents` collisions to note:** Amp already uses `.agents/skills/` and `.agents/checks/` as *native* project dirs — an agentlink canonical `.agents/` store maps 1:1 for skills (good), but `.agents/commands/` is dead and `.agents/checks/*.md` are single files with Amp-specific frontmatter (`severity-default`, `tools`), not rules. — https://ampcode.com/manual/tools.md
8. **Rules dir:** none exists; do not emit `.amp/rules/`. Conditional rules must be expressed as `@`-mentioned Markdown files with `globs:` frontmatter (Amp-only key name; Claude's equivalent `applyTo`/globs concepts differ). — https://ampcode.com/manual/usage.md
9. **Settings format:** JSON/JSONC with mandatory `amp.`-prefixed keys; workspace file must be `.amp/settings.json` (searched upward to repo root, not only at root). Workspace MCP servers trigger a trust prompt (`amp mcp approve`) — syncing MCP config into `.amp/settings.json` does not auto-enable servers. — https://ampcode.com/manual/configuration.md, https://ampcode.com/manual/mcp.md
10. **Compatibility surface agentlink can exploit:** Amp deliberately reads `.claude/skills/`, `~/.claude/skills/`, `~/.claude/plugins/cache/`, and `CLAUDE.md` fallbacks, so Claude-targeted artifacts are partially valid for Amp without translation (disable via `amp.skills.disableClaudeCodeSkills` if undesired). — https://ampcode.com/manual/agent-skills.md, https://ampcode.com/manual/usage.md

### Critical Files for Implementation
- N/A — research-only task, no repo files modified. If wiring Amp into agentlink, the coupling points above map to: config dirs (`.amp/`, `~/.config/amp/`), instructions (`AGENTS.md`), skills (`.agents/skills/<name>/SKILL.md`), MCP (`amp.mcpServers` in `.amp/settings.json`), and hooks (unsupported — plugin-only).