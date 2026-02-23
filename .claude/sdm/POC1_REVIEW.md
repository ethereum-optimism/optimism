# SDM PoC 1 Review

5 February 2026

# Background reading

1. [A new gas model for the OP Stack](https://www.notion.so/external-draft-A-New-Gas-Model-for-the-OP-Stack-2c8f153ee16280ca9b23dbb4195c2bbb?pvs=21)
2. [Fee rebates](https://www.notion.so/Fee-Rebates-2c6f153ee162804ab80bfc48ea85095d?pvs=21)
3. [opgas - the Offchain Unit of Computation: Design Doc](https://raw.githubusercontent.com/ethereum-optimism/design-docs/4ce00b41708d9d06b2d3f43607c1c3ebca3f8b5c/protocol/opgas.md)

# Introduction

SDM PoC 1 is based on the `opgas` design doc linked above and is the first attempt at implementing a PoC with subjective sequencer gas metering. This particular PoC uses the wall-clock based approach described in the design doc.

# Artifacts

Three components from the OP Stack were changed as part of this PoC:

- op-geth
- op-node
- op-batcher
1. Source code:
    1. [optimism repo, branch `nonsense/opgas-geth`](https://github.com/ethereum-optimism/optimism/tree/nonsense/opgas-geth)
    2. [op-geth repo, branch `nonsense/opgas-geth`](https://github.com/ethereum-optimism/op-geth/tree/nonsense/opgas-geth)
2. Private alphanet - `anton-opgas-1`
    
    ```jsx
    op-batcher:
      version: us-docker.pkg.dev/oplabs-tools-artifacts/dev-images/op-batcher:0758307e89
    op-geth:
      version: us-docker.pkg.dev/oplabs-tools-artifacts/dev-images/op-geth:e7947b5a5
    op-node:
      version: us-docker.pkg.dev/oplabs-tools-artifacts/dev-images/op-node:0758307e89
    ```
    
3. Grafana dashboard - [Bedrock networks - anton-opgas-1](https://optimistic.grafana.net/d/nUSlc3d4k/bedrock-networks?orgId=1&from=now-12h&to=now&timezone=America%2FLos_Angeles&var-network=anton-opgas-1-0&var-node=$__all&var-layer=$__all&var-safety=$__all&var-cluster=$__all&var-konaNodes=$__all&refresh=30s)

# Notable changes to the OP Stack

### EIP-7623: Increase calldata cost is disabled

`core/state_transition.go`: `FloorDataGas` is disabled.

### EIP-3529: Reduction in refunds is disabled

`core/state_transition.go`: `calcRefund` is disabled.

### OPContainer struct

Contains a list of pairs `(tx index, opgas refund)` 

### OPContainer field introduced in multiple core types

1. `types.ExecutableData`
2. `types.Header`

### OPGasRefund field introduced in core types

1. `types.Receipt`

### Header serialization updates for JSON, RLP and SSZ

1. JSON marshaling at `internal/ethapi/api.go`
2. SSZ marshaling at `op-service/eth/ssz.go` used for transfer of blocks over gossip (`unsafe` chain) - PoC is not bumping up versioning (stays at `BlockV4`)
3. RLP marshaling of headers at `core/types/gen_header_rlp.go` 

### State transition function changes - introduction of OPGasRefund field

`evmgasToOpgas` calculation used to determine `OPGasRefund` for each sequenced transaction, based on `microseconds_used` and `peak gas used`

# Open questions and concerns

### Wall-clock execution

1. Wall-clock execution time is not a great measure. It could underprice memory allocation and create DoS vectors. it’s dangerous to omit non-computation factors that gas prices in.
2. Wall-clock execution time could lead to blocks that are unprovable because execution in sequencer could have very different performance characteristics in comparison to ZK VM or cannon. Dispute games could not resolve in time.
3. We need to systematically disable all expensive critical path operations, such as database compaction and Go’s STW garbage collection, so that we really measure execution time of transactions, and not some background process.

### Disabled gas refunds EIPs

1. Existing refund mechanisms cap refunds at 20% — EIP-3529 should possibly be disabled if we are to use the gas refund mechanisms from the EVM.
2. Floor Data gas price - EIP-7623 should possibly be disabled if we are to use the gas refund mechanisms from the EVM.

### Worst case gas per microsecond

1. At the moment we hardcode a worst case gas per second at 50M. How do we find and determine a more accurate number?

### Miscellaneous

1. Its unclear if we are addressing both “scaling” as well as “incentives/costs” for users. addressing both is harder and could lead to DoS vectors, addressing only “incentives/costs” (i.e. returning funds to users) reduces attack vectors, but sending ETH directly might introduce tax implications for users.
2. Its unclear what the second-order effects will be on EIP-1559 and on the base fee.
3. Reducing gas (i.e. addressing "scaling") would involve a hard-fork / one-way door. Improving "incentives/costs" could be done with a smart contract and would not require hard-fork, a lot simpler and potentially a feature that an OP chain operator could enable/disable.
4. Is there a way to release and deploy the `opgas` feature, where this decision is not a one-way door, but one we could later come back, revise and revert?
5. What’s the upgrade path for the OP Stack if we go forward with `opgas`? How would the future hard-forks look like?
    1. BlockV5 serialization format for SSZ
    2. OPContainer introduced into the Header in a backward-compatible manner
    3. New `SpanBatch` and `SingularBatch` versions, which contain the OPContainer for a given block range
6. If we go through with `opgas` do we need to implement it in all our clients - `op-reth` and `op-geth` and `kona` and `op-node` ?
7. How do we put `opgas` feature behind a feature flag, so we can deploy it only on one chain initially?
8. How do we communicate OPGasRefund from sequencer to other verifier nodes / batcher — via `header` field, or via `special` transaction? This PoC implements the `header` field modification.

# Next steps / How do we continue to make progress

1. Benchmarking - Backfill / replay 3-or-6-months worth of real-world blocks with transactions into a devnet and review the effects of `opgas` on `gas used` and other metrics. Answer the question of whether “good” users benefit from this change vs “bad” users?
2. Benchmarking - Deploy a copy of the XEN protocol and measure the difference of XEN transactions `gas used` and `gas used with opgas refunds`.
3. Benchmarking - Deploy a copy of Uniswap and measure the difference for user’s `swap` transactions.
4. **Red team** practice
    1. What’s the worst that can happen if we use wall-clock execution time to determine `gas used` as in this SDM PoC 1?
    2. Define contracts and transactions that abuse `opgas` refunds based on wall-clock execution time
    3. Devise a plan of what we could potentially do if this happens in the real-world where `opgas` is deployed and released? Can we dynamically / on-demand start/stop the `opgas` feature?