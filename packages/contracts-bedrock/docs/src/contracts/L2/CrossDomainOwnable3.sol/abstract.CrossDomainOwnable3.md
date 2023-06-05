# CrossDomainOwnable3
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/CrossDomainOwnable3.sol)

**Inherits:**
Ownable

This contract extends the OpenZeppelin `Ownable` contract for L2 contracts to be owned
by contracts on either L1 or L2. Note that this contract is meant to be used with systems
that use the CrossDomainMessenger system. It will not work if the OptimismPortal is
used directly.


## State Variables
### isLocal
If true, the contract uses the cross domain _checkOwner function override. If false
it uses the standard Ownable _checkOwner function.


```solidity
bool public isLocal = true;
```


## Functions
### transferOwnership

Allows for ownership to be transferred with specifying the locality.


```solidity
function transferOwnership(address _owner, bool _isLocal) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|  The new owner of the contract.|
|`_isLocal`|`bool`|Configures the locality of the ownership.|


### _checkOwner

Overrides the implementation of the `onlyOwner` modifier to check that the unaliased
`xDomainMessageSender` is the owner of the contract. This value is set to the caller
of the L1CrossDomainMessenger.


```solidity
function _checkOwner() internal view override;
```

## Events
### OwnershipTransferred
Emits when ownership of the contract is transferred. Includes the
isLocal field in addition to the standard `Ownable` OwnershipTransferred event.


```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner, bool isLocal);
```

