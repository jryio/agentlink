# gemini

## Identity
- **Binary:** `gemini` (npm package `@google/gemini-cli`; also `brew install gemini-cli`; `npx https://github.com/google-gemini/gemini-cli`). Requires Node.js ≥ 20. Source: https://google-gemini.github.io/gemini-cli/
- **Vendor:** Google. Repo: https://github.com/google-gemini/gemini-cli (Apache-2.0, actively maintained — weekly stable/preview/nightly release cadence, 6k+ commits).
- **Docs:**
  - Docs site: https://google-gemini.github.io/gemini-cli/ (Jekyll mirror of the repo README + `docs/`; note: deep `.html` paths like `/docs/cli/skills.html` 404'd — the repo `docs/` markdown is the reliable source).
  - Pages actually read (raw markdown, `main` branch):
    - https://github.com/google-gemini/gemini-cli (repo README page)
    - https://google-gemini.github.io/gemini-cli/ (install/identity)
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/creating-skills.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/using-agent-skills.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills-best-practices.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/gemini-md.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/custom-commands.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/core/subagents.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/index.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/reference.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/tools/mcp-server.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/reference/configuration.md
    - https://raw.githubusercontent.com/google-gemini/gemini-cli/main/packages/core/src/skills/skillLoader.ts (authoritative frontmatter parser)

## Config dirs
- **Project dir:** `.gemini/` in the project root — holds `settings.json`, `commands/`, `skills/`, `agents/`, sandbox profiles (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/reference/configuration.md).
- **User/global dir:** `~/.gemini/` — `settings.json`, `commands/`, `skills/`, `agents/`, `policies/`, plus runtime state (browser profile, session data). `GEMINI.md` also read directly at `~/.gemini/GEMINI.md`.
- **System dirs:** Linux `/etc/gemini-cli/` (`system-defaults.json`, `settings.json`); macOS `/Library/Application Support/GeminiCli/`; Windows `C:\ProgramData\gemini-cli\`. Overridable via `GEMINI_CLI_SYSTEM_DEFAULTS_PATH` / `GEMINI_CLI_SYSTEM_SETTINGS_PATH`.
- **Precedence** (config reference, low→high): defaults → system-defaults file → user settings → project settings → system settings file → env vars → CLI args. ⚠️ **Doc contradiction:** the hooks doc states hook config precedence highest→lowest as "Project → User → System → Extensions" (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/index.md), while the configuration reference says the system settings file "override[s] all other settings files" (highest). Treat project-level as winning for practical purposes but note the disagreement.
- Extensions are an additional config layer (bundled skills, commands, agents, MCP servers, hooks).

## Instructions file
- **Filename:** `GEMINI.md` (default). Customizable to other names via `context.fileName` in `settings.json` (string or list, e.g. `["AGENTS.md", "CONTEXT.md", "GEMINI.md"]`) — so AGENTS.md is supported only if configured, not by default. Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/gemini-md.md
- **Locations/discovery (all concatenated):**
  1. Global: `~/.gemini/GEMINI.md`
  2. Workspace: `GEMINI.md` in configured workspace dirs **and their parent directories** (upward search)
  3. Just-in-time: when a tool touches a file/dir, `GEMINI.md` files in that dir and its ancestors up to a trusted root are auto-loaded
- **Heading/title convention:** none. No required H1; example files use free-form headings. (agentlink's `instructions` normalizer heading regex should treat any/no heading as fine.)
- **Import syntax:** `@file.md` on its own line imports another markdown file (relative or absolute paths; recursive per the Memory Import Processor — `docs/reference/memport.md`). This is unique vs. Claude/Codex.
- Related: `.geminiignore` file controls context exclusion (docs/cli/gemini-ignore.md).

## Skills
- **Supported:** yes (Agent Skills / agentskills.io standard). Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md
- **Dirs (precedence low→high):** built-in → extension-bundled → user `~/.gemini/skills/` (alias `~/.agents/skills/`) → workspace `.gemini/skills/` (alias `.agents/skills/`). Same-name skill in a higher tier wins; within a tier, `.agents/skills/` beats `.gemini/skills/`. Note: **`.agents/skills` is a first-class alias — agentlink's canonical tree maps directly.**
- **File layout:** directory per skill containing `SKILL.md` (required) + optional `scripts/`, `references/`, `assets/`. Loader globs only `SKILL.md` and `*/SKILL.md` — i.e. top-level file or exactly one level of subdirectory, no deeper nesting (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/packages/core/src/skills/skillLoader.ts).
- **Frontmatter keys (exact — from parser source, not just docs):**
  - `name` (string, required) — unique ID; docs say it "should match the directory name". Loader sanitizes it: chars `: \ / < > * ? " |` are replaced with `-`.
  - `description` (string, required) — sole trigger metadata shown to the model pre-activation.
  - **All other keys are silently ignored** by the loader (`parseFrontmatter` destructures only `{name, description}`; YAML parse failures fall back to a line-based parser that also only reads those two keys). So Claude-style keys like `allowed-tools`, `license`, `metadata`, `compatibility` are tolerated but inert — **none are honored**.
  - Keys unique to Gemini CLI: none (it uses a strict subset of the agentskills.io/Claude schema).
- **Size/naming limits:** no enforced limits found in docs or loader. Best-practices doc *recommends* `SKILL.md` body < 5k words and metadata ~100 words (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills-best-practices.md). No lowercase/hyphen naming rule for skills (unlike subagents); the sanitizer only handles path-unsafe chars.
- Activation is consent-gated per session; management via `/skills` and `gemini skills` CLI.

## Hooks
- **Supported:** yes. Sources (cross-checked): https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/index.md and https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/reference.md
- **Config file + format:** JSON, inside `settings.json` (`hooks` top-level key) at any settings layer (`.gemini/settings.json`, `~/.gemini/settings.json`, system). No separate hooks file.
- **Event names (11):** `SessionStart`, `SessionEnd`, `BeforeAgent`, `AfterAgent`, `BeforeModel`, `AfterModel`, `BeforeToolSelection`, `BeforeTool`, `AfterTool`, `PreCompress`, `Notification`. ⚠️ Tool events are `BeforeTool`/`AfterTool` — **not** Claude's `PreToolUse`/`PostToolUse`.
- **Matcher schema:** per event, an array of hook definitions: `{ matcher?: string, sequential?: boolean, hooks: HookConfig[] }`. `matcher` is a **regex** for tool events (matched against tool names like `write_file`, `read_file`, `run_shell_command`; MCP tools are `mcp_<server>_<tool>`), an **exact string** for lifecycle events (e.g. `"startup"`); `"*"` or `""` matches all. `sequential: true` runs the group's hooks serially, else parallel.
- **Command schema:** `{ type: "command" (required, only value), command: string (required), name?: string, timeout?: number (ms, default 60000), description?: string }`.
- **Stdin JSON envelope — base fields (all events):** `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `timestamp` (ISO 8601). Per-event additions:
  - `BeforeTool`: `tool_name`, `tool_input` (object), `mcp_context?`, `original_request_name?`
  - `AfterTool`: `tool_name`, `tool_input`, `tool_response` (`{llmContent, returnDisplay, error?}`), `mcp_context?`, `original_request_name?`
  - `BeforeAgent`: `prompt`; `AfterAgent`: `prompt`, `prompt_response`, `stop_hook_active`
  - `BeforeModel`/`BeforeToolSelection`: `llm_request` (`{model, messages, config, toolConfig}`); `AfterModel`: `llm_request` + `llm_response`
  - `SessionStart`: `source` (`startup|resume|clear`); `SessionEnd`: `reason` (`exit|clear|logout|prompt_input_exit|other`); `Notification`: `notification_type`, `message`, `details`; `PreCompress`: `trigger` (`auto|manual`)
- **Output contract (stdout JSON; stdout must contain ONLY the JSON object — any plain text breaks parsing; stderr for logs):**
  - Common: `systemMessage`, `suppressOutput`, `continue` (false = kill agent loop), `stopReason`, `decision` (`allow`|`deny`, alias `block`), `reason`.
  - `hookSpecificOutput`: `additionalContext` (BeforeAgent/AfterTool/SessionStart), `tool_input` (BeforeTool arg override/merge), `tailToolCallRequest` (`{name, args}`, AfterTool), `llm_request`/`llm_response` (model hooks), `toolConfig.mode` (`AUTO|ANY|NONE`) + `toolConfig.allowedFunctionNames` (BeforeToolSelection), `clearContext` (AfterAgent).
  - **Exit codes:** `0` = success, stdout parsed as JSON (preferred even for intentional denies); `2` = system block, `stderr` becomes the rejection reason (semantics vary per event: blocks tool/turn/response but turn usually continues); any other code = non-fatal warning, execution proceeds.
- **Env for hook processes:** sanitized environment; `GEMINI_PROJECT_DIR`, `GEMINI_PLANS_DIR`, `GEMINI_SESSION_ID`, `GEMINI_CWD`, plus `CLAUDE_PROJECT_DIR` as a compatibility alias. Project hooks are fingerprinted and re-trusted on change.

## Subagents
- **Supported:** yes (behind `experimental.enableAgents`, on by default; disable via `{"experimental": {"enableAgents": false}}`). Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/core/subagents.md
- **Dirs:** project `.gemini/agents/*.md`, user `~/.gemini/agents/*.md`. Extensions can also bundle agents.
- **Format:** Markdown with YAML frontmatter (`---` fences, must start the file); body = the agent's system prompt.
- **Frontmatter keys:**
  - `name` (required) — slug used as tool name; **only lowercase letters, numbers, hyphens, underscores**
  - `description` (required) — shown to main agent for delegation decisions
  - `kind` — `local` (default) | `remote`
  - `tools` — array of tool names; wildcards `*`, `mcp_*`, `mcp_<server>_*`; omitted = inherit all parent tools
  - `mcpServers` — inline per-agent MCP server definitions
  - `model` — default `inherit`
  - `temperature` — 0.0–2.0, default 1
  - `max_turns` — default 30
  - `timeout_mins` — default 10
- Built-in subagents: `codebase_investigator`, `cli_help`, `generalist`, `browser_agent`. Runtime overrides via `settings.json` `agents.overrides` / `modelConfigs.overrides`; policy control via TOML `[[rules]]` with a `subagent` property.

## Commands
- **Supported:** custom slash commands. Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/custom-commands.md
- **Dirs:** user `~/.gemini/commands/`, project `<root>/.gemini/commands/`; project wins on name conflict.
- **Format:** **TOML files, `.toml` extension ("TOML file format (v1)") — NOT Markdown.** Fields: `prompt` (string, required), `description` (string, optional; generic one generated from filename if omitted).
- **Naming/namespacing:** command name = path relative to `commands/` minus `.toml`; subdirectories become namespaces with `:` separator (`commands/git/commit.toml` → `/git:commit`).
- **Templating:** `{{args}}` placeholder (shell-escaped inside `!{...}`); `!{shell command}` injects command output (confirmation-prompted, balanced-brace parsing); `@{path}` injects file content or recursive directory listing (processed first; multimodal-aware). Without `{{args}}`, the raw typed command line is appended to the prompt after two newlines. Reload with `/commands reload`.

## Rules
- **Not supported** as a Cursor-style rules directory — there is no `.gemini/rules/` or equivalent in any official doc. Instructional "rules" live in `GEMINI.md` context files (see Instructions).
- Closest distinct mechanism: the **Policy Engine** — TOML files with `[[rules]]` blocks (fields like `toolName`, `decision`/`action`, `commandPrefix`, `subagent`, `deny_message`) loaded from `~/.gemini/policies/` and additional `policyPaths` in settings (evidence: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/core/subagents.md and docs/reference/configuration.md). These are tool-permission rules, not prompt context.

## MCP
- **Config file:** `settings.json` (any layer: project `.gemini/settings.json`, user `~/.gemini/settings.json`, system) — **no standalone `.mcp.json` equivalent**. Source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/tools/mcp-server.md
- **Format:** JSON. Top-level `mcpServers` object keyed by server name; sibling `mcp` object for globals (`serverCommand`, `allowed`, `excluded`).
- **Server-table shape:** one of `command` (stdio) | `url` (SSE) | `httpUrl` (streamable HTTP) required. Optional: `args` (string[]), `env` (object), `cwd`, `timeout` (ms, default 600000), `trust` (bool, skip confirmations), `includeTools`/`excludeTools` (string[]; exclude wins), `headers` (for url/httpUrl), OAuth fields (`targetAudience`, `targetServiceAccount`, `authProviderType`).
- **Env handling:** values in `env` support `$VAR`, `${VAR}` (all platforms), `%VAR%` (Windows-only); unset vars expand to empty string. Broader `settings.json` strings additionally support `${VAR:-DEFAULT}`. **Sanitization:** host env vars matching sensitive patterns (`GEMINI_API_KEY`, `GOOGLE_API_KEY`, `*TOKEN*`, `*SECRET*`, `*PASSWORD*`, `*KEY*`, `*AUTH*`, `*CREDENTIAL*`, cert/key patterns) are redacted from the spawned server's inherited environment unless explicitly listed in `env`. MCP tool names surface as `mcp_<server>_<tool>` (relevant for hook matchers).

## Translation hazards
What a canonical `.agents` store must drop/rename/reformat to target Gemini CLI:

1. **Instructions file**
   - Rename canonical instructions to `GEMINI.md` (project root and/or `~/.gemini/GEMINI.md`); or set `context.fileName` in `settings.json` to read `AGENTS.md` — default reads only `GEMINI.md`.
   - **No heading convention to normalize** — any H1 (or none) is fine; agentlink's heading-equivalence regex is unnecessary for Gemini.
   - `@path` import lines are live directives — must preserve or inline them; unlike plain markdown links they pull file content into context.
   - Content is concatenated from many locations (global + every ancestor + JIT) — no single-file semantic equality; agentlink should compare per-file, not per-merged-context.

2. **Skills**
   - Layout is compatible: `.agents/skills/<name>/SKILL.md` is a **native alias** — zero path translation needed for skills (best-case of any agent).
   - **Drop/ignore frontmatter keys other than `name` and `description`**: `allowed-tools`, `license`, `metadata`, `compatibility`, etc. are parsed-then-discarded by the loader (inert, not errors). The skill normalizer can strip them when diffing against Gemini targets.
   - Skill `name` is sanitized (`: \ / < > * ? " |` → `-`); a canonical name containing those chars compares unequal to what Gemini actually registered — normalize by applying the same substitution.
   - Discovery is only `SKILL.md` or `*/SKILL.md` (one level deep); deeper nesting is invisible.
   - No documented size limits; soft guidance: body < 5k words.

3. **Hooks**
   - **Event-name mismatch**: canonical/Claude `PreToolUse`→`BeforeTool`, `PostToolUse`→`AfterTool`, `UserPromptSubmit`→`BeforeAgent`, `Stop`→`AfterAgent`, `PreCompact`→`PreCompress`. Gemini-only events with no Claude counterpart: `BeforeModel`, `AfterModel`, `BeforeToolSelection` — canonical hooks for these cannot round-trip to Claude-style peers.
   - **Config location/format**: hooks live inside JSON `settings.json` under `hooks` — not a standalone file; agentlink must merge into settings JSON rather than sync a hook file tree.
   - **Schema differs from Claude**: matcher groups add `sequential`; hook entries add `name`, `timeout` (ms), `description`; only `type: "command"` exists.
   - **Matcher semantics**: regex against **Gemini tool names** (`write_file`, `read_file`, `run_shell_command`, `mcp_<server>_<tool>`) — Claude tool names (`Write`, `Edit`, `Bash`) will never match; translation must rewrite tool-name matchers, not just event names.
   - **Envelope fields differ**: snake_case `session_id`/`cwd`/`hook_event_name`/`tool_name`/`tool_input` (Claude-compatible naming, but `transcript_path`, `timestamp`, `mcp_context`, `original_request_name`, `llm_request`/`llm_response`, `prompt_response`, `stop_hook_active` are Gemini-specific). agentlink `guard`/`remind` envelope parsing should key off `hook_event_name` + `tool_input.file_path` for Gemini tool events.
   - **Output contract differs**: `decision: "deny"` (alias `"block"`) + `reason`, `hookSpecificOutput.additionalContext`, exit code 2 = block with stderr-as-reason; stdout must be JSON-only. Claude's `permissionDecision`/`hookSpecificOutput.permissionDecisionReason` vocabulary does not apply.

4. **Subagents**
   - Dir `.gemini/agents/*.md` with md+frontmatter is structurally close to Claude, but **frontmatter keys differ**: `max_turns`, `timeout_mins`, `temperature`, `model: inherit`, `kind`, inline `mcpServers` are Gemini-specific; Claude keys (`tools` list semantics mostly portable). `name` must be lowercase `[a-z0-9_-]` — canonical names with uppercase/dots must be renamed.

5. **Commands**
   - **Format mismatch: TOML, not Markdown.** A canonical markdown slash-command (frontmatter + body) must be converted: body → `prompt` string, `description` → `description`; markdown frontmatter keys (e.g. `allowed-tools`, `argument-hint`) have no TOML v1 equivalent — drop them.
   - Argument placeholder is `{{args}}` (not `$ARGUMENTS`); shell injection `!{...}` and file injection `@{...}` are Gemini-specific and won't translate outbound.
   - Namespacing via subdirs with `:` separator is compatible in spirit with Claude's.

6. **Rules**
   - Nothing to translate to: no rules dir. Canonical rules must fold into `GEMINI.md`. Policy TOML (`[[rules]]` in `~/.gemini/policies/`) is a permission engine, not prompt content — keep it out of the instructions sync path.

7. **MCP**
   - MCP servers live in JSON `settings.json` → `mcpServers` (not Claude's `.mcp.json` file, not Codex's TOML `config.toml`). Agentlink's MCP comparator must extract the `mcpServers` sub-object from settings JSON and leave sibling settings untouched.
   - Transport keys: `command`+`args` (stdio), `url` (SSE), `httpUrl` (HTTP) — `url` vs `httpUrl` both exist and imply different transports; a canonical single `url`+`transport` pair must split correctly.
   - Extra keys to ignore on compare (Gemini-specific): `trust`, `timeout` (ms, default 600000), `includeTools`, `excludeTools`, `headers`, `cwd`, OAuth fields.
   - Env interpolation syntax `$VAR`/`${VAR}` (and `%VAR%` Windows) means literal values in the file are often references, not secrets — compare env key names only (matches agentlink's existing rule).

## Verified corrections (fact-checker pass)

- Docs site: was "https://google-gemini.github.io/gemini-cli/ (Jekyll mirror of the repo README + docs/)" → now "Official docs live at https://www.geminicli.com/docs/ (linked from the current README); google-gemini.github.io/gemini-cli is a STALE Jekyll mirror still advertising Gemini 2.5 Pro and 100 requests/day quotas — do not cite it as current" (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/README.md)
- npx install: was "npx https://github.com/google-gemini/gemini-cli" → now "npx @google/gemini-cli per the current README; the github-URL npx form appears only on the stale github.io mirror" (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/README.md)
- MCP config file: was "settings.json (any layer) — no standalone .mcp.json equivalent" → now "Correct for the CLI itself (mcpServers lives in settings.json), but mcp-server.md references an `mcp_config.json` used 'if configuring standard MCP clients or remote skills' for explicit env passthrough — the absolute claim needs that caveat" (https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/tools/mcp-server.md)
