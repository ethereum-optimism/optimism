# Sponsored Sender

A simple json-rpc proxy service implementation to faciliate sponsored transactions to different chains. The json-rpc endpoint is available per chain at the `/${chainId}` route. The methods implemented under each route.

- `eth_chainId` simply mirrors back the same chain id in the route
- `eth_account` the senders used to relay transactions
- `eth_sendTransaction` signs the unsigned payload and submits with the configured sender account

This service is not meant for production use. Primarily useful for local testing for services that require the existence of a single sponsored endpoint URL.

## Configuration

Configured entirely with environment variables

```
SPONSORED_SENDER_PRIVATE_KEY=0x
SPONSORED_SENDER_ENDPOINTS=http://localhost:9545,http://localhost:9546
```
