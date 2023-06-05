# Encoding_Invariant
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/Encoding.t.sol)

**Inherits:**
StdInvariant, Test


## State Variables
### actor

```solidity
Encoding_Converter internal actor;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### invariant_round_trip_encoding_AToB


```solidity
function invariant_round_trip_encoding_AToB() external;
```

### invariant_round_trip_encoding_BToA


```solidity
function invariant_round_trip_encoding_BToA() external;
```

