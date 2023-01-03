# `OptimismPortal` Invariants

## Deposits of any value should always succeed unless
**Test:** [`L37`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/echidna/FuzzOptimismPortal.sol#L37)


All deposits, barring creation transactions and transactions sent to `address(0)`, should always succeed. 
