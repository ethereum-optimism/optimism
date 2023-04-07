# Ownable
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/util/Ownable.sol)

**Inherits:**
[IOwnable](/src/interfaces/IOwnable.sol/interface.IOwnable.md)

**Author:**
Adapted from Solmate (https://github.com/transmissions11/solmate/blob/main/src/auth/Owned.sol)

Simple single owner contract.


## State Variables
### _owner
*The owner of the contract.*


```solidity
address internal _owner;
```


## Functions
### onlyOwner


```solidity
modifier onlyOwner() virtual;
```

### constructor


```solidity
constructor(address initialOwner);
```

### owner

Returns the owner of the contract


```solidity
function owner() public view returns (address);
```

### transferOwnership

Transfer ownership to the passed address


```solidity
function transferOwnership(address newOwner) public virtual onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`newOwner`|`address`|The address to transfer ownership to|


## Events
### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed user, address indexed newOwner);
```

