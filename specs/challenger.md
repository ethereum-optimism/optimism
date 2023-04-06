# Challenger Specification

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Description](#description)
- [Terminology](#terminology)
- [Event and Response Lifecycle](#event-and-response-lifecycle)
  - [`GameType.FAULT`](#gametypefault)
  - [`GameType.VALIDITY`](#gametypevalidity)
  - [`GameType.ATTESTATION`](#gametypeattestation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Description

The Challenger is an off-chain agent that listens for faulty claims made about the state of
the L2 on the data availability layer. It is responsible for challenging these incorrect claims
and ensuring the correctness of all finalized claims on the data availability layer.

The Challenger agent is intended to be ran as a permissionless service by participants of the network
alongside an [rollup-node](./rollup-node.md). Challenger agents will be rewarded in the form of the
bond attached to the claims they disprove.

## Terminology

- **data availability layer** - In the context of this document, the data availability layer is the
  generic term for the location where claims about the state of the layer two are made. In the context
  of Optimism, this is Ethereum Mainnet.
- **L2** - In the context of this document, the layer two of the Optimistic Rollup. In the context
  of Optimism, this is the Optimism Mainnet.
- **rollup-node** - In the context of this document, the rollup node describes the
  [rollup-node specification](./rollup-node.md). In the context of Optimism, this is the implementation
  of the [rollup-node specification](./rollup-node.md), the `op-node`.

## Event and Response Lifecycle

The Challenger agent is expected to be able to listen for and respond to several different events
on the data availability layer. These events and responses differ depending on the type of dispute game being
played. For more information on the separate types of dispute games, see the
[Dispute Game Interfaces specification](./dispute-game-interfaces.md) and
[Dispute Game specification](./dispute-game.md).

### `GameType.FAULT`

**TODO**

### `GameType.VALIDITY`

**TODO**

### `GameType.ATTESTATION`

**Events and Responses**

- [`L2OutputOracle.OutputProposed`](../packages/contracts-bedrock/contracts/L1/L2OutputOracle.sol#L57-70)
  The L2OutputOracle contract emits this event when a new output is proposed on the data availability
  layer. Each time an output is proposed, the Challenger should check to see if the output is equal
  the output given by the `optimism_outputAtBlock` endpoint of their `rollup-node`.
  - If it is, the Challenger should do nothing to challenge this output proposal.
  - If it is not, the Challenger should respond by creating a new `DisputeGame` with the
    `DisputeGameType.ATTESTATION` `gameType`, the correct output root as the `rootClaim`, and the abi-encoded
    `l2BlockNumber` of the correct output root as the `extraData`.
- `DisputeGameFactory.DisputeGameCreated` - A new dispute game has been created and is ready to be reviewed. The
  Challenger agent should listen for this event and check if the `rootClaim` of the `AttestationDisputeGame`
  created by the `DisputeGameFactory` is equal to the output root of their `rollup-node` at the game's `l2BlockNumber`.
  - If it is, the Challenger should sign the [EIP-712 typeHash](./dispute-game.md) of the struct containing the
    `AttestationDisputeGame`'s `rootClaim` and `l2BlockNumber`. The Challenger should then submit the abi-encoded
    signature to the `AttetationDisputeGame`'s `challenge` function.
  - If it is not, the Challenger should do nothing in support of this dispute game.

A full diagram and lifecycle of the Challenger's role in the `ATTESTATION` game type can be found below:

![Attestation Diagram](./assets/challenger_attestation.png)
