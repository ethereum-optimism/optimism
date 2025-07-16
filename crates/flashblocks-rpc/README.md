
# Flashblocks RPC Implementation

RPC provider implementation for the Flashblocks specification on OP Stack chains.

## Overview
This component subscribes to a Flashblocks WebSocket stream and provides preconfirmation data through modified Ethereum JSON-RPC endpoints using the pending tag.

## Quick Start

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

## Command Line Options

- `flashblocks.enabled`: Enable flashblocks functionality (default: false)
- `flashblocks.websocket-url`: WebSocket URL for flashblocks stream
- Standard reth/OP Stack options for chain and RPC configuration
