# EchidnaFuzzResourceMetering
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzResourceMetering.sol)

**Inherits:**
[ResourceMetering](/contracts/L1/ResourceMetering.sol/abstract.ResourceMetering.md), StdUtils


## State Variables
### failedMaxGasPerBlock

```solidity
bool internal failedMaxGasPerBlock;
```


### failedRaiseBaseFee

```solidity
bool internal failedRaiseBaseFee;
```


### failedLowerBaseFee

```solidity
bool internal failedLowerBaseFee;
```


### failedNeverBelowMinBaseFee

```solidity
bool internal failedNeverBelowMinBaseFee;
```


### failedMaxRaiseBaseFeePerBlock

```solidity
bool internal failedMaxRaiseBaseFeePerBlock;
```


### failedMaxLowerBaseFeePerBlock

```solidity
bool internal failedMaxLowerBaseFeePerBlock;
```


### underflow

```solidity
bool internal underflow;
```


## Functions
### constructor


```solidity
constructor();
```

### initialize


```solidity
function initialize() internal initializer;
```

### resourceConfig


```solidity
function resourceConfig() public pure returns (ResourceMetering.ResourceConfig memory);
```

### _resourceConfig


```solidity
function _resourceConfig() internal pure override returns (ResourceMetering.ResourceConfig memory);
```

### testBurn

Takes the necessary parameters to allow us to burn arbitrary amounts of gas to test
the underlying resource metering/gas market logic


```solidity
function testBurn(uint256 _gasToBurn, bool _raiseBaseFee) public;
```

### _burnInternal


```solidity
function _burnInternal(uint64 _gasToBurn) private metered(_gasToBurn);
```

### echidna_high_usage_raise_baseFee


```solidity
function echidna_high_usage_raise_baseFee() public view returns (bool);
```

### echidna_low_usage_lower_baseFee


```solidity
function echidna_low_usage_lower_baseFee() public view returns (bool);
```

### echidna_never_below_min_baseFee


```solidity
function echidna_never_below_min_baseFee() public view returns (bool);
```

### echidna_never_above_max_gas_limit


```solidity
function echidna_never_above_max_gas_limit() public view returns (bool);
```

### echidna_never_exceed_max_increase


```solidity
function echidna_never_exceed_max_increase() public view returns (bool);
```

### echidna_never_exceed_max_decrease


```solidity
function echidna_never_exceed_max_decrease() public view returns (bool);
```

### echidna_underflow


```solidity
function echidna_underflow() public view returns (bool);
```

