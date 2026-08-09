# cursor

## Identity

- **Binary**: `agent` (all current docs: `agent`, `agent -p`, `agent update`, `agent --version`; https://cursor.com/docs/cli/overview.md, https://cursor.com/docs/cli/installation.md, https://cursor.com/docs/cli/changelog.md). Older releases/docs called the binary `cursor-agent` (earlier docs.cursor.com/en/cli snapshots); current docs use `agent` exclusively.
- **Vendor**: Anysphere (Cursor). Actively maintained — CLI changelog shows releases through July 2026 (https://cursor.com/docs/cli/changelog.md).
- **Install**: `curl https://cursor.com/install -fsS | bash` (macOS/Linux/WSL); `irm 'https://cursor.com/install?win32=true' | iex` (Windows). Installs to `~/.local/bin`; `~/.cursor/bin` used in CI. Auto-updates; `agent update` to force. (https://cursor.com/docs/cli/installation.md)
- **Auth**: `CURSOR_API_KEY` env var for headless use (https://cursor.com/docs/cli/headless.md).
- **Docs URLs actually read**:
  - https://cursor.com/llms.txt (sitemap; note: docs.cursor.com redirects to cursor.com/docs, every page has a `.md` version)
  - https://cursor.com/docs/cli/overview.md
  - https://cursor.com/docs/cli/installation.md
  - https://cursor.com/docs/cli/using.md
  - https://cursor.com/docs/cli/headless.md
  - https://cursor.com/docs/cli/reference/configuration.md
  - https://cursor.com/docs/cli/reference/slash-commands.md
  - https://cursor.com/docs/cli/changelog.md (grep)
  - https://cursor.com/docs/skills.md
  - https://cursor.com/docs/hooks.md
  - https://cursor.com/docs/rules.md
  - https://cursor.com/docs/subagents.md
  - https://cursor.com/docs/mcp.md
  - https://cursor.com/docs/plugins.md
  - https://cursor.com/docs/reference/plugins.md
  - https://cursor.com/docs/reference/third-party-hooks.md
  - https://cursor.com/help/customization/rules.md
  - https://cursor.com/changelog/2-4

## Config dirs

- **Project dir**: `.cursor/` at project root. Contents: `hooks.json`, `hooks/` (scripts), `rules/*.mdc`, `skills/`, `agents/`, `commands/`, `mcp.json`, `cli.json` (project-level CLI config, permissions only). (https://cursor.com/docs/hooks.md, https://cursor.com/docs/cli/reference/configuration.md)
- **Global/user dir**: `~/.cursor/` (Windows: `$env:USERPROFILE\.cursor\`). Contents: `cli-config.json`, `hooks.json`, `hooks/`, `skills/`, `agents/`, `mcp.json`, `plugins/local/`, `worktrees/<repo>/<name>`, `settings.json` (`enabled_plugins`). Overrides: `CURSOR_CONFIG_DIR` env; on Linux/BSD `$XDG_CONFIG_HOME/cursor/cli-config.json`. (https://cursor.com/docs/cli/reference/configuration.md, https://cursor.com/docs/cli/changelog.md, https://cursor.com/docs/cli/using.md)
- **Enterprise (MDM) dirs**: macOS `/Library/Application Support/Cursor/hooks.json`; Linux/WSL `/etc/cursor/hooks.json`; Windows `C:\ProgramData\Cursor\hooks.json`. (https://cursor.com/docs/hooks.md)
- **Compatibility dirs** (loaded in addition to `.cursor/`): `.claude/skills/`, `.codex/skills/` (+ `~/` variants) for skills (https://cursor.com/docs/skills.md); `.claude/agents/`, `.codex/agents/` (+ `~/` variants) for subagents (https://cursor.com/docs/subagents.md); `.claude/settings.json`, `.claude/settings.local.json`, `~/.claude/settings.json` for hooks, behind a "Third-party skills" setting (https://cursor.com/docs/reference/third-party-hooks.md). Canonical `.agents/skills/` is natively supported at project and user level (https://cursor.com/docs/skills.md).
- **Precedence**: hooks Enterprise → Team → Project → User → Claude project-local → Claude project → Claude user (all matching hooks run; higher priority wins on conflict) (https://cursor.com/docs/reference/third-party-hooks.md). Rules: Team → Project → User (https://cursor.com/docs/rules.md). Subagents: project beats user; `.cursor/` beats `.claude/`/`.codex/` on name conflict (https://cursor.com/docs/subagents.md). Nested `.cursor/skills/` (or `.agents/skills/`) anywhere in a monorepo is discovered and auto-scoped to its subtree (https://cursor.com/docs/skills.md).

## Instructions file

- **Filenames**: `AGENTS.md` at project root, plus nested `AGENTS.md` in any subdirectory — combined with parents, more specific files take precedence (https://cursor.com/docs/rules.md#agentsmd). The CLI "also reads `AGENTS.md` and `CLAUDE.md` at the project root (if present) and applies them as rules alongside `.cursor/rules`" (https://cursor.com/docs/cli/using.md). `CLAUDE.md` is always applied to every conversation regardless of frontmatter (https://cursor.com/help/customization/rules.md).
- **Format**: plain markdown, **no frontmatter, no metadata, no required heading/title convention** (https://cursor.com/docs/rules.md). Example starts with `# Project Instructions` but that's illustrative, not mandated.
- **Import/include syntax**: none for AGENTS.md itself. Rule files (`.mdc`) can pull files into context with `@filename.ext` mentions (https://cursor.com/docs/rules.md FAQ).
- **Legacy**: `.cursorrules` at project root is "legacy and will be deprecated"; migrate to `.cursor/rules/*.mdc` with Always Apply (https://cursor.com/help/customization/rules.md).

## Skills

Supported (since Cursor 2.4, editor + CLI; https://cursor.com/changelog/2-4, https://cursor.com/docs/skills.md).

- **Dirs**: `.agents/skills/`, `.cursor/skills/` (project); `~/.agents/skills/`, `~/.cursor/skills/` (user); compat `.claude/skills/`, `.codex/skills/` (+ `~` variants). Recursive walk of the skills root — any `SKILL.md` at any nesting depth is found; the skill's identity is the folder containing `SKILL.md`.
- **Layout**: `<skill-name>/SKILL.md` + optional `scripts/`, `references/`, `assets/` subdirs.
- **Frontmatter keys** (https://cursor.com/docs/skills.md):
  - `name` (required): lowercase letters, numbers, hyphens only; **must match the parent folder name**.
  - `description` (required): what it does and when to use it; drives agent relevance.
  - `paths` (optional): glob(s) scoping the skill to matching files; list or comma-separated string. Legacy `globs` still accepted as fallback.
  - `disable-model-invocation` (optional, bool): skill only loads when explicitly invoked via `/skill-name`.
  - `metadata` (optional): arbitrary key-value map.
  - `user-invocable` (optional, bool): `false` hides the skill from `/` autocomplete and typed `/skill-name` while keeping it model-available (CLI changelog, July 6 2026: https://cursor.com/docs/cli/changelog.md).
- **Cursor-unique keys** vs Claude's SKILL.md: `paths`, `disable-model-invocation`, `metadata`, `user-invocable`. Claude's `allowed-tools`/`license` are **not documented** for Cursor. The plugins reference lists only `name`+`description` for skill frontmatter (https://cursor.com/docs/reference/plugins.md) — a minor doc contradiction with the full skills page; treat the skills page as authoritative.
- **Naming/size limits**: only the kebab-case + folder-name-match rule for `name`; no documented size limit.

## Hooks

Supported, editor + CLI + cloud agents (https://cursor.com/docs/hooks.md).

- **Config file**: `hooks.json` (JSON). Locations: `<project>/.cursor/hooks.json`, `~/.cursor/hooks.json`, MDM paths (above), team dashboard (Enterprise). Watched and hot-reloaded.
- **Format**: `{ "version": 1, "hooks": { "<event>": [ <hook-def>, ... ] } }`.
- **Hook-def schema**: `command` (string, required — shell string, absolute path, or relative path); `type` (`"command"` default | `"prompt"`); `timeout` (seconds); `loop_limit` (number|null, default 5 for Cursor, applies to `stop`/`subagentStop` follow-ups); `failClosed` (bool, default false → fail-open); `matcher` (string regex — see below). Prompt-type hooks instead use `prompt` (with `$ARGUMENTS` placeholder, auto-appended if absent) + optional `model`; they return `{ ok: boolean, reason?: string }`.
- **Matcher semantics** (a plain string on the hook entry, not a nested object): `preToolUse`/`postToolUse`/`postToolUseFailure` match tool names (`Shell`, `Read`, `Write`, `Grep`, `Delete`, `Task`, or `MCP:<tool_name>`); `subagentStart`/`subagentStop` match subagent type (`generalPurpose`, `explore`, `shell`); `beforeShellExecution`/`afterShellExecution` match the full command string; `beforeReadFile` matches tool type (`TabRead`, `Read`); `afterFileEdit` matches (`TabWrite`, `Write`); `beforeSubmitPrompt` matches the literal `UserPromptSubmit`; `stop` → `Stop`; `afterAgentResponse` → `AgentResponse`; `afterAgentThought` → `AgentThought`.
- **Event names** (camelCase): `sessionStart`, `sessionEnd`, `preToolUse`, `postToolUse`, `postToolUseFailure`, `subagentStart`, `subagentStop`, `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`, `beforeReadFile`, `afterFileEdit`, `beforeSubmitPrompt`, `preCompact`, `stop`, `afterAgentResponse`, `afterAgentThought`, plus Tab hooks `beforeTabFileRead`/`afterTabFileEdit` and app-lifecycle `workspaceOpen`.
- **Working directory**: project hooks run from project root (use `.cursor/hooks/x.sh` paths); user hooks run from `~/.cursor/` (use `./hooks/x.sh`).
- **stdin JSON envelope — common fields on every hook**: `conversation_id`, `generation_id`, `model`, `model_id`, `model_params[{id,value}]`, `hook_event_name`, `cursor_version`, `workspace_roots[]`, `user_email`, `transcript_path`. (`workspaceOpen` omits conversation/generation/model/session/transcript fields.) Key per-event inputs:
  - `preToolUse`: `tool_name`, `tool_input` (object), `tool_use_id`, `cwd`, `agent_message`
  - `postToolUse`: + `tool_output` (JSON-stringified), `duration` (ms)
  - `postToolUseFailure`: `error_message`, `failure_type` (`error|timeout|permission_denied`), `duration`, `is_interrupt`
  - `beforeShellExecution`: `command`, `cwd`, `sandbox`; `afterShellExecution`: `command`, `output`, `duration`, `sandbox`
  - `beforeMCPExecution`: `tool_name`, `tool_input`, plus `url` or `command`; `afterMCPExecution`: + `result_json`, `duration`
  - `afterFileEdit`: `file_path`, `edits[{old_string,new_string}]` (Tab variant adds `range`, `old_line`, `new_line`)
  - `beforeReadFile`: `file_path`, `content`, `attachments[{type,file_path}]`
  - `beforeSubmitPrompt`: `prompt`, `attachments[]`
  - `subagentStart`: `subagent_id`, `subagent_type`, `task`, `parent_conversation_id`, `tool_call_id`, `subagent_model`, `is_parallel_worker`, `git_branch`; `subagentStop`: `status`, `summary`, `duration_ms`, `message_count`, `tool_call_count`, `loop_count`, `modified_files[]`, `agent_transcript_path`
  - `stop`: `status` (`completed|aborted|error`), `loop_count`; `sessionStart`: `session_id`, `is_background_agent`, `composer_mode`; `sessionEnd`: `session_id`, `reason`, `duration_ms`, `final_status`, `error_message`; `preCompact`: `trigger`, `context_usage_percent`, `context_tokens`, `context_window_size`, `message_count`, `messages_to_compact`, `is_first_compaction`
- **Output contract** (JSON on stdout; snake_case): `permission`: `"allow"|"deny"|"ask"` (`ask` accepted-but-not-enforced for `preToolUse`; treated as `deny` for `subagentStart`; supported on `beforeShellExecution`/`beforeMCPExecution`); `user_message`, `agent_message` (shown on deny); `updated_input` (preToolUse); `updated_mcp_tool_output`, `additional_context` (postToolUse); `followup_message` (stop/subagentStop auto-continue, subject to `loop_limit`); `continue` + `user_message` (beforeSubmitPrompt); `env{}` + `additional_context` (sessionStart); `pluginPaths[]` (workspaceOpen). **Exit codes**: 0 = success; 2 = block (≡ `permission:"deny"`); other = fail-open unless `failClosed: true`.
- **Env vars passed to hook scripts**: `CURSOR_PROJECT_DIR`, `CURSOR_VERSION`, `CURSOR_USER_EMAIL`, `CURSOR_TRANSCRIPT_PATH`, `CURSOR_CODE_REMOTE`, and `CLAUDE_PROJECT_DIR` as a compatibility alias. `sessionStart`-returned `env` persists to later hooks in the session.
- **Claude Code compat layer** (https://cursor.com/docs/reference/third-party-hooks.md): maps `PreToolUse→preToolUse`, `PostToolUse→postToolUse`, `UserPromptSubmit→beforeSubmitPrompt`, `Stop→stop`, `SubagentStop→subagentStop`, `SessionStart→sessionStart`, `SessionEnd→sessionEnd`, `PreCompact→preCompact`; accepts nested `hookSpecificOutput` (`permissionDecision`, `permissionDecisionReason`, `updatedInput`) and flat `{decision:"block", reason}` → `followup_message`; maps tools `Bash→Shell`, `Edit→Write`, etc. Claude `Notification`/`PermissionRequest` hooks and `Glob`/`WebFetch`/`WebSearch` matchers are NOT supported.

## Subagents

Supported (2.4+; https://cursor.com/docs/subagents.md).

- **Dirs**: `.cursor/agents/` (project), `~/.cursor/agents/` (user); compat `.claude/agents/`, `.codex/agents/` (+ `~` variants). Project > user; `.cursor/` > `.claude/`/`.codex/` on name conflicts.
- **Format**: one markdown file per subagent, YAML frontmatter + prompt body.
- **Frontmatter keys**: `name` (optional, default derived from filename; lowercase + hyphens), `description` (optional; drives Task-tool delegation), `model` (optional, default `inherit`; or a model ID with bracket params, e.g. `claude-opus-5[effort=high,context=300k]`), `readonly` (bool, default false — no file edits / state-changing shell), `is_background` (bool, default false).
- **Cursor-unique keys**: `model` (with `inherit` + `[param=value]` syntax), `readonly`, `is_background`. No `tools` key (Claude-style tool allowlists are not documented). Built-in subagents: Explore, Bash, Browser.

## Commands

- **Dir**: `.cursor/commands/[command].md` at project level; invoked via `/` menu in Agent (https://cursor.com/changelog/1-6). Plugin bundles also support a `commands/` dir accepting `.md`, `.mdc`, `.markdown`, `.txt` with optional frontmatter `name` + `description` only (https://cursor.com/docs/reference/plugins.md).
- **Deprecation trajectory**: the built-in `/migrate-to-skills` skill converts user-level and workspace-level slash commands into skills with `disable-model-invocation: true` (https://cursor.com/docs/skills.md). No official user-level (`~/.cursor/commands/`) dir is documented. No `argument-hint`/`allowed-tools`-style keys exist.

## Rules

- **Dir**: `.cursor/rules/` (project), files **must use the `.mdc` extension** — plain `.md`/`.markdown` in that dir are ignored (https://cursor.com/docs/rules.md). Nested subfolders allowed. Remote-imported rules land in `.cursor/rules/imported/<repoName>/`.
- **Format**: YAML frontmatter + markdown body. Frontmatter keys: `description` (string), `globs` (comma-separated glob string), `alwaysApply` (bool). Behavior matrix: `alwaysApply: true` → always included (description/globs ignored); `globs` set → auto-attached on file match; `description` only → agent-decides; neither → manual `@rule-name` mention only.
- **User Rules** are not file-based (stored in Cursor settings); **Team Rules** are dashboard-managed free-form text. Precedence Team → Project → User (https://cursor.com/help/customization/rules.md).
- **Conventions**: keep rules < 500 lines; `@filename.ext` references pull files into rule context.

## MCP

- **Config files**: `.cursor/mcp.json` (project), `~/.cursor/mcp.json` (global); also `mcp.json` inside plugin bundles (https://cursor.com/docs/mcp.md). The CLI auto-detects the same `mcp.json` as the editor (https://cursor.com/docs/cli/using.md).
- **Format**: JSON, top-level table `mcpServers` keyed by server name.
- **Server-table shape**:
  - stdio: `type: "stdio"`, `command` (required), `args[]`, `env{}`, `envFile` (path to env file — **stdio only**).
  - remote (HTTP/SSE): `url`, `headers{}`, optional `auth` object with `CLIENT_ID` (req), `CLIENT_SECRET`, `scopes[]` for static OAuth.
- **Env handling**: interpolation in `command`, `args`, `env`, `url`, `headers`: `${env:NAME}`, `${userHome}`, `${workspaceFolder}`, `${workspaceFolderBasename}`, `${pathSeparator}`, `${/}`. Env values may be inline or interpolated. Plugin manifests can declare `${VAR}` placeholders whose values are set in the dashboard (never stored in the repo).
- Transports: stdio, SSE, Streamable HTTP. `/mcp` slash command manages servers in CLI.

## Translation hazards

What a canonical `.agents` store must drop/rename/reformat for Cursor:

1. **Instructions**: Cursor has **no mandated H1 heading** in AGENTS.md/CLAUDE.md (agentlink's `(?im)^# .*?(Claude|Codex)` heading normalizer needs a Cursor arm only if a convention is chosen — safest is to emit no heading or a plain `# Project Instructions`). AGENTS.md must stay **frontmatter-free plain markdown** — strip any YAML frontmatter when targeting AGENTS.md. Nested AGENTS.md files are merged with more-specific-wins, so a single canonical instructions file maps 1:1 to root AGENTS.md only.
2. **Rules (if used instead of AGENTS.md)**: must rename files to `.mdc` (`.md` is silently ignored); must add Cursor-specific frontmatter (`description`, `globs` as a **comma-separated string**, `alwaysApply` bool) — no other agent uses this triple; drop any other keys. Keep bodies < 500 lines.
3. **Skills**: `name` must equal the parent folder name and be lowercase `[a-z0-9-]` — rename/re-folder on mismatch. Drop Claude-only keys (`allowed-tools`, `license` not documented for Cursor). Rename legacy `globs` → `paths` (list or comma string). `metadata` arbitrary map IS supported (unlike some agents) — keep it. `disable-model-invocation`/`user-invocable` are Cursor-native; preserve if present, drop if the canonical store doesn't model them. SKILL.md lives at `<folder>/SKILL.md`, never a bare `<name>.md`.
4. **Hooks — event names**: canonical PascalCase/Claude names must map to lowerCamelCase: `PreToolUse→preToolUse`, `PostToolUse→postToolUse`, `UserPromptSubmit→beforeSubmitPrompt`, `Stop→stop`, `SubagentStop→subagentStop`, `SessionStart→sessionStart`, `SessionEnd→sessionEnd`, `PreCompact→preCompact`. Claude `Notification`/`PermissionRequest` have NO Cursor equivalent — drop. Codex-style hooks have no mapping at all.
5. **Hooks — config shape**: unwrap Claude's nested `[{matcher, hooks:[{type, command}]}]` into Cursor's flat `[{command, matcher, timeout, loop_limit, failClosed}]` entries directly under `hooks.<event>`; add top-level `"version": 1`. Cursor's `matcher` is a bare regex string on the entry, not a wrapper object.
6. **Hooks — envelope fields**: stdin uses `conversation_id`/`generation_id`/`hook_event_name`/`cursor_version`/`workspace_roots` (NOT Claude's `session_id`/`hook_event_name`/`cwd`-centric envelope; note Cursor preToolUse *does* include `cwd`). File-edit input uses `file_path` + `edits[]` (old/new string pairs), not Claude's `tool_input.file_path`. Shell hooks use `command` + `sandbox`. agentlink's `guard`/`remind` parsers must accept `file_path`, `edits`, `command`, `tool_input` (object form), `tool_output` (JSON-stringified string), and `hook_event_name` lowerCamel values.
7. **Hooks — output**: snake_case keys (`permission`, `user_message`, `agent_message`, `updated_input`, `followup_message`, `additional_context`), not Claude's `permissionDecision`/`hookSpecificOutput` nesting (Cursor accepts both on input from Claude configs, but canonical emitters should use Cursor-native). `ask` is unenforced for `preToolUse` and means `deny` for `subagentStart` — don't rely on it. Exit code 2 ≡ deny.
8. **Hooks — tool/matcher names**: rename `Bash→Shell`, `Edit→Write`; MCP matchers use `MCP:<tool_name>`; `Glob`/`WebFetch`/`WebSearch` matchers are unsupported — drop or fold into `preToolUse` without matcher.
9. **Hooks — script paths**: project-hook commands run from the project root, so paths must be `.cursor/hooks/x.sh` (a canonical store generating `./hooks/x.sh` would break, since that resolves to `<project>/hooks/`).
10. **Subagents**: rename Claude keys → Cursor: no `tools` key (drop), `model` accepts `inherit` or `id[param=value]` brackets (reformat any other model syntax), add `readonly`/`is_background` only if modeled canonically. Filenames are free-form `.md`; `name` defaults from filename.
11. **Commands**: emit `.md` files in `.cursor/commands/` with at most `name`/`description` frontmatter — drop Claude command keys (`argument-hint`, `allowed-tools`, `model`). Prefer emitting skills with `disable-model-invocation: true` going forward (official migration direction).
12. **MCP**: emit JSON `{"mcpServers": {...}}` (not TOML). Server entries: stdio = `type/command/args/env/envFile`; remote = `url/headers/auth`. Convert any TOML `[mcp_servers.x]` env tables to JSON objects; convert `${VAR}` styles to `${env:VAR}`; `envFile` is stdio-only (drop for remote). Env VALUES are stored inline or interpolated — agentlink's name-only comparison must read `env` as a JSON object of the entry.
13. **Compat-dirs caveat**: Cursor reads `.claude/` and `.codex/` skills/agents/hooks natively — a sync tool targeting Cursor can either emit Cursor-native paths (`.cursor/…`) or rely on the compat layer, but precedence rules (`.cursor/` wins) mean mixed deployments can shadow each other.
14. **CLI config**: `cli-config.json`/`cli.json` are pure JSON (no comments), and only `permissions.allow`/`permissions.deny` (exact strings like `Shell(ls)`) may be set at project level — don't sync other keys into `.cursor/cli.json`.

### Critical Files for Implementation

- `internal/link/normalize.go` — add Cursor arms to the `instructions` heading regex, `skill` frontmatter canonicalization (keep `paths`/`metadata`/`disable-model-invocation`, drop `allowed-tools`/`license`), and `hook` normalizer (Cursor command/path conventions).
- `internal/link/mcp.go` — add `.cursor/mcp.json` (JSON, `mcpServers` table) as a third MCP peer alongside `.mcp.json` and `config.toml`; env-name extraction from JSON objects.
- `internal/config/schema.go` — add the `cursor` endpoint kind with paths `.cursor/` (project), `~/.cursor/` (user), plus the `.agents/skills/` native pairing.
- `internal/hookinput/input.go` — accept Cursor hook envelopes (`conversation_id`, `hook_event_name` camelCase events, `file_path`+`edits[]`, `command`, object-form `tool_input`).
- `internal/adopt/adopt.go` — verify the generic `.<agent>/skills → .agents/skills` mapping holds for `.cursor/` (it does; Cursor reads `.agents/skills/` natively) and that `.cursor/rules/*.mdc` / `hooks.json` land under `.agents/cursor/*`.

## Verified corrections (fact-checker pass)

- was: Installs to `~/.local/bin`; `~/.cursor/bin` used in CI → now: Installs to `~/.local/bin` only; no official source documents `~/.cursor/bin` as a CI install path — current installation docs mention only `~/.local/bin` and PATH setup (https://cursor.com/docs/cli/installation.md)
- was: No official user-level (`~/.cursor/commands/`) dir is documented → now: No user-level commands directory path is documented, but official docs do acknowledge user-level slash commands: /migrate-to-skills 'converts both user-level and workspace-level commands', so a user-level commands concept exists even though its on-disk path is undocumented (https://cursor.com/docs/skills.md)
- was: `matcher` (string regex); Matcher semantics (a plain string on the hook entry, not a nested object) → now: All official examples use a plain string matcher (supporting the brief), but the Per-Script Configuration Options table types `matcher` as `object` — the docs are internally inconsistent; treat string form as de facto correct per examples (https://cursor.com/docs/hooks.md)
