# eth_sendBundle API Reference

## Overview

The `eth_sendBundle` method is a JSON-RPC endpoint that allows searchers to submit transactions with advanced execution control. Unlike regular transaction submission via `eth_sendTransaction`, bundles allow for the following features:
- **Execution Timing**: Specify exact block ranges, flashblock ranges, or timestamps when transactions should execute
- **Revert Protection**: Reverting transactions don't land on-chain so they don't cost any gas

## Prerequisites

The `eth_sendBundle` endpoint is only available when revert protection is enabled with the `--builder.enable-revert-protection` flag.

## Bundle Structure (JSON-RPC params)

```json
{
  "txs": ["0x..."],                    // Array of raw transaction bytes
  "revertingTxHashes": ["0x..."],      // Optional: transactions allowed to revert
  "minBlockNumber": "0x1",             // Optional: minimum block number
  "maxBlockNumber": "0xa",             // Optional: maximum block number
  "minFlashblockNumber": "0x64",       // Optional: minimum flashblock number
  "maxFlashblockNumber": "0x68",       // Optional: maximum flashblock number
  "minTimestamp": 1640995200,          // Optional: minimum timestamp (Unix epoch)
  "maxTimestamp": 1640995800           // Optional: maximum timestamp (Unix epoch)
}
```

### Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `txs` | `string[]` | ✅ | Array of RLP-encoded transaction data (exactly one transaction) |
| `revertingTxHashes` | `string[]` | ❌ | Transaction hashes allowed to revert without failing the bundle |
| `minBlockNumber` | `number` | ❌ | Earliest block number for execution |
| `maxBlockNumber` | `number` | ❌ | Latest block number for execution |
| `minFlashblockNumber` | `number` | ❌ | Earliest flashblock iteration for execution |
| `maxFlashblockNumber` | `number` | ❌ | Latest flashblock iteration for execution |
| `minTimestamp` | `number` | ❌ | Earliest timestamp for execution (Unix epoch seconds) |
| `maxTimestamp` | `number` | ❌ | Latest timestamp for execution (Unix epoch seconds) |

## Response

```json
{
  "bundleHash": "0x..." // Transaction hash of the submitted transaction
}
```

## Validation Rules

### Block Number Validation

1. **Range Validity**: If both `minBlockNumber` and `maxBlockNumber` are specified, min ≤ max
2. **Past Block Protection**: `maxBlockNumber` must be greater than the current block number
3. **Range Limits**: Block ranges cannot exceed 10 blocks (`MAX_BLOCK_RANGE_BLOCKS`)
4. **Default Maximum**: If no `maxBlockNumber` is specified, defaults to `current_block + 10`

### Flashblock Number Validation

If both `minFlashblockNumber` and `maxFlashblockNumber` are specified, min ≤ max.

### Block Number + Flashblock Number interaction

When both block number and flashblock number ranges are specified, they act independently of each other. For example, if the builder receives a bundle request with parameters like
```
"minBlockNumber": 100,
"maxBlockNumber": 105,
"minFlashblockNumber": 1,
"maxFlashblockNumber": 3,
```
Then the builder will only execute the bundle if the current block number is between 100 and 105 AND the current flashblock number is between 1 and 3.

### Transaction Constraints

1. **Single Transaction**: Bundles must contain exactly one transaction
2. **Valid Format**: Transaction must be properly RLP-encoded

### Timestamp Constraints (⚠️ Caution)

Timestamp-based constraints depend on the builder node's clock and may not be perfectly synchronized with network time. Block number or flashblock number constraints are preferred.

## Error Responses

| Error | Description | Solution |
|-------|-------------|----------|
| `bundle must contain exactly one transaction` | Bundle has 0 or >1 transactions | Include exactly one transaction |
| `block_number_max (X) is a past block` | Max block is ≤ current block | Use future block number |
| `block_number_max (X) is too high` | Block range exceeds 10 blocks | Reduce block range |
| `flashblock_number_min (X) is greater than flashblock_number_max (Y)` | Invalid flashblock range | Ensure min ≤ max |
| `method not found` | Revert protection disabled | Enable revert protection |

## Usage Examples

### Basic Bundle Submission

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "method": "eth_sendBundle",
    "params": [{
      "txs": ["0x02f86c0182..."], // Raw transaction bytes
      "maxBlockNumber": 10        // Execute within next 10 blocks
    }],
    "id": 1,
    "jsonrpc": "2.0"
  }'
```

### Bundle with Revert Protection

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "method": "eth_sendBundle",
    "params": [{
      "txs": ["0x02f86c0182..."],
      "revertingTxHashes": ["0xabc123..."], // Allow this tx to revert
      "minBlockNumber": 5,
      "maxBlockNumber": 10
    }],
    "id": 1,
    "jsonrpc": "2.0"
  }'
```

### Flashblock Bundle

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "method": "eth_sendBundle",
    "params": [{
      "txs": ["0x02f86c0182..."],
      "minFlashblockNumber": 1,  // Flashblock 1
      "maxFlashblockNumber": 4   // Flashblock 4
    }],
    "id": 1,
    "jsonrpc": "2.0"
  }'
```

## Monitoring and Debugging

### Check Bundle Status
Use `eth_getTransactionReceipt` to check if your bundle was included:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{
    "method": "eth_getTransactionReceipt",
    "params": ["0x..."],  // Bundle hash (is also tx hash) from response
    "id": 1,
    "jsonrpc": "2.0"
  }'
```