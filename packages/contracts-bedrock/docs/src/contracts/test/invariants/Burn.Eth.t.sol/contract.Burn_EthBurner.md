# Burn_EthBurner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/Burn.Eth.t.sol)

**Inherits:**
StdUtils


## State Variables
### vm

```solidity
Vm internal vm;
```


### failedEthBurn

```solidity
bool public failedEthBurn;
```


## Functions
### constructor


```solidity
constructor(Vm _vm);
```

### burnEth

Takes an integer amount of eth to burn through the Burn library and
updates the contract state if an incorrect amount of eth moved from the contract


```solidity
function burnEth(uint256 _value) external;
```

