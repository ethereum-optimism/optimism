# EchidnaFuzzBurnGas
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzBurn.sol)

**Inherits:**
StdUtils


## State Variables
### failedGasBurn

```solidity
bool internal failedGasBurn;
```


## Functions
### testGas

Takes an integer amount of gas to burn through the Burn library and
updates the contract state if at least that amount of gas was not burned
by the library


```solidity
function testGas(uint256 _value) public;
```

### echidna_burn_gas


```solidity
function echidna_burn_gas() public view returns (bool);
```

