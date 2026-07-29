# Reading Flashblocks Data over Ethereum JSON RPC

The Flashblocks RPC implementation provides preconfirmation data through modified Ethereum JSON-RPC endpoints using the `pending` tag. This allows applications to access transaction and state information from Flashblocks before they are finalized on the blockchain.

To run a node capable of serving Flashblocks, consider using https://github.com/base/node-reth/ or https://github.com/paradigmxyz/reth implementations.

At the time of writing base version has more advanced flashblocks support, including flashblocks sync over multiple blocks in case canonical head is lagging.

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