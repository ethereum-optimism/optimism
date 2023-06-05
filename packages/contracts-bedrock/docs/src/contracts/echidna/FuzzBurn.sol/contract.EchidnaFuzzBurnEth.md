# EchidnaFuzzBurnEth
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzBurn.sol)

**Inherits:**
StdUtils


## State Variables
### failedEthBurn

```solidity
bool internal failedEthBurn;
```


## Functions
### testBurn

Takes an integer amount of eth to burn through the Burn library and
updates the contract state if an incorrect amount of eth moved from the contract


```solidity
function testBurn(uint256 _value) public;
```

### echidna_burn_eth


```solidity
function echidna_burn_eth() public view returns (bool);
```

