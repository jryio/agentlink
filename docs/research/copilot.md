# copilot

## Identity — binary, vendor, install, docs URLs (list every URL you actually read)

- **Binary:** `copilot` (interactive TUI; `copilot -p "..."` for programmatic mode). Vendor: **GitHub**. Actively maintained; this is the 2025 agentic terminal agent, distinct from the retired `gh copilot` alias extension.
- **Install:** `npm install -g @github/copilot` (requires Node.js 22+), `winget install GitHub.Copilot`, `brew install --cask copilot-cli` — https://docs.github.com/en/copilot/how-tos/copilot-cli/cli-getting-started
- **Repo:** https://github.com/github/copilot-cli (public issue tracker; closed-source runtime)
- **Docs base:** https://docs.github.com/en/copilot
- **URLs actually read (all official docs.github.com unless noted):**
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/cli-getting-started
  - https://docs.github.com/en/copilot/concepts/agents/copilot-cli/about-copilot-cli
  - https://docs.github.com/en/copilot/concepts/agents/copilot-cli/comparing-cli-features
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions
  - https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/add-custom-instructions/add-repository-instructions
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills
  - https://docs.github.com/en/copilot/concepts/agents/about-agent-skills
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/use-hooks
  - https://docs.github.com/en/copilot/reference/hooks-reference
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/invoke-custom-agents
  - https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/create-custom-agents
  - https://docs.github.com/en/copilot/reference/custom-agents-configuration
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers
  - https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
  - https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
  - https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference
  - https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/plugins-creating
  - https://github.com/github/copilot-cli/issues/1004 (official repo issue; prompt-file feature request, closed as duplicate of #618, "not planned")

## Config dirs — project dir, global/user dir, discovery/precedence rules

- **Global/user dir:** `~/.copilot/` (macOS/Linux), `%USERPROFILE%\.copilot\` (Windows). `COPILOT_HOME` env var replaces the entire path. Contents include `agents/`, `skills/`, `hooks/`, `instructions/`, `copilot-instructions.md`, `mcp-config.json`, `settings.json` (user settings, JSONC), `config.json` (auto-managed app state — user settings were migrated out of it into `settings.json`), `permissions-config.json`, `lsp-config.json`, `extensions/`, `installed-plugins/`, `session-state/`, `logs/` — https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- **Project config:** there is **no single `.copilot/` project dir**. Project-level config is spread across:
  - `.github/agents/`, `.github/skills/`, `.github/hooks/*.json`, `.github/instructions/`, `.github/copilot-instructions.md`, `.github/copilot/settings.json` + `.github/copilot/settings.local.json`, `.github/mcp.json`
  - Repo root: `.mcp.json`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`
  - Cross-tool locations also read: `.claude/skills/`, `.claude/agents/`, `.claude/commands/`, `.claude/settings.json`, `.claude/settings.local.json`, `.claude/CLAUDE.md`, and `.agents/skills/` (project) / `~/.agents/skills/` (personal)
- **Discovery:** repo instruction/agent files are discovered in "standard locations": the repository root, the cwd, intermediate directories between them, and directories nested in the path of a file being worked on; the CLI walks cwd → git root loading `.github/agents/` (and `.claude/agents/`) at every ancestor level, deepest wins — https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions and https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference
- **Settings precedence (later overrides earlier):** built-in defaults → MDM managed settings → `~/.copilot/settings.json` → `.github/copilot/settings.json` → `.github/copilot/settings.local.json` → env vars → CLI flags — https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
- **⚠ Doc contradiction (agent precedence):** https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/invoke-custom-agents and https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli say a **user-level agent overrides a repository-level** one on name conflict; https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference says "**User-level agents have lower priority than project-level agents**." Both are current official pages; treat agent precedence as unreliable.

## Instructions file — filename(s), location, heading/title conventions, import/include syntax if any

All of these are loaded and **combined** (identical copies deduped; "does not define a general precedence order" in the CLI) — https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions:

| File | Scope |
|---|---|
| `AGENTS.md` | Agent instructions, any standard location (repo root, cwd, ancestors, nested dirs) |
| `CLAUDE.md` (and `.claude/CLAUDE.md`) | Same discovery |
| `GEMINI.md` | Same discovery |
| `.github/copilot-instructions.md` | Repository-wide |
| `~/.copilot/copilot-instructions.md` | User-level, all repos |
| `.github/instructions/**/*.instructions.md` | Path-specific (frontmatter `applyTo` glob; `excludeAgent` optional) |
| `~/.copilot/instructions/**/*.instructions.md` | User-level modular |
| dirs in `COPILOT_CUSTOM_INSTRUCTIONS_DIRS` (comma-separated) | Extra `AGENTS.md` + `*.instructions.md` |

- **No heading/title convention** — plain Markdown body; "whitespace between instructions is ignored" (https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/add-custom-instructions/add-repository-instructions). Note: the GitHub.com (non-CLI) docs say "the nearest `AGENTS.md` file in the directory tree will take precedence," which differs from the CLI's combine-all behavior — same page.
- **Import syntax:** `@<relative-path>` includes another file, recursively; supported in `.github/copilot-instructions.md`, `AGENTS.md`, `CLAUDE.md` only — **not** in `GEMINI.md` or `*.instructions.md`. Absolute paths and `~/` paths are not loaded; references must stay inside the repo (or the custom-instructions dir) — https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions
- `--no-custom-instructions` disables loading; `/instructions` lists/toggles files per session.

## Skills — supported? dir, file layout, EXACT frontmatter keys with meaning, which keys are unique to this tool, size/naming limits

**Supported.** Follows the open Agent Skills spec. Layout: `<skills-dir>/<skill-name>/SKILL.md` plus optional sibling scripts/resources (auto-discovered on invocation). File **must** be named `SKILL.md`; directory names should be lowercase with hyphens — https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills

**Locations, priority order (first found wins on duplicate names)** — https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference:
1. `.github/skills/` (project) 2. `.agents/skills/` (project) 3. `.claude/skills/` (project, Claude-compatible) 4. parent-dir `.github/skills/` (monorepo) 5. `~/.copilot/skills/` 6. `~/.agents/skills/` 7. plugin dirs 8. `COPILOT_SKILLS_DIRS` (comma-separated) + `skillDirectories` setting 9. built-in 10. remote org/enterprise skills. Plugin name collisions get plugin-qualified invocation (`/plugin/skill`).

**Frontmatter keys** (CLI command reference, "Skill frontmatter fields"):
- `name` (required) — unique ID; **letters, numbers, hyphens only; max 64 chars**
- `description` (required) — what/when; **max 1024 chars**
- `argument-hint` (optional) — freeform arg hint shown in picker
- `allowed-tools` (optional) — comma-separated string or YAML array of pre-approved tools; `"*"` = all
- `user-invocable` (optional, default `true`) — whether `/SKILL-NAME` works
- `disable-model-invocation` (optional, default `false`) — blocks auto-invocation by the agent
- `license` (optional) — listed only in the how-to page (https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-skills), absent from the CLI reference table

**Keys unique/notable vs other tools:** `argument-hint`, `user-invocable`, `disable-model-invocation` (shared with Copilot commands/agents but absent from Claude/Codex skill specs in this form); `license`. No `metadata` key documented. Invocation via `/skill-name` in a prompt or automatic by description; `/skills list|info|reload|add|remove` and `copilot skill ...` manage them.

## Hooks — supported? config file + format, event names, matcher schema, command schema, stdin JSON envelope fields, output contract

**Supported.** Full reference: https://docs.github.com/en/copilot/reference/hooks-reference; how-to: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/use-hooks

**Config files (all combined; all entries from all sources run):**
- Repo: `.github/hooks/*.json`
- User: `~/.copilot/hooks/*.json` (or `$COPILOT_HOME/hooks/`)
- Inline `hooks` key in `.github/copilot/settings.json`, `.github/copilot/settings.local.json`, `~/.copilot/settings.json`, and cross-tool `.claude/settings.json` / `.claude/settings.local.json`
- Policy (machine-wide, can't be disabled): `/etc/github-copilot/policy.d/*.json` (POSIX, root-owned), `%ProgramData%\GitHub\Copilot\policy.d\*.json`, Windows Registry
- Plugin-contributed `hooks.json`

**Format:** JSON, `{ "version": 1, "disableAllHooks": false, "hooks": { "<event>": [ <entry>, ... ] } }`. **Flat array per event** — no nested matcher-group level like Claude Code.

**Entry schemas:**
- Command: `{ "type": "command", "bash": "...", "powershell": "...", "command": "...", "cwd": "...", "env": {...}, "timeoutSec": 30, "matcher": "..." }` — one of `bash`/`powershell`/`command` required; `command` is cross-platform fallback copied to both; `timeout` aliases `timeoutSec`; default timeout 30 s.
- HTTP: `{ "type": "http", "url": "https://...", "headers": {...}, "allowedEnvVars": [...], "timeoutSec": 30 }` — POSTs the input payload; https-only except localhost opt-in.
- Prompt: `{ "type": "prompt", "prompt": "..." }` — `sessionStart` only, interactive new sessions only.

**Event names** — two casings select two payload formats: camelCase = native (camelCase fields), PascalCase = "VS Code compatible" (snake_case fields + `hook_event_name`, ISO 8601 timestamps): `sessionStart`/`SessionStart`, `sessionEnd`/`SessionEnd`, `userPromptSubmitted`/`UserPromptSubmit`, `userPromptTransformed`, `preToolUse`/`PreToolUse`, `postToolUse`/`PostToolUse`, `postToolUseFailure`/`PostToolUseFailure`, `agentStop`/`Stop`, `subagentStart`, `subagentStop`/`SubagentStop`, `errorOccurred`/`ErrorOccurred`, `preCompact`/`PreCompact`, `notification`/`Notification`, `permissionRequest`/`PermissionRequest`.

**Matcher schema:** optional `matcher` on an entry; anchored regex `^(?:PATTERN)$` matched against `toolName` (preToolUse/postToolUse/permissionRequest), `trigger` (preCompact), `agentName` (subagentStart), `notification_type` (notification). Invalid regex = entry skipped. **PascalCase `PreToolUse`/`PermissionRequest` switch to Claude matcher semantics** (`*`, `|`-alternation, literal names) and payloads report **Claude tool names** (`Bash`, `Read`, `Edit`, …) via a documented mapping table (runtime `bash`→`Bash`, `view`→`Read`, `edit`→`Edit`, `grep`→`Grep`, `task`→`Agent`, etc.).

**stdin JSON envelope (camelCase):** common `sessionId`, `timestamp` (Unix **ms number**), `cwd`; per-event additions: `toolName`, `toolArgs` (unknown; in examples a **JSON-encoded string**, e.g. `"toolArgs":"{\"command\":\"ls\"}"`), `toolResult: { resultType: "success", textResultForLlm }`, `error` / `error: {message,name,stack}` + `errorContext` + `recoverable`, `prompt`, `transformedPrompt`, `transcriptPath`, `stopReason`, `stop_hook_active`, `agentName`/`agentId`/`agentType`/`response`, `trigger`, `customInstructions`, `reason`, `source`, `initialPrompt`, `message`/`title`/`notification_type`.

**Output contract:** hook writes **one** JSON object to stdout (after stripping any line-oriented `{"type":"progress",...}` messages); unparsable/empty = no output. Cap 10 MiB.
- `preToolUse`: `{ permissionDecision: "allow"|"deny"|"ask", permissionDecisionReason, modifiedArgs }`
- `permissionRequest`: `{ behavior: "allow"|"deny", message, interrupt }`
- `postToolUse`: `{ modifiedResult: {resultType:"success", textResultForLlm}, additionalContext }`
- `agentStop`/`subagentStop`: `{ decision: "block"|"allow", reason, modifiedResponse (subagentStop only) }`
- `sessionStart`/`notification`: `{ additionalContext }`; `userPromptSubmitted`: `modifiedPrompt` (SDK-only); `userPromptTransformed`: `modifiedTransformedPrompt`
- **Exit codes:** `0` = parse stdout; `2` = warning generally, but **deny** for `preToolUse`/`permissionRequest` (stderr ignored for permissionRequest, stdout merged into deny); other non-zero = fail-open logged, **except `preToolUse` command hooks which are fail-closed**; timeouts are always fail-open, even for `preToolUse` and policy hooks.

## Subagents — dir + file format + frontmatter

Supported as **custom agents** (Markdown "agent profiles"); the CLI also has built-in agents (`explore`, `task`, `general-purpose`, `code-review`, `research`, `rubber-duck`, `security-review`) — https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference

- **Dirs:** project `.github/agents/` or `.claude/agents/` (walked at every ancestor from cwd to git root; `.github/agents/` beats `.claude/agents/` at same level); user `~/.copilot/agents/`; plugin `<plugin>/agents/`; org/enterprise `/agents/` in the org's `.github`/`.github-private` repo — https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/invoke-custom-agents
- **File format:** `NAME.agent.md` or `NAME.md`; filename (minus extension) is the agent ID; filename charset limited to `. - _ a-z A-Z 0-9` (https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/create-custom-agents). Markdown body = the agent prompt, **max 30,000 chars**.
- **Frontmatter** (https://docs.github.com/en/copilot/reference/custom-agents-configuration + CLI reference): `description` (**required**), `name` (optional display name; defaults to filename), `tools` (list or comma string; omit = all tools; `[]` = none; supports `server/tool` and `server/*` namespacing and aliases `read`/`edit`/`search`/`execute`/`agent`/`web`/`todo`), `mcp-servers` (agent-scoped MCP map, YAML version of the MCP JSON shape; `stdio` mapped to `local`), `model`, `infer` (auto-delegation, default true — marked **retired** in the configuration reference but still documented in the CLI reference; replaced by `disable-model-invocation`), `disable-model-invocation`, `user-invocable` (default true), `target` (`vscode`|`github-copilot`), `metadata` (string map). `argument-hint`/`handoffs` from VS Code are ignored.

## Commands — slash-command dir + format

- **Supported only in Claude-compatible form:** individual `.md` files in **`.claude/commands/`** (documented under "Commands (alternative skill format)" in https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference). Name derived from filename; no `name` field required; frontmatter supports **`argument-hint`, `description`, `allowed-tools`, `disable-model-invocation`**. Commands have **lower priority than a skill with the same name**. Plugins can also contribute command directories via the `commands` field in `plugin.json` (https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference).
- **Not supported:** `.github/prompts/*.prompt.md` (VS Code "prompt files") are **not read by the CLI** — official repo feature request closed as duplicate/"not planned" (https://github.com/github/copilot-cli/issues/1004 → #618). Skills invoked as `/skill-name` are the intended substitute. No user-level commands dir is documented.

## Rules — rules dir + format (if distinct from instructions)

No `.cursor/rules`-style dir. The closest analog is **path-specific instruction files**: `.github/instructions/**/*.instructions.md` (repo) and `~/.copilot/instructions/**/*.instructions.md` (user) — Markdown with frontmatter `applyTo` (required glob, comma-separated list allowed, e.g. `"**/*.ts,**/*.tsx"`) and optional `excludeAgent: "code-review" | "cloud-agent"` — https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-custom-instructions. `@` file references are **not** expanded in these files. `/instructions` toggles them per session.

## MCP — config file, format, server-table shape, env handling

https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers

- **User config:** `~/.copilot/mcp-config.json` (`$COPILOT_HOME/mcp-config.json`). **Project config:** `.mcp.json` (discovered walking cwd → repo root) and `.github/mcp.json`; same-dir `.mcp.json` wins; name conflicts resolve toward the file closer to cwd; **project beats user**. Project files load only in trusted directories (skipped silently otherwise; `GITHUB_COPILOT_PROMPT_MODE_WORKSPACE_MCP=true` forces load in `-p` mode).
- **Format:** JSON. Top-level either `{"mcpServers": { "<name>": {...} }}` or (project files only) a **bare** `{ "<name>": {...} }` map. `.vscode/mcp.json` is **not read** (its `servers` key is unsupported).
- **Server shape:** `{ "type": "local"|"stdio"|"http"|"sse", "command": "...", "args": [...], "env": {...}, "tools": ["*"], "url": "...", "headers": {...} }`. `local` ≡ `stdio` (stdio recommended for cross-client compat; cloud agent maps `stdio`→`local`). `tools` filters exposed tools (`*` default). `sse` = deprecated legacy transport. The **GitHub MCP server is built in** with no config.
- **Env handling:** `PATH` is auto-inherited; **every other env var must be declared in `env`**. Remote servers use `headers` for secrets. Agent-profile `mcp-servers` (cloud) additionally support `${VAR}`, `${VAR:-default}`, `${{ secrets.X }}`, `${{ vars.X }}` expansion (https://docs.github.com/en/copilot/reference/custom-agents-configuration). CLI stores MCP secrets in OS keychain with `~/.copilot/mcp-secrets/` fallback.
- Management: `/mcp add|show|edit|delete|disable|enable|search` (search is experimental), `copilot mcp add|list|get|remove`.

## Translation hazards — every concrete thing a canonical `.agents` store must drop/rename/reformat to target this agent

**Good news first:** Copilot CLI natively reads `.agents/skills/` (project) and `~/.agents/skills/` (personal), plus `AGENTS.md` anywhere — a canonical `.agents` tree needs no symlink for those two.

1. **No project `.copilot/` dir exists.** Sibling sync must target `.github/<thing>` (agents, skills, hooks, instructions, `copilot-instructions.md`, `copilot/settings.json`, `mcp.json`), root `.mcp.json`, and root instruction files — not one dotted dir.
2. **Instructions heading normalization:** Copilot has **no title/heading convention** and **combines all instruction files with no defined precedence** — unlike agentlink's current regex that treats `# Claude Code instructions` ≈ `# Codex instructions`. Any heading is inert content for Copilot; a canonical normalizer should strip/normalize agent-named H1s for this target too.
3. **`@path` imports:** only valid in `AGENTS.md`/`CLAUDE.md`/`.github/copilot-instructions.md`; must be relative, in-repo, no `~/` or absolute. Emitting imports into `GEMINI.md` or `*.instructions.md` silently does nothing.
4. **Skill frontmatter — drop/rename:** canonical keys not in Copilot's list (`metadata`, Claude-style `model`, etc.) are undocumented and should be dropped. Keep only `name`, `description`, `license`, `allowed-tools`, `argument-hint`, `user-invocable`, `disable-model-invocation`. **Limits:** `name` = letters/numbers/hyphens, ≤64 chars; `description` ≤1024 chars; dir lowercase-hyphen; file **must** be `SKILL.md` (not `<name>.md`). `allowed-tools` accepts string or array and `"*"`.
5. **Hooks — format is incompatible with Claude Code's nesting:** Copilot wants a **flat array per event** with `matcher` on each entry (`{"version":1,"hooks":{"preToolUse":[{"type":"command","bash":"...","matcher":"bash|edit"}]}}`), not Claude's `[{"matcher": "...", "hooks": [{"type":"command","command":"..."}]}]` groups. Command field is **`bash`/`powershell`/`command`**, not Claude's single `command` (though `command` exists it means "cross-platform fallback"). Timeout key is `timeoutSec` (seconds number), not Claude's ms `timeout` (Copilot's `timeout` alias is still seconds).
6. **Hooks — event names:** canonical store must either emit camelCase (`preToolUse`, `sessionStart`, `userPromptSubmitted`, `agentStop`, `subagentStop`, …) or PascalCase (`PreToolUse`, `SessionStart`, `UserPromptSubmit`, `Stop`, `SubagentStop`, …). **Renames vs Claude Code:** `Stop`→`agentStop`, `UserPromptSubmit`→`userPromptSubmitted`, `SubagentStop`→`subagentStop`; Copilot-only events (`permissionRequest`, `postToolUseFailure`, `userPromptTransformed`, `errorOccurred`, `notification`) have no canonical equivalent; Claude's `SessionStart` source values and `PreCompact` mostly map. PascalCase selects the **VS Code/Claude-style snake_case payload** (`hook_event_name`, `session_id`, `tool_name`, `tool_input`, ISO timestamp) and Claude matcher semantics + Claude tool names (`Bash`/`Read`/`Edit`) — the lowest-friction target for a canonical Claude-shaped store.
7. **Hooks — stdin envelope (camelCase native):** `sessionId`/`timestamp` (ms **number**)/`cwd`/`toolName`/`toolArgs` — note `toolArgs` is delivered as a **JSON string** in examples, not an object; `toolResult.textResultForLlm` instead of Claude's `tool_response`. `agentlink guard`/`remind` envelope handling needs a Copilot variant.
8. **Hooks — output contract:** decision keys differ from Claude Code: `permissionDecision: allow|deny|ask` + `permissionDecisionReason` + `modifiedArgs` (vs Claude's `hookSpecificOutput.permissionDecision`); `permissionRequest` uses `behavior`/`message`/`interrupt`; blocking continuation uses `decision: "block"` + `reason`. Exit `2` semantics differ per event (deny only for `preToolUse`/`permissionRequest`). `preToolUse` command hooks are fail-closed on error but fail-open on timeout — normalizers must not assume Claude's exit-code table.
9. **Hook file placement:** repo = `.github/hooks/*.json` (any filename, must contain `"version": 1`); user = `~/.copilot/hooks/*.json`; inline `hooks` key also legal in `settings.json` files (including `.claude/settings.json`, which Copilot reads for a cross-tool subset: `hooks`, `disableAllHooks`, `enabledPlugins`, `extraKnownMarketplaces`, `companyAnnouncements`).
10. **MCP — rename/reformat:** top-level key is **`mcpServers`** (or bare map in project files); **`.vscode/mcp.json`'s `servers` key is unread**. Server `type` is `local|stdio|http|sse` (Codex/others may use different enums); extra keys `tools` (array filter, default `["*"]`) and `headers` must be preserved or intentionally dropped. All env vars except `PATH` must be explicit in `env` — no ambient inheritance, so a canonical entry relying on inherited env breaks. File targets: root `.mcp.json` or `.github/mcp.json` (project), `~/.copilot/mcp-config.json` (user) — **not** TOML, not `.codex/config.toml`.
11. **Custom agents — rename/reformat:** files must be `NAME.agent.md` (or `.md`) in `.github/agents/` (or `.claude/agents/`); filename charset only `. - _ a-z A-Z 0-9`; body ≤30,000 chars. Required frontmatter is **`description`** only; canonical keys to translate: `tools` (aliases `read/edit/search/execute/...`, case-insensitive; `server/tool` namespacing), `mcp-servers` (YAML, `stdio`→`local`), `model`, `infer`/`disable-model-invocation`, `user-invocable`, `target`, `metadata`. Drop VS Code-only `argument-hint`/`handoffs` (ignored) and any Claude-only keys. Precedence between user and project agents is **contradicted across official docs** — don't rely on either direction.
12. **Commands:** if the canonical store has slash commands, emit single `.md` files to `.claude/commands/` with frontmatter limited to `description`, `argument-hint`, `allowed-tools`, `disable-model-invocation` (no `name` key — name comes from filename). They lose to same-named skills. Do **not** emit `.prompt.md` files — unread by the CLI.
13. **Rules:** path-scoped rules must become `NAME.instructions.md` under `.github/instructions/` with `applyTo` glob frontmatter; Cursor-style `description`/`alwaysApply`/`globs` keys are undocumented — reduce to `applyTo` (+optional `excludeAgent`).
14. **Trust gate:** project-level config (MCP, and effectively anything the agent acts on) is inert until the user confirms folder trust; synced canonical config will not activate in `-p`/CI without pre-trusted dirs or `GITHUB_COPILOT_PROMPT_MODE_WORKSPACE_MCP=true`.
15. **`COPILOT_HOME`:** global dir is relocatable; a sync tool must resolve `$COPILOT_HOME` before defaulting to `~/.copilot` (legacy `--config-dir` and old XDG locations are migrated/deprecated).