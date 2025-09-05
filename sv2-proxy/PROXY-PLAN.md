## SV2 Transparent Proxy — Plan

### Goal
- Replace the legacy `op-node` container in Kurtosis deployments with a fully transparent, bidirectional proxy (`sv2-proxy`).
- Keep all existing wiring and ports unchanged for consumers.
- Route traffic so that `supervisor-v2` replaces op-node’s behavior without other services noticing.

### Context
- Current interop wiring:
  - supervisor v1 ↔ op-node ↔ EL
- Target minimal-diff wiring:
  - supervisor v2 ↔ sv2-proxy ↔ EL
- The proxy must forward requests 1:1 (HTTP and WebSocket), preserving headers, bodies, and JWT where applicable, with no payload translation.
- Supervisor v2 already exposes an embedded op-node user-RPC via an HTTP reverse proxy at `/opnode/{chainId}/`. We will leverage that as the upstream for rollup RPC consumers that formerly talked to the standalone op-node.

### Surfaces to Proxy (transparent pass-through)
1) Rollup RPC (was op-node user RPC)
   - Downstream clients (batcher/proposer/tools) connect to the same host:port as before.
   - Upstream target: supervisor-v2 at `http://sv2-host:sv2-port/opnode/{chainId}/`.
   - Requirements: HTTP 1.1 + WebSocket upgrades; no path/body rewrites beyond the base-path mapping.

2) EL User RPC (e.g., 8545)
   - Downstream clients connect exactly as before.
   - Upstream target: the actual EL user RPC.
   - Requirements: HTTP 1.1 + optional WebSocket upgrades; no auth handling beyond pass-through.

3) EL Auth RPC (e.g., 8551)
   - Downstream producer (engine API clients) connect as before.
   - Upstream target: the actual EL auth RPC (engine API).
   - Requirements: Preserve Authorization/JWT headers, HTTP keep-alives; no JWT validation at proxy.

4) EL → Supervisor Interop RPC
   - If EL is configured to call an interop/supervisor endpoint (e.g., mempool filtering), the proxy exposes the same endpoint and forwards to supervisor v2 (JSON-RPC over HTTP). If supervisor v2 offers only HTTP endpoints for some checks, we can add a minimal translator later; initial version will focus on direct pass-through if an equivalent JSON-RPC exists.

### Behavior
- Fully transparent TCP/HTTP/WS forwarding in both directions.
- No schema/method translation; no authentication at the proxy; no buffering beyond stream forwarding.
- Health: optional `/healthz` that only checks listener readiness (no upstream probing by default).

### Configuration
- Static bind ports to match legacy `op-node` and EL endpoints (from Kurtosis template):
  - rollup RPC (op-node replacement): same port as prior op-node RPC
  - EL user RPC: 8545 (or template value)
  - EL auth RPC: 8551 (or template value)
  - interop (if present): same port as prior interop endpoint
- Env vars for upstreams:
  - `SV2_PROXY_CHAIN_ID` (required for rollup RPC path)
  - `SV2_PROXY_SUPERVISOR_URL` (e.g., `http://sv2:9750`)
  - `SV2_PROXY_EL_USERRPC` (e.g., `http://el:8545`)
  - `SV2_PROXY_EL_AUTHRPC` (e.g., `http://el:8551`)
  - `SV2_PROXY_INTEROP_UPSTREAM` (optional, for EL→SV2 interop)
  - Optional: `SV2_PROXY_LISTEN_ADDR_*` and `SV2_PROXY_LISTEN_PORT_*` overrides per surface

### Implementation Outline
- Language: Go (net/http reverse proxy + TCP passthrough for WS).
- Components:
  - HTTP reverse proxy handler for rollup RPC → supervisor v2 (`/opnode/{chainId}/` base-path).
  - HTTP reverse proxy for EL user RPC (supports WS upgrade).
  - HTTP reverse proxy for EL auth RPC (preserve Authorization header; supports WS if used).
  - Optional HTTP reverse proxy for interop → supervisor v2.
  - Simple health endpoint.
- Start-up:
  - Read env config, validate URLs/ports, bind listeners.
  - Log mappings (downstream → upstream).

### Kurtosis Changes (minimal diffs)
- In `kurtosis-devnet/interop.yaml`:
  - Swap `op-node` image with `sv2-proxy` image but keep service name and ports.
  - Swap `op-supervisor` image to `op-supervisor-v2` and add its extra params, including `sv2.config` mount if needed.
  - Point supervisor v2 L2 RPCs (auth/user) to the proxy endpoints.
  - Keep EL configurations and ports unchanged.

### Risks & Mitigations
- WebSocket support: Ensure reverse proxy handles WS correctly (use `httputil` with proper upgrade headers or a library that supports WS proxying).
- JWT pass-through: Do not alter headers; treat as opaque.
- Path mapping: Only apply for rollup RPC to sv2 (`/opnode/{chainId}/`); others are root-to-root.
- Backpressure/large payloads: Use streaming io.Copy where needed; avoid buffering.

### Testing Plan
- Unit: proxy config parsing; URL/port validation; basic handler routing.
- Integration (local):
  - Rollup RPC: `eth_chainId`, `eth_blockNumber` via proxy against supervisor v2.
  - EL RPCs: `eth_blockNumber` and basic tx submission via proxy to EL.
  - Auth RPC: simple `engine_` method smoke via proxy with JWT header.
  - WebSocket: subscribe/filter logs via WS proxy.
- Kurtosis smoke: deploy interop.yaml variant and run existing health checks (no consumer changes).

### Future Enhancements
- Optional JSON-RPC compatibility shim (supervisor v1 method names) if EL requires supervisor_checkAccessList.
- Metrics: basic counters and latencies per surface.
- mTLS between proxy and upstreams if required later.


