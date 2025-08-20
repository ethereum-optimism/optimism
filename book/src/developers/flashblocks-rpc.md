# Reading Flashblocks Data over Ethereum JSON RPC

The Flashblocks RPC implementation provides preconfirmation data through modified Ethereum JSON-RPC endpoints using the `pending` tag. This allows applications to access transaction and state information from Flashblocks before they are finalized on the blockchain.

## Quick Start

To run a node capable of serving Flashblocks, run the `flashblocks-rpc` crate inside the rollup-boost repository.

Build:

```bash
cargo build --release
```

Run:

```bash
./target/release/flashblocks-rpc \
    --flashblocks.enabled=true \
    --flashblocks.websocket-url=ws://localhost:8080/flashblocks \
    --chain=optimism \
    --http \
    --http.port=8545
```

## Supported RPC Endpoints

The following Ethereum JSON-RPC endpoints are modified to support Flashblocks data when queried with the `pending` tag:

### `eth_getBlockByNumber`

Retrieves block information including transactions from the pending Flashblock.

**Request:**
```json
{
  "method": "eth_getBlockByNumber",
  "params": ["pending", true],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "hash": "0x0",
    "parentHash": "0x...",
    "stateRoot": "0x...",
    "transactionsRoot": "0x...",
    "receiptsRoot": "0x...",
    "number": "0x...",
    "gasUsed": "0x...",
    "gasLimit": "0x...",
    "timestamp": "0x...",
    "extraData": "0x...",
    "mixHash": "0x...",
    "nonce": "0x...",
    "transactions": [
      {
        "hash": "0x...",
        "nonce": "0x...",
        "blockHash": "0x0",
        "blockNumber": null,
        "transactionIndex": "0x0",
        "from": "0x...",
        "to": "0x...",
        "value": "0x...",
        "gas": "0x...",
        "gasPrice": "0x...",
        "input": "0x...",
        "v": "0x...",
        "r": "0x...",
        "s": "0x..."
      }
    ]
  }
}
```

**Notes:**
- Returns an empty hash (`0x0`) as the block hash since the block is not yet finalized
- Includes all transactions from received Flashblocks
- Implements an append-only pattern - multiple queries show expanding transaction lists

### `eth_getTransactionReceipt`

Retrieves transaction receipt information from the pending Flashblock.

**Request:**
```json
{
  "method": "eth_getTransactionReceipt",
  "params": ["0x..."],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "transactionHash": "0x...",
    "transactionIndex": "0x0",
    "blockHash": "0x0",
    "blockNumber": null,
    "from": "0x...",
    "to": "0x...",
    "cumulativeGasUsed": "0x...",
    "gasUsed": "0x...",
    "contractAddress": null,
    "logs": [],
    "status": "0x1",
    "logsBloom": "0x...",
    "effectiveGasPrice": "0x..."
  }
}
```

**Notes:**
- Returns receipt data for transactions in the pending Flashblock
- `blockHash` is `0x0` and `blockNumber` is `null` since the block is not finalized
- Includes gas usage and execution status information

### `eth_getBalance`

Retrieves account balance from the pending state.

**Request:**
```json
{
  "method": "eth_getBalance",
  "params": ["0x...", "pending"],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response:**
```json
"0x..."
```

**Notes:**
- Returns balance reflecting all changes from preconfirmed transactions
- Uses cached state changes from Flashblocks metadata
- Falls back to standard RPC if no pending data is available

### `eth_getTransactionCount`

Retrieves the transaction count (nonce) for an account from the pending state.

**Request:**
```json
{
  "method": "eth_getTransactionCount",
  "params": ["0x...", "pending"],
  "id": 1,
  "jsonrpc": "2.0"
}
```

**Response:**
```json
"0x..."
```

**Notes:**
- Returns nonce reflecting all preconfirmed transactions
- Combines latest confirmed nonce with pending transaction count
- Useful for determining the next valid nonce for the next Flashblock

## Data Flow

The Flashblocks RPC implementation follows this data flow:

1. **WebSocket Connection**: Establishes connection to rollup-boost Flashblocks endpoint
2. **Payload Reception**: Receives `FlashblocksPayloadV1` messages containing transaction batches
3. **Cache Processing**: Updates in-memory cache with new transaction and state data
4. **RPC Queries**: Responds to client requests using cached pending data
5. **State Management**: Maintains consistency between confirmed and pending states

## Flashblocks Payload Structure

Flashblocks payloads contain the following information:

```rust,ignore
pub struct FlashblocksPayloadV1 {
    pub version: PayloadVersion,
    pub execution_payload: OpExecutionPayloadEnvelope,
    pub metadata: Metadata,
}

pub struct Metadata {
    pub receipts: HashMap<String, OpReceipt>,
    pub new_account_balances: HashMap<String, String>,
    pub block_number: u64,
}
```