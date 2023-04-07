# SafeCall
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/lib/SafeCall.sol)

**Author:**
ethereum-optimism (https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/libraries/SafeCall.sol)

Perform low level calls, assuring a valid return.


## Functions
### call

Perform a low level call without copying any returndata


```solidity
function call(address _target, uint256 _gas, uint256 _value, bytes memory _calldata) internal returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|  Address to call|
|`_gas`|`uint256`|     Amount of gas to pass to the call|
|`_value`|`uint256`|   Amount of value to pass to the call|
|`_calldata`|`bytes`|Calldata to pass to the call|


### callWithMinGas

Perform a low level call without copying any returndata. This function
will revert if the call cannot be performed with the specified minimum
gas.


```solidity
function callWithMinGas(address _target, uint256 _minGas, uint256 _value, bytes memory _calldata)
    internal
    returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|  Address to call|
|`_minGas`|`uint256`|  The minimum amount of gas that may be passed to the call|
|`_value`|`uint256`|   Amount of value to pass to the call|
|`_calldata`|`bytes`|Calldata to pass to the call|


