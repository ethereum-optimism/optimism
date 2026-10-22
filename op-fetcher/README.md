## op-fetcher

This is a service that takes `SystemConfigProxy` and `L1StandardBridgeProxy` as inputs, and uses those addresses to fetch additional chain configuration information from onchain. It reads the L1 contracts with `eth_call`s at the latest L1 block, falling back to legacy getters for older contract releases, and can be used as a go library in addition to a cli tool. Current information fetched from onchain with this service (can be expanded in the future):
* all L1 contract addresses except `SystemConfigProxy` and `L1StandardBridgeProxy`
* all L1 roles
* `fault_proof_status` fields (to determine no fault proofs vs. permissioned vs. permissionless)

## Usage
Fetch data onchain for a single opchain and print results
```
go run ./cmd fetch \
  --l1-rpc-url "<l1-rpc-url>" \
  --system-config "<system-config-proxy-address>" \
  --l1-standard-bridge "<l1-standard-bridge-proxy-address>"
```
