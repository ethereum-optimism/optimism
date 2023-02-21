# `L2OutputOracle` Invariants

## The block number of the output root proposals should monotonically increase.
**Test:** [`L2OutputOracle.t.sol#L32`](../contracts/test/invariants/L2OutputOracle.t.sol#L32)

When a new output is submitted, it should never be allowed to correspond to a block number that is less than the current output. 
