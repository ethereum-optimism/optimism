# boba-chain-ops

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

## Compilation

Run `make boba-migrate`.

