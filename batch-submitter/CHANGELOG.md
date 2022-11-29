# @eth-optimism/batch-submitter-service

## 0.1.13

### Patch Changes

- 7a8812d14: Update go-ethereum to v1.10.26

## 0.1.12

### Patch Changes

- 6f458607: Bump go-ethereum to 1.10.17

## 0.1.11

### Patch Changes

- b2aa08e0: Add MAX_PLAINTEXT_BATCH_SIZE parameter to max out compression

## 0.1.10

### Patch Changes

- 526eac8d: feat: bss less strict min-tx-size

## 0.1.9

### Patch Changes

- 160f4c3d: Update docker image to use golang 1.18.0
- 0c4d4e08: l2geth: Revert transaction pubsub feature

## 0.1.8

### Patch Changes

- 88601cb7: Refactored Dockerfiles
- 6856b215: Count reverted transactions in failed_submissions
- 9678b357: Add Min/MaxStateRootElements configuration
- f8348862: l2geth: Sync from Backend Queue
- 727b0582: Enforce min/max tx size on plaintext batch encoding

## 0.1.7

### Patch Changes

- aca0684e: Add 20% buffer to gas estimation on tx-batch submission to prevent OOG reverts
- 75040ca5: Adds MIN_L1_TX_SIZE configuration

## 0.1.6

### Patch Changes

- 6af67df5: Move L2 dial logic out of bss-core to avoid l2geth dependency
- fe680568: Enable the usage of typed batches and type 0 zlib compressed batches

## 0.1.5

### Patch Changes

- 6f2ea193: Update to go-ethereum v1.10.16
- 87359fd2: Refactors the bss-core service to use a metrics interface to allow
  driver-specific metric extensions

## 0.1.4

### Patch Changes

- bcbde5f3: Fixes a bug that causes the txmgr to not wait for the configured numConfirmations

## 0.1.3

### Patch Changes

- 69118ac3: Switch num_elements_per_batch from Histogram to Summary
- df98d134: Remove extra space in metric names
- 3ec06301: Default to JSON logs, add LOG_TERMINAL flag for debugging
- fe321618: Unify metric name format
- 93a26819: Fixes a bug where clearing txs are rejected on startup due to missing gas limit

## 0.1.2

### Patch Changes

- c775ffbe: fix BSS log-level flag parsing
- d093a6bb: Adds a fix for the BSS to account for the new timestamp logic in L2Geth
- d4c2e01b: Restructure to use bss-core package

## 0.1.1

### Patch Changes

- 5905f3dc: Update golang version to support HTTP/2
- c1eba2e6: use EIP-1559 txns for tx/state batches

## 0.1.0

### Minor Changes

- 356b7271: Add multi-tx support, clear pending txs on startup

### Patch Changes

- 85aa148d: Adds confirmation depth awareness to txmgr

## 0.0.2

### Patch Changes

- d6e0de5a: Fix metrics server
