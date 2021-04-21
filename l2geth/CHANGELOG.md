# Changelog

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
