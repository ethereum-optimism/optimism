---
title: Configuration
lang: en-US
---

The OP Stack is a flexible platform with various configuration values that you can tweak to fit your specific needs. If you’re looking to fine-tune your deployment, look no further.

::: warning 🚧 Work in Progress

OP Stack configuration is an active work in progress and will likely evolve significantly as time goes on. If something isn’t working about your configuration, check back with this page to see if anything has changed.

:::

## New Blockchain Configuration

New OP Stack blockchains are currently configured with a JSON file inside the Optimism repository. The file is `<optimism repository>/packages/contracts-bedrock/deploy-config/<chain name>.json`. For example, [this is the configuration file for the tutorial blockchain](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/deploy-config/getting-started.json). 


### Admin accounts

| Key | Type | Description | Default / Recommended value |
| --- | --- | --- | --- |
| `finalSystemOwner` | L1 Address | Address that will own all ownable contracts on L1 once the deployment is finished, including the `ProxyAdmin` contract. | It is recommended to have a single admin account to retain a common security model. |
| `controller` | L1 Address | Address that will own the `SystemDictator` contract and can therefore control the flow of the deployment or upgrade.  | It is recommended to have a single admin account to retain a common security model. |
| `proxyAdminOwner` | L2 Address | Address that will own the `ProxyAdmin` contract on L2. The L2 `ProxyAdmin` contract owns all of the `Proxy` contracts for every predeployed contract in the range `0x42...0000` to `0x42..2048`. This makes predeployed contracts easily upgradeable. | It is recommended to have a single admin account to retain a common security model. |


### Fee recipients

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `baseFeeVaultRecipient` | L1 Address | L1 address that the base fees from all transactions on the L2 can be withdrawn to. | It is recommended to have a single admin account to retain a common security model. |
| `l1FeeVaultRecipient` | L1 Address | L1 address that the L1 data fees from all transactions on the L2 can be withdrawn to. | It is recommended to have a single admin account to retain a common security model. |
| `sequencerFeeVaultRecipient` | L1 Address | L1 address that the tip fees from all transactions on the L2 can be withdrawn to. | It is recommended to have a single admin account to retain a common security model. |


### Misc.

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `numDeployConfirmations` | Number of blocks | Number of confirmations to wait when deploying smart contracts to L1. | 1 |
| `l1StartingBlockTag` | Block hash | Block tag for the L1 block where the L2 chain will begin syncing from. Generally recommended to use a finalized block to avoid issues with reorgs.  |  |
| `l1ChainID` | Number | Chain ID of the L1 chain. | 1 for L1 Ethereum mainnet, <br> 5 for the Goerli test network. <br> [See here for other blockchains](https://chainlist.org/?testnets=true). |
| `l2ChainID` | Number | Chain ID of the L2 chain. | 42069 |


### Blocks

These fields apply to L2 blocks: Their timing, when do they need to be written to L1, and how they get written.

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `l2BlockTime` | Number of seconds | Number of seconds between each L2 block. | 2 |
| `maxSequencerDrift` | Number of seconds | How far the L2 timestamp can differ from the actual L1 timestamp | 600 (10 minutes) |
| `sequencerWindowSize` | Number of blocks | Maximum number of L1 blocks that a Sequencer can wait to incorporate the information in a specific L1 block. For example, if the window is `10` then the information in L1 block `n` must be incorporated by L1 block `n+10`. | 3600 (12 hours) |
| `channelTimeout` | Number of blocks | Maximum number of L1 blocks that a transaction channel frame can be considered valid. A transaction channel frame is a chunk of a compressed batch of transactions. After the timeout, the frame is dropped. | 300 (1 hour) |
| `p2pSequencerAddress` | L1 Address | Address of the key that the Sequencer uses to sign blocks on the p2p network. | Sequencer, an address for which you own the private key |
| `batchInboxAddress` | L1 Address | Address that Sequencer transaction batches are sent to on L1. | 0xff00…0042069 |
| `batchSenderAddress` | L1 Address | Address of the account that nodes will filter for when searching for Sequencer transaction batches being sent to the `batchInboxAddress`. Can be updated later via the `SystemConfig` contract on L1. | Batcher, an address for which you own the private key |


### Proposal fields

These fields apply to output root proposals.

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `l2OutputOracleStartingBlockNumber` | Number | Block number of the first OP Stack block. Typically this should be zero, but this may be non-zero for networks that have been upgraded from a legacy system (like Optimism Mainnet). Will be removed with the addition of permissionless proposals. | 0 |
| `l2OutputOracleStartingTimestamp` | Number | Timestamp of the first OP Stack block. This MUST be the timestamp corresponding to the block defined by the `l1StartingBlockTag`. Will be removed with the addition of permissionless proposals. |  |
| `l2OutputOracleSubmissionInterval` | Number of blocks | Number of blocks between proposals to the `L2OutputOracle`. Will be removed with the addition of permissionless proposals. | 120 (24 minutes) |
| `finalizationPeriodSeconds` | Number of seconds | Number of seconds that a proposal must be available to challenge before it is considered finalized by the `OptimismPortal` contract. | We recommend 12 on test networks, seven days on production ones |
| `l2OutputOracleProposer` | L1 Address | Address that is allowed to submit output proposals to the `L2OutputOracle` contract. Will be removed when we have permissionless proposals. |  |
| `l2OutputOracleChallenger` | L1 Address | Address that is allowed to challenge output proposals submitted to the `L2OutputOracle`. Will be removed when we have permissionless challenges. | It is recommended to have a single admin account to retain a common security model. |



### L1 data fee

These fields apply to the cost of the [L1 data fee](https://community.optimism.io/docs/developers/build/transaction-fees/#the-l1-data-fee) for L2 transactions.

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `gasPriceOracleOverhead` | Number | Fixed L1 gas overhead per transaction. Default value will likely be adjusted with more information from the Optimism Goerli deployment. | 2100 |
| `gasPriceOracleScalar` | Number | Dynamic L1 gas overhead per transaction, given in 6 decimals. Default value of 1000000 implies a dynamic gas overhead of exactly 1x (no overhead). | 1000000 |


### EIP 1559 gas algorithm

These fields apply to [the EIP 1559 algorithm](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1559.md) used for the [L2 execution costs](https://community.optimism.io/docs/developers/build/transaction-fees/#the-l2-execution-fee) of transactions on the blockchain.

| Key | Type | Description | Default value | Value on L1 Ethereum |
| --- | --- | --- | --- | --- |
| `eip1559Denominator` | Number | Denominator used for the [EIP1559 gas pricing mechanism on L2](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1559.md). A larger denominator decreases the amount by which the base fee can change in a single block. | 50 | 8 |
| `eip1559Elasticity` | Number | Elasticity for the [EIP1559 gas pricing mechanism on L2](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1559.md). A larger elasticity increases the maximum allowable gas limit per block. | 10 | 2 |
| `l2GenesisBlockGasLimit` | String | Initial block gas limit, represented as a hex string. Default is 25m, implying a 2.5m target when combined with a 10x elasticity. | 0x17D7840 |  |
| `l2GenesisBlockBaseFeePerGas` | String | Initial base fee, used to avoid an unstable EIP1559 calculation out of the gate. Initial value is 1 gwei. | 0x3b9aca00 |  |


### Governance token

The governance token is a side-effect of use of the OP Stack in the Optimism Mainnet network. It may not be included by default in future releases.

| Key | Type | Description | Default value |
| --- | --- | --- | --- |
| `governanceTokenOwner` | L2 Address | Address that will own the token contract deployed by default to every OP Stack based chain.  |  |
| `governanceTokenSymbol` | String | Symbol for the token deployed by default to each OP Stack chain. | OP |
| `governanceTokenName` | String | Name for the token deployed by default to each OP Stack chain. | Optimism |