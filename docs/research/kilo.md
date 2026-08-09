# kilo

## Identity — binary, vendor, install, docs URLs

- **Binary**: `kilo` (npm package `@kilocode/cli`). Kilo CLI 1.0 is a fork of OpenCode, built from `Kilo-Org/kilocode`; config behavior is "identical to OpenCode" per the plugins page.
- **Vendor**: Kilo-Org (Kilo Code), kilo.ai.
- **Install**: `npm install -g @kilocode/cli` (also curl `https://kilo.ai/cli/install | bash`, pnpm, bun, `brew install Kilo-Org/tap/kilo`, AUR `kilo-bin`, or binaries from GitHub Releases incl. `-baseline` variants for non-AVX CPUs).
- **Maintenance**: actively maintained; docs carry a "applies to Kilo version 1.0 and later" notice and the repo ships frequent releases.
- **URLs actually read**:
  - https://kilo.ai/docs/code-with-ai/platforms/cli (also served at https://kilo.ai/docs/cli — same page)
  - https://kilo.ai/docs/customize/skills
  - https://kilo.ai/docs/customize/custom-subagents
  - https://kilo.ai/docs/customize/custom-rules
  - https://kilo.ai/docs/customize/custom-instructions
  - https://kilo.ai/docs/customize/workflows
  - https://kilo.ai/docs/automate/mcp/using-in-kilo-code
  - https://kilo.ai/docs/automate/extending/plugins

## Config dirs — project dir, global/user dir, discovery/precedence rules

Source: https://kilo.ai/docs/code-with-ai/platforms/cli#configuration

- **Global/user dir**: `~/.config/kilo/` (XDG-style). Config file `kilo.json[c]`; legacy `opencode.json[c]` also read. TUI settings in `tui.json[c]` in the same dir.
- **Project**: `./kilo.json[c]` (project root), legacy `./opencode.json[c]`, or config inside `./.kilo/` (legacy `./.kilocode/` also read). Project-level config takes precedence over global.
- **No more opencode fallback**: Kilo no longer reads `~/.config/opencode` or `./.opencode/` configs.
- **Trusted vs project config**: `{env:VAR}` and `{file:...}` references resolve ONLY in trusted config (global `~/.config/kilo`, `KILO_CONFIG`/`KILO_CONFIG_CONTENT`, or MDM-managed config). Project-level `kilo.json` cannot use `{env:VAR}` (ignored + warning); `{file:...}` works but only for files inside the project root.
- **Env overrides**: `KILO_PROVIDER`, `KILO_<FIELD>` (other providers), `KILOCODE_<FIELD>` (kilocode provider), `KILO_ORG_ID`, `KILO_PURE=1` (disable external plugins), `KILO_DISABLE_SKILL_SHELL`.

## Instructions file — filename(s), location, heading/title conventions, import/include syntax if any

Source: https://kilo.ai/docs/customize/custom-instructions and https://kilo.ai/docs/code-with-ai/platforms/cli

- **Primary filename**: `AGENTS.md` (built-in `/init` command "Create/update AGENTS.md file for the project"). Also auto-discovered: `CLAUDE.md` (compatibility) and `CONTEXT.md`. Discovery walks up parent directories (`findUp`) from the project root.
- **Global instructions**: `~/.config/kilo/AGENTS.md`; also `~/.claude/CLAUDE.md` (Claude-compatible). Project-level loads before global.
- **Per-directory**: `AGENTS.md` in any subdirectory is injected as `<system-reminder>` when the Read tool touches files there (monorepo support).
- **Extra instruction sources**: `instructions` array key in `kilo.jsonc` — paths, globs, or URLs (URLs fetched at session start, 5s timeout, silently skipped on failure). Example from docs: `["CONTRIBUTING.md", ".cursor/rules/*.md"]`.
- **Heading conventions**: none documented — content is free-form markdown; no title-normalization convention like Claude Code's.
- **Import/include syntax**: none inside the files themselves; inclusion is via the `instructions` config array or `{file:./path}` references in config (trusted-config rules apply).
- **Legacy**: `.kilocoderules` files still auto-migrated.

## Skills — supported? dir, file layout, EXACT frontmatter keys with meaning, which keys are unique to this tool, size/naming limits

**Supported.** Source: https://kilo.ai/docs/customize/skills (implements the open Agent Skills spec, agentskills.io).

- **Dirs**:
  - Project: `.kilo/skills/<skill-name>/SKILL.md`
  - Global: `~/.kilo/skills/` (macOS/Linux; `\Users\<user>\.kilo\skills\` on Windows). Note: this is `~/.kilo` for skills while general config lives in `~/.config/kilo` — two distinct global roots.
  - Compatibility dirs loaded by default / when enabled: `.agents/skills/` (loaded by default) and `.claude/skills/` (when Claude Code Compatibility enabled); trusted global equivalents `~/.agents/skills/`, `~/.claude/skills/`.
  - Extra locations: `skills.paths` (absolute, `~/`-relative, or project-relative) and `skills.urls` (remote skill dirs serving an `index.json` manifest) in `kilo.jsonc`.
- **Layout**: `<skill-name>/SKILL.md` required; optional `scripts/`, `references/`, `assets/` subdirs.
- **Frontmatter keys** (YAML):
  - `name` (required) — max 64 chars, lowercase letters/numbers/hyphens, must not start/end with hyphen.
  - `description` (required) — max 1024 chars; drives agent's skill-selection decision.
  - `license` (optional)
  - `compatibility` (optional) — environment requirements
  - `metadata` (optional) — arbitrary key-value map
- **Tool-unique keys**: none — all keys come from the Agent Skills spec; Kilo adds no proprietary frontmatter keys (no `allowed-tools` equivalent).
- **Naming rule contradiction (docs)**: one section says `name` **must match** the parent directory name (error table lists "name doesn't match directory"), while the Troubleshooting section says "The `name` does not need to match the directory name but should be unique". Both from https://kilo.ai/docs/customize/skills — safest target behavior: make `name` == directory name.
- **Precedence**: project `.kilo/skills/` > global `~/.kilo/skills/` on name collision; compat dirs and `skills.paths`/`urls` load alongside.
- **Extra**: SKILL.md body may embed shell commands via `` !`command` `` — executed only for trusted skills (global dirs, built-ins, absolute paths in global config), after a permission prompt; never for project or remote-URL skills. `KILO_DISABLE_SKILL_SHELL` kills it.

## Hooks — supported? config file + format, event names, matcher schema, command schema, stdin JSON envelope fields, output contract

**Shell-command hooks (Claude-Code style PreToolUse/PostToolUse JSON): not supported.** Kilo has no hook config file, no matchers, no stdin JSON envelope, no stdout output contract. Instead it has a **JavaScript/TypeScript plugin system** (inherited from OpenCode). Source: https://kilo.ai/docs/automate/extending/plugins

- **Mechanism**: `.ts`/`.js` modules exporting `async (ctx) => Hooks`. Loaded from `plugin` array in `kilo.json[c]` (npm specifiers, `file:` URLs, local paths) or auto-registered from plugin dirs: global `~/.config/kilo/plugin/`, project `.kilo/plugin/` (legacy `.kilocode/plugin/`). Both `plugin/` and `plugins/` folder names work.
- **Hook event names** (object keys, NOT Claude names):
  - Lifecycle: `config`, `event` (catch-all bus)
  - Tools: `tool`, `tool.execute.before`, `tool.execute.after`, `tool.definition`
  - Chat: `chat.message`, `chat.params`, `chat.headers`, `permission.ask`, `command.execute.before`, `shell.env`
  - Providers: `auth`, `provider`
  - Experimental: `experimental.chat.messages.transform`, `experimental.chat.system.transform`, `experimental.session.compacting`, `experimental.compaction.autocontinue`, `experimental.text.complete`
  - Bus events (via `event` hook): `session.created/updated/idle/error/deleted/compacted/diff/status`, `message.updated/removed`, `message.part.updated/removed`, `tool.execute.before/after`, `permission.asked/replied`, `file.edited`, `file.watcher.updated`, `shell.env`, `command.executed`, `lsp.updated`, `lsp.client.diagnostics`, `todo.updated`, `server.connected`, `installation.updated`.
- **Input/output contract**: JS function signature `(input, output)` — e.g. `tool.execute.before` receives `input.tool` and mutable `output.args`; blocking is done by `throw new Error(...)`. No stdin/stdout, no JSON envelope, no exit codes.
- **Closest analog to a permission hook**: `permission.ask` can auto-allow/auto-deny prompts; `tool.execute.before` can block dangerous ops.
- **Kill switch**: `KILO_PURE=1` skips all external plugins.

## Subagents — dir + file format + frontmatter

**Supported** (called "agents"; "subagent" is a mode). Source: https://kilo.ai/docs/customize/custom-subagents

- **Dirs**: project `.kilo/agents/*.md`; global `~/.config/kilo/agents/*.md`. Filename (minus `.md`) is the agent name. Symlinked external dirs need `permission.markdown_source` allow rules in global config.
- **Alternate definition**: `agent` object in `kilo.jsonc` (keys = agent names; `prompt` may use `{file:./path}`).
- **Format**: Markdown with YAML frontmatter; body = system prompt.
- **Frontmatter/JSON keys**: `description`, `mode` (`subagent`|`primary`|`all`, default `all`), `model` (`provider/model-id`), `prompt` (JSON only), `temperature`, `top_p`, `permission` (object, `allow`/`ask`/`deny` per tool with glob sub-rules), `hidden`, `steps` (max agentic iterations), `color`, `disable`. Unknown extra keys are passed through to the model provider (e.g. `reasoningEffort`).
- **Precedence** (later wins): built-ins < global config < project config < global agent md files < project agent md files.
- Built-in subagents: `general`, `explore`.

## Commands — slash-command dir + format

**Supported** (docs call them "Workflows"). Source: https://kilo.ai/docs/customize/workflows

- **Dirs**: project `.kilo/commands/`; global `~/.config/kilo/commands/`. Filename minus `.md` = `/name`. External symlinks need `permission.markdown_source` allow rules.
- **Format**: Markdown + optional YAML frontmatter.
- **Frontmatter keys**: `description` (picker text), `agent` (which agent runs it), `model` (override), `subtask` (`true` = run as sub-agent session).
- Legacy `.kilocode/workflows/` auto-migrated.

## Rules — rules dir + format (if distinct from instructions)

**Supported, but wired through the `instructions` config key, not auto-discovered dirs.** Source: https://kilo.ai/docs/customize/custom-rules

- **Project rules**: files conventionally in `.kilo/rules/*.md`, each path or glob listed in the `instructions` array of project `kilo.jsonc`. **Global rules**: same via `instructions` in `~/.config/kilo/kilo.jsonc`.
- **Format**: plain text or Markdown (Markdown recommended); no frontmatter, no `.mdc` extension, no per-rule metadata — so no globs/scoping per rule file.
- **Load order**: global instructions first, then project; globs in filesystem order; project wins conflicts.
- Legacy `.kilocode/rules/` dirs auto-included for backward compat.

## MCP — config file, format, server-table shape, env handling

**Supported.** Source: https://kilo.ai/docs/automate/mcp/using-in-kilo-code and https://kilo.ai/docs/code-with-ai/platforms/cli

- **Config file**: `mcp` key inside `kilo.json[c]` — global `~/.config/kilo/kilo.jsonc`, project `./kilo.jsonc` or `./.kilo/kilo.jsonc`. Project > global. JSONC (comments allowed). No separate `.mcp.json` file.
- **Server-table shape** (top-level key = server name):
  - Local: `{ "type": "local", "command": ["node", "/path/server.js"], "environment": {"API_KEY": "..."}, "enabled": true, "timeout": 10000 }` — note `command` is an **array** (argv), env map key is `environment`.
  - Remote: `{ "type": "remote", "url": "https://...", "headers": {...}, "enabled": true, "timeout": 15000, "oauth": false }` — StreamableHTTP tried first, SSE fallback; OAuth 2.0 automatic unless disabled.
  - Timeouts in ms: default 10s local, 15s remote.
- **Env handling**: literal values in `environment`/`headers`, or `{env:VAR_NAME}` references — but `{env:...}` resolves ONLY in trusted (global) config, never project config. `KILO_<FIELD>`/`KILOCODE_<FIELD>` env overrides exist for provider config, not MCP.
- **Permissions**: MCP tools are permissioned as `{server}_{tool}` namespaced keys with wildcard support (`my_server_*`).

## Translation hazards — every concrete thing a canonical `.agents` store must drop/rename/reformat to target this agent

1. **Hooks: untranslatable 1:1.** Claude-Code-style shell hooks (PreToolUse/PostToolUse, matchers, stdin JSON, exit codes) have NO config-file equivalent. Nearest mapping is a generated TS plugin: `PreToolUse` → `tool.execute.before` (block via `throw`), `PostToolUse` → `tool.execute.after`, permission hooks → `permission.ask`. Event names differ entirely (`tool.execute.before` not `PreToolUse`); there is no JSON envelope, no matcher schema — matching is done in JS code (`input.tool === "bash"`). A canonical store should either drop hooks or emit `.kilo/plugin/*.ts` shims.
2. **Config format**: single JSONC file (`kilo.jsonc`), NOT separate per-concern files, NOT TOML. Schema URL `https://app.kilo.ai/config.json`. `~/.config/kilo/` (XDG) for global config, but `~/.kilo/` for global skills — two different global roots; do not collapse them.
3. **Instructions**: canonical `AGENTS.md` maps directly (Kilo's primary file IS `AGENTS.md`); `CLAUDE.md` content also works. Extra rule files must be listed by path/glob in the `instructions` array of `kilo.jsonc` — dropping a rule file into `.kilo/rules/` alone does nothing unless registered (only legacy `.kilocode/rules/` auto-loads).
4. **Skills**: keep spec-standard frontmatter only — `name`, `description`, `license`, `compatibility`, `metadata`. Drop Claude-only keys (`allowed-tools`, `model`, etc.) — docs don't list them and Kilo follows the agentskills.io spec. Enforce `name` == directory name (one docs section requires it; the contradiction means matching is the only safe choice). Enforce name charset: ≤64 chars, `[a-z0-9-]`, no leading/trailing hyphen; description ≤1024 chars. Strip `` !`cmd` `` shell placeholders for project-deployed skills (they never execute from project dirs and render as untrusted markers).
5. **Skill dirs**: project `.kilo/skills/` (Kilo loads `.agents/skills/` natively as a compat dir — a canonical `.agents/skills` tree works out of the box); global `~/.kilo/skills/` (NOT `~/.config/kilo/skills/`).
6. **Subagents**: rename canonical agent frontmatter to Kilo keys: `tools` → `permission` (object of `allow|ask|deny`, not a list), add `mode: subagent` (default `all` would expose it as a primary agent). Drop unsupported canonical keys; note Kilo-only keys (`mode`, `hidden`, `steps`, `color`, `disable`, `top_p`) shouldn't leak back into the canonical store. Body markdown = prompt; no `prompt` key needed in md files. Filename = agent name.
7. **Commands**: dir is `.kilo/commands/` (global `~/.config/kilo/commands/`); frontmatter keys are `description`, `agent`, `model`, `subtask` — drop Claude-style `allowed-tools`, `argument-hint`, etc. No documented argument-substitution placeholders in command bodies.
8. **MCP**: reformat server blocks — `command` must be a single **argv array** (split canonical `command` + `args` into one array), env map key is `environment` (not `env`), each server needs explicit `type: "local"|"remote"`, key is `mcp` (not `mcpServers` — the docs' legacy SSE example uses `mcpServers` but current format is `mcp`). `{env:VAR}` in MCP config only works in global config — project-targeted translations must inline values or warn. Drop `alwaysAllow`/`disabled` legacy keys → use `enabled` and `permission` rules.
9. **Trust model**: anything using `{env:...}` or `{file:...}` in project-level config (`kilo.jsonc`, `.kilo/agents/*.md`, `.kilo/commands/*.md`) silently fails unless config is trusted (global) — translations targeting project scope must inline or relocate to global config. External symlinks for agents/commands/skills need `permission.markdown_source` allow rules in global config.
10. **Naming/legacy aliases to avoid emitting**: `.kilocode/`, `.opencode/`, `opencode.json`, `.kilocode/rules/`, `.kilocode/workflows/`, `.kilocoderules` are all legacy/compat read paths — translations should target `.kilo/` + `kilo.jsonc` only. Also note the `kilo plugin` CLI still writes `.kilo/opencode.jsonc` (docs show this) — an internal inconsistency; prefer `kilo.jsonc` when writing directly.

## Verified corrections (fact-checker pass)

- Hazard #4: 'Drop Claude-only keys (allowed-tools, model, etc.)' → allowed-tools is NOT Claude-only; it is an optional (experimental) field defined in the Agent Skills spec itself — Kilo's docs merely omit it from their frontmatter table, so dropping it for Kilo targets is still right but the rationale is wrong (https://agentskills.io/specification)
- Hooks mechanism: '.ts/.js modules exporting async (ctx) => Hooks' → plugin modules must default-export a module descriptor { id, server } where server is the async (ctx) => Hooks function; id is required for local-file plugins, and bare function exports are only accepted as legacy named exports — emitted .kilo/plugin/*.ts shims need the { id, server } default export (https://kilo.ai/docs/automate/extending/plugins)
- Skill name rule stated as '≤64 chars, [a-z0-9-], no leading/trailing hyphen' → the spec additionally forbids consecutive hyphens (--) and caps compatibility at 500 chars; enforce those too when validating generated SKILL.md files (https://agentskills.io/specification)
