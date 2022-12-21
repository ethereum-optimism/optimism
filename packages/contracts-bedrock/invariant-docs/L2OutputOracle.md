# `L2OutputOracle` Invariants

## The block number of the output root proposals should monotonically increase.
**Test:** [`L58`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/L2OutputOracle.t.sol#L58)

When a new output is submitted, it should never be allowed to correspond to a block number that is less than the current output. 


## The block number of the output root proposals should monotonically increase.
**Test:** [`L85`](https://github.com/ethereum-optimism/optimism/tree/develop/packages/contracts-bedrock/contracts/test/invariants/L2OutputOracle.t.sol#L85)

When a new output is submitted, it should never be allowed to correspond to a block number that is less than the current output. 
This is a stripped version of `invariant_monotonicBlockNumIncrease` that gives foundry's invariant fuzzer less context. 
