# check-jovian

A tool to verify that the Jovian upgrade has been successfully applied to an OP Stack chain.

## Overview

This tool checks three key aspects of the Jovian upgrade:

1. **GasPriceOracle Contract**: Verifies that `GasPriceOracle.isJovian()` returns `true`
2. **L1Block Contract**: Verifies that `L1Block.DAFootprintGasScalar()` returns a valid number
3. **Block Headers**: Verifies that the latest block header has a non-nil `BlobGasUsed` field

## Usage

### Prerequisites

Set the L2 RPC endpoint via environment variable:
```bash
export CHECK_JOVIAN_L2=http://localhost:9545
```

Or use the command-line flag:
```bash
--l2 http://localhost:9545
```

### Commands

#### Check all Jovian features
```bash
go run . all
```

#### Check individual features

Check GasPriceOracle contract:
```bash
go run . contracts gpo
```

Check L1Block contract:
```bash
go run . contracts l1block
```

Check block header:
```bash
go run . block-header
```

## Build

From the `optimism` directory:
```bash
go build ./op-chain-ops/cmd/check-jovian
```

## Implementation Details

The tool uses the `op-e2e/bindings` package to interact with the L2 contracts and verify:

- **GasPriceOracle.isJovian**: Returns `true` after the Jovian upgrade is activated
- **L1Block.DAFootprintGasScalar**: Returns the DA footprint gas scalar value (warns if 0, as SystemConfig needs to update)
- **Block Header BlobGasUsed**: Non-nil after Jovian activation (used to track DA footprint limits)

## Pattern

This tool follows the same pattern as `check-ecotone` and `check-fjord`, providing a systematic way to verify upgrade completion.

