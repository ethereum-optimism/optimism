# MeterUser
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ResourceMetering.t.sol)

**Inherits:**
[ResourceMetering](/contracts/L1/ResourceMetering.sol/abstract.ResourceMetering.md)


## State Variables
### innerConfig

```solidity
ResourceMetering.ResourceConfig public innerConfig;
```


## Functions
### constructor


```solidity
constructor();
```

### initialize


```solidity
function initialize() public initializer;
```

### resourceConfig


```solidity
function resourceConfig() public view returns (ResourceMetering.ResourceConfig memory);
```

### _resourceConfig


```solidity
function _resourceConfig() internal view override returns (ResourceMetering.ResourceConfig memory);
```

### use


```solidity
function use(uint64 _amount) public metered(_amount);
```

### set


```solidity
function set(uint128 _prevBaseFee, uint64 _prevBoughtGas, uint64 _prevBlockNum) public;
```

### setParams


```solidity
function setParams(ResourceMetering.ResourceConfig memory newConfig) public;
```

