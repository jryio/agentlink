# Configuration

`agentlink.yaml` is strict. Unknown fields, duplicate IDs, unsafe paths, and
invalid globs are errors.

```yaml
version: 1
sources: {}
pairs: []
mcp_servers: []
activations: []
exceptions: []
ignore: []
limits: {}
```

`version`, `sources`, and `pairs` are required.

## Sources

```yaml
sources:
  shared:
    root: ~/Dropbox/agent-config
    optional: true
  project:
    root: .
    relative_to: cwd
```

`root` accepts absolute paths, `~`, and environment variables.

Relative roots use the config directory by default. Set `relative_to: cwd` for
a global config that should follow the invocation directory.

An unavailable optional source is skipped. Other source errors fail the check.

## Pairs

```yaml
pairs:
  - id: global-skills
    kind: tree
    claude: {source: shared, path: claude/skills}
    codex: {source: shared, path: codex/skills}
    normalizer: skill
    sync: manual
    ignore: ["**/generated/**"]
    optional: false
```

`optional` treats two missing peers as clean. One missing peer is still drift.

### Kinds

| Kind | Behavior |
| --- | --- |
| `file` | Compare two files. |
| `tree` | Compare relative files below two directories. |
| `siblings` | Find matching peer filenames recursively. |

Tree endpoints may not overlap. All endpoint paths use `/` and remain below
their source root.

### Normalizers

| Normalizer | Behavior |
| --- | --- |
| `exact` | Compare bytes. |
| `text` | Normalize line endings, trailing space, and final newline. |
| `instructions` | Normalize text and Claude/Codex headings. |
| `skill` | Canonicalize frontmatter and remove tool-only keys. |
| `hook` | Normalize text and agent-specific reminder commands. |
| `presence` | Require both files without comparing content. |

Non-`exact` normalizers reject binary files.

### Sync

`exact` and `text` pairs default to `sync: copy`. Other normalizers default to
`sync: manual`.

Use `sync: copy` only when raw content is valid for both peers.

## Ignore and exceptions

Globs use `/`. `*` matches one path component. `**` crosses directories and
must be a complete component.

Use `ignore` for generated or runtime files.

Use `exceptions` for intended differences:

```yaml
exceptions:
  - pair: hooks
    paths: [claude-only/**]
    reason: Codex does not expose this event.
```

Paths are relative to the pair. `.` covers the whole pair.

## MCP wiring

```yaml
mcp_servers:
  - id: issues
    claude:
      config: {source: project, path: .mcp.json}
      server: issues
    codex:
      config: {source: project, path: .codex/config.toml}
      server: issues
    compare_public: true
    required_env: [ISSUES_TOKEN]
```

Supported formats are JSON, TOML, and YAML.

Checks cover:

- server presence
- `command`, `args`, `transport`, and `url`
- required environment key names

Secret values are neither compared nor printed. Provider identity checks should
run as separate trusted hook or CI steps. Project configuration never executes
commands.

## Activations

```yaml
activations:
  - id: codex-skills-live
    expected: {source: shared, path: codex/skills}
    live: {source: home, path: .codex/skills}
```

An activation requires `live` to be a symlink to `expected`. `agentlink` checks
links but never creates them.

## Limits

```yaml
limits:
  max_file_size: 4194304
  max_files: 25000
```

Zero or omission uses the shown default. Configuration files are limited to
4 MiB. Hook input is limited to 8 MiB.

## Discovery

Configuration is selected in this order:

1. `--config`
2. `AGENTLINK_CONFIG`
3. `agentlink.yaml` or `agentlink.yml` in the current directory or a parent
4. the user configuration directory

## Editor support

`agentlink init` writes a local schema reference. To write the bundled schema
elsewhere:

```sh
agentlink schema > agentlink.schema.json
```
