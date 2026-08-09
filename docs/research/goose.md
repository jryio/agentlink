# goose

## Identity
- **Binary**: `goose` (CLI); separate desktop app "Goose". Written in Rust (`crates/goose-cli`) (https://raw.githubusercontent.com/aaif-goose/goose/main/crates/goose/src/hooks/mod.rs)
- **Vendor**: originally Block; project moved to the **Agentic AI Foundation (AAIF)** in April 2026. GitHub `block/goose` now redirects to **`github.com/aaif-goose/goose`** (observed redirect; announcement banner on every docs page: "goose has moved to the Agentic AI Foundation (AAIF)" — https://goose-docs.ai/docs/guides/config-files)
- **Install** (https://goose-docs.ai/docs/getting-started/installation):
  - CLI: `brew install block-goose-cli` (Homebrew formula `block-goose-cli`), or `curl -fsSL https://github.com/aaif-goose/goose/releases/download/stable/download_cli.sh | bash`, or PowerShell `download_cli.ps1` on Windows; `goose update` self-updates
  - Desktop: `brew install --cask block-goose`, or zips from GitHub releases (`Goose.zip` / `Goose_intel_mac.zip`)
- **Docs**: **`https://goose-docs.ai`** — the old `block.github.io/goose` now serves only a meta-refresh redirect to goose-docs.ai, and its deep links 404 (observed raw HTML at https://block.github.io/goose/index.html). Blog continues through 2026-07-30; repo shows 5k+ commits — actively maintained.
- **URLs actually read**: the two above plus `https://goose-docs.ai/sitemap.xml`, `/docs/guides/config-files`, `/docs/guides/context-engineering/using-goosehints`, `/docs/guides/context-engineering/using-skills`, `/docs/guides/context-engineering/hooks`, `/docs/guides/context-engineering/subagents`, `/docs/guides/context-engineering/slash-commands`, `/docs/guides/context-engineering/custom-agents`, `/docs/guides/context-engineering/using-persistent-instructions`, `/docs/getting-started/installation`, `/docs/getting-started/using-extensions`, `/docs/mcp/skills-mcp`, and source files `crates/goose/src/skills/mod.rs`, `crates/goose/src/hooks/mod.rs`, `crates/goose/src/sources.rs` on `aaif-goose/goose@main`.

## Config dirs
- **Global/user dir**: `~/.config/goose/` on macOS/Linux; `%APPDATA%\Block\goose\config\` on Windows (https://goose-docs.ai/docs/guides/config-files). Contents: `config.yaml` (settings + extensions), `permission.yaml`, `secrets.yaml` (plaintext secrets when keyring unavailable), `permissions/tool_permissions.json` (auto-managed), `prompts/` (prompt templates), `settings.json` (e.g. `disabledPlugins`).
- **Project dir**: there is **no** `.goose/` project config dir for config.yaml. Project-scoped state is: `<project>/.config/goose/settings.json` (project-level `disabledPlugins`, per hooks doc) and `<project>/.agents/{skills,agents,plugins}/` (shared cross-agent convention, see below). Legacy dirs `.goose/skills`, `.goose/agents` are still discovered for backward compatibility (https://goose-docs.ai/docs/guides/context-engineering/using-skills).
- **Precedence**: environment variables > `config.yaml` > defaults (https://goose-docs.ai/docs/guides/config-files §Configuration Priority). Hints: local `.goosehints` wins over global on conflict (https://goose-docs.ai/docs/guides/context-engineering/using-goosehints).

## Instructions file
- **Filenames**: `.goosehints` and `AGENTS.md`, both looked for by default; default list is `["AGENTS.md", ".goosehints"]` ("goose looks for `AGENTS.md` then `.goosehints`"). Customizable via env var `CONTEXT_FILE_NAMES` (JSON array, e.g. `["CLAUDE.md", ".goosehints"]`) (https://goose-docs.ai/docs/guides/context-engineering/using-goosehints).
- **Locations**: project-local at working dir and **hierarchically nested** up to repo root; nested files auto-load when goose touches files in subdirectories. Global: `~/.config/goose/.goosehints`. Requires the Developer extension enabled.
- **Heading/title conventions**: none — free-form natural-language text/markdown; no required H1 or title.
- **Import/include syntax**: `@filename.md` or `@relative/path` inside hints **inlines file content into context immediately**; a plain path reference (no `@`) is a lazy pointer goose reads on demand.
- Separate mechanism: **persistent instructions** via `GOOSE_MOIM_MESSAGE_TEXT` / `GOOSE_MOIM_MESSAGE_FILE` env vars, re-injected every turn, capped at 64 KB (https://goose-docs.ai/docs/guides/context-engineering/using-persistent-instructions).

## Skills
**Supported** via a built-in platform extension ("Skills", enabled by default; in v1.25.0+ the standalone extension is deprecated in favor of the "Summon" extension — https://goose-docs.ai/docs/mcp/skills-mcp).
- **Dirs** (https://goose-docs.ai/docs/guides/context-engineering/using-skills):
  - Project: `<project>/.agents/skills/<name>/SKILL.md`
  - Global: `~/.agents/skills/<name>/SKILL.md`
  - Plugin-provided: `~/.agents/plugins/<plugin-name>/...` (namespaced `plugin:skill`)
  - Backward-compat discovery: `.goose/skills/`, `.claude/skills/`, `~/.claude/skills/`, `~/.config/agents/skills/`, and `~/.config/goose/skills` (docs + `inferred_discoverable_skill_root` in `crates/goose/src/skills/mod.rs`)
- **Layout**: directory per skill, mandatory `SKILL.md`, plus arbitrary supporting files (scripts, templates) in the same dir.
- **Frontmatter keys** (docs say `name` and `description` required; source `SkillFrontmatter` shows `name: Option<String>` (falls back to directory name), `description: String` (defaults to `""`), and `metadata: HashMap<String, Value>` — a free-form nested map per the agentskills.io spec; unknown extra keys are also tolerated and flattened into `properties`, e.g. `argument-hint` and `arguments` are read from properties in `skills/mod.rs`):
  - `name` — skill identifier used for listing/loading
  - `description` — shown in session instructions for relevance matching
  - `metadata` — goose/agentskills-specific free-form map (unique vs. Claude, which uses top-level `license`/`allowed-tools`/`metadata`)
- **Naming/size limits** (source `validate_skill_name`): name ≤ 64 chars; only lowercase ASCII letters, digits, hyphens; must not start or end with `-`. No documented size limit on SKILL.md.
- Invocation: auto-matched, `goose skills list`, or in-session `/skills <name>...`. Advertised as compatible with Claude Agent Skills (agentskills.io).

## Hooks
**Supported** (added 2026; blog announcement https://goose-docs.ai/blog/2026/05/14/goose-hooks; guide https://goose-docs.ai/docs/guides/context-engineering/hooks; source `crates/goose/src/hooks/mod.rs` cross-checked). Follows the Open Plugins hooks spec.
- **Config file + format**: hooks **cannot be configured standalone** — they live inside a *plugin*: `<plugin>/hooks/hooks.json` (JSON), discovered from `~/.agents/plugins/<plugin-name>/` (user) and `<project>/.agents/plugins/<plugin-name>/` (project). Plugin also needs a `plugin.json` manifest (`{"name","version","description"}`). Disable via `"disabledPlugins": ["name"]` in `~/.config/goose/settings.json` or `<project>/.config/goose/settings.json`.
- **Event names** (exactly 11; unknown names ignored): `SessionStart`, `SessionEnd`, `Stop`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `BeforeReadFile`, `AfterFileEdit`, `BeforeShellExecution`, `AfterShellExecution`. `SubagentStart`/`SubagentStop` are **not** emitted.
- **Matcher schema**: per-rule optional `matcher` — a **regex, not a glob**, unanchored, matched against an event-specific target (tool name / prompt text / file path / shell command). A bare `"*"` is an invalid regex → rule **silently skipped**; use `".*"` or omit.
- **Command schema**: rule = `{ "matcher": "...", "hooks": [ { "type": "command", "command": "...", "timeout": 30 } ] }`. Only `type: "command"` is executed. Runs via `sh -c`. `${PLUGIN_ROOT}` expands in the command and `PLUGIN_ROOT` is set in env. Default timeout 30 s.
- **stdin JSON envelope fields**: `event`, `session_id`, `matcher_context` (the matched string), `tool_name` (namespaced, e.g. `developer__shell`), `tool_input` (tool-specific args object; built-in developer tools use `command`, `path`/`content`, `path`/`before`/`after`, etc.), `message` (UserPromptSubmit), `last_assistant_message` (Stop), `working_dir`. Source also serializes `tool_output` (present in `HookContext` but **not** in the doc table — undocumented extra). Optional fields are omitted when not applicable.
- **Output contract**: only `PreToolUse` and `Stop` can block. Block signals: **exit code 2** (reason from stderr), or stdout starting with `{` containing `{"decision":"block","reason":"..."}` (checked regardless of exit code). Any other output/failure = **fail open**. Blocked tool calls return a "Tool call denied by policy hook `<plugin>`: <reason>" message to the model. `Stop` blocks are capped (`GOOSE_STOP_HOOK_BLOCK_CAP` raises the cap). No `additionalContext`/context-injection output channel documented — hooks are observation-only otherwise.

## Subagents
- **No dedicated subagent file dir/format.** Subagents are spawned dynamically: natural-language requests or **recipes** (YAML files in the working directory or in `GOOSE_RECIPE_PATH`, with keys `id, version, title, description, instructions, activities, extensions, parameters, prompt` with `{{param}}` templating) (https://goose-docs.ai/docs/guides/context-engineering/subagents). External CLIs (e.g. Codex) can be registered as a `stdio` extension named `subagent` in config.yaml.
- **Custom agents** are the closest file-based equivalent (https://goose-docs.ai/docs/guides/context-engineering/custom-agents): Markdown files with YAML frontmatter in `~/.agents/agents/` (global) and `<project>/.agents/agents/` (project); compat discovery also reads `.goose/agents/`, `.claude/agents/`, `~/.goose/agents/`, `~/.claude/agents/`. Frontmatter: `name` (required), `description` (optional), `model` (optional — goose-specific); body = instructions. Source (`sources.rs`) shows `name`/`description` plus arbitrary flattened extra properties. Invoked via `@name` mention, "Use the X agent…", or delegation.

## Commands
- **No markdown prompt-command directory** (no `.claude/commands` equivalent). Custom slash commands are aliases for **recipes**, declared in `config.yaml` (https://goose-docs.ai/docs/guides/context-engineering/slash-commands):
  ```yaml
  slash_commands:
    - command: "run-tests"
      recipe_path: "/path/to/recipe.yaml"
  ```
- Rules: case-insensitive, unique, no spaces, must not collide with built-ins (`/recipe`, `/compact`, `/help`, `/skills`, `/r`, …); at most one positional parameter; missing/invalid recipe → text sent to model verbatim. Recipes themselves live in the working dir, `~/.local/share/goose/recipes/`, or `GOOSE_RECIPE_PATH`.

## Rules
**Not supported as a distinct concept** — there is no rules directory or rule-file format (no `.cursor/rules` analog). The role is covered by `.goosehints`/`AGENTS.md` (session-start + nested), persistent instructions via `GOOSE_MOIM_MESSAGE_*` (every-turn, 64 KB cap), and skills (on-demand). Evidence: no rules page exists in the docs sitemap (https://goose-docs.ai/sitemap.xml); the context-engineering section comprises goosehints, persistent instructions, skills, custom agents, subagents, slash commands, prompt templates, hooks, plugins.

## MCP
- **Config file**: `~/.config/goose/config.yaml`, YAML, under the top-level **`extensions:`** map keyed by extension id (https://goose-docs.ai/docs/guides/config-files §Extensions Configuration; cross-checked https://goose-docs.ai/docs/getting-started/using-extensions §Config Entry). There is **no project-level MCP file** and no separate `.mcp.json`.
- **Server-table shape** (per-extension fields): `type` (`builtin` | `platform` | `stdio` | `streamable_http` | `frontend` | `inline_python`; legacy `sse` read only for compatibility), `name`, `enabled`, `bundled`, `display_name`, `timeout` (seconds), `available_tools` (tool allow-list; empty = all). stdio adds `cmd` (string), `args` (string list); streamable_http adds `uri` and `headers` (map).
- **Env handling**: two distinct keys — `envs` (map of literal `KEY: value`) and `env_keys` (list of variable *names* goose resolves from the system keyring/`secrets.yaml`/environment, keeping values out of config). Provider API keys placed in `config.yaml` are **ignored by design**; secrets live in the OS keyring or `secrets.yaml`. Also installable via `goose://extension?cmd=...&arg=...` deeplinks.

## Translation hazards
Concrete adjustments when targeting goose from a canonical `.agents` store:

1. **Skills land natively**: `.agents/skills/<name>/SKILL.md` is goose's canonical project path — no move needed. But:
   - **Drop/relocate Claude-only frontmatter keys**: goose does not read `allowed-tools`, `license`, or top-level `metadata` fields other than its own; extra keys survive as inert `properties`. Move tool gating out of `allowed-tools` (no equivalent; closest is per-extension `available_tools` in config.yaml). Arbitrary metadata should be nested under `metadata:` per agentskills.io, not flattened.
   - **Enforce name rules**: lowercase ASCII + digits + hyphens, ≤64 chars, no leading/trailing hyphen — rename skills like `Code_Review` or `My Skill`.
   - `name` may be omitted (falls back to dir name), but docs call it required — always emit it.
2. **Instructions**: target filename is **`.goosehints`** (dotfile, not `GOOSE.md`) — although `AGENTS.md` is read natively first, so a canonical AGENTS.md can be synced as-is; if writing `.goosehints`, strip any convention H1 like `# Claude Code instructions` (goose has no heading convention; plain text). Convert any Claude `@path` imports carefully — goose also uses `@file` to mean *eager inline inclusion* (same syntax, same semantics; safe), but plain paths stay lazy.
3. **Hooks — full reformat required**:
   - Config location: not a top-level `.agents/hooks.json`; must wrap into a **plugin**: `.agents/plugins/<plugin>/plugin.json` + `.agents/plugins/<plugin>/hooks/hooks.json`.
   - JSON (not YAML/TOML); structure `{hooks: {Event: [{matcher, hooks:[{type, command, timeout}]}]}}`.
   - **Event-name mapping**: Claude `Notification`/`PreCompact`/`SubagentStop` → **no goose equivalent, drop**. `SubagentStart`/`SubagentStop` are never emitted (documented) — drop them. `PostToolUse` in goose fires **only on success**; failure handlers must move to goose-only `PostToolUseFailure`. Claude's `PreToolUse` matcher on `Write|Edit` maps to goose tool names `developer__write|developer__edit` (namespaced with `__`), not `Write|Edit`.
   - **Matcher is regex, not glob**: translate globs (`*.rs` → `\.rs$`); never emit `"*"` (silently skipped) — omit matcher or use `".*"`.
   - **stdin envelope differs**: fields are `event`, `session_id`, `matcher_context`, `tool_name`, `tool_input`, `message`, `last_assistant_message`, `working_dir` — Claude's `hook_event_name`, `cwd`, `tool_response`, `transcript_path`, `session_id` must be renamed/re-mapped (`hook_event_name`→`event`, `cwd`→`working_dir`); there is no `tool_response` documented. File-path extraction from `tool_input` is tool-specific (`path` for `developer__write`/`developer__edit`).
   - **Output contract differs**: blocking = exit 2 (reason on stderr) or `{"decision":"block","reason":...}` on stdout; only `PreToolUse` and `Stop` block; everything else fails open. Claude's `hookSpecificOutput` / `additionalContext` / `permissionDecision` JSON outputs have no goose equivalent — drop.
   - Commands run via `sh -c` with `${PLUGIN_ROOT}` substitution; `timeout` in seconds (default 30).
4. **Subagents/agents**: rename canonical agent files to goose's flat `*.md` under `.agents/agents/` with frontmatter keys limited to `name`, `description`, `model` — drop Claude keys like `tools`, `color`, `permissionMode` (unsupported). Recipe-based subagents are YAML and not convertible to Markdown agent files.
5. **Slash commands**: cannot translate Markdown prompt commands — goose only aliases slash commands to recipe YAML paths via `slash_commands:` in global `config.yaml` (project-level not supported). Converting a prompt-command means authoring a recipe YAML and editing the global config — out of tree-sync scope.
6. **MCP**: target is YAML `extensions:` in global `~/.config/goose/config.yaml`, not a project JSON file. Rename JSON `command`→`cmd`, keep `args` as a list, map `env` to `envs` (literal values) and prefer `env_keys` (name-only, keyring-resolved) for secrets — note `envs` values sit in plaintext YAML. Map transport: `stdio`→`stdio`, `http`/`sse`→`streamable_http` (`sse` is legacy-only), and add goose-required bookkeeping keys (`name`, `enabled: true`, `bundled: false`, `timeout`, `type`). No `disabled` flag — use `enabled`. Env-var expansion syntax like `${VAR}` is not documented for `envs`.
7. **Rules**: no target — merge rule files into `.goosehints` or a skill; there is nothing else to write to.
8. **Global/user scope mismatch**: goose's global hints live at `~/.config/goose/.goosehints`, but global skills/agents/plugins live under `~/.agents/...` (shared with other tools), not under `~/.config/goose/` — adopt mapping must split by kind.
9. **Naming**: docs and config are inconsistent about the subagent extension (`name: subagent` stdio entry) — treat as recipe/extension territory, not file-sync territory.

## Verified corrections (fact-checker pass)

- Recipe discovery locations: was 'Recipes themselves live in the working dir, ~/.local/share/goose/recipes/, or GOOSE_RECIPE_PATH' → now 'recipes are discovered from the current directory, GOOSE_RECIPE_PATH directories, the global library ~/.config/goose/recipes/, project-local ./.goose/recipes/, or a GitHub repo via GOOSE_RECIPE_GITHUB_REPO; ~/.local/share/goose/recipes/ appears only as an illustrative example path and is not a documented storage location' (https://goose-docs.ai/docs/guides/recipes/storing-recipes)
- CONTEXT_FILE_NAMES default: was 'default list is ["AGENTS.md", ".goosehints"]' → now 'official docs contradict each other: the goosehints guide states the default is ["AGENTS.md", ".goosehints"], but the environment-variables reference table states the default is [".goosehints", "AGENTS.md"]; the claim should flag the discrepancy rather than assert one order' (https://goose-docs.ai/docs/guides/environment-variables vs https://goose-docs.ai/docs/guides/context-engineering/using-goosehints)
- Recipe schema keys: was 'recipes ... with keys id, version, title, description, instructions, activities, extensions, parameters, prompt' → now 'the official recipe schema has no id field (id appears only in a subagents-doc example); documented fields are title and description (both required), instructions and prompt (at least one required), plus activities, extensions, parameters, response, retry, settings, sub_recipes, and version' (https://goose-docs.ai/docs/guides/recipes/recipe-reference)
- Skills extension status: was 'in v1.25.0+ the standalone extension is deprecated in favor of the "Summon" extension' → now 'the skills-mcp page states the Skills extension is deprecated and only available in v1.16.0–v1.24.0 (i.e., removed, not merely deprecated, in v1.25.0+, with Summon as the replacement), while the using-skills guide still describes a built-in Skills platform extension enabled by default — the docs are internally inconsistent and the brief should say so' (https://goose-docs.ai/docs/mcp/skills-mcp)
