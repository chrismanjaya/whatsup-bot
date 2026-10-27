# App name

One paragraph: what this service does and for whom.

## Quick start
```bash
cp .env.example .env   # fill in values
make tools             # install pinned golangci-lint
make run
```

## Configuration
| Variable | Required | Default | Description |
|---|---|---|---|
| `DB_DSN` | yes | — | Database connection string |
| `LOG_LEVEL` | no | `info` | debug, info, warn, error |
| `LOG_FORMAT` | no | `text` | `json` in production |

## Architecture
See `.agents/skills/go-architecture/SKILL.md`. Layout:

```
cmd/                  composition root (main.go, wire.go)
internal/domain       entities + pure rules
internal/usecase/<f>  one package per feature; ports in port.go
internal/infra        delivery/, storage/, external/ adapters
internal/constant     error codes, fixed values
internal/util         error wrapping, logging, helpers
```

## Development
`make check` runs fmt, vet, lint, test and vulnerability scan — the same as CI.
