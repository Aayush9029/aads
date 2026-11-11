# aads

A Go CLI for the Apple Ads Campaign Management API 5. It ships through `aayush9029/tap`. Status: active.

## Commands

| Task | Command |
|------|---------|
| Build | `make build` |
| Test | `make test` |
| Full check (race, vet, build; same as CI) | `make check` |

## Architecture

- `main.go` starts `internal/cli`.
- `internal/commands/registry.go` declares each command: the HTTP method, the API path, and the flags. Add a command there, not in the CLI layer.
- `internal/auth` builds the OAuth client secret from an EC P-256 private key.
- `internal/api` holds the HTTP client, the pagination, and the error decoder. `internal/types` holds the API models.
- `docs/` holds the usage guide, the command reference, and the map from commands to Apple endpoints.

## Rules

- Configuration order: command flags, then `AADS_*` environment variables, then `~/.aads/config.yaml`. The config file stores the private key path, never the key or the tokens.
- Each mutation command supports `--dry-run`. It prints the request with redacted headers and does not call Apple. Keep this for each new mutation; the tap formula test uses it.
- Normal output goes to stdout. Progress, retry notices, and errors go to stderr.
- When you add or change a command, update `docs/commands/aads.md` and `docs/OFFICIAL_APPLE_ADS_DOCS.md`.
