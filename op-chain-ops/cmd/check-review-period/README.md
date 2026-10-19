# check-review-period

A tool to verify that an upgrade leaves a chain's review period delays unchanged.

## Overview

The tool reads three delays per chain from L1:

1. **Proof maturity delay**: `OptimismPortal2.proofMaturityDelaySeconds()`
2. **Dispute game finality delay**: `AnchorStateRegistry.disputeGameFinalityDelaySeconds()`
3. **DelayedWETH withdrawal delay**: `DelayedWETH.delay()`

The contracts are resolved from each chain's `SystemConfig` with `eth_call`s to `optimismPortal()`, `delayedWETH()`,
and the portal's `anchorStateRegistry()`.

## Usage

Run `check` after the upgrade, with either the upgrade transaction or the last block before it:

```bash
go run ./op-chain-ops/cmd/check-review-period check \
  --l1 <L1_RPC> \
  --system-config <SYSTEM_CONFIG_A> \
  --system-config <SYSTEM_CONFIG_B> \
  --upgrade-tx <UPGRADE_TX_HASH>

go run ./op-chain-ops/cmd/check-review-period check \
  --l1 <L1_RPC> \
  --system-config <SYSTEM_CONFIG_A> \
  --before-block <BLOCK_BEFORE_THE_UPGRADE>
```

For each chain, `check` reads the delays at the block before the upgrade and at the latest block, logs each delay, and
exits non-zero if any delay changed. It also fails if none of the chain's proxies (`SystemConfig`, `OptimismPortal2`,
`AnchorStateRegistry`, `DelayedWETH`) emitted `Upgraded(address)` in the upgrade transaction or, with `--before-block`,
after that block. A changed contract address is logged as a warning but does not fail the check.

The L1 RPC must serve state at the block before the upgrade, so older upgrades need an archive node.

Flags can also be set through environment variables: `CHECK_REVIEW_PERIOD_L1`, `CHECK_REVIEW_PERIOD_SYSTEM_CONFIG`,
`CHECK_REVIEW_PERIOD_UPGRADE_TX` and `CHECK_REVIEW_PERIOD_BEFORE_BLOCK`.

## Build

From the `optimism` directory:

```bash
go build ./op-chain-ops/cmd/check-review-period
```

## Implementation Details

The reads live in the `reviewperiod` package, which the `op-acceptance-tests` review period tests also use, so CI
exercises the same code that runs against live networks.
