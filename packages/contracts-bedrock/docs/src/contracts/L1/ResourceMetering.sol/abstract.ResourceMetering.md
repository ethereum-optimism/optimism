# ResourceMetering
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/ResourceMetering.sol)

**Inherits:**
Initializable

ResourceMetering implements an EIP-1559 style resource metering system where pricing
updates automatically based on current demand.


## State Variables
### params
EIP-1559 style gas parameters.


```solidity
ResourceParams public params;
```


### __gap
Reserve extra slots (to a total of 50) in the storage layout for future upgrades.


```solidity
uint256[48] private __gap;
```


## Functions
### metered

Meters access to a function based an amount of a requested resource.


```solidity
modifier metered(uint64 _amount);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_amount`|`uint64`|Amount of the resource requested.|


### _metered

An internal function that holds all of the logic for metering a resource.


```solidity
function _metered(uint64 _amount, uint256 _initialGas) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_amount`|`uint64`|    Amount of the resource requested.|
|`_initialGas`|`uint256`|The amount of gas before any modifier execution.|


### _resourceConfig

Virtual function that returns the resource config. Contracts that inherit this
contract must implement this function.


```solidity
function _resourceConfig() internal virtual returns (ResourceConfig memory);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`ResourceConfig`|ResourceConfig|


### __ResourceMetering_init

Sets initial resource parameter values. This function must either be called by the
initializer function of an upgradeable child contract.


```solidity
function __ResourceMetering_init() internal onlyInitializing;
```

## Structs
### ResourceParams
Represents the various parameters that control the way in which resources are
metered. Corresponds to the EIP-1559 resource metering system.


```solidity
struct ResourceParams {
    uint128 prevBaseFee;
    uint64 prevBoughtGas;
    uint64 prevBlockNum;
}
```

### ResourceConfig
Represents the configuration for the EIP-1559 based curve for the deposit gas
market. These values should be set with care as it is possible to set them in
a way that breaks the deposit gas market. The target resource limit is defined as
maxResourceLimit / elasticityMultiplier. This struct was designed to fit within a
single word. There is additional space for additions in the future.


```solidity
struct ResourceConfig {
    uint32 maxResourceLimit;
    uint8 elasticityMultiplier;
    uint8 baseFeeMaxChangeDenominator;
    uint32 minimumBaseFee;
    uint32 systemTxMaxGas;
    uint128 maximumBaseFee;
}
```

