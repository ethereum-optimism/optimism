# @eth-optimism/endpoint-monitor

The endpoint-monitor runs websocket checks on edge-proxyd endpoints and downstream infra provider endpoints.

## Setup

Install go1.19

```bash
make build

source .env.example # (or copy to .envrc if using direnv)
./bin/endpoint-monitor
```
