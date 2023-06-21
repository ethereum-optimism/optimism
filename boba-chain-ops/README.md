# boba-chain-ops

## boba-migrate
This package performs state regenesis. It takes the following input:

1. An `alloaction.json` file that contains a list of pre-allocated accounts.
2. A `genesis.json` file that contains the genesis block configuration.
3. A list of addresses that transacted on the network prior to this past regenesis.
4. A list of addresses that performed approvals on prior versions of the OVM ETH contract.
5. A list of msg information that stores the cross domain message from L2 to L1

It creates an initialized Bedrock erigon database as output. It does this by performing the following steps:

1. Create genesis (types.Genesis) from `genesis.json` and `allocation.json` in memory.
2. Set up the proxy contract for the predeploy contracts and update the bytecode of the predeploy contracts.
3. Migrate Boba legacy proxy implementation contract the new proxy contract and delete the slots
4. Migrate the ETH balance from the storage of OVM ETH contract to the balance field in genesis and delete the ETH balance storage slot.
5. Migrate the msg information from the storage of OVM_CrossDomainMessenger contract to the new slot.

It performs the following integrity checks:

1. OVM ETH storage slots must be completely accounted for.
2. The total supply of OVM ETH migrated must match the total supply of the OVM ETH contract.

### Compilation

Run `make boba-migrate`

## boba-rollover

This package performs state regenesis for creating a legacy chain in erigon. The new chain is only readable and is not compatible with v3. It takes the following input:

1. An `alloaction.json` file that contains a list of pre-allocated accounts.
2. A `genesis.json` file that contains the genesis block configuration.

3. A list of addresses that transacted on the network prior to this past regenesis.

It creates an initialized erigon database as output. It does this by performing the following steps:

1. Migrate the ETH balance from the storage of OVM ETH contract to the balance field in genesis and delete the ETH balance storage slot.

It performs the following integrity checks:

1. OVM ETH storage slots must be completely accounted for.
2. The total supply of OVM ETH migrated must match the total supply of the OVM ETH contract.

### Compilation

Run `make boba-rollover`

## boba-regenerate

This pacakge performs the chain regeneration via calling engine api. It does this by performing the following steps:

1. Call the legacy block chain to get the block and transaction information.
2. Build `PayloadAttributes` and call `engine_forkchoiceUpdatedV1` to get `PayloadID`
3. Call `engine_getPayloadV1` with the `PayloadID` from the last step to get `executionPayload`
4. Call `engine_newPayloadV1` with the `executionPayload` to execute the transaction
5. Call `engine_forkchoiceUpdatedV1` to build the block

It performs the following integrity checks:

1. The block hash and transaction hash must match the legacy block chain

### Compilation

Run `make boba-regenerate`

## boba-crawler

This package performs the process of getting addresses that send or receive ETH from the legacy block chain. It does this by performing the following steps:

1. Call `debug_traceTransaction` to find out addresses that send and receive ETH from internal transactions.
2. Call `eth_getLogs` to find out addresses that receive ETH from the ` ETH Mint` event

It performs the following integrity checks:

1. The address list can be used to compute the all storage keys of `OVM_ETH` contract from the allocation file.

### Compilation

Run `make boba-crawler`

## boba-devnet

This package generates a clean genesis file for devent. It only includes the predeployed contracts for L2. It takes the following input to generate the genesis file:

1. The deployment configuration for the l2
2. The hardhat deployment path
3. The l1 PRC endpoint for quering the block information

### Compilation

Run `make boba-devnet`

## boba-connect

This package generates a transition block between the legacy and new systems. It does this by performing the following steps:

1. A configuration file to get the timestamp of the transition block.
2. Use the engine api to create an empty block with the right block timestamp

### Compilation

Run `make boba-connect`
