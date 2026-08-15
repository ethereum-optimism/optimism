# Kona Node Service V2

`kona-node-service-v2` is an additive rewrite of `kona-node-service` around long-running async
workflows, explicit ownership, and narrow semantic interfaces. V1 remains intact as a behavioral
reference; the V2 binary does not compile or route through its actor runtime.

## Architecture

```text
                              node
                startup, task ownership, shutdown
                                │
                     shared canonical L1 watcher
                   ┌────────────┴────────────┐
                   │                         │
          independent L1 reader     independent L1 reader
                   │                         │
                   ▼                         ▼
             ┌──────────┐  safe/finalized  ┌────────────┐
             │  engine  │◀─────────────────│ derivation │
             └────┬─────┘                  └────────────┘
                  │ RPC query/admin adapters       │ reset adapter
                  └──────────────┬─────────────────┘
                                 ▼
                               ┌─────┐
                               │ RPC │
                               └─────┘
```

The three top-level domain services are:

- **Engine**: exclusively owns raw Engine API calls, authoritative forkchoice, unsafe validation,
  local sequencing, conductor authorization, gossip, P2P, safe reconciliation, and finalization.
- **Derivation**: owns local or delegated L1 derivation, finality mapping, L1 reorg recovery, and
  derivation-only pipeline reset. Its Engine capability exposes only `update_safe` and
  `update_finalized`.
- **RPC**: owns transport and routes narrow query/admin adapters without implementing domain logic.

Node owns a coherent L1-label watcher. Engine and Derivation use direct provider readers, local
caches, and independent canonical cursors; snapshots are catch-up targets rather than event logs.

Every service is an ordinary long-running `run` future. Node owns every top-level `JoinHandle`,
observes unexpected termination, closes lifecycle channels in dependency order, and awaits all
children through explicit shutdown. V2 does not use `NodeActor::step`, a generic actor lifecycle,
a cancellation-token hierarchy, or runtime-scope task detachment.
