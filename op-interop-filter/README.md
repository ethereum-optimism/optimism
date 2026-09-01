# op-interop-filter

A lightweight service that validates interop executing messages for op-geth or op-reth transaction filtering.

Any reorg will trigger the failsafe which disables all interop transactions. If `--reorg-recovery-enabled` is set, reorg-triggered failsafe is automatically resolved by rewinding logs DB state to finalized.

## Usage

### Build from source

```bash
just op-interop-filter
./bin/op-interop-filter --help
```

### Run from source

```bash
go run ./cmd --help
```

### Build docker image

```bash
docker buildx bake op-interop-filter
```

## Rendered positions

By default the filter stores every log a block emitted, at the block-level index it actually had.
That is correct for any ordinary chain, where a log's position on the chain *is* its identity.

It is not correct for a **private chain in a public dependency set**. Such a chain keeps its blocks
private and publishes only its cross-chain messages, as a separate public "rendering" chain built by
replaying them. A message's identity — the `(block, log index)` that counterparties, judges,
relayers and tooling all reference — is its position on that *rendering*, not its position among the
private chain's own logs, which no one outside the operator can see. A filter storing raw private
positions files every message under a number nobody will ever cite, and then rejects the traffic it
was deployed to admit.

The generated rendering `rollup.json` fixes that in process. When a chain's rollup config contains
the `private_interop` section, blocks and receipts are
fetched and verified exactly as for every other chain — the source is an ordinary execution client
serving self-consistent blocks, and nothing about verification is relaxed. The transformation is
then applied to the already-verified logs, between the fetch and the logs DB:

- logs outside the emitter set are dropped;
- the survivors keep their original interleaved order and are renumbered densely from zero;
- the block is still sealed under its **real** hash, with its real number and timestamp.

Only the log sequence changes. In short: store the real L2 block, but only the right logs, at the
right indexes.

The transformation is `render.RenderedLogs` from `op-private-interop/render` — imported, never
reimplemented. It is the same call the rendering builder makes when it constructs the replay
transactions, and the filter's admission decisions only mean anything if the two agree about every
index.

### Topology for a private chain's sequencer

```
counterparty chains ──► op-interop-filter  (ordinary rollup config; stock ingestion)
own private chain   ──► op-interop-filter  (rendering rollup config; rendered positions)
```

Both routes read a real node with full verification. Only a chain whose generated config marks it
as a rendering is transformed; counterparty configs are ordinary, so their logs remain canonical
as fetched.

**Why transform in process rather than wait for the rendering chain.** Admission gating happens while
the sequencer is building a block, so the filter needs its own chain's canonical message positions at
unsafe-head latency. Those positions only become readable from the rendering chain itself after the
private chain's messages have been batched, posted and derived — a claim-cadence round trip that
admission cannot wait for. The rendering transformation is a pure function of the private block, so
applying it in place gives the same positions immediately.

### Emitter set

The rendering config's `private_interop.extra_emitters` adds emitter addresses on top of the two
standard interop predeploys, which are always included. It is generated from the deployment intent
and is the sole runtime source for both this filter and the rendering batcher; neither process
accepts an independent override that could silently renumber messages.

### Failure containment

A misconfiguration here is contained to the rendering chain. A chain's ingested data is only consulted
to decide whether that chain's own executing messages may be admitted, so the blast radius of a
wrong emitter set is the operator's own sequencer admitting or rejecting its own transactions — it
cannot corrupt another chain's ingested data. Every config-activated transform is logged as a
warning at startup.
