# AddressManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/AddressManager.sol)

**Inherits:**
Ownable

AddressManager is a legacy contract that was used in the old version of the Optimism
system to manage a registry of string names to addresses. We now use a more standard
proxy system instead, but this contract is still necessary for backwards compatibility
with several older contracts.


## State Variables
### addresses
Mapping of the hashes of string names to addresses.


```solidity
mapping(bytes32 => address) private addresses;
```


## Functions
### setAddress

Changes the address associated with a particular name.


```solidity
function setAddress(string memory _name, address _address) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_name`|`string`|   String name to associate an address with.|
|`_address`|`address`|Address to associate with the name.|


### getAddress

Retrieves the address associated with a given name.


```solidity
function getAddress(string memory _name) external view returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_name`|`string`|Name to retrieve an address for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address associated with the given name.|


### _getNameHash

Computes the hash of a name.


```solidity
function _getNameHash(string memory _name) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_name`|`string`|Name to compute a hash for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hash of the given name.|


## Events
### AddressSet
Emitted when an address is modified in the registry.


```solidity
event AddressSet(string indexed name, address newAddress, address oldAddress);
```

