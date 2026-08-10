# Configuration

`agentlink.yaml` is strict. Unknown fields, duplicate IDs, unsafe paths, and
invalid globs are errors.

```yaml
version: 2
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
    peers:
      agents: {source: shared, path: skills}
      claude: {source: shared, path: claude/skills}
    normalizer: skill
    sync: manual
    ignore: ["**/generated/**"]
    optional: false
```

`peers` names exactly two endpoints keyed by registered agent ID. The
built-in ID `agents` is the canonical `.agents` hub store. The registered
agent IDs are: amp, claude, codex, copilot, crush, cursor, devin, droid,
gemini, goose, hermes, kilo, kimi, mastracode, omp, opencode, pi, qodercli.
Unknown IDs and agents without the relevant capability (for example a hook
pair involving pi, whose hooks are TypeScript modules) are validation errors.

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
| `instructions` | Normalize text and agent-name headings/prose (each registry agent's display names converge on "Agent"). |
| `skill` | Canonicalize frontmatter and keep only keys both peers understand (renames like cursor's `paths` are inverted first). |
| `hook` | Parse each peer's hook document and compare canonical form: event names, timeout units, and `--agent`/`remind-` command tokens are normalized; events the other peer cannot express are ignored. |
| `presence` | Require both files without comparing content. |

Non-`exact` normalizers reject binary files.

### Sync

`exact` and `text` pairs default to `sync: copy`. Other normalizers default to
`sync: manual`.

Use `sync: copy` only when raw content is valid for both peers.

Use `sync: translate` (with `normalizer` `skill`, `instructions`, or `hook`)
to rewrite the source artifact into the target's native shape on write:
unsupported frontmatter keys are dropped, canonical keys are renamed (for
example `globs` becomes `paths` for cursor), hook events are mapped to the
target's names with unsupported events dropped and reported, and hook
documents embedded in settings files (claude, gemini, qodercli, crush) are
merged rather than replacing unrelated settings. The sync source is always
canonicalized first, so either peer — not just the `.agents` hub — can be
`--from`: a claude settings file synced to the hub lands as a bare canonical
hooks document, not as a settings dump. Translation never fabricates values.

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
    peers:
      agents:
        config: {source: project, path: .agents/mcp.json}
        server: issues
      codex:
        config: {source: project, path: .codex/config.toml}
        server: issues
    compare_public: true
    required_env: [ISSUES_TOKEN]
```

Supported formats are JSON, JSONC (comments and trailing commas tolerated),
TOML, and YAML, chosen from each peer's registry entry. The server table is
located per agent (`mcpServers`, `mcp_servers`, `mcp`, `amp.mcpServers`, or
goose's `extensions`, where only `stdio`/`streamable_http` entries count).
Server fields are normalized before comparison: opencode's single-array
`command` splits into command and args, `local`/`streamable-*` transport
aliases converge, and environment key names come from each agent's env field
(`env`, `environment`, or codex's `env_vars` passthrough).

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

`max_file_size` is clamped to a hard ceiling of 64 MiB and `max_files` to
1,000,000; larger values are rejected by `validate` and clamped at runtime so
repository-controlled configuration cannot remove the process resource budget.
`pairs`, `mcp_servers`, and `activations` are each capped at 4096 entries.

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
