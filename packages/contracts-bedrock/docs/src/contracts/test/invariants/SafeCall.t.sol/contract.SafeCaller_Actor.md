# SafeCaller_Actor
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/SafeCall.t.sol)

**Inherits:**
StdUtils


## State Variables
### FAILS

```solidity
bool internal immutable FAILS;
```


### vm

```solidity
Vm internal vm;
```


### numCalls

```solidity
uint256 public numCalls;
```


## Functions
### constructor


```solidity
constructor(Vm _vm, bool _fails);
```

### performSafeCallMinGas


```solidity
function performSafeCallMinGas(uint64 gas, uint64 minGas, address to, uint8 value) external;
```

