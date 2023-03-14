---
title: Featured Hacks
lang: en-US
---

::: warning 🚧 OP Stack Hacks are explicitly things that you can do with the OP Stack that are *not* currently intended for production use

OP Stack Hacks are not for the faint of heart. You will not be able to receive significant developer support for OP Stack Hacks — be prepared to get your hands dirty and to work without support.

:::

## Overview

Featured Hacks is a compilation of some of the cool stuff people are building on top of the OP Stack!

## OPCraft

### Author

[Lattice](https://lattice.xyz/)

### Description

OPCraft was an OP Stack chain that ran a modified EVM as the backend for a fully onchain 3D voxel game built with [MUD](https://mud.dev/).

### OP Stack Configuration

- Data Availability: Ethereum DA (Goerli)
- Sequencer: Single Sequencer
- Derivation: Standard Rollup
- Execution: Modified Rollup EVM

### Links

- [Announcing OPCraft: an Autonomous World built on the OP Stack](https://dev.optimism.io/opcraft-autonomous-world/)
- [OPCraft Explorer](https://opcraft.mud.dev/)
- [OPCraft on GitHub](https://github.com/latticexyz/opcraft)
- [MUD](https://mud.dev/)

## Ticking Optimism

### Author

[@therealbytes](https://twitter.com/therealbytes)

### Description

Ticking Optimism is a proof-of-concept implementation of an OP Stack chain that calls a `tick` function every block. By using the OP Stack, Ticking Optimism avoids the need for off-chain infrastructure to execute a function on a regular basis. Ticking Conway is a system that uses Ticking Optimism to build [Conway’s Game of Life](https://conwaylife.com/) onchain.

### OP Stack Configuration

- Data Availability: Ethereum DA (any)
- Sequencer: Single Sequencer
- Derivation: Standard Rollup with custom `tick` function
- Execution: Rollup EVM

### Links

- [Ticking Optimism on GitHub](https://github.com/therealbytes/ticking-optimism)
- [Ticking Conway on GitHub](https://github.com/therealbytes/ticking-conway)