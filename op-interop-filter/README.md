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

## Rendered positions (`--render-transform-chains`)

By default the filter stores every log a block emitted, at the block-level index it actually had.
That is correct for any ordinary chain, where a log's position on the chain *is* its identity.

It is not correct for a **private chain in a public dependency set**. Such a chain keeps its blocks
private and publishes only its cross-chain messages, as a separate public "rendering" chain built by
replaying them. A message's identity — the `(block, log index)` that counterparties, judges,
relayers and tooling all reference — is its position on that *rendering*, not its position among the
private chain's own logs, which no one outside the operator can see. A filter storing raw private
positions files every message under a number nobody will ever cite, and then rejects the traffic it
was deployed to admit.

`--render-transform-chains` fixes that in process. For a listed chain, blocks and receipts are
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
counterparty chains ──► op-interop-filter          (stock ingestion, untouched)
own private chain   ──► op-interop-filter --render-transform-chains <own chain ID>
```

Both routes read a real node with full verification. Only the listed chain's logs are transformed,
and a counterparty chain must never be listed: its logs are already canonical as fetched.

**Why transform in process rather than wait for the rendering chain.** Admission gating happens while
the sequencer is building a block, so the filter needs its own chain's canonical message positions at
unsafe-head latency. Those positions only become readable from the rendering chain itself after the
private chain's messages have been batched, posted and derived — a claim-cadence round trip that
admission cannot wait for. The rendering transformation is a pure function of the private block, so
applying it in place gives the same positions immediately.

### Emitter set

`--render-extra-emitters` adds emitter addresses on top of the two standard interop predeploys,
which are always included. This is a genesis-time property of the chain, not a per-message policy,
and it **must match the rendering builder's configuration**: the filter and the builder derive the
same positions only if they filter by the same rule, and a mismatch silently renumbers messages.

### Failure containment

A misconfiguration here is contained to the listed chain. A chain's ingested data is only consulted
to decide whether that chain's own executing messages may be admitted, so the blast radius of a
wrong emitter set is the operator's own sequencer admitting or rejecting its own transactions — it
cannot corrupt another chain's ingested data. `Config.Check` rejects a listed chain that has no
rollup config, which is the one form of this mistake that would otherwise be silent.

Both flags are dormant unless set, and every listed chain is logged as a warning at startup.
