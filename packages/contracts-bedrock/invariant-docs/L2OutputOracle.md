# `L2OutputOracle` Invariants

## The block number of the output root proposals should monotonically increase.
**Test:** [`L36`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/L2OutputOracle.t.sol#L36)

When a new output is submitted, it should never be allowed to correspond to a block number that is less than the current output. 
