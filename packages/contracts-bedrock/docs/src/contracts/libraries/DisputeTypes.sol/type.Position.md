# Position
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/DisputeTypes.sol)

A `Position` represents a position of a claim within the game tree.

*The packed layout of this type is as follows:
┌────────────┬────────────────┐
│    Bits    │     Value      │
├────────────┼────────────────┤
│ [0, 128)   │ Depth          │
│ [128, 256) │ Index at depth │
└────────────┴────────────────┘*


```solidity
type Position is uint256;
```

