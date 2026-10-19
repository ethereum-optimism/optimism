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

Run `snapshot` before the upgrade and `check` after it:

```bash
# 1. Before the upgrade
go run ./op-chain-ops/cmd/check-review-period snapshot \
  --l1 <L1_RPC> \
  --system-config <SYSTEM_CONFIG_A> \
  --system-config <SYSTEM_CONFIG_B> \
  --out before.json

# 2. Run the upgrade

# 3. After the upgrade
go run ./op-chain-ops/cmd/check-review-period check \
  --l1 <L1_RPC> \
  --snapshot before.json
```

`check` reads the chains listed in the snapshot again, logs each delay, and exits non-zero if any delay changed. A
changed contract address is logged as a warning but does not fail the check.

Flags can also be set through environment variables: `CHECK_REVIEW_PERIOD_L1`, `CHECK_REVIEW_PERIOD_SYSTEM_CONFIG`,
`CHECK_REVIEW_PERIOD_OUT` and `CHECK_REVIEW_PERIOD_SNAPSHOT`.

## Build

From the `optimism` directory:

```bash
go build ./op-chain-ops/cmd/check-review-period
```

## Implementation Details

The reads live in the `reviewperiod` package, which the `op-acceptance-tests` review period tests also use, so CI
exercises the same code that runs against live networks.
