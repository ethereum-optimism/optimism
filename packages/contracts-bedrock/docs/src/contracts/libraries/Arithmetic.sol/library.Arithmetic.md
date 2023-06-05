# Arithmetic
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Arithmetic.sol)

Even more math than before.


## Functions
### clamp

Clamps a value between a minimum and maximum.


```solidity
function clamp(int256 _value, int256 _min, int256 _max) internal pure returns (int256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_value`|`int256`|The value to clamp.|
|`_min`|`int256`|  The minimum value.|
|`_max`|`int256`|  The maximum value.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`int256`|The clamped value.|


### cdexp

(c)oefficient (d)enominator (exp)onentiation function.
Returns the result of: c * (1 - 1/d)^exp.


```solidity
function cdexp(int256 _coefficient, int256 _denominator, int256 _exponent) internal pure returns (int256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_coefficient`|`int256`|Coefficient of the function.|
|`_denominator`|`int256`|Fractional denominator.|
|`_exponent`|`int256`|   Power function exponent.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`int256`|Result of c * (1 - 1/d)^exp.|


