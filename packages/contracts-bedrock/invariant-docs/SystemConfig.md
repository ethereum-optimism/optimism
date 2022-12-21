# SystemConfig Invariants

## The gas limit of the `SystemConfig` contract can never be lower
**Test:** [`L23`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/invariant-docs/SystemConfig.t.sol)

than the hard-coded lower bound.
