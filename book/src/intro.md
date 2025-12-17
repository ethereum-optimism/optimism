# Rollup Boost

Rollup Boost is a lightweight sidecar for rollups that enables rollup extensions. These extensions provide an efficient, decentralized, and verifiable block building platform for rollups.

## What is Rollup Boost?

[Rollup Boost](https://github.com/flashbots/rollup-boost/) is a sequencer sidecar that uses the [Engine API](https://specs.optimism.io/protocol/exec-engine.html#engine-api) in the Optimism stack to enable its rollup extensions. Its scalable modules allow for faster confirmation times, stronger user guarantees, and more.

It requires no modification to the OP stack software and allows rollup operators to connect to an external builder.

It is designed and developed by [Flashbots](https://flashbots.net/) under a MIT license. We invite developers, rollup operators, and researchers to join in developing this open-source software to achieve efficiency and decentralization in the rollup ecosystem.

## Who is this for?

Rollup Boost is designed for:

- **Rollup Operators**: Teams running OP Stack or compatible rollups who want to improve efficiency, reduce confirmation times and have more control over block construction.

- **Rollup Developers**: Engineers building on rollups to unlock new use cases with rollup extensions.

- **Block Builders**: Teams focused on MEV who want to offer specialized block building services for rollups.

- **Researchers**: Those exploring new approaches to MEV and block production strategies.

## What are the design goals of Rollup Boost?

**Simplicity**

Rollup Boost was designed with minimal complexity, maintaining a lightweight setup that avoids additional latency. 

By focusing on simplicity, Rollup Boost integrates smoothly into the OP stack without disrupting existing op-node and execution engine communication.

This way, we prioritize the reliability of the block proposal hot path over feature richness. We achieve this by striving for a stateless design and limit scope creep.

**Modularity**

Recognizing that rollups have varying needs, Rollup Boost was built with extensibility in mind. It is designed to support custom block-building features, allowing operators to implement modifications for custom block selection rules.

**Liveness Protection**

To safeguard against liveness risks, Rollup Boost offers a local block production fallback if there is a failure in the external builder.

Rollup Boost also guards against invalid blocks from the builder by validating the blocks against the local execution engine. This ensures the proposer always have a fallback path and minimizes liveness risks of the chain.

All modules in Rollup Boost are designed and tested with a fallback scenario in mind to maintain high performance and uptime.

**Compatibility**

Rollup Boost allows operators to continue using the standard `op-node` and `op-geth` / `op-reth` software without any custom forks. 

It is also compatible with `op-conductor` when running sequencers with a high availability setup.

It aims to be on parity with the performance of using a vanilla proposer setup without Rollup Boost.

<br>

For more details on the design philosophy and testing strategy of Rollup Boost, see this [doc](https://github.com/flashbots/rollup-boost/blob/main/docs/design-philosophy-testing-strategy.md).

## Is it secure?

Rollup Boost underwent a security audit on May 11, 2025, with [Nethermind](https://www.nethermind.io). See the full report [here](https://github.com/flashbots/rollup-boost/blob/main/docs/NM_0411_0491_Security_Review_World_Rollup_Boost.pdf).

In addition to the audit, we have an extensive suite of integration tests in our CI to ensure the whole flow works end-to-end with the OP stack as well as with external builders.

## Getting Help

- GitHub Issues: Open an [issue](https://github.com/flashbots/rollup-boost/issues/new) for bugs or feature requests
- Forum: Join the [forum](https://collective.flashbots.net/c/rollup-boost) for development updates and research discussions

## Sections

Here are some useful sections to jump to:

- Run Rollup Boost with the full Op stack setup locally by following this [guide](./operators/local.md).
- Read about how [flashblocks](./modules/flashblocks.md) work in Rollup Boost
- Query the [JSON-RPC](./developers/flashblocks-rpc.md) of a flashblocks enabled node Foundry's `cast` or `curl`.