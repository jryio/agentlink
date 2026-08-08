# agentlink

`agentlink` keeps Claude Code and Codex configuration as peer implementations of the same developer behavior. It detects drift, explains the exact counterpart that needs attention, and can safely reconcile an explicitly chosen side.

It is one fast Go binary. There is no Python runtime, shell-script engine, Git API, Dropbox API, daemon, or database. A source is simply a directory on the local filesystem, so it can be:

- a project directory;
- a version-controlled dotfiles directory;
- a mounted Dropbox, iCloud Drive, Syncthing, or network-share directory;
- a live tool configuration directory; or
- any combination of those roots.

The operating model comes from the peer-artifact approach in [How I keep Claude Code and Codex in sync](https://stafforini.com/notes/how-i-keep-claude-code-and-codex-in-sync/): explicit counterparts, edit-time reminders, commit-time enforcement, deterministic audits, documented divergences, and secret-safe MCP wiring checks.

## Why it feels good to use

- `agentlink` with no arguments is the fast drift check.
- YAML is strict: typos are errors, not silently ignored fields.
- `agentlink init` writes a matching JSON Schema, so editors autocomplete and validate the config immediately.
- `siblings` pairs discover nested `CLAUDE.md` / `AGENTS.md` files automatically.
- Skill frontmatter is compared semantically while tool-only keys are ignored.
- Sync is preview-only until `--apply`; deletion additionally requires `--prune`.
- Every read and write stays inside a configured `os.Root`; sync writes are atomic and refuse to replace symlinks.
- Hooks receive changed paths from any producer. Git is one possible producer, not an architectural dependency.
- Human output is concise; `--json` and stable exit codes make CI straightforward.

## Install

Requires a security-patched Go toolchain: Go 1.25.12 or newer on the 1.25 line, or Go 1.26.5 or newer. Earlier patch releases are affected by [`GO-2026-4970`](https://pkg.go.dev/vuln/GO-2026-4970) in `os.Root`, which is central to agentlink's filesystem confinement.

```sh
go install github.com/jryio/agentlink/cmd/agentlink@latest
```

For a checkout:

```sh
task tools
task build
./bin/agentlink version
```

## Sixty-second start

From a project root:

```sh
agentlink init
agentlink check
```

`init` creates `agentlink.yaml` and `agentlink.schema.json`. The starter config checks every sibling `CLAUDE.md` / `AGENTS.md` pair recursively and the project-local skill trees.

When a counterpart is missing or different:

```sh
agentlink sync --from claude             # inspect the plan
agentlink sync --from claude --apply     # atomically apply copies
agentlink check                          # deterministic verification
```

Target-only files remain untouched unless pruning is explicit:

```sh
agentlink sync --from claude --prune             # preview deletions
agentlink sync --from claude --prune --apply     # apply them
```

Neither Claude nor Codex is canonical. You choose the source for each reconciliation.

## Configuration

Here is a compact configuration spanning a synced global directory and a local project:

```yaml
# yaml-language-server: $schema=./agentlink.schema.json
version: 1

sources:
  shared:
    root: ~/Dropbox/agent-config
  project:
    root: .
    relative_to: config

pairs:
  - id: global-instructions
    kind: file
    claude: {source: shared, path: claude/CLAUDE.md}
    codex: {source: shared, path: codex/AGENTS.md}
    normalizer: instructions
    sync: manual

  - id: global-skills
    kind: tree
    claude: {source: shared, path: claude/skills}
    codex: {source: shared, path: codex/skills}
    normalizer: skill
    sync: manual

  - id: project-instructions
    kind: siblings
    claude: {source: project, path: CLAUDE.md}
    codex: {source: project, path: AGENTS.md}
    normalizer: instructions
    optional: true

  - id: project-hooks
    kind: tree
    claude: {source: project, path: .claude/hooks}
    codex: {source: project, path: .codex/hooks}
    normalizer: hook
    optional: true

ignore:
  - "**/.git/**"
  - "**/cache/**"
  - "**/tmp/**"
```

Relative source roots resolve beside the config by default. Set `relative_to: cwd` when one global config should apply to the directory where `agentlink` is invoked. `~` and environment variables are supported. Missing environment variables are configuration errors.

Pair kinds are:

- `file`: one exact Claude/Codex file pair;
- `tree`: two recursively compared directories; and
- `siblings`: recursively discovers counterpart filenames at matching directory levels.

Normalizers are `exact`, `text`, `instructions`, `skill`, `hook`, and `presence`. `presence` is useful for registration files whose formats are deliberately tool-specific: both files must exist, but their content is not claimed to be equivalent.

Raw copying is automatic by default only for `exact` and `text`. Semantic pairs default to `sync: manual`, because equal behavior may require different syntax. Set `sync: copy` only when raw content is valid for both tools; the generated starter does this explicitly for ordinary project instruction Markdown.

See [the full configuration guide](docs/configuration.md) and the checked-in [JSON Schema](agentlink.schema.json).

## Drift, audit, and JSON

`check`, `status`, and `audit` are aliases:

```sh
agentlink
agentlink check --pair global-skills
agentlink --json check
agentlink --quiet check
```

Exit codes are deliberately small:

| Code | Meaning |
| ---: | --- |
| `0` | clean or command succeeded |
| `1` | drift exists, a guard blocked, or an operation failed |
| `2` | invalid command usage |

Optional unavailable roots—useful for intermittently mounted file-sharing services—are shown as skipped and do not create false drift.

## Provider-neutral reminders and guards

`guard` and `remind` accept paths as arguments, one path per input line, or a Claude/Codex JSON hook payload. They inspect current normalized content, so a tool-only skill frontmatter change that preserves semantic parity does not create busywork.

```sh
agentlink guard CLAUDE.md .claude/skills/review/SKILL.md
printf '%s\n' CLAUDE.md | agentlink guard
some-change-producer | agentlink remind --agent claude
some-change-producer | agentlink remind --agent codex
```

For a Git pre-commit hook, Git can provide the paths while `agentlink` remains unaware of repositories:

```sh
git diff --cached --name-only --diff-filter=ACMR | agentlink guard
```

The same command works with another VCS, an editor file list, or a sync-service event adapter. See [integration recipes](docs/integrations.md).

## Live activation checks

Durable files are only useful when the tools actually load them. `activations` verify that a live path is a symlink to the expected tracked or synced artifact:

```yaml
sources:
  home: {root: ~}

activations:
  - id: codex-skills-live
    expected: {source: shared, path: codex/skills}
    live: {source: home, path: .codex/skills}
```

The audit reports a missing live path, a real file where a symlink is required, or a link resolving to the wrong target. It never creates or replaces live links automatically.

## Secret-safe MCP parity

Claude and Codex do not automatically share MCP configuration. `mcp_servers` verifies that both server entries exist, compares only public `command`, `args`, `transport`, and `url` fields, and checks required environment-variable **names**. Secret values are never compared or printed.

```yaml
mcp_servers:
  - id: tasks-mcp
    claude:
      config: {source: project, path: .mcp.json}
      server: tasks
    codex:
      config: {source: project, path: .codex/config.toml}
      server: tasks
    required_env: [TASKS_TOKEN]
```

JSON, TOML, and YAML MCP configuration files are supported.

Service-specific account identity probes remain explicit external checks. `agentlink` does not execute commands from an automatically discovered project YAML file: doing so would turn a drift audit into arbitrary code execution. Run a trusted provider-specific identity command alongside `agentlink check` in your hook or CI when account identity must also be verified.

## Intentional divergence

An exception is explicit, scoped, and requires a reason:

```yaml
exceptions:
  - pair: global-skills
    paths: [claude-only/**]
    reason: Claude exposes an event Codex does not currently support.
```

Exceptions are visible in `agentlink list`. They are not magic ignore files hidden elsewhere.

## Command map

```text
agentlink [check]       deterministic drift audit
agentlink sync          preview/apply one-way reconciliation
agentlink guard         block drifting changed paths
agentlink remind        emit human/Claude/Codex hook context
agentlink list          resolve and display the topology
agentlink validate      strict YAML validation
agentlink doctor        validate and safely open sources
agentlink schema        print the bundled JSON Schema
agentlink init          create starter YAML + schema
agentlink version       print build version
```

Configuration discovery checks `--config`, then `AGENTLINK_CONFIG`, then walks upward for `agentlink.yaml` / `agentlink.yml`, then checks the user configuration directory.

## Development

```sh
task test       # race-enabled and shuffled
task test:e2e   # init → drift → preview → apply → verify → guard
task cover
task ci         # tidy, strict lint, vet, vulnerability scan, coverage, build
```

The tests exercise the complete loop: initialize, detect one-sided drift, prove preview does not mutate, apply an atomic sync, verify nested counterparts, block drift through hook input, prune explicitly, and validate secret-safe MCP comparisons.
