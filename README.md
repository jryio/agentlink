# agentlink

<one-sentence description>

## Requirements

- Go 1.25 or newer
- [Task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest`)

## Quick start

```bash
task tools     # install pinned dev tools
task           # lint + vet + test
task build     # binaries → ./bin
```

## Layout

- `cmd/` — binaries
- `internal/` — private packages
- `docs/` — extended docs

## Common tasks

| Command      | What it does                     |
| ------------ | -------------------------------- |
| `task`       | Default: lint, vet, test         |
| `task test`  | Race-enabled unit tests          |
| `task cover` | HTML coverage report             |
| `task lint`  | golangci-lint v2                 |
| `task vuln`  | govulncheck                      |
| `task build` | Build all `cmd/*` binaries       |
| `task ci`    | Run the full CI pipeline locally |

## Contributing

See [AGENTS.md](AGENTS.md) for engineering conventions (humans and agents).

## License

<license>
