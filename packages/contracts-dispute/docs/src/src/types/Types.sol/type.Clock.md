# Clock
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/types/Types.sol)

A `Clock` represents a packed `Duration` and `Timestamp`

*The packed layout of this type is as follows:
┌────────────┬────────────────┐
│    Bits    │     Value      │
├────────────┼────────────────┤
│ [0, 128)   │ Duration       │
│ [128, 256) │ Timestamp      │
└────────────┴────────────────┘*


```solidity
type Clock is uint256;
```

