## op-fetcher

This is a service that takes `SystemConfigProxy` and `L1StandardBridgeProxy` as inputs, and uses those addresses to fetch additional chain configuration information from onchain. All of the L1 rpc calls are made via a single solidity script, `FetchChainInfo.s.sol`. The compiled `forge-artifacts` for that script are then embedded in the go code so that `op-fetcher` can be used as a go library in addition to a cli tool. Current information fetched from onchain with this service (can be expanded in the future):
* all L1 contract addresses except `SystemConfigProxy` and `L1StandardBridgeProxy`
* all L1 roles
* `fault_proof_status` fields (to determine no fault proofs vs. permissioned vs. permissionless)

## Usage
1. Build (compiles solidity script used by this service, as well as go code):
```
cd <repo-root>/op-fetcher
just build-all
```
2. Fetch data onchain for a single opchain and print results
```
go run ./cmd fetch \
  --l1-rpc-url "<l1-rpc-url>" \
  --system-config "<system-config-proxy-address>" \
  --l1-standard-bridge "<l1-standard-bridge-proxy-address>"
```