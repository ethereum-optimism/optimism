# LibClock
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/lib/LibClock.sol)

**Author:**
clabby <https://github.com/clabby>

This library contains helper functions for working with the `Clock` type.


## Functions
### wrap

Packs a `Duration` and `Timestamp` into a `Clock` type.


```solidity
function wrap(Duration _duration, Timestamp _timestamp) internal pure returns (Clock _clock);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_duration`|`Duration`|The `Duration` to pack into the `Clock` type.|
|`_timestamp`|`Timestamp`|The `Timestamp` to pack into the `Clock` type.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_clock`|`Clock`|The `Clock` containing the `_duration` and `_timestamp`.|


### duration

Pull the `Duration` out of a `Clock` type.


```solidity
function duration(Clock _clock) internal pure returns (Duration _duration);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_clock`|`Clock`|The `Clock` type to pull the `Duration` out of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_duration`|`Duration`|The `Duration` pulled out of `_clock`.|


### timestamp

Pull the `Timestamp` out of a `Clock` type.


```solidity
function timestamp(Clock _clock) internal pure returns (Timestamp _timestamp);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_clock`|`Clock`|The `Clock` type to pull the `Timestamp` out of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_timestamp`|`Timestamp`|The `Timestamp` pulled out of `_clock`.|


