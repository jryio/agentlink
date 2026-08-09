# pi

## Identity
- **Binary:** `pi`. Install: `npm install -g --ignore-scripts @earendil-works/pi-coding-agent` or `curl -fsSL https://pi.dev/install.sh | sh` (https://github.com/earendil-works/pi/blob/main/packages/coding-agent/README.md, https://pi.dev/docs/latest).
- **Vendor:** Mario Zechner (badlogic), now under the `earendil-works` GitHub org / Earendil Inc. **Renames to note:** repo `github.com/badlogic/pi-mono` now redirects to `github.com/earendil-works/pi`; npm package renamed `@mariozechner/pi-coding-agent` → `@earendil-works/pi-coding-agent` (README badge + install command). Actively maintained (repo docs commits days old).
- **Docs:** https://pi.dev/docs/latest (rendered mirror of the repo's `packages/coding-agent/docs/*.md`; each page links back to the GitHub source). Canonical doc files: README.md, docs/skills.md, docs/extensions.md, docs/settings.md, docs/prompt-templates.md in the repo.
- Philosophy (README "Philosophy" section): deliberately **no MCP, no sub-agents, no permission popups, no plan mode, no built-in to-dos** — all expected to be built as TypeScript extensions or third-party pi packages.

## Config dirs
- **Global/user dir:** `~/.pi/agent/` — holds `settings.json`, `AGENTS.md`, `SYSTEM.md`/`APPEND_SYSTEM.md`, `skills/`, `prompts/`, `extensions/`, `themes/`, `keybindings.json`, `models.json`, `sessions/`, `trust.json`, and package stores `git/` + `npm/`. Overridable via env `PI_CODING_AGENT_DIR` (README CLI Reference → Environment Variables).
- **Project dir:** `.pi/` — holds `settings.json`, `skills/`, `prompts/`, `extensions/`, `themes/`, `SYSTEM.md`/`APPEND_SYSTEM.md`, and project-local package stores `.pi/git/`, `.pi/npm/` (`pi install -l`).
- **Precedence:** `.pi/settings.json` (project) overrides `~/.pi/agent/settings.json` (global); nested objects deep-merge (docs/settings.md "Project Overrides"). Session dir precedence: `--session-dir` > `PI_CODING_AGENT_SESSION_DIR` > `sessionDir` setting.
- **Discovery gate — project trust:** project-local `.pi/*` resources and project `.agents/skills` load **only after the project is trusted** (interactive prompt, persisted in `~/.pi/agent/trust.json`; `defaultProjectTrust` setting `ask|always|never`; `--approve`/`--no-approve` per-run). Non-interactive modes never prompt (README "Project Trust", docs/settings.md).
- Resource paths can also be injected via `settings.json` arrays `packages`, `extensions`, `skills`, `prompts`, `themes` (glob + `!exclude` + `+/-path` syntax supported).

## Instructions file
- **Filenames:** `AGENTS.md` **or** `CLAUDE.md` (either is read). `AGENTS.override.md` in a directory **replaces** that directory's `AGENTS.md`/`CLAUDE.md` (README "Context Files").
- **Locations:** `~/.pi/agent/AGENTS.md` (global), then every parent directory walking up from cwd, then cwd. **All matching files are concatenated** — there is no single-winner precedence.
- **Heading/title convention:** none documented; no required `# Title` heading (contrast with agentlink's Claude/Codex heading normalizer — nothing to normalize for pi).
- **Import/include syntax:** none documented.
- System-prompt files (adjacent concept): `.pi/SYSTEM.md` / `~/.pi/agent/SYSTEM.md` replace the default system prompt; `APPEND_SYSTEM.md` appends. Disable context files with `--no-context-files`/`-nc`.

## Skills
**Supported.** Implements the Agent Skills standard (agentskills.io) "warning about most violations but remaining lenient" (docs/skills.md).
- **Dirs:** global `~/.pi/agent/skills/` and `~/.agents/skills/`; project `.pi/skills/` and `.agents/skills/` discovered in cwd **and all ancestors** up to git repo root (filesystem root outside a repo) — project locations require trust. Also `skills` array in settings, `--skill <path>` CLI, and package `skills/` dirs / `pi.skills` package.json entries.
- **Layout:** directory containing `SKILL.md` (required; frontmatter + body), discovered recursively; freeform `scripts/`, `references/`, `assets/` alongside. Exception: in `~/.pi/agent/skills/` and `.pi/skills/` only, bare root-level `.md` files are also discovered as individual skills; root `.md` files in `.agents/skills/` are ignored.
- **Frontmatter keys** (docs/skills.md "Frontmatter"):
  | key | required | meaning |
  |---|---|---|
  | `name` | yes | ≤64 chars, lowercase `a-z0-9-`, no leading/trailing/consecutive hyphens. **Pi deviation: name need NOT match parent dir** (standard requires it; pi explicitly rejects that rule for shared cross-harness skill dirs). |
  | `description` | yes | ≤1024 chars; **missing description ⇒ skill is not loaded** (only hard failure). |
  | `license` | no | license name or bundled-file reference |
  | `compatibility` | no | ≤500 chars, environment requirements |
  | `metadata` | no | arbitrary key-value map |
  | `allowed-tools` | no | space-delimited pre-approved tools (**experimental**) |
  | `disable-model-invocation` | no | `true` hides skill from system prompt; only `/skill:name` invokes (**pi-unique key**) |
- Unknown frontmatter fields are **ignored** (safe to leave foreign keys). Most violations warn-but-load. Name collisions across locations: warn, first found wins.
- Invocation: auto via description match (agent `read`s the SKILL.md), or `/skill:name [args]` slash command (toggle `enableSkillCommands`, default `true`).
- Cross-harness: docs officially suggest pointing `skills` setting at `~/.claude/skills` / `~/.codex/skills` (docs/skills.md "Using Skills from Other Harnesses") — evidence pi's skill format is Claude-compatible by design.

## Hooks
**Not supported as config-file hooks.** There is no hooks key in `settings.json` (full key list in docs/settings.md contains none), no shell-command matcher schema, no stdin JSON envelope, no exit-code output contract. The entire hook surface is **in-process TypeScript extension events** via `pi.on(event, handler)` in `.pi/extensions/` or `~/.pi/agent/extensions/` (docs/extensions.md).
- **Event names (snake_case, not Claude's PascalCase):** `project_trust`, `resources_discover`, `session_start`, `session_info_changed`, `session_before_switch`, `session_before_fork`, `session_before_compact`, `session_compact`, `session_before_tree`, `session_tree`, `session_shutdown`, `before_agent_start`, `agent_start`, `agent_end`, `agent_settled`, `turn_start`, `turn_end`, `message_start`, `message_update`, `message_end`, `context`, `before_provider_headers`, `before_provider_request`, `after_provider_response`, `model_select`, `thinking_level_select`, `tool_execution_start`, `tool_call`, `tool_execution_update`, `tool_result`, `tool_execution_end`, `user_bash`, `input`.
- **Closest Claude analogs:** PreToolUse ≈ `tool_call` (payload `{toolName, toolCallId, input}`; `input` mutable; return `{block: true, reason?, terminate?}` to block); PostToolUse ≈ `tool_result` (return partial patch `{content?, details?, isError?, usage?}`; chained in load order); UserPromptSubmit ≈ `before_agent_start` (return `{message?, systemPrompt?}`); SessionStart ≈ `session_start` (`event.reason`: `startup|reload|new|resume|fork`).
- **"Command" schema:** handlers are async TS functions, not shell commands; interaction via `ctx.ui.confirm/notify/...`. No external process is spawned per event.

## Subagents
**Not supported.** "Pi ships with powerful defaults but skips features like sub agents and plan mode"; "**No sub-agents.** … Spawn pi instances via tmux, or build your own with extensions" (README Philosophy). No `.pi/agents/` dir, no subagent file format or frontmatter exists. Available only via third-party pi packages/extensions.

## Commands
**Supported as "prompt templates"** (pi's slash-command equivalent; docs/prompt-templates.md).
- **Dirs:** global `~/.pi/agent/prompts/*.md`; project `.pi/prompts/*.md` (trust-gated); package `prompts/`; `prompts` settings array; `--prompt-template <path>`. **Discovery is non-recursive.**
- **Format:** Markdown file; filename minus `.md` becomes the command (`review.md` → `/review`). Optional YAML frontmatter keys: `description` (falls back to first non-empty line), `argument-hint` (e.g. `"<PR-URL>"`, shown in autocomplete). No other frontmatter documented.
- **Argument substitution:** `$1`, `$2`…, `$@`/`$ARGUMENTS`, `${1:-default}`, `${@:-default}`, `${@:N}`, `${@:N:L}`.
- Skills also surface as `/skill:name`; extensions register arbitrary `/commands` via `pi.registerCommand()`.

## Rules
**Not supported** as a distinct directory/format. Project rules live in the instructions files (`AGENTS.md`/`CLAUDE.md`, concatenated from global + ancestors + cwd — README "Context Files"). No `.cursor/rules`-style equivalent exists.

## MCP
**Not supported.** "**No MCP.** Build CLI tools with READMEs (see Skills), or build an extension that adds MCP support" (README Philosophy, linking https://mariozechner.at/posts/2025-11-02-what-if-you-dont-need-mcp/). There is no MCP config file, no server table, no env handling for MCP in core; docs list no mcp page. MCP is achievable only through third-party extensions (extensions.md lists "MCP server integration" as an example use case). agentlink's MCP pair/sync feature has **no pi endpoint** to target.

## Translation hazards
Concrete drops/renames/reformats when targeting pi from a canonical `.agents` store:
1. **MCP blocks: drop entirely.** No config file exists; warn user that MCP servers are untranslatable to pi core.
2. **Hooks: cannot be translated.** Claude-style hooks (settings JSON + `PreToolUse`/`PostToolUse` matchers + shell command + stdin JSON envelope with `tool_input.file_path` etc.) have no declarative equivalent. Closest target is a generated `.pi/extensions/*.ts` file subscribing to snake_case events (`tool_call` ≈ PreToolUse, `tool_result` ≈ PostToolUse, `before_agent_start` ≈ UserPromptSubmit, `session_start` ≈ SessionStart) — a code-generation task, not config mapping. `agentlink guard`/`remind`'s hook-envelope parsing is irrelevant for pi; pi never shells out to hook commands.
3. **Subagents: drop.** No agents dir/format; `.agents/agents/*` content has no pi destination.
4. **Rules dirs: fold into instructions.** Canonical rules must be merged into `AGENTS.md` (pi concatenates AGENTS.md files; no per-rule files). Note pi also accepts `CLAUDE.md` and the `AGENTS.override.md` replacement semantic — agentlink must not write both `AGENTS.md` and `AGENTS.override.md` in the same dir.
5. **Instructions heading normalizer: no-op for pi** — pi has no required `# <Tool> instructions` heading convention; writing a neutral or pi-titled heading is safe.
6. **Skills — mostly drop-in, with caveats:**
   - Layout identical to Claude (`<name>/SKILL.md`), and pi natively reads `.agents/skills/` (project, ancestor-walking) and `~/.agents/skills/` — the canonical store path is a first-class pi location, so `.agents/skills` → pi needs **no symlink** for skills (unlike `.claude/skills`).
   - `allowed-tools` is supported (space-delimited string; experimental) — keep, but note format is a **string, not a YAML array** per the agentskills.io spec pi follows.
   - Claude-only keys (`license`, `metadata`, `compatibility`) are all valid pi keys too — **do not strip** them for pi (they're ignored-or-used, never errors; unknown fields are ignored anyway).
   - Pi-unique key `disable-model-invocation` has no Claude/Codex analog — namespace it or drop it when syncing pi → canonical.
   - Enforce limits when generating: `name` ≤64 chars lowercase-hyphen (pi warns but loads), `description` **required** (missing ⇒ skill silently not loaded), ≤1024 chars.
   - Skill `name` need not equal directory name in pi (unlike the strict standard) — safe to keep canonical dir names.
   - Root-level loose `.md` files placed in `.agents/skills/` are **ignored** by pi — every canonical skill must be a `SKILL.md`-bearing directory.
   - Name collisions across locations keep the **first** found — precedence ordering matters if agentlink materializes duplicates.
7. **Commands → prompt templates:** rename canonical slash-command frontmatter to pi's minimal set: keep only `description` and `argument-hint`; drop Claude-specific keys (`allowed-tools`, `model`, etc.). Convert argument placeholders to pi's `$1`/`$@`/`${1:-default}` syntax if canonical uses a different one. Flatten any nested command dirs — discovery is non-recursive. Target dirs: `.pi/prompts/` or `~/.pi/agent/prompts/` (note: **not** `.agents/`).
8. **Config file format:** everything is JSON (`settings.json`, `keybindings.json`, `models.json`, `trust.json`) — no TOML/YAML config. Project-level settings in `.pi/settings.json` deep-merge over global.
9. **Trust gate:** anything agentlink symlinks/writes under `.pi/` (or project `.agents/skills`) is inert until the user trusts the project (`/trust` or `defaultProjectTrust: "always"`) — document this; non-interactive runs silently skip untrusted project resources.
10. **Env-var dir override:** if agentlink manages the global dir, respect `PI_CODING_AGENT_DIR` (default `~/.pi/agent`).
11. **Adopt mapping note:** pi's native support for `.agents/skills` means the generic adopt rule `.pi/skills → .agents/skills` is optional for skills (both work); but `.pi/prompts`, `.pi/extensions`, `.pi/settings.json` have no `.agents` counterpart and would map to `.agents/pi/*` per the existing fallback rule.

## Verified corrections (fact-checker pass)

- was -> MCP is achievable only through third-party extensions (extensions.md lists "MCP server integration" as an example use case) -> now -> "MCP server integration" appears as an example extension use case in the README's Extensions section ("What's possible"), NOT in docs/extensions.md, which contains zero occurrences of "MCP"; substance of the claim (no core MCP, extension-only path) is unchanged (https://github.com/earendil-works/pi/blob/main/packages/coding-agent/README.md#extensions; https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/extensions.md)
