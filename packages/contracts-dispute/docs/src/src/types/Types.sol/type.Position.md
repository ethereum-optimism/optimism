# Position
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/types/Types.sol)

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

