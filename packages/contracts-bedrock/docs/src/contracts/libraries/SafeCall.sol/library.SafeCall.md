# SafeCall
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/SafeCall.sol)

Perform low level safe calls


## Functions
### send

Performs a low level call without copying any returndata.

*Passes no calldata to the call context.*


```solidity
function send(address _target, uint256 _gas, uint256 _value) internal returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|  Address to call|
|`_gas`|`uint256`|     Amount of gas to pass to the call|
|`_value`|`uint256`|   Amount of value to pass to the call|


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


### hasMinGas

Helper function to determine if there is sufficient gas remaining within the context
to guarantee that the minimum gas requirement for a call will be met as well as
optionally reserving a specified amount of gas for after the call has concluded.

*!!!!! FOOTGUN ALERT !!!!!
1.) The 40_000 base buffer is to account for the worst case of the dynamic cost of the
`CALL` opcode's `address_access_cost`, `positive_value_cost`, and
`value_to_empty_account_cost` factors with an added buffer of 5,700 gas. It is
still possible to self-rekt by initiating a withdrawal with a minimum gas limit
that does not account for the `memory_expansion_cost` & `code_execution_cost`
factors of the dynamic cost of the `CALL` opcode.
2.) This function should *directly* precede the external call if possible. There is an
added buffer to account for gas consumed between this check and the call, but it
is only 5,700 gas.
3.) Because EIP-150 ensures that a maximum of 63/64ths of the remaining gas in the call
frame may be passed to a subcontext, we need to ensure that the gas will not be
truncated.
4.) Use wisely. This function is not a silver bullet.*


```solidity
function hasMinGas(uint256 _minGas, uint256 _reservedGas) internal view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_minGas`|`uint256`|     The minimum amount of gas that may be passed to the target context.|
|`_reservedGas`|`uint256`|Optional amount of gas to reserve for the caller after the execution of the target context.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|`true` if there is enough gas remaining to safely supply `_minGas` to the target context as well as reserve `_reservedGas` for the caller after the execution of the target context.|


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


