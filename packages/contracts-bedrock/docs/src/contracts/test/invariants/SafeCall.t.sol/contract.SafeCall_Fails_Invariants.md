# SafeCall_Fails_Invariants
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/SafeCall.t.sol)

**Inherits:**
Test


## State Variables
### actor

```solidity
SafeCaller_Actor actor;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### invariant_callWithMinGas_neverForwardsMinGas_reverts


```solidity
function invariant_callWithMinGas_neverForwardsMinGas_reverts() public;
```

### performSafeCallMinGas


```solidity
function performSafeCallMinGas(address to, uint64 minGas) external payable;
```

