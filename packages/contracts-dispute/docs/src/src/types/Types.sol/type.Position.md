# Position
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/types/Types.sol)

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

