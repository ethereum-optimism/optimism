# ResolvedDelegateProxy
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/ResolvedDelegateProxy.sol)

ResolvedDelegateProxy is a legacy proxy contract that makes use of the AddressManager to
resolve the implementation address. We're maintaining this contract for backwards
compatibility so we can manage all legacy proxies where necessary.


## State Variables
### implementationName
Mapping used to store the implementation name that corresponds to this contract. A
mapping was originally used as a way to bypass the same issue normally solved by
storing the implementation address in a specific storage slot that does not conflict
with any other storage slot. Generally NOT a safe solution but works as long as the
implementation does not also keep a mapping in the first storage slot.


```solidity
mapping(address => string) private implementationName;
```


### addressManager
Mapping used to store the address of the AddressManager contract where the
implementation address will be resolved from. Same concept here as with the above
mapping. Also generally unsafe but fine if the implementation doesn't keep a mapping
in the second storage slot.


```solidity
mapping(address => AddressManager) private addressManager;
```


## Functions
### constructor


```solidity
constructor(AddressManager _addressManager, string memory _implementationName);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_addressManager`|`AddressManager`| Address of the AddressManager.|
|`_implementationName`|`string`|implementationName of the contract to proxy to.|


### fallback

Fallback, performs a delegatecall to the resolved implementation address.


```solidity
fallback() external payable;
```

