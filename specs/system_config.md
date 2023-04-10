# System Config

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [System config contents (version 0)](#system-config-contents-version-0)
  - [`batcherHash` (`bytes32`)](#batcherhash-bytes32)
  - [`overhead` and `scalar` (`uint256,uint256`)](#overhead-and-scalar-uint256uint256)
  - [`gasLimit` (`uint64`)](#gaslimit-uint64)
  - [`unsafeBlockSigner` (`address`)](#unsafeblocksigner-address)
- [Writing the system config](#writing-the-system-config)
- [Reading the system config](#reading-the-system-config)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The `SystemConfig` is a contract on L1 that can emit rollup configuration changes as log events.
The rollup [block derivation process](./derivation.md) picks up on these log events and applies the changes.

## System config contents (version 0)

Version 0 of the system configuration contract defines the following parameters:

### `batcherHash` (`bytes32`)

A versioned hash of the current authorized batcher sender(s), to rotate keys as batch-submitter.
The first byte identifies the version.

Version `0` embeds the current batch submitter ethereum address (`bytes20`) in the last 20 bytes of the versioned hash.

In the future this versioned hash may become a commitment to a more extensive configuration,
to enable more extensive redundancy and/or rotation configurations.

### `overhead` and `scalar` (`uint256,uint256`)

The L1 fee parameters, also known as Gas Price Oracle (GPO) parameters,
are updated in conjunction and apply new L1 costs to the L2 transactions.

### `gasLimit` (`uint64`)

The gas limit of the L2 blocks is configured through the system config.
Changes to the L2 gas limit are fully applied in the first L2 block with the L1 origin that introduced the change,
as opposed to the 1/1024 adjustments towards a target as seen in limit updates of L1 blocks.

### `unsafeBlockSigner` (`address`)

Blocks are gossiped around the p2p network before they are made available on L1.
To prevent denial of service on the p2p layer, these unsafe blocks must be
signed with a particular key to be accepted as "canonical" unsafe blocks.
The address corresponding to this key is the `unsafeBlockSigner`. To ensure
that its value can be fetched with a storage proof in a storage layout independent
manner, it is stored at a special storage slot corresponding to
`keccak256("systemconfig.unsafeblocksigner")`.

Unlike the other values, the `unsafeBlockSigner` only operates on blockchain
policy. It is not a consensus level parameter.

## Writing the system config

The `SystemConfig` contract applies authentication to all writing contract functions,
the configuration management can be configured to be any type of ethereum account or contract.

On a write, an event is emitted for the change to be picked up by the L2 system,
and a copy of the new written configuration variable is retained in L1 state to read with L1 contracts.

## Reading the system config

A rollup node initializes its derivation process by finding a starting point based on its past L2 chain:

- When started from L2 genesis, the initial system configuration is retrieved from the rollup chain configuration.
- When started from an existing L2 chain, a previously included L1 block is determined as derivation starting point,
  and the system config can thus be retrieved from the last L2 block that referenced the L1 block as L1 origin:
  - `batcherHash`, `overhead` and `scalar` are retrieved from the L1 block info transaction.
  - `gasLimit` is retrieved from the L2 block header.
  - other future variables may also be retrieved from other contents of the L2 block, such as the header.

After preparing the initial system configuration for the given L1 starting input,
the system configuration is updated by processing all receipts from each new L1 block.

The contained log events are filtered and processed as follows:

- the log event contract address must match the rollup `SystemConfig` deployment
- the first log event topic must match the ABI hash of `ConfigUpdate(uint256,uint8,bytes)`
- the second topic determines the version. Unknown versions are critical derivation errors.
- the third topic determines the type of update. Unknown types are critical derivation errors.
- the remaining event data is opaque, encoded as ABI bytes (i.e. includes offset and length data),
  and encodes the configuration update. In version `0` the following types are supported:
  - type `0`: `batcherHash` overwrite, as `bytes32` payload.
  - type `1`: `overhead` and `scalar` overwrite, as two packed `uint256` entries.
  - type `2`: `gasLimit` overwrite, as `uint64` payload.
  - type `3`: `unsafeBlockSigner` overwrite, as `address` payload.

Note that individual derivation stages may be processing different L1 blocks,
and should thus maintain individual system configuration copies,
and apply the event-based changes as the stage traverses to the next L1 block.
