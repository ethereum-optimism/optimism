# External Block Building

Much like [mev-boost](https://github.com/flashbots/mev-boost) on Ethereum layer 1, rollup-boost is a sidecar that runs alongside the sequencer in Optimism chains to source blocks from external block builders. Unlike mev-boost, there are no code changes or extra config to support external block building in the consensus node as rollup-boost reuses the existing Engine API to source external blocks.

![architecture](https://raw.githubusercontent.com/flashbots/rollup-boost/refs/heads/main/assets/rollup-boost-architecture.png)

Instead of pointing to the local execution client, the rollup operator simply needs to point the consensus node to rollup-boost and configure rollup-boost to connect to both the local execution client and the external block builder. 

To read more about the design, check out the [design doc](https://github.com/ethereum-optimism/design-docs/blob/main/protocol/external-block-production.md) for how rollup-boost integrates into the Optimism stack.
