# Clock
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/DisputeTypes.sol)

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

