# codex

## Identity

- Binary: `codex` (npm package `@openai/codex`; Homebrew cask `codex`; standalone installers `https://chatgpt.com/codex/install.sh` / `.ps1`). Repo: https://github.com/openai/codex (Rust, actively maintained).
- Vendor: OpenAI.
- Docs: developers.openai.com/codex — its pages serve canonical markdown twins at `https://learn.chatgpt.com/docs/<slug>.md`. Index: https://learn.chatgpt.com/llms.txt.
- URLs actually read: https://github.com/openai/codex/tree/main/docs (stubs redirecting to docs site), https://github.com/openai/codex/blob/main/docs/config.md, https://github.com/openai/codex/blob/main/docs/skills.md, https://github.com/openai/codex/blob/main/docs/agents_md.md, https://learn.chatgpt.com/docs/codex/cli.md, https://learn.chatgpt.com/docs/config-file/config-basic.md, https://learn.chatgpt.com/docs/agent-configuration/agents-md.md, https://learn.chatgpt.com/docs/build-skills.md, https://learn.chatgpt.com/docs/hooks.md, https://learn.chatgpt.com/docs/agent-configuration/subagents.md, https://learn.chatgpt.com/docs/agent-configuration/rules.md, https://learn.chatgpt.com/docs/custom-prompts.md, https://learn.chatgpt.com/docs/extend/mcp.md, https://agentskills.io/specification.

## Config dirs

- Project: `.codex/` at the project root (config, hooks, rules, agents). Loaded **only for trusted projects**; untrusted projects silently skip all project `.codex/` layers (https://learn.chatgpt.com/docs/config-file/config-basic.md).
- User/global: `~/.codex/`; relocated entirely by `CODEX_HOME` env var (https://learn.chatgpt.com/docs/agent-configuration/agents-md.md).
- System: `/etc/codex/` (`/etc/codex/config.toml`, `/etc/codex/skills`).
- Profiles: `~/.codex/<profile-name>.config.toml` selected via `--profile`.
- Precedence (highest first): CLI flags / `-c` overrides → project `.codex/config.toml` files ordered project-root→cwd (closest wins) → profile file → `~/.codex/config.toml` → `/etc/codex/config.toml` → built-in defaults. Admins can additionally enforce `requirements.toml` (managed layer; can pin `[features].hooks`, `allow_managed_hooks_only = true`) (https://learn.chatgpt.com/docs/config-file/config-basic.md, https://github.com/openai/codex/blob/main/docs/config.md).

## Instructions file

- Filenames: `AGENTS.md` and `AGENTS.override.md` (override wins within the same directory; only one file per directory is used). Extra names via `project_doc_fallback_filenames` (e.g. `["TEAM_GUIDE.md", ".agents.md"]`) (https://learn.chatgpt.com/docs/agent-configuration/agents-md.md).
- Discovery: global scope reads `$CODEX_HOME/AGENTS.override.md` else `$CODEX_HOME/AGENTS.md`. Project scope walks from project root (git root) **down to cwd**, checking `AGENTS.override.md` → `AGENTS.md` → fallback names per directory. Files are concatenated root→cwd joined by blank lines; later (closer) files win by appearing later in the prompt.
- Limits: empty files skipped; combined cap `project_doc_max_bytes` (default 32 KiB).
- Heading conventions: no required H1; docs examples use `# AGENTS.md` plus `##` sections. A `## Code Review Rules` section is a recognized convention consumed by Codex code review.
- Import/include syntax: none documented; files are plain concatenated markdown.

## Skills

Supported. Codex follows the open agent skills standard (https://agentskills.io/specification).

- Dirs (NOT under `.codex/`): `$CWD/.agents/skills`, each ancestor `.agents/skills` up to `$REPO_ROOT/.agents/skills` (repo scope); `$HOME/.agents/skills` (user); `/etc/codex/skills` (admin); bundled system skills. Symlinked skill folders are followed. Duplicate `name`s are not merged (https://learn.chatgpt.com/docs/build-skills.md).
- Layout: `<skill-name>/SKILL.md` (required) + optional `scripts/`, `references/`, `assets/`, and Codex-specific `agents/openai.yaml`.
- SKILL.md frontmatter (YAML): `name` (required; 1–64 chars, lowercase alphanumerics + hyphens, no leading/trailing/consecutive hyphens, must match parent dir name), `description` (required; 1–1024 chars; drives implicit matching), optional `license`, `compatibility` (≤500 chars), `metadata` (string→string map), `allowed-tools` (space-separated, experimental, support varies) (https://agentskills.io/specification).
- Codex-unique: `agents/openai.yaml` with `interface.*` (display_name, short_description, icons, brand_color, default_prompt), `policy.allow_implicit_invocation` (bool), `dependencies.tools[]` (mcp tool declarations); `[[skills.config]]` entries in `config.toml` (`path` to SKILL.md, `enabled = false`) to disable by path.
- Size limits: initial skills list (name+description+path) capped at ≤2% of context window or 8,000 chars if unknown; SKILL.md recommended <500 lines/<5000 tokens per spec.
- Invocation: `$skill-name` mention, `/skills` picker, or implicit via description.

## Hooks

Supported (feature flag `[features].hooks = true`, default on; legacy alias `codex_hooks`) (https://learn.chatgpt.com/docs/hooks.md).

- Config files: `hooks.json` **or** inline `[hooks]` TOML in `config.toml`, discovered at `~/.codex/`, `<repo>/.codex/` (trusted only), plugin bundles (`hooks/hooks.json` or manifest `hooks` entry in `.codex-plugin/plugin.json`), and managed `requirements.toml`. All matching hooks from all layers run; layers don't replace each other.
- Events: `SessionStart`, `SessionEnd`, `SubagentStart`, `SubagentStop`, `PreToolUse`, `PermissionRequest`, `PostToolUse`, `PreCompact`, `PostCompact`, `UserPromptSubmit`, `Stop`.
- Matcher schema: per-event array of groups `{ "matcher": "<regex>", "hooks": [ ...handlers ] }`; `matcher` is a regex (string, `"*"`/empty/omitted = match all). Matched field per event: `SessionStart`→`source` (`startup|resume|clear|compact`), `SessionEnd`→`reason` (only `other`), `PreToolUse`/`PostToolUse`/`PermissionRequest`→tool name (`Bash`, `apply_patch` with aliases `Edit`/`Write`, `mcp__<server>__<tool>`, other function tool names, `Agent` for spawn_agent), `PreCompact`/`PostCompact`→`trigger` (`manual|auto`), `SubagentStart`/`SubagentStop`→`agent_type`; `UserPromptSubmit`/`Stop` ignore matcher.
- Command schema: only `type: "command"` runs (`prompt`/`agent` parsed, skipped). Fields: `command` (string), `timeout` (sec; default 600, `SessionEnd` default 1 / max 3), `statusMessage`, `additionalContextLimit` (approx token threshold for `additionalContext`, default 2500, 0 = unlimited), `commandWindows`/`command_windows` (Windows override); `async` parsed but unsupported. cwd = session cwd.
- stdin JSON envelope (one object): `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model` (Codex-specific), plus per-event: `turn_id` (Codex-specific), `permission_mode` (`default|acceptEdits|plan|dontAsk|bypassPermissions`), `tool_name`, `tool_use_id`, `tool_input` (Bash/apply_patch use `tool_input.command`), `tool_response` (PostToolUse), `prompt` (UserPromptSubmit), `source`, `reason`, `trigger`, `agent_id`, `agent_type`, `agent_transcript_path`, `stop_hook_active`, `last_assistant_message`.
- Output contract: exit 0 + no output = success. JSON on stdout: shared `{continue, stopReason, systemMessage, suppressOutput}` (suppressOutput parsed, unimplemented; `continue`/`stopReason` unsupported for PreToolUse/PermissionRequest — emitting them marks the run failed). `hookSpecificOutput: {hookEventName, additionalContext}` injects developer context (SessionStart, SubagentStart, UserPromptSubmit, PreToolUse, PostToolUse). PreToolUse: `hookSpecificOutput.permissionDecision: "deny"|"allow"` + optional `updatedInput` (`"ask"` unsupported/fails); legacy `{decision:"block", reason}` accepted. PermissionRequest: `hookSpecificOutput.decision.behavior: "allow"|"deny"` (+`message`); any deny wins. Stop/SubagentStop: `decision:"block"`+`reason` = continue with reason as new prompt; plain text invalid for these two. Exit code 2 + stderr = block/feedback. Non-managed hooks must be reviewed and trusted (hash-pinned) via `/hooks` before they run; `--dangerously-bypass-hook-trust` skips this.
- Full generated schemas: https://github.com/openai/codex/tree/main/codex-rs/hooks/schema/generated.

## Subagents

Supported. Custom agents are standalone **TOML** files: `~/.codex/agents/*.toml` (user) and `.codex/agents/*.toml` (project) (https://learn.chatgpt.com/docs/agent-configuration/subagents.md).

- Required keys: `name`, `description`, `developer_instructions` (string; the agent's system-level instructions — there is no markdown body). `name` is the identity; filename need not match. A custom name matching a built-in (`default`, `worker`, `explorer`) takes precedence.
- Optional keys: any config.toml keys, e.g. `model`, `model_reasoning_effort`, `sandbox_mode`, `[mcp_servers.*]`, `[[skills.config]]`. Omitted settings inherit from the parent session.
- Global knobs in `config.toml` `[agents]`: `enabled`, `max_concurrent_threads_per_session` (legacy alias `max_threads`), `default_subagent_model`, `default_subagent_reasoning_effort`, `interrupt_message`. CLI control: `/agent`.

## Commands

- Custom slash commands = "custom prompts": `~/.codex/prompts/*.md` (top-level only; user-level only, not project-shareable). **Deprecated** in favor of skills (https://learn.chatgpt.com/docs/custom-prompts.md).
- Frontmatter: `description` (shown in menu), `argument-hint` (e.g. `[FILES=<paths>]`). Body placeholders: `$1`–`$9`, `$ARGUMENTS`, named `$UPPERCASE` via `KEY=value`, `$$` for literal `$`. Invoked as `/prompts:<name>`.
- Skills invoked with `$name` are the supported replacement for reusable prompts.

## Rules

Supported, but **not** prompt-style rules: Codex "rules" are sandbox-escalation exec policy (experimental) (https://learn.chatgpt.com/docs/agent-configuration/rules.md).

- Dir/format: `*.rules` files under `rules/` beside each config layer (`~/.codex/rules/`, `<repo>/.codex/rules/` trusted-only); TUI allow-list writes `~/.codex/rules/default.rules`.
- Language: Starlark; `prefix_rule(pattern=[...], decision="allow|prompt|forbidden", justification=..., match=[...], not_match=[...])`. Most restrictive decision wins; shell chains split via tree-sitter when safe. Test with `codex execpolicy check`.

## MCP

- File: `config.toml` — `~/.codex/config.toml` or project `.codex/config.toml` (trusted only). CLI: `codex mcp add <name> --env K=V -- <cmd>`, `codex mcp list/login` (https://learn.chatgpt.com/docs/extend/mcp.md).
- Shape: `[mcp_servers.<name>]` tables. Stdio: `command` (req), `args`, `env` (inline table of literal values), `env_vars` (names to forward, or `{name, source="local"|"remote"}`), `cwd`, `experimental_environment`. Streamable HTTP: `url` (req), `auth` (`oauth` default | `chatgpt`), `bearer_token_env_var`, `http_headers` (static map), `env_http_headers` (header→env-var-name map). Common: `startup_timeout_sec` (10), `tool_timeout_sec` (60), `enabled`, `required`, `enabled_tools`, `disabled_tools`, `default_tools_approval_mode` (`auto|prompt|writes|approve`), `[mcp_servers.<name>.tools.<tool>].approval_mode`. OAuth callbacks: top-level `mcp_oauth_callback_port`/`mcp_oauth_callback_url`.
- Env handling: values can be literals (`env`), forwarded by name (`env_vars`), or referenced by env-var name for HTTP auth/headers — no `.env` file, no `${VAR}` expansion documented. MCP tool names surface as `mcp__<server>__<tool>`.

## Translation hazards

Concrete drops/renames/reformats a canonical `.agents` store must apply to target codex:

1. **MCP**: canonical JSON server blocks (Claude `.mcp.json`-style) must be re-emitted as TOML `[mcp_servers.<name>]` in `config.toml`. Keep `command`/`args`/`url`; Claude's `type: "sse"` has no equivalent (only stdio + streamable HTTP); `env` maps to the `env` inline table, but header auth must become `bearer_token_env_var` / `env_http_headers` / `http_headers`; drop Claude-only fields (`headers` JSON style, `timeout` ms units — Codex uses `startup_timeout_sec`/`tool_timeout_sec` in seconds). Project-scoped MCP requires project trust or it is silently ignored.
2. **Instructions**: canonical instructions must be written to `AGENTS.md` (never `CLAUDE.md` etc. unless added to `project_doc_fallback_filenames`). No import/include syntax — inline all `@import`-style references before writing. Neutralize tool-specific H1s (agentlink's heading normalizer already equates `# Claude Code instructions` / `# Codex instructions`). Never write `AGENTS.override.md` from a canonical store (it would shadow `AGENTS.md`). Respect the 32 KiB combined cap (`project_doc_max_bytes`). A `## Code Review Rules` section has special semantics — preserve it verbatim if present.
3. **Skills**: dir is `.agents/skills` (repo) / `~/.agents/skills` (user) — notably *not* under `.codex/`. File must be `SKILL.md` with only spec keys `name`, `description`, `license`, `compatibility`, `metadata`, `allowed-tools`; drop any other tool-specific frontmatter keys (e.g. Claude plugin keys). Enforce `name` rules: ≤64 chars, lowercase `[a-z0-9-]`, no leading/trailing/double hyphens, must equal directory name; `description` ≤1024 chars. `allowed-tools` is experimental — don't rely on Claude-style `Bash(git:*)` scoping working. Codex-only `agents/openai.yaml` (interface/policy/dependencies) has no canonical counterpart — drop on export to other agents; generate only if targeting Codex UI. Per-skill disable is by absolute path in `[[skills.config]]`, not frontmatter.
4. **Hooks**: emit `hooks.json` (JSON) or `[hooks]` TOML — YAML canonical must be converted to the nested `{Event: [{matcher, hooks:[{type:"command", command, timeout, statusMessage, additionalContextLimit, commandWindows}]}]}` shape. Event-name mismatches vs Claude: Codex adds `PermissionRequest`, `PreCompact`/`PostCompact`, `SubagentStart`/`SubagentStop`; `SessionStart` matcher gains `compact`; `Notification` event does not exist (drop). Tool-name remap: file-edit matchers must target `apply_patch` (aliases `Edit`/`Write`), shell is `Bash`, agents match `Agent`. Output contract pitfalls: `permissionDecision: "ask"` and `updatedPermissions` **fail the hook run** (not ignored) for PreToolUse/PermissionRequest — strip them; `continue:false` unsupported on PreToolUse/PermissionRequest; `suppressOutput` unimplemented; PermissionRequest decisions use `decision.behavior` (not `permissionDecision`). stdin envelope differs: adds `model`, `turn_id`; `permission_mode` values are Codex-specific (`dontAsk`, `bypassPermissions`). SessionEnd timeout default 1s/max 3s. Trust model: synced hook files are **skipped until interactively trusted** (hash-pinned) — any byte change re-triggers review; automation needs `--dangerously-bypass-hook-trust` or managed `requirements.toml`.
5. **Subagents**: convert markdown+frontmatter agent files (Claude `.claude/agents/*.md`) to TOML `.codex/agents/*.toml`: frontmatter `description` → `description`, body → `developer_instructions` string, `name` → `name` (identity is the key, not filename — sanitize to a spawn-safe token). Drop Claude-only frontmatter (`tools` list — Codex has no per-agent tool allowlist; approximate with `sandbox_mode = "read-only"`), rename `model` values to Codex model slugs and `model_reasoning_effort` levels (`low|medium|high|xhigh|max|ultra`).
6. **Commands**: `~/.codex/prompts/*.md` is deprecated and user-only — translate canonical slash commands to **skills** (`.agents/skills/<name>/SKILL.md`) instead. If emitted as prompts: frontmatter limited to `description` + `argument-hint` (drop `allowed-tools`, `model`, etc.), convert argument syntax to `$1..$9`/`$ARGUMENTS`/`$NAME`, escape `$` as `$$`, and invoke as `/prompts:<name>`.
7. **Rules**: `.rules` files are Starlark exec policy (`prefix_rule`), not natural-language rules — never translate markdown rule files into `.rules`; keep prompt-style rules in AGENTS.md.
8. **Global**: `CODEX_HOME` relocates the entire user dir (`~/.codex` assumptions break); project layers require trust, so freshly synced `.codex/` content may not load until the user trusts the project.

## Verified corrections (fact-checker pass)

- Hazard #4 event-name mismatch: 'Codex adds PermissionRequest, PreCompact/PostCompact, SubagentStart/SubagentStop' → Claude Code also has all of these events (PermissionRequest, PreCompact, PostCompact, SubagentStart, SubagentStop all appear in Claude's hook event table); none are Codex-unique — only the 'Notification does not exist in Codex (drop)' part holds (https://code.claude.com/docs/en/hooks)
- Hazard #4: 'SessionStart matcher gains `compact`' → Claude's SessionStart source values are also `startup`, `resume`, `clear`, `compact` (plus `fork`), so `compact` is not a Codex addition; if anything Codex lacks Claude's `fork` source (https://code.claude.com/docs/en/hooks)
- Hazard #4: '`permission_mode` values are Codex-specific (`dontAsk`, `bypassPermissions`)' → Claude hook `permission_mode` uses the same values: `default`, `plan`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions` (Claude additionally has `auto`); these values are shared, not Codex-specific (https://code.claude.com/docs/en/hooks)
