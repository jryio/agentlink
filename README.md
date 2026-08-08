# agentlink

Keep Claude Code and Codex configuration in sync.

`agentlink` compares peer instructions, skills, hooks, MCP wiring, and live
symlinks. It works with any local directory, including repositories and mounted
sync services. It does not depend on Git or a storage provider.

## Install

```sh
go install github.com/jryio/agentlink/cmd/agentlink@latest
```

Use Go 1.25.12+ or 1.26.5+. Earlier versions contain an `os.Root`
[security issue](https://pkg.go.dev/vuln/GO-2026-4970).

## Start

```sh
agentlink init
agentlink check
```

`init` creates:

- `agentlink.yaml`
- `agentlink.schema.json`

The schema provides completion and validation in YAML-aware editors.

When peers drift:

```sh
agentlink sync --from claude           # preview
agentlink sync --from claude --apply   # write
agentlink check
```

Neither tool is canonical. Choose `--from claude` or `--from codex` for each
sync.

## Configure

```yaml
# yaml-language-server: $schema=./agentlink.schema.json
version: 1

sources:
  project:
    root: .
    relative_to: config

pairs:
  - id: instructions
    kind: siblings
    claude: {source: project, path: CLAUDE.md}
    codex: {source: project, path: AGENTS.md}
    normalizer: instructions
    sync: copy
    optional: true

  - id: skills
    kind: tree
    claude: {source: project, path: .claude/skills}
    codex: {source: project, path: .codex/skills}
    normalizer: skill
    optional: true

ignore:
  - "**/.git/**"
  - "**/node_modules/**"
  - "**/cache/**"
```

Sources are plain filesystem roots. A root may be:

- project-local
- tracked in a dotfiles repository
- mounted from Dropbox, Syncthing, iCloud Drive, or a network share
- relative to the config or current working directory

Pair kinds:

- `file`: one file on each side
- `tree`: matching directory trees
- `siblings`: every matching `CLAUDE.md` and `AGENTS.md` below a root

Semantic pairs default to manual sync. Set `sync: copy` only when the same file
is valid for both tools.

See [Configuration](docs/configuration.md) for the full schema.

## Commands

```text
agentlink                 check all configured peers
agentlink check           check all configured peers
agentlink check --pair ID check one peer
agentlink sync            preview or apply one-way sync
agentlink guard           reject drifting changed paths
agentlink remind          emit hook context
agentlink list            show resolved configuration
agentlink validate        validate YAML
agentlink doctor          validate YAML and source roots
agentlink schema          print the JSON Schema
agentlink init            create starter files
```

Use `agentlink --json check` for structured output. Exit status is `0` for
success, `1` for drift, and `2` for invalid usage.

## Guard changes

`guard` accepts path arguments, newline-delimited paths, or Claude/Codex hook
JSON.

```sh
agentlink guard CLAUDE.md
printf '%s\n' CLAUDE.md | agentlink guard
git diff --cached --name-only --diff-filter=ACMR | agentlink guard
```

Git is only the path producer. The same interface works with another VCS, an
editor, CI, or a sync-service adapter.

See [Integrations](docs/integrations.md) for hook examples.

## Safety

- Sync previews by default.
- Writes require `--apply`.
- Deletes also require `--prune`.
- Semantic pairs require explicit copy permission.
- Reads are bounded and confined with `os.Root`.
- Writes are atomic and do not replace symlinks.
- MCP checks compare public fields and environment key names, never secret
  values.
- Configuration never executes identity-check commands.

Live paths can be checked with `activations`. Intentional differences belong in
`exceptions` with a reason.

## Develop

```sh
task tools
task ci
task test:e2e
task build
```

`task ci` runs formatting checks, lint, vet, race-enabled tests, coverage,
`govulncheck`, and a build.
