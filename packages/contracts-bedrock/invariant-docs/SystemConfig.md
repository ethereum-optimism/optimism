# `SystemConfig` Invariants

## The gas limit of the `SystemConfig` contract can never be lower
**Test:** [`L24`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/SystemConfig.t.sol#L24)
than the hard-coded lower bound. 
