# kimi

## Identity

- **Binary**: `kimi` (Python package `kimi-cli` on PyPI; Python 3.12–3.14, 3.13 recommended) — https://moonshotai.github.io/kimi-cli/en/guides/getting-started.md
- **Vendor**: Moonshot AI (月之暗面). Repo: https://github.com/MoonshotAI/kimi-cli (11k stars, active commits as of 2026-08).
- **Install**: `curl -LsSf https://code.kimi.com/install.sh | bash` (installs uv, then the package) or `uv tool install --python 3.13 kimi-cli`; Windows PowerShell script also available — https://moonshotai.github.io/kimi-cli/en/guides/getting-started.md
- **Docs base URL**: https://moonshotai.github.io/kimi-cli/en/ (VitePress; full page index at https://moonshotai.github.io/kimi-cli/llms.txt)
- **Maintenance status**: Actively maintained, BUT officially winding down: "Kimi CLI is evolving into Kimi Code CLI… This project will be gradually wound down; the docs and existing installations remain available." — https://github.com/MoonshotAI/kimi-cli README. Successor: https://github.com/MoonshotAI/kimi-code (TypeScript rewrite; auto-migrates config/sessions).
- URLs actually read: see sources_read (docs pages listed inline per claim below, plus repo source files via jsdelivr CDN).

## Config dirs

- **Global/user dir**: `~/.kimi/` — holds `config.toml`, `kimi.json` (metadata), `mcp.json`, `credentials/`, `mcp-oauth/`, `sessions/`, `plans/`, `user-history/`, `logs/`, `plugins/` (plugins dir confirmed in source: `get_plugins_dir() -> get_share_dir()/"plugins"` — https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/src/kimi_cli/plugin/manager.py). The entire share dir is relocatable via `KIMI_SHARE_DIR` env var — https://moonshotai.github.io/kimi-cli/en/configuration/data-locations.md and https://moonshotai.github.io/kimi-cli/en/configuration/env-vars.md
- **Project dir**: `.kimi/` exists at project level but ONLY for: `.kimi/skills/`, `.kimi/AGENTS.md`, and hook scripts by convention (`.kimi/hooks/*.sh` in doc examples). There is **no project-level config file** — the only config file is `~/.kimi/config.toml` (or `--config-file <path>`) — https://moonshotai.github.io/kimi-cli/en/configuration/config-files.md
- **Config format**: TOML (default) or JSON (auto-migrated to TOML with `.bak` backup) — config-files.md
- **Precedence**: env vars > CLI flags > config file — https://moonshotai.github.io/kimi-cli/en/configuration/overrides.md
- **Project root definition** (used for skills + AGENTS.md discovery): nearest `.git` ancestor of the work dir, falling back to work dir itself — https://moonshotai.github.io/kimi-cli/en/customization/skills.md

## Instructions file

- **Filename**: `AGENTS.md` (also lowercase `agents.md`; the two are mutually exclusive per directory, uppercase wins). Additionally `.kimi/AGENTS.md` per directory, loaded alongside and BEFORE the root-level file — source docstring in `soul/agent.py` (https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/src/kimi_cli/soul/agent.py) and tests (https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/tests/core/test_load_agents_md.py)
- **Discovery**: hierarchical merge from project root (nearest `.git` ancestor) down to the work dir; per directory the order is `.kimi/AGENTS.md` then `AGENTS.md`/`agents.md`; files concatenated root→leaf with `<!-- From: ... -->` annotations; total capped at **32 KiB** (`_AGENTS_MD_MAX_BYTES = 32 * 1024`), budget allocated leaf-first (deeper files never truncated for shallower ones). Empty files skipped. Without `.git`, only the work dir is searched.
- **Generation**: `/init` slash command analyzes the project and generates `AGENTS.md` — https://moonshotai.github.io/kimi-cli/en/reference/slash-commands.md
- **Heading/title conventions**: none documented.
- **Import/include syntax**: none for `AGENTS.md`. (Jinja2 `{% include %}` and `${VAR}` substitution exist only in custom-agent system-prompt templates — https://moonshotai.github.io/kimi-cli/en/customization/agents.md)
- No `CLAUDE.md` support. Injected into the default agent's system prompt as `${KIMI_AGENTS_MD}` — agents.md.

## Skills

**Supported.** Cross-tool compatible (agentskills.io spec) — https://moonshotai.github.io/kimi-cli/en/customization/skills.md

- **Layout**: `<name>/SKILL.md` (canonical) OR flat `<name>.md` directly in a skills dir (name defaults to filename stem; subdirectory wins on tie). Auxiliary `scripts/`, `references/`, `assets/` dirs allowed. Recommendation: keep SKILL.md under 500 lines.
- **Directories** (priority: Project > User > Extra > Built-in; within user/project, brand group beats generic group on same-name clashes):
  - Project (relative to project root): brand `.kimi/skills/` > `.claude/skills/` > `.codex/skills/`; generic `.agents/skills/`
  - User: brand `~/.kimi/skills/` > `~/.claude/skills/` > `~/.codex/skills/`; generic `~/.config/agents/skills/` (recommended) > `~/.agents/skills/`
  - `merge_all_available_skills = true` (default) merges ALL existing brand dirs (kimi > claude > codex); `false` = first-existing-only — skills.md + confirmed in source https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/src/kimi_cli/skill/__init__.py
  - `--skills-dir <path>` (repeatable) REPLACES user/project auto-discovery (scoped as `extra`, top priority); `extra_skill_dirs` config array is ADDITIVE (absolute, `~/`, or project-root-relative) — skills.md + source
  - Plugin install dir `~/.kimi/plugins/` is also scanned for skills as `extra` scope (source, undocumented in docs)
  - Skill paths are NOT affected by `KIMI_SHARE_DIR` — env-vars.md
- **EXACT frontmatter keys** (YAML):
  | key | meaning | limit |
  |---|---|---|
  | `name` | skill name; lowercase letters, numbers, hyphens ONLY; 1–64 chars; defaults to directory name | No (optional) |
  | `description` | purpose/use cases; 1–1024 chars; fallback chain: first non-empty body line (truncated at 240 chars) → `"No description provided."` | No |
  | `license` | license name or file reference | No |
  | `compatibility` | environment requirements, ≤500 chars | No |
  | `metadata` | additional key-value attributes | No |
  | `type: flow` | marks a **flow skill** (multi-step Mermaid/D2 diagram workflow, one `BEGIN` + one `END` node required; decision branches chosen via `<choice>name</choice>` output) | No |
- **Kimi-unique keys**: `type` (flow) is Kimi-specific; the rest follow the cross-tool agentskills.io spec. Notably there is **no `allowed-tools` key** (Claude-specific keys are ignored).
- **Invocation**: `/skill:<name> [extra text]` loads SKILL.md as a prompt; `/flow:<name>` executes a flow skill — slash-commands.md.

## Hooks

**Supported (Beta)** — https://moonshotai.github.io/kimi-cli/en/customization/hooks.md (cross-checked against the `[[hooks]]` schema in https://moonshotai.github.io/kimi-cli/en/configuration/config-files.md — consistent).

- **Config file + format**: `~/.kimi/config.toml`, `[[hooks]]` TOML array. No separate hooks file.
- **Entry schema**: `event` (required string), `command` (required shell command string; receives JSON on stdin), `matcher` (optional **regex string**, empty = match all), `timeout` (optional integer **seconds**, default 30). Hooks for the same event run in parallel; identical commands deduplicated; ALL failures (timeout/crash/regex error) are fail-open (allow).
- **13 events** (matcher filter in parens): `PreToolUse` (tool name), `PostToolUse` (tool name), `PostToolUseFailure` (tool name), `UserPromptSubmit` (none), `Stop` (none), `StopFailure` (error type), `SessionStart` (`startup`/`resume`), `SessionEnd` (reason), `SubagentStart` (agent name), `SubagentStop` (agent name), `PreCompact` (trigger), `PostCompact` (trigger), `Notification` (sink name).
- **stdin JSON envelope**: common fields `session_id`, `cwd`, `hook_event_name`, plus per-event fields: `tool_name`, `tool_input`, `tool_call_id` (PreToolUse); `tool_output` (PostToolUse); `error` (PostToolUseFailure); `prompt` (UserPromptSubmit, SubagentStart); `stop_hook_active` (Stop; true on the single allowed re-trigger — anti-loop); `error_type`, `error_message` (StopFailure); `source` (SessionStart); `reason` (SessionEnd); `agent_name` (Subagent*); `response` (SubagentStop); `trigger`, `token_count` (PreCompact); `estimated_token_count` (PostCompact); `sink`, `notification_type`, `title`, `body`, `severity` (Notification).
- **Output contract**: exit `0` = allow, non-empty stdout is added to context; exit `2` = block, stderr fed back to the LLM as correction; any other code = allow, stderr only logged. Structured JSON on stdout with exit 0: `{"hookSpecificOutput": {"hookEventName": ..., "permissionDecision": "deny", "permissionDecisionReason": "..."}}` — `deny` blocks and the reason goes to the LLM.
- Hook scripts conventionally live in `.kimi/hooks/` (doc examples); `/hooks` lists configured hooks.

## Subagents

**Supported, but NOT folder-discovered.** No `.kimi/agents/` directory convention — https://moonshotai.github.io/kimi-cli/en/customization/agents.md

- **Format**: YAML agent spec files loaded explicitly via `--agent-file /path/to/agent.yaml` (or built-ins `default`, `okabe` via `--agent`).
- **Schema**: `version: 1` top-level; `agent:` map with `extend` (`default` or relative path to another agent file), `name`, `system_prompt_path` (relative to the agent file), `system_prompt_args` (custom `${VAR}` values), `tools` (list of Python `module:ClassName` paths, e.g. `kimi_cli.tools.shell:Shell`), `exclude_tools`, `subagents`. Source-only (undocumented in docs) fields: `model`, `when_to_use`, `allowed_tools` — https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/src/kimi_cli/agentspec.py
- **Subagent definitions**: inline in the agent YAML: `subagents: { coder: { path: ./coder-sub.yaml, description: "..." } }`; each subagent file is itself a full agent YAML (usually `extend`-ing the main agent). Subagents cannot nest (no `Agent` tool inside subagents).
- **Built-in subagent types**: `coder` (read/write/shell), `explore` (read-only), `plan` (read-only, no Shell) — launched via the `Agent` tool with `subagent_type`.
- Runtime instance state (context.jsonl, wire.jsonl, meta.json, prompt.txt, output) lives under the session dir `subagents/<agent_id>/` — data-locations.md.

## Commands

- **No custom slash-command directory.** All slash commands are built-in (`/help`, `/login`, `/model`, `/init`, `/plan`, `/sessions`, `/compact`, `/mcp`, `/hooks`, `/skill:<name>`, `/flow:<name>`, etc.) — https://moonshotai.github.io/kimi-cli/en/reference/slash-commands.md
- The user-extensible equivalent of commands is **skills**: `/skill:<name>` invokes any skill with optional trailing text as the task — skills.md + slash-commands.md.

## Rules

- **Not supported** as a distinct mechanism. No rules directory; `AGENTS.md` (see Instructions file) is the only project-rules channel. Evidence: no rules concept anywhere in the doc index https://moonshotai.github.io/kimi-cli/llms.txt; customization docs cover only skills, hooks, agents, MCP, plugins, print/wire modes.

## MCP

- **Config file**: `~/.kimi/mcp.json`, JSON, standard `{"mcpServers": { ... }}` shape ("format compatible with other MCP clients") — https://moonshotai.github.io/kimi-cli/en/customization/mcp.md and data-locations.md
- **Server shapes**:
  - HTTP: `{ "url": "https://...", "transport": "http", "headers": { "KEY": "value" } }`; OAuth via `kimi mcp add --auth oauth` + `kimi mcp auth <name>` (tokens cached in `~/.kimi/mcp-oauth/`, outside the config file).
  - stdio: `{ "command": "npx", "args": [...], "env": { "SOME_VAR": "value" } }`.
- **Management**: `kimi mcp add [--transport stdio|http] [-e KEY=VALUE] [-H "KEY:VALUE"] [--auth oauth] NAME TARGET...`, `list`, `remove`, `auth`, `reset-auth`, `test` — https://moonshotai.github.io/kimi-cli/en/reference/kimi-mcp.md
- **Ad-hoc**: `--mcp-config-file /path/to/mcp.json` or `--mcp-config '<json>'` CLI flags — mcp.md / README.
- **Env handling**: stdio `env` is a plain JSON map (values stored verbatim); HTTP secrets go in `headers`. Client behavior knob `[mcp.client] tool_call_timeout_ms` (default 60000) lives in `config.toml` — config-files.md.

## Translation hazards

Concrete drops/renames/reformats a canonical `.agents` store must apply when targeting Kimi CLI:

1. **Config format**: main config is TOML (`~/.kimi/config.toml`), not JSON/YAML; JSON is accepted but auto-migrated to TOML. There is NO project-level config file — anything project-scoped must go to `~/.kimi/config.toml` (global) or be dropped.
2. **Hooks live inside config.toml** as `[[hooks]]` TOML array-of-tables — a canonical JSON/YAML hook store must be serialized into TOML and merged into the user's global config (risky shared file).
3. **Hook matcher is a bare regex string** on the entry (`matcher = "WriteFile|StrReplaceFile"`), NOT Claude's `{matcher, hooks: [{type, command}]}` nested structure. Flatten: one `[[hooks]]` entry per (event, matcher-regex, command); `type: "command"` and `timeout` object nesting must be dropped/flattened (`timeout` = plain seconds integer, default 30).
4. **Event-name mapping**: Claude's `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `Stop`, `SessionStart`, `SessionEnd`, `SubagentStop`, `PreCompact`, `Notification` carry over verbatim, but Kimi adds `PostToolUseFailure`, `StopFailure`, `SubagentStart`, `PostCompact` (no Claude equivalents) and LACKS Claude-specific nuances; Codex's hook events don't map 1:1. Unknown events are invalid.
5. **Matcher targets Kimi tool names**, not Claude's: `Shell` (not `Bash`), `WriteFile` (not `Write`), `StrReplaceFile` (not `Edit`), `ReadFile`, `ReadMediaFile`, `Glob`, `Grep`, `SearchWeb`, `FetchURL`, `SetTodoList`, `AskUserQuestion`, `Agent`, `TaskList/TaskOutput/TaskStop`, `EnterPlanMode/ExitPlanMode` — https://moonshotai.github.io/kimi-cli/en/customization/agents.md. Canonical matchers referencing `Bash|Edit|Write` MUST be rewritten.
6. **Hook envelope**: fields `session_id`, `cwd`, `hook_event_name` match Claude's shape (good for agentlink's `guard`/`remind`), but `tool_input` contents follow Kimi tool schemas (`Shell.command`, `WriteFile.path`/`content`, `StrReplaceFile.edit.old/new`) — file-path extraction must read `tool_input.file_path` only for `WriteFile`/`StrReplaceFile` and `tool_input.path` for `ReadFile` [INFERENCE from doc examples: hooks.md uses `.tool_input.file_path`; agents.md shows `path` params]. Non-tool events use entirely different fields (`prompt`, `trigger`, `stop_hook_active`, ...).
7. **Hook output contract**: exit 2 blocks (same as Claude), but structured-output passthrough is limited to `hookSpecificOutput.{hookEventName, permissionDecision: "deny", permissionDecisionReason}` — other Claude output keys (`continue`, `suppressOutput`, `decision`, `systemMessage`, per-event `additionalContext`) are undocumented for Kimi and must be dropped.
8. **Fail-open semantics**: timeouts/crashes allow the operation — a canonical store relying on fail-closed guards must not assume blocking on error.
9. **Skills — frontmatter**: supported keys are exactly `name`, `description`, `license`, `compatibility`, `metadata`, `type`. DROP Claude-only keys (`allowed-tools`, etc.) — ignored but noise. Kimi-unique `type: flow` must be dropped when translating AWAY from Kimi. Enforce `name`: lowercase `[a-z0-9-]`, 1–64 chars (rename on conflict with canonical names containing uppercase/underscores). `description` ≤1024 chars (truncate). `compatibility` ≤500 chars.
10. **Skills — layout**: both `<name>/SKILL.md` and flat `<name>.md` work; flat files lose aux dirs. Kimi reads `.agents/skills/` (project) and `~/.config/agents/skills/` or `~/.agents/skills/` (user) natively — the canonical store path is directly consumable, BUT brand dirs (`.kimi/skills`, `.claude/skills`, `.codex/skills`) outrank the generic group on name clashes, so a canonical skill can be shadowed.
11. **Instructions file**: filename is `AGENTS.md` (same as Codex — no rename needed from a Codex-style canonical store). No heading convention exists, so agentlink's instructions normalizer heading regex (`(Claude|Codex)`) needs no Kimi variant unless a heading is injected. Remember the **32 KiB merged cap** and hierarchical `.kimi/AGENTS.md` layering — a canonical file >32 KiB will be truncated (leaf-first). No `@import`/include syntax — inlined content only.
12. **Subagents**: cannot be expressed as folder + Markdown frontmatter. Canonical `.agents/agents/*.md` must be compiled into: one YAML file per agent (`version: 1`, `agent:` map) + a separate system-prompt `.md` file referenced by `system_prompt_path`, and users must pass `--agent-file` explicitly (no auto-discovery). `tools` entries are Python import paths (`kimi_cli.tools.*:Class`) — not portable to/from other agents; subagent references are relative `path` + `description` pairs, not name-only.
13. **Commands**: no slash-command dir — canonical commands must be converted to skills (SKILL.md) and invoked as `/skill:<name>`, or dropped.
14. **Rules**: nothing to target — fold rules into `AGENTS.md`.
15. **MCP**: `mcpServers` JSON shape is directly compatible; keep `command`/`args`/`env` (stdio) and `url`/`headers` (http). Kimi adds `transport: "http"` on http servers (harmless extra key for other tools; canonical comparators should treat it as noise). Never translate OAuth tokens (live in `~/.kimi/mcp-oauth/`). `mcp.json` is GLOBAL only (`~/.kimi/`) — no project-scope MCP file.
16. **Env vars**: `KIMI_SHARE_DIR` relocates `~/.kimi` (config/sessions/mcp.json move) but NOT skill search paths — sync logic must resolve both independently.

## Verified corrections (fact-checker pass)

- was → Built-in subagent type `explore` is 'read-only' → now: `explore` is NOT strictly read-only — agents.md grants it `Shell` alongside ReadFile/ReadMediaFile/Glob/Grep/SearchWeb/FetchURL ('no write tools', but Shell itself can write); only `plan` is truly read-only with no Shell (https://moonshotai.github.io/kimi-cli/en/customization/agents.md)
- was → `tool_call_id` is a PreToolUse-only envelope field → now: `tool_call_id` is also sent on PostToolUse and PostToolUseFailure payloads in source (the hooks.md docs table omits it for those events) (https://cdn.jsdelivr.net/gh/MoonshotAI/kimi-cli@main/src/kimi_cli/hooks/events.py)
