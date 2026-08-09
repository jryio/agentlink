# hermes

## Identity

- **Binary:** `hermes` (Python package `hermes_cli` in-repo; also `hermes --tui` for an Ink TUI). Exists and is actively maintained — 21,227 commits, releases cut within days of research date (2026-08).
- **Vendor:** Nous Research. Product name: **Hermes Agent** (self-improving autonomous agent with a CLI, messaging gateway, TUI, and desktop app). Released February 2026. Note: this is a general autonomous agent, not purely a coding CLI, but it has full coding-agent config surfaces (context files, skills, hooks, MCP).
- **Repo:** https://github.com/NousResearch/hermes-agent
- **Install:** `curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash` (Linux/macOS/WSL2/Termux); `iex (irm https://hermes-agent.nousresearch.com/install.ps1)` (native Windows); or the Hermes Desktop installer.
- **Docs base:** https://hermes-agent.nousresearch.com/docs/ (Docusaurus; machine-readable `/docs/llms.txt` and `/docs/llms-full.txt`).
- **URLs actually read:**
  - https://hermes-agent.nousresearch.com/docs/
  - https://github.com/NousResearch/hermes-agent
  - https://hermes-agent.nousresearch.com/docs/llms.txt
  - https://hermes-agent.nousresearch.com/docs/user-guide/configuration
  - https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files
  - https://hermes-agent.nousresearch.com/docs/user-guide/features/skills
  - https://hermes-agent.nousresearch.com/docs/developer-guide/creating-skills
  - https://hermes-agent.nousresearch.com/docs/user-guide/features/hooks
  - https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp
  - https://hermes-agent.nousresearch.com/docs/reference/mcp-config-reference
  - https://hermes-agent.nousresearch.com/docs/user-guide/features/delegation
  - https://hermes-agent.nousresearch.com/docs/user-guide/import-from-other-agents

## Config dirs

- **Global/user dir:** `~/.hermes/` — this is `HERMES_HOME`, overridable via the `HERMES_HOME` env var; named profiles (`hermes profile create <name>`) each get their own `HERMES_HOME`. "All settings are stored in the `~/.hermes/` directory" (https://hermes-agent.nousresearch.com/docs/user-guide/configuration). Layout:
  ```text
  ~/.hermes/
  ├── config.yaml     # all non-secret settings (model, mcp_servers, hooks, skills, ...)
  ├── .env            # API keys and secrets
  ├── auth.json       # OAuth provider credentials
  ├── SOUL.md         # global personality (slot #1 in system prompt)
  ├── memories/       # MEMORY.md, USER.md
  ├── skills/         # skills — single source of truth
  ├── skill-bundles/  # <slug>.yaml bundle files
  ├── hooks/          # gateway hooks (<name>/HOOK.yaml + handler.py)
  ├── agent-hooks/    # conventional home for shell-hook scripts
  ├── plugins/        # Python plugins
  ├── cron/  sessions/  logs/  pending/  mcp-tokens/
  └── shell-hooks-allowlist.json
  ```
- **Project config dir:** **none for settings.** There is no project-scoped `.hermes/config.yaml` or settings file; all settings are global-to-profile. Projects contribute only *context files* (see Instructions). The one project-local `.hermes/` path that exists is `.hermes/plans/`, an **output** directory the bundled `plan` skill writes plans into (https://hermes-agent.nousresearch.com/docs/user-guide/features/skills) — not configuration.
- **Precedence (settings):** CLI args → `~/.hermes/config.yaml` → `~/.hermes/.env` (secrets) → built-in defaults (https://hermes-agent.nousresearch.com/docs/user-guide/configuration). A system-level "Managed Scope" lets orgs pin values (referenced at /docs/user-guide/managed-scope, not read).
- **Precedence (project context files):** first match wins in order `.hermes.md`/`HERMES.md` → `AGENTS.md` → `CLAUDE.md` → `.cursorrules`; only ONE project context type is loaded per session. `SOUL.md` loads independently, from `HERMES_HOME` only (https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files).

## Instructions file

- **Filenames** (https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files):
  | File | Discovery |
  |---|---|
  | `.hermes.md` / `HERMES.md` | project instructions, highest priority; walks up to git root |
  | `AGENTS.md` | "primary project context file"; CWD at startup + **progressive subdirectory discovery** during the session (nested AGENTS.md injected when the agent touches that subtree; walks up to 5 parents; each dir checked once per session) |
  | `CLAUDE.md` | same progressive discovery as AGENTS.md |
  | `.cursorrules` + `.cursor/rules/*.mdc` | CWD only, lowest priority fallback |
  | `SOUL.md` | global personality, `$HERMES_HOME/SOUL.md` only; never probed in CWD |
- **Heading/title conventions:** none required inside the file. Hermes wraps loaded content itself: everything goes under a `# Project Context` header with one `## <filename>` section per file; SOUL.md content is appended verbatim with no wrapper. Any `# ...` heading the user writes is just content.
- **Import/include syntax:** none. No `@file` imports documented.
- **Size limits:** startup files truncated at `context_file_max_chars` (config.yaml) else dynamic with model context window (floor 20,000 / ceiling 500,000 chars), 70% head / 20% tail / 10% marker. Progressively discovered subdirectory files are capped at 8,000 chars each. All files pass a prompt-injection scanner; matches are blocked.

## Skills

**Supported** — core feature ("procedural memory"). Sources: https://hermes-agent.nousresearch.com/docs/user-guide/features/skills and https://hermes-agent.nousresearch.com/docs/developer-guide/creating-skills

- **Dir:** `~/.hermes/skills/` — primary, read-write, single source of truth. Additional read/scan dirs via `skills.external_dirs` in config.yaml (supports `~` and `${VAR}`); **local dir wins on name collision**; non-existent dirs silently skipped. WARNING for agentlink: external dirs are *not* write-protected — the agent edits skills in place wherever found.
- **Layout:** `<category>/<skill-name>/SKILL.md` (category dir optional; bundled skills use categories, agent-created skills may not). `SKILL.md` required; optional `references/`, `templates/`, `scripts/`, `examples/`, `assets/` subdirs. Hub state in `skills/.hub/` (lock.json, taps.json, quarantine); `skills/.bundled_manifest` tracks seeded bundled skills. Skills are agentskills.io-spec compatible.
- **Frontmatter keys** (exact, from the creating-skills guide):
  - `name` — skill identifier; valid-identifier regex `^[a-z][a-z0-9_-]*$` (used in URL-install name resolution)
  - `description` — shown in skill search/index; house standard is ≤60 chars (guidance, not a documented hard limit)
  - `version`, `author`, `license` — standard agentskills.io metadata
  - `platforms` — list subset of `[macos, linux, windows]`; hides skill on other OSes
  - `metadata.hermes.tags`, `metadata.hermes.category`, `metadata.hermes.related_skills` — catalog grouping
  - `metadata.hermes.requires_toolsets` / `requires_tools` — hide skill unless listed toolsets/tools ARE available
  - `metadata.hermes.fallback_for_toolsets` / `fallback_for_tools` — hide skill when listed toolsets/tools ARE available
  - `metadata.hermes.config[]` — `{key, description, default, prompt}` non-secret settings persisted to `skills.config.*` in config.yaml
  - `metadata.hermes.blueprint` — `{schedule, deliver, prompt, no_agent}`; marks the skill a runnable cron automation (opt-in via `/suggestions`)
  - `required_environment_variables[]` — `{name, prompt, help, required_for}`; secrets prompted at load, stored in `~/.hermes/.env`, auto-passthrough to sandboxes. Legacy alias: `prerequisites.env_vars`
  - `required_credential_files[]` — `{path, description}` relative to `~/.hermes/`; mounted into Docker/Modal sandboxes
- **Hermes-unique keys:** `platforms`, the entire `metadata.hermes.*` block (`requires_*`, `fallback_for_*`, `config`, `blueprint`, `category`, `related_skills`), `required_environment_variables` (with prompt/help/required_for sub-keys), `required_credential_files`. NOT Hermes-unique: `name`, `description`, `version`, `author`, `license`, `metadata` (agentskills.io standard, shared with Claude/Codex skills). Notably there is **no `allowed-tools` equivalent** — skills cannot restrict tool use.
- **Naming/limits:** dir name = install slug; dirs starting with `.` or `_` are ignored in taps; `name` should match `^[a-z][a-z0-9_-]*$`; description ≤60 chars is the documented authoring standard. No documented hard size cap on SKILL.md (context-file caps don't apply to skills; inline-shell snippet output capped at 4,000 chars). Body template tokens `${HERMES_SKILL_DIR}` and `${HERMES_SESSION_ID}` are substituted at load (disable via `skills.template_vars: false`); `` !`cmd` `` inline shell is opt-in (`skills.inline_shell: true`).
- Also: `~/.hermes/skill-bundles/<slug>.yaml` group skills under one slash command (`name`, `description`, `skills[]`, `instruction`); bundles shadow same-named skills.

## Hooks

**Supported — four systems** (https://hermes-agent.nousresearch.com/docs/user-guide/features/hooks):

1. **Shell hooks** (the agentlink-relevant one): `hooks:` block in `~/.hermes/config.yaml`, runs in CLI + gateway.
   - **Config schema:**
     ```yaml
     hooks:
       <event_name>:
         - matcher: "<regex>"        # optional; pre/post_tool_call only; matches tool NAME
           command: "<shell command>" # required; shlex.split, shell=False
           timeout: 60                 # optional; capped at 300
           fail_closed: false          # optional; pre_tool_call only; `failClosed` also accepted
     hooks_auto_accept: false
     ```
   - **Event names** (snake_case, plugin-hook set = `VALID_HOOKS`): `pre_tool_call`, `post_tool_call`, `pre_llm_call`, `post_llm_call`, `pre_verify`, `on_session_start`, `on_session_end`, `on_session_finalize`, `on_session_reset`, `subagent_start`, `subagent_stop`, `pre_gateway_dispatch`, `pre_approval_request`, `post_approval_response`, `transform_tool_result`, `transform_terminal_output`, `transform_llm_output`, plus `kanban_task_claimed/completed/blocked`. Only `pre_tool_call` can block; only `pre_llm_call` injects context; only `pre_verify` can force continuation; transforms rewrite strings.
   - **stdin JSON envelope:**
     ```json
     {"hook_event_name": "pre_tool_call", "tool_name": "terminal", "tool_input": {"command": "..."}, "session_id": "sess_...", "cwd": "/path", "extra": {"task_id": "...", "...": "event-specific kwargs"}}
     ```
     `tool_name`/`tool_input` are `null` for non-tool events; `extra` carries all event kwargs (`user_message`, `child_role`, `duration_ms`, ...).
   - **Output contract (stdout JSON):** `{"action": "block", "message": "..."}` (Hermes-canonical) or `{"decision": "block", "reason": "..."}` (Claude-Code style — both accepted); `{"context": "..."}` for `pre_llm_call` injection; `{"action": "continue", "message": "..."}` (or Claude Stop-style `{"decision":"block","reason":"..."}`) for `pre_verify`. **Exit code 2 blocks a `pre_tool_call`** even with no JSON (stderr ≤400 chars becomes the message) — Claude Code/Cursor compatible. Empty/`{}` output = no-op. Default fail-open; `fail_closed: true` blocks on spawn error/timeout/bad JSON. Malformed output never crashes the agent.
   - **Consent:** first-use prompt per `(event, command)` pair, persisted to `~/.hermes/shell-hooks-allowlist.json` (`{"approvals": [{"event", "command"}]}`); bypass via `--accept-hooks`, `HERMES_ACCEPT_HOOKS=1`, or `hooks_auto_accept: true`. Script edits are silently trusted (keyed on command string, not hash).
2. **Gateway hooks:** `~/.hermes/hooks/<name>/HOOK.yaml` (`name`, `description`, `events: [...]`) + `handler.py` (Python `handle(event_type, context)`). Gateway-only events with `:` names: `gateway:startup`, `session:start/end/reset/compress`, `agent:start/step/end`, `reaction:added/removed`, `command:*` (wildcards supported). Do NOT fire in CLI.
3. **Plugin hooks:** Python plugins in `~/.hermes/plugins/<name>/` calling `ctx.register_hook()` — same event names as shell hooks.
4. **Outbound webhooks:** `hooks.outbound:` list in config.yaml — `{name, url, events[], secret_env, timeout, matcher}`; POSTs signed JSON (same envelope + `delivery_id`, `timestamp`; `X-Hermes-Signature-256` HMAC header). Notify-only, cannot block.

## Subagents

**Not folder-based.** Subagents are spawned programmatically via the `delegate_task` tool (`goal` + `context` fields; up to 3 concurrent by default, configurable). Each child is a fresh `AIAgent` with isolated context built at call time — there is no `agents/` directory, no per-subagent markdown file, no frontmatter (https://hermes-agent.nousresearch.com/docs/user-guide/features/delegation). Observability only via `subagent_start`/`subagent_stop` hooks. A canonical `.agents/agents/` store has **no Hermes target**.

## Commands

**No custom slash-command directory.** Every installed skill is automatically a slash command (`/<skill-name> [args]`, stackable up to 5), and bundles add `/<bundle-name>` (https://hermes-agent.nousresearch.com/docs/user-guide/features/skills). Decisive evidence: `hermes import-agent claude-code` **skips `commands/*.md` "with a note — convert them into skills"** (https://hermes-agent.nousresearch.com/docs/user-guide/import-from-other-agents). Built-in slash commands are fixed (see /docs/reference/slash-commands, not read). To target Hermes, slash commands must be rewritten as SKILL.md skills.

## Rules

**No dedicated rules directory.** `.cursorrules` and `.cursor/rules/*.mdc` are consumed only as the lowest-priority *project context file*, CWD only, and only when no `.hermes.md`/`AGENTS.md`/`CLAUDE.md` exists (https://hermes-agent.nousresearch.com/docs/user-guide/features/context-files). Standing agent-behavior rules otherwise live in SOUL.md (global personality), `agent.coding_instructions` in config.yaml, or skills.

## MCP

- **Config file:** `~/.hermes/config.yaml`, key `mcp_servers:` (YAML; snake_case — NOT `mcpServers`). Global only; no project-level MCP file (https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp, https://hermes-agent.nousresearch.com/docs/reference/mcp-config-reference).
- **Server-table shape:**
  ```yaml
  mcp_servers:
    <name>:
      command: "npx"          # stdio
      args: [...]
      env: {KEY: "${VAR}"}
      # OR
      url: "https://..."      # HTTP
      headers: {...}
      auth: oauth             # HTTP OAuth 2.1 + PKCE; tokens cached in ~/.hermes/mcp-tokens/<server>.json (0o600)
      transport: sse          # default streamable-HTTP
      ssl_verify: true        # bool | CA bundle path
      client_cert / client_key: ...
      enabled: true
      timeout: 300            # tool call seconds
      connect_timeout: 60
      keepalive_interval: 180
      supports_parallel_tool_calls: false
      skip_preflight: false
      idle_timeout_seconds / max_lifetime_seconds: 0   # stdio lifecycle
      tools: {include: [...], exclude: [...], resources: true, prompts: true}  # fnmatch globs; include wins
      sampling / elicitation: {...}
      trust: full | untrusted  # untrusted => approval for every non-readOnlyHint tool
  ```
- **Env handling:** `${VAR}` and Cursor-style `${env:VAR}` both resolve (from profile secret scope incl. `~/.hermes/.env`, falling back to process env); unset vars stay literal. Also `${userHome}`, `${workspaceFolder}`, `${workspaceFolderBasename}`, `${pathSeparator}`/`${/}`. Secrets belong in `~/.hermes/.env`, not inline; `hermes import-agent` strips secret-looking env/header values (`*_TOKEN`, `*_API_KEY`, `Authorization`) on import.

## Translation hazards

Concrete drops/renames/reformats a canonical `.agents` store needs to target Hermes:

1. **Single-file YAML config.** Hermes has no per-feature config files — MCP servers, hooks, skill settings all live in `~/.hermes/config.yaml` (YAML). No JSON (Claude `.mcp.json`/`settings.json`) and no TOML (Codex `config.toml`) targets exist; translators must merge into YAML and tolerate unrelated keys.
2. **No project-level config at all.** Everything under `~/.hermes/` is global-to-profile. Project repos can only carry context files (AGENTS.md etc.). agentlink's project-scope pairs must exclude Hermes settings; the adopt mapping `.hermes/* → .agents/hermes/*` only makes sense for the *global* home, not a project checkout. (`.hermes/plans/` in a workspace is agent output, not config — do not sync it.)
3. **MCP key rename:** `mcpServers` → `mcp_servers`. Drop Hermes-only server keys when exporting FROM Hermes (`tools` include/exclude/resources/prompts, `trust`, `sampling`, `elicitation`, `enabled`, `skip_preflight`, `keepalive_interval`, `idle/max_lifetime_seconds`, `supports_parallel_tool_calls`); when importing TO Hermes these are safely omittable. Convert `type: "stdio"|"http"` discriminators to Hermes' implicit shape (presence of `command` vs `url`; `transport: sse` for SSE). Keep secret VALUES out — use `${VAR}`/`${env:VAR}` refs; Hermes itself strips `*_TOKEN`/`*_API_KEY`/`Authorization` values on import.
4. **Instructions filename priority trap:** only ONE project context file type loads — if a repo has both AGENTS.md and CLAUDE.md, CLAUDE.md is silently ignored (`.hermes.md` > `AGENTS.md` > `CLAUDE.md` > `.cursorrules`). Syncing CLAUDE.md content to Hermes requires writing/merging AGENTS.md (or `.hermes.md`), not a parallel file. Also: Hermes wraps content in its own `# Project Context` / `## <filename>` headers — no heading convention to preserve, and no import/include syntax to translate (strip `@import` lines).
5. **Subdirectory context files are first-class** (progressive discovery, 8,000-char cap each) — nested AGENTS.md trees transfer, but files >8k chars in subdirs get truncated; startup files cap at `context_file_max_chars` (floor 20k).
6. **Skills are global-only.** No project skills dir; a canonical project `.agents/skills/` must either be copied/symlinked into `~/.hermes/skills/` or registered via `skills.external_dirs` — and external_dirs entries are **mutated in place** by the agent (`skill_manage`), so a symlinked canonical store is live-writable by Hermes. Local `~/.hermes/skills/` wins name collisions with external dirs.
7. **Skill frontmatter drops (canonical → Hermes):** Claude's `allowed-tools` has NO Hermes equivalent — drop it (Hermes skills can't gate tools; conditional visibility uses `metadata.hermes.requires_*/fallback_*` instead, which is semantics, not a rename). Keep `name`/`description`/`version`/`author`/`license`/`metadata` (agentskills.io-standard). Hermes-extra keys (`platforms`, `metadata.hermes.*`, `required_environment_variables`, `required_credential_files`) must be dropped when targeting other agents; `platforms` uses `macos` not `darwin`. Enforce name regex `^[a-z][a-z0-9_-]*$` and ≤60-char descriptions. Strip body directives `[[as_document]]`/`[[audio_as_voice]]`, `${HERMES_SKILL_DIR}`/`${HERMES_SESSION_ID}` tokens, and `` !`...` `` inline shell when exporting to other agents.
8. **Hook event-name mismatches** (Claude Code → Hermes): `PreToolUse` → `pre_tool_call`; `PostToolUse` → `post_tool_call`; `SessionStart` → `on_session_start`; `SubagentStop` → `subagent_stop`; `UserPromptSubmit` → `pre_llm_call` (explicitly the same slot per the docs); `Stop` ≈ `pre_verify` but NOT equivalent (only fires when code was edited, return is `continue` not `block`); `Notification` → `pre_approval_request` (approx). Hermes-only events with no Claude counterpart: `pre_gateway_dispatch`, `transform_*`, `post_approval_response`, `on_session_finalize/reset`, `kanban_task_*`. Gateway `HOOK.yaml`/`handler.py` events (`agent:start`, `command:*`) are a different namespace entirely and gateway-only — not translatable to shell hooks.
9. **Hook config schema:** entries are a flat YAML list per event (`matcher`/`command`/`timeout`/`fail_closed`), not Claude's nested `hooks: [{matcher, hooks: [{type, command}]}]`; `type: "command"` is implicit. `matcher` is a regex against the **tool name only** (no path/glob matching like Claude's `Write|Edit` + file-path matchers, no Codex-style matchers). `command` is `shlex.split` with `shell=False` — pipes, redirects, and shell builtins in the command string break; wrap in a script file. Timeout is seconds, capped at 300.
10. **Hook stdin envelope:** fields `hook_event_name`, `tool_name`, `tool_input`, `session_id`, `cwd`, `extra{}` — close to Claude's envelope but NO `transcript_path`; everything event-specific is buried in the `extra` bag (agentlink's `guard`/`remind` must read paths from `tool_input.path` / `tool_input.file_path` and fall back to `extra`).
11. **Hook output contract:** stdout JSON accepts both `{"action":"block","message"}` and Claude's `{"decision":"block","reason"}`; exit code 2 = block (Claude-compatible). BUT context injection is `{"context": "..."}` on `pre_llm_call` — NOT Claude's `hookSpecificOutput.additionalContext` on `UserPromptSubmit`; that shape must be rewritten. `fail_closed`/`failClosed` is Hermes/Cursor-style; exit-2 semantics only block on `pre_tool_call`.
12. **Hook consent gate:** synced hook configs won't run on a fresh machine until each `(event, command)` pair is approved (allowlist `~/.hermes/shell-hooks-allowlist.json` keyed on the exact command string) or `hooks_auto_accept: true`/`HERMES_ACCEPT_HOOKS=1` is set. Non-TTY runs silently skip unapproved hooks.
13. **Slash commands:** must be converted to skills (SKILL.md in `~/.hermes/skills/`); Hermes' own importer refuses Claude `commands/*.md`. Argument substitution conventions (`$ARGUMENTS` etc.) have no documented equivalent — the skill body + trailing text is the mechanism.
14. **Subagent definitions:** `.claude/agents/*.md`-style files have no target — drop them or rewrite as skills.
15. **Rules dir:** no target; fold rules into AGENTS.md. Note `.cursor/rules/*.mdc` is read ONLY when no higher-priority context file exists — a repo with AGENTS.md never loads `.cursor/rules`, so don't rely on it as a side-channel.
16. **SOUL.md / MEMORY.md / USER.md:** Hermes-only global files (`~/.hermes/`); SOUL.md is never read from a project dir. No counterpart in Claude/Codex config trees; exclude from canonical sync or map explicitly. Global CLAUDE.md/AGENTS.md content maps to Hermes `memories/MEMORY.md` entries per the official importer, not to a global instructions file.

Cross-check notes: hooks claims verified against both the hooks page (shell-hook schema, envelope, exit-2 semantics) and the configuration page (config.yaml location/precedence); skills frontmatter verified against both the user-guide skills page and the developer-guide creating-skills page (consistent; creating-skills is the superset). No contradictions found between official pages.