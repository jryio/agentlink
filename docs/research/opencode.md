# opencode

## Identity
- **Binary**: `opencode` (run `opencode` for TUI; also desktop app / IDE extension). Docs spell the product "OpenCode" but the CLI is lowercase `opencode`.
- **Vendor**: created by SST; the GitHub org moved from `sst` to `anomalyco` — the repo is now **github.com/anomalyco/opencode** (`sst/opencode` redirects). Site/docs remain opencode.ai. Actively maintained (~15k commits, `dev` default branch, 195k stars as of 2026-08).
- **Install**: `curl -fsSL https://opencode.ai/install | bash`; `npm install -g opencode-ai`; `brew install anomalyco/tap/opencode` (recommended tap; the core `opencode` formula lags); also pacman/AUR, choco, scoop, mise, Docker, GitHub Releases (https://opencode.ai/docs/).
- **URLs read**:
  - https://opencode.ai/docs/ (intro/install/init)
  - https://opencode.ai/docs/config/ (config locations, precedence, variable substitution)
  - https://opencode.ai/docs/rules/ (AGENTS.md instructions)
  - https://opencode.ai/docs/skills/ (Agent Skills)
  - https://opencode.ai/docs/plugins/ (hook system)
  - https://opencode.ai/docs/agents/ (subagents)
  - https://opencode.ai/docs/commands/ (slash commands)
  - https://opencode.ai/docs/mcp-servers/ (MCP)
  - https://github.com/anomalyco/opencode (repo; reached via github.com/sst/opencode redirect)
  - https://github.com/sst (org page: "We've moved to https://github.com/anomalyco")

## Config dirs
- **Project**: `opencode.json` or `opencode.jsonc` (JSON **and JSONC** both supported) at project root; discovery starts at cwd and traverses up to the nearest Git directory. Safe to commit (https://opencode.ai/docs/config/).
- **Project directory**: `.opencode/` — holds `agents/`, `commands/`, `plugins/`, `skills/` ("agents, commands, modes, and plugins" per config page; `skills/` per skills page). Loaded as its own precedence tier.
- **Global/user**: `~/.config/opencode/` (XDG-style, **not** `~/.opencode`) containing `opencode.json`, `tui.json` (TUI settings), `AGENTS.md`, `agents/`, `commands/`, `plugins/`, `skills/` (https://opencode.ai/docs/config/, https://opencode.ai/docs/rules/).
- **Data dir (separate from config)**: `~/.local/share/opencode/` — e.g. OAuth tokens in `mcp-auth.json` (https://opencode.ai/docs/mcp-servers/); plugin npm cache in `~/.cache/opencode/node_modules/` (https://opencode.ai/docs/plugins/).
- **Custom overrides**: `OPENCODE_CONFIG` (single file path), `OPENCODE_CONFIG_DIR` (extra directory searched like `.opencode/`), `OPENCODE_CONFIG_CONTENT` (inline JSON).
- **Precedence** (later overrides earlier, files are **merged** not replaced): 1) remote org defaults from `.well-known/opencode` → 2) global `~/.config/opencode/opencode.json` → 3) `OPENCODE_CONFIG` → 4) project `opencode.json` → 5) `.opencode` directories → 6) `OPENCODE_CONFIG_CONTENT` → 7) managed files (`/Library/Application Support/opencode/` macOS, `/etc/opencode/` Linux, `%ProgramData%\opencode` Windows) → 8) macOS MDM `.mobileconfig` (domain `ai.opencode.managed`) (https://opencode.ai/docs/config/).

## Instructions file
- **Filename**: `AGENTS.md` — project root (discovered by traversing up from cwd) and global `~/.config/opencode/AGENTS.md`. `/init` generates/updates it (https://opencode.ai/docs/rules/).
- **Claude Code fallbacks**: project `CLAUDE.md` (only if no `AGENTS.md`), global `~/.claude/CLAUDE.md` (only if no `~/.config/opencode/AGENTS.md`); disable with `OPENCODE_DISABLE_CLAUDE_CODE=1`, `OPENCODE_DISABLE_CLAUDE_CODE_PROMPT=1`, `OPENCODE_DISABLE_CLAUDE_CODE_SKILLS=1`. First matching file wins per category (https://opencode.ai/docs/rules/).
- **Heading/title convention**: none mandated; the docs' example starts with a free-form `# SST v3 Monorepo Project` H1 — content is plain markdown instructions.
- **Import/include syntax**: **none parsed**. `@path` references inside AGENTS.md are convention only (opencode "doesn't automatically parse file references"). Real includes go through the `instructions` array in `opencode.json`: accepts relative paths, **globs** (`".cursor/rules/*.md"`, `"packages/*/AGENTS.md"`) and **remote URLs** (fetched with 5 s timeout); all combined with AGENTS.md files (https://opencode.ai/docs/rules/).

## Skills
- **Supported**: yes — "Agent Skills", loaded on demand via the native `skill` tool (https://opencode.ai/docs/skills/).
- **Dirs** (one folder per skill, `SKILL.md` inside): project `.opencode/skills/<name>/SKILL.md`; global `~/.config/opencode/skills/<name>/SKILL.md`; plus compatibility locations `.claude/skills/`, `~/.claude/skills/`, **`.agents/skills/`**, **`~/.agents/skills/`** (same layout). Project discovery walks up from cwd to the git worktree.
- **File layout**: `<name>/SKILL.md`, YAML frontmatter + markdown body. Frontmatter keys recognized (**only these**):
  - `name` (required) — skill identifier
  - `description` (required) — shown to the agent in `<available_skills>` listing
  - `license` (optional)
  - `compatibility` (optional) — e.g. `compatibility: opencode`
  - `metadata` (optional) — string→string map
  - Unknown frontmatter fields are **ignored**.
- **Keys unique/notable vs Claude**: `compatibility` is opencode/agentskills-spec-specific; Claude's `allowed-tools` is **not supported** (silently ignored) — gating is done instead via `permission.skill` glob→`allow|ask|deny` in `opencode.json` or agent frontmatter, and `tools: {skill: false}` disables the tool.
- **Limits**: `name` 1–64 chars, regex `^[a-z0-9]+(-[a-z0-9]+)*$` (lowercase alnum, single hyphens, no leading/trailing/double hyphen) and **must match the containing directory name**; `description` 1–1024 chars (https://opencode.ai/docs/skills/).

## Hooks
- **Supported, but NOT shell-command hooks** — opencode's hook mechanism is **JS/TS plugins**, not config-declared commands (https://opencode.ai/docs/plugins/).
- **Config file + format**: plugin modules (`.js`/`.ts`) in `.opencode/plugins/` (project) or `~/.config/opencode/plugins/` (global), auto-loaded at startup; or npm package names in the `plugin` array in `opencode.json` (installed with Bun into `~/.cache/opencode/node_modules/`). Load order: global config plugins → project config plugins → global dir → project dir.
- **Schema**: a plugin exports `async ({project, client, $, directory, worktree}) => ({ ...hooks })`. Hook keys are dotted event names; values are `async (input, output)` functions. Key hooks:
  - `tool.execute.before` — `input.tool` = tool name (e.g. `"bash"`, `"read"`), `output.args` = tool arguments (mutable); **throw an Error to block** the tool call.
  - `tool.execute.after` — post-execution equivalent.
  - `shell.env` — `output.env` map to inject env into all shells; `input.cwd`.
  - `experimental.session.compacting` — mutate `output.context[]` or replace `output.prompt`.
  - `event` — generic subscriber receiving `{event}` with `event.type` ∈ `command.executed`, `file.edited`, `file.watcher.updated`, `installation.updated`, `lsp.client.diagnostics`, `lsp.updated`, `message.part.removed|updated`, `message.removed|updated`, `permission.asked|replied`, `server.connected`, `session.created|compacted|deleted|diff|error|idle|status|updated`, `todo.updated`, plus TUI hooks `tui.prompt.append`, `tui.command.execute`, `tui.toast.show`.
- **Matcher schema**: none — matching is imperative JS inside the hook (e.g. `if (input.tool === "bash")`).
- **Command schema / stdin JSON envelope / output contract**: **not supported** — there is no spawned shell command, no stdin JSON envelope, and no stdout/exit-code contract; the plugin runs in-process and signals outcomes by mutating `output` or throwing (https://opencode.ai/docs/plugins/).

## Subagents
- **Dir + format**: markdown files, one per agent — project `.opencode/agents/*.md`, global `~/.config/opencode/agents/*.md`. **Filename (sans `.md`) becomes the agent name** (`review.md` → `@review`) (https://opencode.ai/docs/agents/).
- **Frontmatter** (YAML): `description` (**required**, drives auto-invocation), `mode` (`primary` | `subagent` | `all`, default `all`), `model` (`provider/model-id`), `temperature`, `top_p`, `prompt` (inline or `{file:./prompts/x.txt}` relative to config file), `steps` (max agentic iterations), `disable` (bool), `hidden` (bool; hide from `@` autocomplete, still callable via Task tool), `color` (hex or theme name — UI only), `permission` (map: keys `read|edit|glob|grep|list|bash|task|external_directory|todowrite|webfetch|websearch|lsp|skill|question|doom_loop` → `allow|ask|deny` or glob→action object), `tools` (**deprecated** in favor of `permission`). Any other frontmatter/config keys are passed through to the provider as model options (e.g. `reasoningEffort`). Body = system prompt.
- Agents can also be defined in JSON under the `agent` key of `opencode.json`. Built-ins: primary `build`, `plan`; subagents `general`, `explore`, `scout`; hidden system agents `compaction`, `title`, `summary`.

## Commands
- **Dir + format**: markdown files — project `.opencode/commands/*.md`, global `~/.config/opencode/commands/*.md`; filename = slash-command name (`test.md` → `/test`). Custom commands override built-ins of the same name (https://opencode.ai/docs/commands/).
- **Frontmatter**: `description`, `agent` (which agent executes it), `model`, `subtask` (bool — force subagent invocation). Body is the prompt **template** supporting `$ARGUMENTS`, positional `$1`/`$2`/…, shell-output injection `` !`command` `` (runs in project root), and `@file` references (content inlined). JSON equivalent: `command` key in `opencode.json` with `template` (required), `description`, `agent`, `model`, `subtask`.

## Rules
- **Not a separate concept.** opencode has no rules directory/format distinct from instructions: rules = `AGENTS.md` files plus the `instructions` array (paths/globs/URLs) in `opencode.json` (https://opencode.ai/docs/rules/). The docs explicitly frame AGENTS.md as "similar to Cursor's rules".

## MCP
- **Config file**: the `mcp` key inside `opencode.json` / `opencode.jsonc` (project or global) — there is no separate MCP file (https://opencode.ai/docs/mcp-servers/).
- **Server-table shape** (map of name → server object, keyed by arbitrary name; `enabled: false` disables without removal):
  - Local: `{ "type": "local", "command": ["npx", "-y", "pkg"], "cwd": "…", "environment": {"K": "V"}, "enabled": true, "timeout": 5000 }` — **`command` is a single array** (binary + args together), env key is named **`environment`** (not `env`), `timeout` in ms (default 5000).
  - Remote: `{ "type": "remote", "url": "https://…", "headers": {"Authorization": "Bearer …"}, "oauth": {…} | false, "enabled": true, "timeout": 5000 }` — OAuth auto-detected (RFC 7591 dynamic registration), tokens stored in `~/.local/share/opencode/mcp-auth.json`; `oauth` object supports `clientId`, `clientSecret`, `scope`.
- **Env handling**: `environment` map for local servers; anywhere in config, `{env:VAR_NAME}` substitutes environment variables and `{file:path}` substitutes file contents (works for headers, `oauth.clientSecret`, apiKey, etc.) (https://opencode.ai/docs/mcp-servers/, https://opencode.ai/docs/config/).
- MCP tools are gated like built-in tools via `tools` globs (`"my-mcp*": false`) or per-agent.

## Translation hazards
Concrete drops/renames/reformats a canonical `.agents` store must apply to target opencode:
1. **Hooks are untranslatable as data.** Claude/Codex-style hook config (matcher + shell command + stdin JSON envelope + exit-code/stdout contract) has **no opencode equivalent**. Canonical hooks must be dropped or rewritten by hand as `.opencode/plugins/*.js|ts` modules. Closest event mapping: `PreToolUse` → `tool.execute.before`, `PostToolUse` → `tool.execute.after`; there is no matcher field (match on `input.tool` in code), no stdin envelope (blocking = `throw new Error(...)`), and no per-tool regex matching of args beyond manual inspection of `output.args`. Session/lifecycle events use dotted names (`session.idle`, `session.created`, …), not PascalCase.
2. **Skill frontmatter keys**: keep only `name`, `description`, `license`, `compatibility`, `metadata`. Drop `allowed-tools` (ignored by opencode; re-express as `permission.skill` globs in `opencode.json` — note this moves the data out of the skill file into config). Unknown keys are ignored (not errors), so loss is silent.
3. **Skill naming rules**: `name` must be 1–64 chars, match `^[a-z0-9]+(-[a-z0-9]+)*$`, and **equal the directory name** — canonical names with uppercase/underscores must be slugified and the directory renamed to match. `description` capped at 1024 chars — truncate longer ones. Layout is strictly `<name>/SKILL.md` (not `<name>.md`).
4. **Skills dirs**: opencode natively reads `.agents/skills/<name>/SKILL.md` (project) and `~/.agents/skills/<name>/SKILL.md` (global) as compatibility locations, so a canonical store can be used in place; the native locations are `.opencode/skills/` and `~/.config/opencode/skills/`.
5. **MCP shape**: JSON only (no TOML). `command` must be one array `[bin, ...args]` — split any canonical `command` string + `args` array into it. Rename env table to `environment` (not `env`). Remote servers need explicit `"type": "remote"` + `url` (+ `headers`); there is no `transport` field (stdio vs HTTP implied by `local`/`remote`). Secrets become literal values or `{env:VAR}` placeholders — opencode never resolves a separate env-reference indirection.
6. **Instructions**: canonical content targets `AGENTS.md` (project root or `~/.config/opencode/AGENTS.md`). No heading convention to rewrite, but **no include/import syntax exists** — canonical `@import`-style directives must be flattened or moved into the `instructions` array of `opencode.json` (which accepts globs and https URLs). Extra canonical instruction files can't just sit in a directory; they must be enumerated in `instructions`.
7. **Subagent files**: dir is `.opencode/agents/` (**plural**), filename becomes the agent name (must be a valid mentionable identifier — use the same slug rules as skills in practice). Required frontmatter: `description`; canonical `tools: [...]` allow-lists must be converted to `permission` maps (`allow|ask|deny`, glob-capable) since `tools` is deprecated; mode uses `mode: subagent|primary|all`. Claude-style `allowed-tools` comma strings have no direct slot.
8. **Commands**: dir is `.opencode/commands/`, `.md` files with frontmatter `description|agent|model|subtask`; template variables `$ARGUMENTS`/`$1..$N` are compatible with Claude's, but canonical `allowed-tools`/other frontmatter keys are not documented and should be dropped; `` !`cmd` `` shell injection and `@file` refs are opencode-native.
9. **Config file naming**: project config is `opencode.json`/`opencode.jsonc` at root plus a `.opencode/` directory — do not emit `opencode.yaml`/TOML. Global config lives under `~/.config/opencode/` (XDG), not `~/.opencode`. Optional `"$schema": "https://opencode.ai/config.json"` header.
10. **No rules directory**: anything modeled as a canonical `rules/` dir must be folded into AGENTS.md content or the `instructions` array.
