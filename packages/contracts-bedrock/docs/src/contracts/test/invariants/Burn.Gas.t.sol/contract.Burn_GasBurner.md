# Burn_GasBurner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/Burn.Gas.t.sol)

**Inherits:**
StdUtils


## State Variables
### vm

```solidity
Vm internal vm;
```


### failedGasBurn

```solidity
bool public failedGasBurn;
```


## Functions
### constructor


```solidity
constructor(Vm _vm);
```

### burnGas

Takes an integer amount of gas to burn through the Burn library and
updates the contract state if at least that amount of gas was not burned
by the library


```solidity
function burnGas(uint256 _value) external;
```

