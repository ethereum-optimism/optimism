# CustomMeterUser
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ResourceMetering.t.sol)

**Inherits:**
[ResourceMetering](/contracts/L1/ResourceMetering.sol/abstract.ResourceMetering.md)

A simple wrapper around `ResourceMetering` that allows the initial
params to be set in the constructor.


## State Variables
### startGas

```solidity
uint256 public startGas;
```


### endGas

```solidity
uint256 public endGas;
```


## Functions
### constructor


```solidity
constructor(uint128 _prevBaseFee, uint64 _prevBoughtGas, uint64 _prevBlockNum);
```

### _resourceConfig


```solidity
function _resourceConfig() internal pure override returns (ResourceMetering.ResourceConfig memory);
```

### use


```solidity
function use(uint64 _amount) public returns (uint256);
```

