<!-- DOCTOC SKIP -->
# Optimism Bedrock specs

This directory contains the plain english specs for Optimism, a minimal optimistic rollup protocol
that maintains 1:1 compatibility with Ethereum.

## Specification Contents

- [Introduction](introduction.md)
- [Overview](overview.md)
- [Deposits](deposits.md)
- [Withdrawals](withdrawals.md)
- [Execution Engine](exec-engine.md)
- [L2 Output Root Proposals](proposals.md)
- [Rollup Node](rollup-node.md)
- [Rollup Node P2p](rollup-node-p2p.md)
- [L2 Chain Derivation](derivation.md)
- [Network Upgrades](network-upgrades.md)
- [System Config](system_config.md)
- [Batch Submitter](batcher.md)
- [Guaranteed Gas Market](guaranteed-gas-market.md)
- [Messengers](messengers.md)
- [Bridges](bridges.md)
- [Predeploys](predeploys.md)
- [Glossary](glossary.md)

## Design Goals

Our aim is to design a protocol specification that is:

- **Fast:** When users send transactions, they get reliable confirmations with low-latency.
  For example when swapping on Uniswap you should see that your transaction succeed in less than 2
  seconds.
- **Scalable:** It should be possible to handle an enormous number of transactions
  per second which will enable the system to charge low fees.
  V1.0 will enable Optimism to scale up to and even past the gas limit on L1.
  Later iterations should scale much further.
- **Modular:** Our designs will use modularity to reduce complexity and enable parallel
  contributions. Coming up with good conceptual frameworks & composable atoms of software enables us
  to build extremely complex software even when any one person cannot hold that much in their brain.
- **Minimal:** Rollups should be minimal to best take advantage of the battle-tested infrastructure
  (like Geth) that already runs Ethereum. An ideal optimistic rollup design should be representable
  as a *diff* against Ethereum client software.
- **Developer Driven:** Our designs will be developer driven to ensure we are actually building
  something that people want to use. We must constantly engage with the developers who will be using
  our software to avoid creating a system no one wants to use.
- **Clear and Readable:** The specs we write are written to be read. So tight feedback loop with the
  systems team consuming the spec is also key!
- **Secure:** This is self-evident.
  User’s assets are at stake. Every component of the system must be incredibly secure.
- **Decentralizable:** Optimism must be designed to avail itself of the security and
  censorship-resistant guarantees achieved by a decentralized system.
  Currently centralized components of the system should have a clear path towards decentralization.
  Already decentralized components of the system should be protected and preserved.
