# Kona Node Service V2

`kona-node-service-v2` is an additive rewrite of `kona-node-service` around long-running async workflows and narrow semantic interfaces.

The original service remains the production implementation while V2 is developed and compared against it. V2 intentionally does not depend on the original service's actor framework.

## Architecture

```text
                         ┌──▶ unsafe_chain ──▶ engine
l1 ──────────────────────┤
                         └──▶ safe_chain ────▶ engine

network ─────────────────────▶ unsafe_chain
rpc ─────────────────────────▶ subsystem control handles
```

- `engine` exclusively owns raw Engine API calls and forkchoice state.
- `unsafe_chain` acquires unsafe blocks from local sequencing or network gossip.
- `safe_chain` derives safe and finalized L2 progress from L1.
- `l1` provides shared L1 access while unsafe and safe processing maintain independent cursors.
- `node` is the composition root and task supervisor.

Each service is expressed as a normal long-running `run` future. Private bounded channels may be used to provide narrow cloneable clients, but V2 does not use `NodeActor::step` or a generic actor lifecycle.

## Migration

V2 is implemented in vertical slices while V1 remains intact. Removal and production cutover are explicitly deferred until behavioral parity has been demonstrated.

The crate temporarily contains a copied V1 runtime so a `kona-node-v2` binary can be exercised by the same acceptance suite from the start. New subsystem modules replace that copied implementation one at a time; they do not wrap or modify the original `kona-node-service` crate.
