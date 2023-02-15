# Changelog

## 0.5.33

### Patch Changes

- 33acb7c6a: Has l2geth return a NonceToHigh response if the txn nonce is greater than the expected nonce.

## 0.5.32

### Patch Changes

- ea817097b: Use default cas gap of 25 million

## 0.5.31

### Patch Changes

- ccbca22c3: Patch release for additional instrumentation for the Bedrock upgrade.

## 0.5.30

### Patch Changes

- 0e8652c29: Close down the syncservice more cleanly

## 0.5.29

### Patch Changes

- 4e65ceb9d: Dedupe dumper addresses in memory

## 0.5.28

### Patch Changes

- ac0f14f5: Fix state dumper
- 5005da9a: Fixes a small l2geth bug when trying to dump state

## 0.5.27

### Patch Changes

- 596c974e: Kick the build

## 0.5.26

### Patch Changes

- 397b27ee: Add data exporter

## 0.5.25

### Patch Changes

- 89f1abfa: add --rpc.evmtimeout flag to configure timeout for eth_call

## 0.5.24

### Patch Changes

- c3e66e57: Add the gas estimation block tag to `eth_estimateGas` to be RPC compliant

## 0.5.23

### Patch Changes

- c3363225: fix NPE in debug_standardTraceBlockToFile

## 0.5.22

### Patch Changes

- ff0723aa: Have L2Geth Verifier sync in parallel with the DTL.

## 0.5.21

### Patch Changes

- 248f73c5: Rerelease the previous version

## 0.5.20

### Patch Changes

- 359bc604: Patch for L1 syncing nodes that got stuck after DTL batch sync errors

## 0.5.19

### Patch Changes

- 1bcee8f1: Fix `eth_getBlockRange`
- c799535d: Add system addresses for nightly goerli

## 0.5.18

### Patch Changes

- 935a98e6: rollup: fix log.Crit usage
- 81f09f16: l2geth: Record rollup transaction metrics

## 0.5.17

### Patch Changes

- 13524da4: Style fix in the sync service
- 160f4c3d: Update docker image to use golang 1.18.0
- 1a28ba5f: Skip account cmd tests
- 45582fcc: Skip unused tests in l2geth
- 0c4d4e08: l2geth: Revert transaction pubsub feature

## 0.5.16

### Patch Changes

- a01a2eb1: Skip TestWSAttachWelcome
- 23ad6068: Skip some geth console tests that flake in CI
- 6926b293: Adds a flag for changing the genesis fetch timeout

## 0.5.15

### Patch Changes

- 88601cb7: Refactored Dockerfiles
- f8348862: l2geth: Sync from Backend Queue

## 0.5.14

### Patch Changes

- 962f36e4: Add support for system addresses

## 0.5.13

### Patch Changes

- 0002b1df: Remove dead code in l2geth
- 1187dc9a: Don't block read rpc requests when syncing
- bc342ec4: Fix queue index comparison

## 0.5.12

### Patch Changes

- 84e6a158: Bump the timeout to download the genesis file on l2geth

## 0.5.11

### Patch Changes

- 9ef215b8: Various small changes to reduce our upstream Geth diff

## 0.5.10

### Patch Changes

- 2e7f6a55: Fixes incorrect timestamp handling for L1 syncing verifiers
- 81d90563: Bring back RPC methods that were previously blocked

## 0.5.9

### Patch Changes

- e631c39c: Implement berlin hardfork

## 0.5.8

### Patch Changes

- 949916f8: Add a better error message for when the sequencer url is not configured when proxying user requests to the sequencer for `eth_sendRawTransaction` when running as a verifier/replica
- 300f79bf: Fix nonce issue
- ae96d784: Add reinitialize-by-url command, add dump chain state command
- c7569a16: Fix blocknumber monotonicity logging bug

## 0.5.7

### Patch Changes

- d4bf299f: Add support to fully unmarshal Receipts with Optimism fields
- 8be69ca7: Add changeset for https://github.com/ethereum-optimism/optimism/pull/2011 - replicas forward write requests to the sequencer via a configured parameter `--sequencer.clienthttp` or `SEQUENCER_CLIENT_HTTP`
- c9fd6ec2: Correctly parse fee enforcement via config to allow turning off L2 fees for development

## 0.5.6

### Patch Changes

- 3a77bbcc: Implement updated timestamp logic
- 3e3c07a3: changed the default address to be address(0) in `call`

## 0.5.5

### Patch Changes

- 2924845d: expose ErrNonceTooHigh from miner

## 0.5.4

### Patch Changes

- d205c1d6: surface sequencer low-level sequencer execution errors

## 0.5.3

### Patch Changes

- 5febe10f: fixes empty block detection and removes empty worker tasks
- 272d20d6: renames l2geth package name to github.com/ethereum-optimism/optimism/l2geth

## 0.5.2

### Patch Changes

- d141095c: Allow for unprotected transactions

## 0.5.1

### Patch Changes

- 7f2898ba: Fixes deadlock

## 0.5.0

### Minor Changes

- e03dcead: Start refactor to new version of the OVM
- e4a1129c: Adds aliasing to msg.sender and tx.origin to avoid xdomain attacks
- 299a459e: Introduces a new opcode L1BLOCKNUMBER to replace old functionality where blocknumber would return the L1 block number and the L2 block number was inaccessible.
- 872f5976: Removes various unused OVM contracts
- 65289e63: Add optimistic ethereum specific fields to the receipt. These fields are related to the L1 portion of the fee. Note that this is a consensus change as it will impact the blockhash through the receipts root
- 92c9692d: Opcode tweaks. Coinbase returns SequencerFeeVault address. Difficulty returns zero.
- 1e63ffa0: Refactors and simplifies OVM_ETH usage
- d3cb1b86: Reintroduces the whitelist into the v2 system
- 81ccd6e4: `regenesis/0.5.0` release
- f38b8000: Removes ERC20 and WETH9 features from OVM_ETH
- 3605b963: Adds refactored support for the L1MESSAGESENDER opcode
- 3f28385a: Removes all custom genesis initialization

### Patch Changes

- 8988a460: Cleanup `time.Ticker`s
- fbdd06f5: Set the latest queue index and index after the tx has been applied to the chain
- 5c0e90aa: Handle policy/consensus race condition for balance check
- 8c8807c0: Refactor to simplify the process of generating the genesis json file
- 95a0d803: Remove calls to `syncBatchesToTip` in the main `sequence()` loop
- da99cc43: Remove dead `debug_ingestTransactions` endpoint and `txType` from RPC transactions
- 6bb040b7: Remove complex mutex logic in favor of simple mutex logic in the `SyncService`
- 7bd88e81: Use `OVM_GasPriceOracle` based L1 base fee instead of fetching it from remote
- b70ee70c: upgraded to solidity 0.8.9
- 3c56126c: Handle race condition in L2 geth for fee logic
- c39165f8: Remove dead L1 gas price fetching code
- 95c0463c: Fix various geth tests
- e11c3ea2: Use minimal EIP-2929 for state accessing opcodes

## 0.4.15

### Patch Changes

- 5c9b6343: Fix execution manager run

## 0.4.14

### Patch Changes

- 0d429564: Add ROLLUP_ENABLE_ARBITRARY_CONTRACT_DEPLOYMENT_FLAG

## 0.4.13

### Patch Changes

- dfe3598f: Lower per tx fee overhead to more accurately represent L1 costs

## 0.4.12

### Patch Changes

- 0e14855c: Add in min accepted L2 gas limit config flag

## 0.4.11

### Patch Changes

- f331428f: Update the memory usage in geth

## 0.4.10

### Patch Changes

- eb1eb327: Ensure that L2 geth doesn't reject blocks from the future

## 0.4.9

### Patch Changes

- 3c420ec3: Reduce the geth diff
- 9d1ff999: Allow transactions via RPC to `address(0)`
- 101b942c: Removes `id` field from EVM and no longer logs the EVM execution id
- 4cf68ade: Style fix in the `RollupClient`
- 6dbb9293: Remove dead code in `blockchain.go` and `miner/worker.go`

## 0.4.8

### Patch Changes

- a8e37aac: Style fix to the ovm state manager precompile
- 616b7a28: Small fixes to miner codepath
- 7ee76c23: Remove an unnecessary use of `reflect` in l2geth
- 75d8dcd3: Remove layer of indirection in `callStateManager`
- f0a02385: Update the start script to work with the latest regenesis, `0.4.0`
- 75ec2869: Return correct value in L2 Geth fee too high error message
- 7acbab74: Delete stateobjects in the miner as blocks are produced to prevent a build up of memory
- 0975f738: Remove diffdb
- 8f9bb36f: Quick syntax fix in the sync service
- 11d46182: Make the extradata deterministic for deterministic block hashes

## 0.4.7

### Patch Changes

- bb7b916e: revert rpcGasCap logic to upstream geth behavior

## 0.4.6

### Patch Changes

- 32a9f494: Give a better error message for when the fee is too high when sending transactions to the sequencer
- 735ef774: Fix a bug in the fee logic that allowed for fees that were too low to get through

## 0.4.5

### Patch Changes

- 53b37978: Fixes the flags to use float64 instead of bools for the `--rollup.feethresholddown` and `-rollup.feethresholdup` config options
- 709c85d6: Prevents the sequencer from accepting transactions with a too high nonce

## 0.4.4

### Patch Changes

- 0404c964: Allow zero gas price transactions from the `OVM_GasPriceOracle.owner` when enforce fees is set to true. This is to prevent the need to manage an additional hot wallet as well as prevent any situation where a bug causes the fees to go too high that it is not possible to lower the fee by sending a transaction
- c612a903: Add sequencer fee buffer with config options `ROLLUP_FEE_THRESHOLD_UP` and `ROLLUP_FEE_THRESHOLD_DOWN` that are interpreted as floating point numbers

## 0.4.3

### Patch Changes

- 6e2074c5: Update the `RollupClient` transaction type to use `hexutil.Big`

## 0.4.2

### Patch Changes

- 7e04137d: Handle errors correctly in the RollupClient and retry in the SyncService when initially attempting to connect to the DTL

## 0.4.1

### Patch Changes

- 40b99a6e: Add new RPC endpoint `rollup_gasPrices`

## 0.4.0

### Minor Changes

- e04de624: Add support for ovmCALL with nonzero ETH value

### Patch Changes

- 01646a0a: Add new config `ROLLUP_GAS_PRICE_ORACLE_OWNER_ADDRESS` to set the owner of the gas price oracle at runtime
- 8fee7bed: Add extra overflow protection for the DTL types
- 5fc728da: Add a new Standard Token Bridge, to handle deposits and withdrawals of any ERC20 token.
  For projects developing a custom bridge, if you were previously importing `iAbs_BaseCrossDomainMessenger`, you should now
  import `iOVM_CrossDomainMessenger`.
- 257deb70: Prevent overflows in abi encoding of ovm codec transaction from geth types.Transaction
- 08873674: Update queueOrigin type
- 01646a0a: Removes config options that are no longer required. `ROLLUP_DATAPRICE`, `ROLLUP_EXECUTION_PRICE`, `ROLLUP_GAS_PRICE_ORACLE_ADDRESS` and `ROLLUP_ENABLE_L2_GAS_POLLING`. The oracle was moved to a predeploy 0x42.. address and polling is always enabled as it no longer needs to be backwards compatible
- 0a7f5a46: Removes the gas refund for unused gas in geth since it is instead managed in the smart contracts
- e045f582: Adds new SequencerFeeVault contract to store generated fees
- 25a5dbdd: Removes the SignatureHashType from l2geth as it is deprecated and no longer required.

## 0.3.9

### Patch Changes

- f409ce75: Fixes an off-by-one error that would sometimes break replica syncing when stopping and restarting geth.
- d9fd67d2: Correctly log 'end of OVM execution' message.

## 0.3.8

### Patch Changes

- 989a3027: Optimize main polling loops
- cc6c7f07: Bump golang version to 1.15

## 0.3.7

### Patch Changes

- cb4a928b: Make block hashes deterministic by using the same clique signer key
- f1b27318: Fixes incorrect type parsing in the RollupClient. The gasLimit became greater than the largest safe JS number so it needs to be represented as a string
- a64f8161: Implement the next fee spec in both geth and in core-utils
- 5e4eaea1: fix potential underflow when launching the chain when the last verified index is 0
- 1293825c: Fix gasLimit overflow
- a25acbbd: Refactor the SyncService to more closely implement the specification. This includes using query params to select the backend from the DTL, trailing syncing of batches for the sequencer, syncing by batches as the verifier as well as unified code paths for transaction ingestion to prevent double ingestion or missed ingestion
- c2b6e14b: Implement the latest fee spec such that the L2 gas limit is scaled and the tx.gasPrice/tx.gasLimit show correctly in metamask

## 0.3.6

### Patch Changes

- f091e86: Fix to ensure that L1 => L2 success status is reflected correctly in receipts
- f880479: End to end fee integration with recoverable L2 gas limit

## 0.3.5

### Patch Changes

- d4c9793: Fixed a bug where reverts without data would not be correctly propagated for eth_call
- 3958644: Adds the `debug_ingestTransactions` endpoint that takes a list of RPC transactions and applies each of them to the state sequentially. This is useful for testing purposes
- c880043: Fix gas estimation logic for simple ETH transfers
- 467d6cb: Adds a test for contract deployments that run out of gas
- 4e6c3f9: add an env var METRICS_ENABLE for MetricsEnabledFlag

## 0.3.4

### Patch Changes

- e2b70c1: Don't panic on a monotonicity violation

## 0.3.3

### Patch Changes

- f5185bb: Fix bug with replica syncing where contract creations would fail in replicas but pass in the sequencer. This was due to the change from a custom batched tx serialization to the batch serialzation for txs being regular RLP encoding

## 0.3.2

### Patch Changes

- 20242af: Fixes a bug in L2geth that causes it to skip the first deposit if there have been no deposits batch-submitted yet
- cf3cfe4: Allow for dynamically set configuration of the gasLimit in the contracts by setting the storage slot at runtime
- de5e3dc: Updates `scripts/start.sh` with the mainnet config by default

## 0.3.1

### Patch Changes

- 9231063: Prevent montonicity errors in the miner

## 0.3.0

### Minor Changes

- b799caa: Updates to use RLP encoded transactions in batches for the `v0.3.0` release

### Patch Changes

- b799caa: Add value parsing to the rollup client
- b799caa: Removes the extra setting of the txmeta in the syncservice and instead sets the raw tx in the txmeta at the rpc layer
- b799caa: Fill in the raw transaction into the txmeta in the `eth_sendTransaction` codepath
- b799caa: Add support for parsed revert reasons in DoEstimateGas
- b799caa: Update minimum response from estimate gas
- b799caa: Add value transfer support to ECDSAContractAccount
- b799caa: Ignore the deprecated type field in the API
- b799caa: Return bytes from both ExecutionManager.run and ExecutionManager.simulateMessage and be sure to properly ABI decode the return values and the nested (bool, returndata)
- b799caa: Block access to RPCs related to signing transactions
- b799caa: Add ExecutionManager return data & RLP encoding
- b799caa: Update gas related things in the RPC to allow transactions with high gas limits and prevent gas estimations from being too small
- 9b7dd4b: Update `scripts/start.sh` to parse the websocket port and pass to geth at runtime
- b799caa: Remove the OVMSigner
- b799caa: Prevent 0 value transactions with calldata via RPC

## 0.2.6

### Patch Changes

- a0a0052: Add value parsing to the rollup client
- 20df745: Protect a possible `nil` reference in `eth_call` when the blockchain is empty
- 9f1529c: Update the start script to be more configurable
- 925675d: Update `scripts/start.sh` to regenesis v0.2.0

## 0.2.5

### Patch Changes

- 79f66e9: Use constant execution price, which is set by the sequencer
- 5b9be2e: Correctly set the OVM context based on the L1 values during `eth_call`. This will also set it during `eth_estimateGas`. Add tests for this in the integration tests

## 0.2.4

### Patch Changes

- 7e9ca1e: Add batch API to rollup client
- 6e8fe1b: Removes mockOVM_ECDSAContractAccount and OVM_ProxySequencerEntrypoint, two unused contracts.
- 76c4ceb: Calculate data fees based on if a byte was zero or non-zero

## 0.2.3

### Patch Changes

- d6734f6: Change ROLLUP_BASE_TX_SIZE to camelcase for standard style
- 5e0d0fc: Commit go.sum after a `make test`
- 8a2c24a: Set default timestamp refresh threshold to 3 minutes
- ba2e043: Add `VerifiedIndex` to db and api
- ef40ed7: Allow gas estimation for replicas

## 0.2.2

### Patch Changes

- b290cfe: CPU Optimization by caching ABI methods
- c4266fa: Fix logger error

## 0.2.1

### Patch Changes

- 3b00b7c: bump private package versions to try triggering a tag

## v0.1.3

- Integrate data transport layer
- Refactor `SyncService`
- New RPC Endpoint `eth_getBlockRange`

## v0.1.2

Reduce header cache size to allow L2Geth to spin back up.

## v0.1.1

Pre-minnet fixes.

- gaslimit: fix eth_call (#186)
- rollup: safer historical log syncing (#173)
- config: flag for max acceptable calldata size (#181)
- debug rpc: debug_setL1Head and better l1 timestamp management (#184)
- Fix for hasEmptyAccount (#182)
- gasLimit: error on gas limit too high for queue origin sequencer txs (#180)
- Fixes issue with broken gas limit (#183)

## v0.1.0

Initial Release

- Feature complete for minnet
- OVM runtime implemented for deterministic transaction execution on L1
- Runs in either Sequencer mode or Verifier mode
- `rollup` package includes the `SyncService` for syncing the Canonical
  Transaction Chain
- New configuration options for rollup related features
- No P2P networking
- Maintains RPC compatibility with geth
