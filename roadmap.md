# Optimistic Roadmap

![Roadmap Diagram](./assets/roadmap.svg)

## Design Goals

The roadmap & abstractions are designed to enable the independent development of each component. The 4 major components are: 
1. the optimistic mainnet deployment,
2. the fraud proof infrastructure,
3. stateless clients, 
4. sharding. 

Each component will produce incremental and independent releases, each driving closer to unification and Optimistic Ethereum nirvana.

## Summary of each component

### Optimism Mainnet

Optimism mainnet is the first Optimistic Ethereum network. It serves as:

1. A live deployment of the latest stable Optimistic Ethereum spec; and
2. The source of funding of Optimistic Ethereum protocol development and other Ethereum public goods.

This network will start out as being a rollup using eth1 as a data availability engine, and will migrate once sharding is ready.

### Fraud Proof

The fraud proof contracts and infrastructure are a pluggable set of EVM smart contracts, off-chain witness generators, challenge agents, and more. The near term fraud proof design is a 1:1 implementation of the Ethereum protocol written in the EVM. In the medium to long term, after statelessness is introduced to Ethereum (see the next project), the fraud proof should become entirely native so that Ethereum clients remain fully compatible with layer 2.

### eth1.x stateless clients

State expiry and stateless clients are key to solving the Ethereum state bloat problem **and** for native EVM fraud proofs. Once stateless clients are ready we can transition the Optimistic Ethereum fraud proof to a native component of the EVM.

Note: Although critical to the OE roadmap, this project is tracked [external](https://github.com/ethereum/stateless-ethereum-specs/) to this repository.

### eth2

The eth2 merge API and sharding will allow for native integration into execution engines as well as massive scalability ([~100k TPS](https://vitalik.ca/general/2021/01/05/rollup.html)) by greatly increasing Ethereum's data availability bandwidth.

Note: Although critical to the OE roadmap, this project is tracked [external](https://github.com/ethereum/consensus-specs) to this repository.

***Combining the eth2 merge API, native fraud proofs, and sharding - we'll reach Optimistic Ethereum nirvana.***
