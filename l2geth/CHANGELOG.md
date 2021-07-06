# Changelog

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
