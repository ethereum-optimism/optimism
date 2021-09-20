# Optimistic Roadmap
This document contains the Optimistic Ethereum protocol roadmap. That includes the project goals and a high level roadmap diagram.

## Goals
The fundamental goal of the Optimistic Ethereum protocol is to scale Ethereum while preserving the security properties of L1. To achieve this we must create:

### Ultra-Minimal Rollup Node
A rollup client, expressible as a minimal diff against Ethereum. Minimalism is key because it enables compatibility with existing Ethereum tooling as well as security.

#### Completion Criteria
1. 1:1 equivalence with the Ethereum yellow paper.
    - This may require changes to Ethereum in order to natively support rollups (eg. the merge API).
2. Multi-client support.
3. Minimal batch submission cost including the integration of eth2 sharding when ready.
4. Support for sequencer consensus.

### Pluggable State Oracle
A set of pluggable contracts for proving L2 rollup state to L1. Pluggability is key because it will allow parallel development across the OE rollup stack.

#### Completion Criteria
1. Proposal manager capable of being managed by any dispute game.
    - Should eventually support validity proofs.
2. Dispute game capable of adjudicating any fraud proof VM.

### Future Proof Fraud Proof VM
Create a VM built for executing fraud proofs against an EVM equivalent rollup. Future proof-ness is key because the fraud proof must evolve as the Ethereum protocol evolves.

#### Completion Criteria
1. Must correctly implement the Optimistic Ethereum protocol defined in the Ultra-Minimal Rollup Node specs.
2. Easy to update as Ethereum hard forks are released.
3. Divergences between client implementations and the fraud proof are gracefully handled.

## Roadmap Diagram
We will be working towards these goals iteratively. The following diagram is an approximation of what the ordering of work will look like and what streams of work can be parallelized:

![Roadmap Diagram](./assets/roadmap.svg)

The roadmap & abstractions are designed to enable independent development of each component. The 4 major components are:

1. the optimistic mainnet deployment,
2. the fraud proof infrastructure,
3. stateless clients,
4. sharding.

Each component will produce incremental and independent releases, each driving closer to unification and Optimistic Ethereum nirvana.

### Summary of each component

#### Optimism Mainnet
Optimism mainnet is the first Optimistic Ethereum network. It serves as:

1. A live deployment of the latest stable Optimistic Ethereum spec; and
2. The source of funding of Optimistic Ethereum protocol development and other Ethereum public goods.

This network will start out as being a rollup using eth1 as a data availability engine, and will migrate once sharding is ready.

#### Fraud Proof
The fraud proof contracts and infrastructure are a pluggable set of EVM smart contracts, off-chain witness generators, challenge agents, and more. The near term fraud proof design is a 1:1 implementation of the Ethereum protocol written in the EVM. In the medium to long term, after statelessness is introduced to Ethereum (see the next project), the fraud proof should become entirely native so that Ethereum clients remain fully compatible with layer 2.

#### eth1.x stateless clients
State expiry and stateless clients are key to solving the Ethereum state bloat problem **and** for native EVM fraud proofs. Once stateless clients are ready we can transition the Optimistic Ethereum fraud proof to a native component of the EVM.

Note: Although critical to the OE roadmap, this project is tracked [external](https://github.com/ethereum/stateless-ethereum-specs/) to this repository.

#### eth2
The eth2 merge API and sharding will allow for native integration into execution engines as well as massive scalability ([~100k TPS](https://vitalik.ca/general/2021/01/05/rollup.html)) by greatly increasing Ethereum's data availability bandwidth.

Note: Although critical to the OE roadmap, this project is tracked [external](https://github.com/ethereum/consensus-specs) to this repository.

***Combining the eth2 merge API, native fraud proofs, and sharding - we'll reach Optimistic Ethereum nirvana.***
