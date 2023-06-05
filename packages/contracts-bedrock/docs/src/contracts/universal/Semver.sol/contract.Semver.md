# Semver
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/Semver.sol)

Semver is a simple contract for managing contract versions.


## State Variables
### MAJOR_VERSION
Contract version number (major).


```solidity
uint256 private immutable MAJOR_VERSION;
```


### MINOR_VERSION
Contract version number (minor).


```solidity
uint256 private immutable MINOR_VERSION;
```


### PATCH_VERSION
Contract version number (patch).


```solidity
uint256 private immutable PATCH_VERSION;
```


## Functions
### constructor


```solidity
constructor(uint256 _major, uint256 _minor, uint256 _patch);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_major`|`uint256`|Version number (major).|
|`_minor`|`uint256`|Version number (minor).|
|`_patch`|`uint256`|Version number (patch).|


### version

Returns the full semver contract version.


```solidity
function version() public view returns (string memory);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`string`|Semver contract version as a string.|


