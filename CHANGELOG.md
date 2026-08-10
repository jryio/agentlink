# Changelog

## Unreleased

- Support 18 CLI coding agents (amp, claude, codex, copilot, crush, cursor, devin, droid, gemini, goose, hermes, kilo, kimi, mastracode, omp, opencode, pi, qodercli) plus the canonical `.agents` hub via a data-driven registry in `internal/agent`; per-agent layouts documented in `docs/research/` and locked to the registry by a machine-checkable `agentlink` block in `docs/research/design-matrix.json`.
- **Breaking**: configuration moves to `version: 2`. Pair and MCP endpoints move from fixed `claude:`/`codex:` keys to a `peers:` map keyed by agent ID; `missing_claude`/`missing_codex` finding states become `missing` with a `peer` field, and findings carry a `paths` map keyed by agent ID. Version-1 files fail with a migration-directed error.
- Add `sync: translate` (internal/format): rewrite artifacts between any two agents' native shapes — every sync source is canonicalized through the shared `Formatter.Canonicalize` seam before target emission, so spokes are as safe a source as the hub. Skill frontmatter key drop/rename, instruction headings, hook event mapping with TOML/YAML/JSONC/settings-file emission and merge-on-write for settings documents.
- Normalizers are agent-parameterized: instruction headings and prose names converge per peer pair, skill comparison keeps only mutually supported frontmatter keys, and hook documents compare structurally across JSON/JSONC/TOML/YAML dialects. Malformed hook events surface as error findings instead of comparing clean.
- MCP checks generalize across agents: per-agent server table keys, JSONC support, command-array and transport-alias normalization, env key names from each agent's env field (including codex's object-form `env_vars` entries), goose extension filtering.
- `init` detects agents present in the project and scaffolds hub-and-spoke pairs, activations for non-native skill readers, and hook translation pairs.
- `guard`/`remind --agent` accept any registered agent with declarative hooks; hook input parsing recognizes cursor/copilot envelope fields.

## 0.1.0

- Add the `agentlink` CLI.
- Add strict YAML configuration and a bundled JSON Schema.
- Compare file, tree, sibling, MCP, and live-link peers.
- Add preview-first sync with atomic writes and explicit pruning.
- Add provider-neutral reminder and guard input.
- Confine and bound filesystem access.
