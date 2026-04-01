# polygon-rpc-proxy

A minimal JSON-RPC 2.0 reverse proxy for Polygon, written in Go.
Forwards all requests transparently to an upstream RPC node (default: `https://polygon.drpc.org`), adds per-IP rate limiting, and structured logging.

## Features

- Transparent proxy — single and batch JSON-RPC requests passed through unchanged
- Per-IP token-bucket rate limiting (`golang.org/x/time/rate`)
- Structured JSON logging via [zerolog](https://github.com/rs/zerolog)
- YAML config with `${VAR:-default}` env substitution
- `GET /health` endpoint
- Minimal Docker image (`FROM scratch`)

## Configuration

Config is loaded from `config.yaml` (override with `CONFIG_FILE` env var).
Every value supports environment variable substitution.

```yaml
server:
  port: ${PORT:-8080}
  read_timeout: ${READ_TIMEOUT:-15s}
  write_timeout: ${WRITE_TIMEOUT:-35s}
  idle_timeout: ${IDLE_TIMEOUT:-60s}

proxy:
  upstream_url: "${UPSTREAM_URL:-https://polygon.drpc.org}"
  timeout: ${TIMEOUT:-30s}

rate_limit:
  enabled: ${RATE_LIMIT_ENABLED:-true}
  requests_per_second: ${RATE_LIMIT_RPS:-100}
  burst: ${RATE_LIMIT_BURST:-200}

log:
  level: "${LOG_LEVEL:-info}"   # debug | info | warn | error
  pretty: ${LOG_PRETTY:-false}
```

## Running

**Locally:**
```sh
go run ./cmd/server
```

**With custom upstream:**
```sh
UPSTREAM_URL=https://my-node.example.com go run ./cmd/server
```

**Docker:**
```sh
docker build -t polygon-rpc-proxy .
docker run -p 8080:8080 polygon-rpc-proxy
```

## Usage

```sh
# single call
curl -s -X POST http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' | jq

# batch
curl -s -X POST http://localhost:8080 \
  -H 'Content-Type: application/json' \
  -d '[{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]},{"jsonrpc":"2.0","id":2,"method":"eth_chainId","params":[]}]' | jq

# health
curl http://localhost:8080/health
```

## Testing

```sh
# unit tests only
go test -short ./...

# all tests including live integration against polygon.drpc.org
go test ./...
```

## Project layout

```
.
├── cmd/server/         # main entrypoint
├── pkg/
│   ├── config/         # YAML config loader with env substitution
│   ├── middleware/     # per-IP rate limiting middleware
│   └── proxy/          # core JSON-RPC proxy handler
├── config.yaml         # default configuration
└── Dockerfile          # multi-stage build, scratch final image
```
