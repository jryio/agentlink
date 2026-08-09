# Integrations

`guard` and `remind` accept:

- path arguments
- one path per input line
- agent hook JSON containing `path`, `file_path`, `filePath`, `file`, or patch
  input (Claude, Codex, Cursor, Copilot, and similar envelopes)

Relative paths use the current working directory.

## Remind an agent

```sh
your-path-producer | agentlink remind --agent claude
your-path-producer | agentlink remind --agent codex
```

`remind` always succeeds. It emits context only when touched peers drift.
Agent adapters use the shared `hookSpecificOutput` `PostToolUse` envelope.

## Block drift

```sh
your-path-producer | agentlink guard
```

`guard` exits `1` when a touched peer drifts and prints the counterpart to
update.

Git can provide staged paths:

```sh
git diff --cached --name-only --diff-filter=ACMR | agentlink guard
```

The core does not invoke Git. Any VCS, editor, CI job, or file-event adapter can
provide paths.

## Audit in CI

```sh
agentlink --quiet check
```

For annotations or stored results:

```sh
agentlink --json check
```
