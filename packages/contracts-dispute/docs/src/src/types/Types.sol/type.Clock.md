# Clock
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/types/Types.sol)

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

