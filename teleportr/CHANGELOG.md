# @eth-optimism/teleportr

## 0.0.11

### Patch Changes

- 29ff7462: Revert es target back to 2017

## 0.0.10

### Patch Changes

- ed3a39fb: Fix panic

## 0.0.9

### Patch Changes

- 23dcba53: Better availability endpoint + retries

## 0.0.8

### Patch Changes

- 487a9731: Improve metrics
- b5ee3c70: Increase max disbursements to 15

## 0.0.7

### Patch Changes

- 6f458607: Bump go-ethereum to 1.10.17
- cd15c40a: Only do 5 disbursements at a time

## 0.0.6

### Patch Changes

- df61d215: Add disburser balance to status
- 32639605: Fix teleportr FailedDatabaseOperations method for upsert_disbursement
- 32639605: Expose metrics server

## 0.0.5

### Patch Changes

- 44c293d8: Fix confs_remaining calculation to prevent underflow

## 0.0.4

### Patch Changes

- 7a320e22: Use L2 gas price in driver

## 0.0.3

### Patch Changes

- 160f4c3d: Update docker image to use golang 1.18.0

## 0.0.2

### Patch Changes

- f101d38b: Add metrics for balances

## 0.0.1

### Patch Changes

- 172c3d74: Add SuggestGasTipCap fallback
- 6856b215: Count reverted transactions in failed_submissions
- f4f3054a: Add teleportr API server
- 3e57f559: Bump `go-ethereum` to `v1.10.16`
- bf1cc8f4: Restructure Deposit and CompletedTeleport to use struct embeddings
- bced4fa9: Add LoadInTeleport method to database
- e5732d97: Add btree index on deposit.txn_hash and deposit.address
