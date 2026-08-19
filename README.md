# agentlink

Keep coding-agent configuration in sync across Claude Code, Codex, Cursor,
Copilot, Gemini, and a growing registry of CLI agents.

`agentlink` compares peer instructions, skills, hooks, MCP wiring, and live
symlinks. It works with any local directory, including repositories and mounted
sync services. It does not depend on Git or a storage provider.

Shared files live in `.agents`; agents that cannot read `.agents` natively get
their own directory linked or translated from it:

```text
.agents/
└── skills/
    ├── review/
    │   └── SKILL.md
    └── search/
        └── SKILL.md
.claude/
└── skills -> ../.agents/skills
.cursor/
└── skills/          (translated from .agents by `sync: translate`)
```

Most agents — Codex, Cursor, Kimi, Copilot, and others — read `.agents/skills`
natively, so no pair is needed for them. Run `agentlink init` in a project and
it detects which agents you use and scaffolds only the pairs you need.

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
agentlink sync --from agents           # preview
agentlink sync --from agents --apply   # write
agentlink check
```

No tool is canonical by force. The `.agents` hub is the usual source; choose
`--from <agent>` (any registered ID) per sync. With `sync: translate` the
source artifact is rewritten into the target's native shape — frontmatter keys
are dropped or renamed, hook events are mapped, settings files are merged
rather than replaced.

## Adopt an existing project

Move any project-relative agent configuration file or directory into `.agents`
and replace the original with a relative symlink:

```sh
agentlink adopt --from .claude/skills           # preview
agentlink adopt --from .claude/skills --apply   # copy, then link back
```

The default maps `.<agent>/skills` to `.agents/skills`, matching the shared
`.agents/skills` layout. Other paths stay agent-specific: for example,
`.claude/settings.local.json` becomes
`.agents/claude/settings.local.json`. Use `--to PATH` to choose a different
destination beneath `.agents`.

If the selected destination already exists, `adopt` prints a warning and
refuses to replace it until `--force --apply` is supplied. The selected root
must be a regular file or directory inside the project. Relative links inside a
copied tree are dereferenced only when they stay inside the project; external
links and an unmanaged linked root are refused.

## Configure hierarchies

Agentlink searches from the working directory toward the filesystem root. It
uses the first `agentlink.yaml` that it finds. Parent and child configuration
files do not merge.

### Shared library for sibling repositories

Place a shared skill library and its configuration above related repositories:

```text
cloudx/
├── agentlink.yaml                 # shared configuration
├── skills/
│   └── review/
│       └── SKILL.md
├── api/
│   └── .agents/
│       └── skills/                # materialized shared skills and local skills
└── web/
    └── .agents/
        └── skills/
```

Put this configuration in `cloudx/agentlink.yaml`:

```yaml
version: 2

sources:
  shared:
    root: skills
  project:
    root: .
    relative_to: cwd

pairs:
  - id: cloudx-skills
    kind: tree
    base: agents
    peers:
      agents: {source: shared, path: .}
      codex: {source: project, path: .agents/skills}
    normalizer: skill
    sync: copy
```

Run these commands from `cloudx/`:

```sh
agentlink check --repos .
agentlink sync --repos . --from agents --apply
```

The shared library supplies required skills. A repository can keep extra
skills in `.agents/skills`; `base: agents` leaves those files unmanaged.

### Repository configuration that shadows its parent

Give a repository its own configuration when its agent setup differs:

```text
cloudx/
├── agentlink.yaml                 # applies to repositories without a local config
├── api/
│   ├── .git/
│   ├── agentlink.yaml             # selected for commands in api/
│   ├── .agents/
│   │   └── skills/
│   └── .claude/
│       └── skills/
└── web/
    └── .git/
```

Put this configuration in `cloudx/api/agentlink.yaml`:

```yaml
version: 2

sources:
  project:
    root: .
    relative_to: cwd

pairs:
  - id: api-skills
    kind: tree
    peers:
      agents: {source: project, path: .agents/skills}
      claude: {source: project, path: .claude/skills}
    normalizer: skill
    sync: copy
```

Run `agentlink check` from `cloudx/api/` to use the repository configuration.
`agentlink check --repos cloudx/` also uses it for `api/`. Add every pair that
the repository needs because the local file replaces, rather than extends, the
parent configuration.

## Normalizers

Each agent expresses the same intent differently: headings name the tool, and
skill frontmatter carries keys only one tool understands. Compared byte for
byte, equivalent files look like drift. Normalizers remove these surface
differences before comparing, using the agent registry (internal/agent) to
learn each peer's conventions.

`instructions` treats these headings as equal:

```markdown
# Claude Code instructions
# Agent instructions
```

`skill` keeps only the frontmatter keys both peers understand, so these
compare equal:

```yaml
---
name: review
description: Review a pull request
allowed-tools: [Bash, Read]
---
```

```yaml
---
name: review
description: Review a pull request
---
```

`hook` parses each peer's hook document (JSON, TOML, or YAML; bare, wrapped,
or embedded in a settings file) and canonicalizes event names, timeout units,
and agent-specific command tokens before comparing.

## Configure

```yaml
# yaml-language-server: $schema=./agentlink.schema.json
version: 2

sources:
  project:
    root: .
    relative_to: config

pairs:
  - id: instructions
    kind: siblings
    peers:
      agents: {source: project, path: AGENTS.md}
      claude: {source: project, path: CLAUDE.md}
    normalizer: instructions
    sync: translate
    optional: true

  - id: skills
    kind: tree
    peers:
      agents: {source: project, path: .agents/skills}
      claude: {source: project, path: .claude/skills}
    normalizer: skill
    optional: true

ignore:
  - "**/.git/**"
  - "**/node_modules/**"
  - "**/cache/**"
```

Each pair names exactly two peers keyed by registered agent ID; `agents` is
the built-in canonical hub. Registered agents: amp, claude, codex, copilot,
crush, cursor, devin, droid, gemini, goose, hermes, kilo, kimi, mastracode,
omp, opencode, pi, qodercli.

Sources are plain filesystem roots. A root may be:

- project-local
- tracked in a dotfiles repository
- mounted from Dropbox, Syncthing, iCloud Drive, or a network share
- relative to the config or current working directory

Pair kinds:

- `file`: one file on each side
- `tree`: matching directory trees
- `siblings`: every matching `CLAUDE.md` and `AGENTS.md` below a root

Semantic pairs default to manual sync. Set `sync: copy` only when the same
file is valid for both peers, or `sync: translate` to have the source
artifact rewritten into the target's native shape (skill frontmatter,
instruction headings, hook documents).

See [Configuration](docs/configuration.md) for the full schema.

## Commands

```text
agentlink                 check all configured peers
agentlink check           check all configured peers
agentlink check --repos DIR check every repository below DIR
agentlink check --pair ID check one peer
agentlink sync --repos DIR preview or apply one-way sync in each repository
agentlink sync            preview or apply one-way sync
agentlink guard           reject drifting changed paths
agentlink adopt            preview moving selected configuration into .agents
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

`guard` accepts path arguments, newline-delimited paths, or agent hook JSON
(Claude, Codex, Cursor, Copilot, and friends share the `path`/`file_path`/
`filePath`/`file` envelope fields it understands).

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
