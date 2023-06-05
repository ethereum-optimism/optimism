# ProxyAdmin
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/ProxyAdmin.sol)

**Inherits:**
Ownable

This is an auxiliary contract meant to be assigned as the admin of an ERC1967 Proxy,
based on the OpenZeppelin implementation. It has backwards compatibility logic to work
with the various types of proxies that have been deployed by Optimism in the past.


## State Variables
### proxyType
A mapping of proxy types, used for backwards compatibility.


```solidity
mapping(address => ProxyType) public proxyType;
```


### implementationName
A reverse mapping of addresses to names held in the AddressManager. This must be
manually kept up to date with changes in the AddressManager for this contract
to be able to work as an admin for the ResolvedDelegateProxy type.


```solidity
mapping(address => string) public implementationName;
```


### addressManager
The address of the address manager, this is required to manage the
ResolvedDelegateProxy type.


```solidity
AddressManager public addressManager;
```


### upgrading
A legacy upgrading indicator used by the old Chugsplash Proxy.


```solidity
bool internal upgrading;
```


## Functions
### constructor


```solidity
constructor(address _owner) Ownable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|Address of the initial owner of this contract.|


### setProxyType

Sets the proxy type for a given address. Only required for non-standard (legacy)
proxy types.


```solidity
function setProxyType(address _address, ProxyType _type) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_address`|`address`|Address of the proxy.|
|`_type`|`ProxyType`|   Type of the proxy.|


### setImplementationName

Sets the implementation name for a given address. Only required for
ResolvedDelegateProxy type proxies that have an implementation name.


```solidity
function setImplementationName(address _address, string memory _name) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_address`|`address`|Address of the ResolvedDelegateProxy.|
|`_name`|`string`|   Name of the implementation for the proxy.|


### setAddressManager

Set the address of the AddressManager. This is required to manage legacy
ResolvedDelegateProxy type proxy contracts.


```solidity
function setAddressManager(AddressManager _address) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_address`|`AddressManager`|Address of the AddressManager.|


### setAddress

Set an address in the address manager. Since only the owner of the AddressManager
can directly modify addresses and the ProxyAdmin will own the AddressManager, this
gives the owner of the ProxyAdmin the ability to modify addresses directly.


```solidity
function setAddress(string memory _name, address _address) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_name`|`string`|   Name to set within the AddressManager.|
|`_address`|`address`|Address to attach to the given name.|


### setUpgrading

Set the upgrading status for the Chugsplash proxy type.


```solidity
function setUpgrading(bool _upgrading) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_upgrading`|`bool`|Whether or not the system is upgrading.|


### isUpgrading

Legacy function used to tell ChugSplashProxy contracts if an upgrade is happening.


```solidity
function isUpgrading() external view returns (bool);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not there is an upgrade going on. May not actually tell you whether an upgrade is going on, since we don't currently plan to use this variable for anything other than a legacy indicator to fix a UX bug in the ChugSplash proxy.|


### getProxyImplementation

Returns the implementation of the given proxy address.


```solidity
function getProxyImplementation(address _proxy) external view returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`address`|Address of the proxy to get the implementation of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the implementation of the proxy.|


### getProxyAdmin

Returns the admin of the given proxy address.


```solidity
function getProxyAdmin(address payable _proxy) external view returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`address payable`|Address of the proxy to get the admin of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the admin of the proxy.|


### changeProxyAdmin

Updates the admin of the given proxy address.


```solidity
function changeProxyAdmin(address payable _proxy, address _newAdmin) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`address payable`|   Address of the proxy to update.|
|`_newAdmin`|`address`|Address of the new proxy admin.|


### upgrade

Changes a proxy's implementation contract.


```solidity
function upgrade(address payable _proxy, address _implementation) public onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`address payable`|         Address of the proxy to upgrade.|
|`_implementation`|`address`|Address of the new implementation address.|


### upgradeAndCall

Changes a proxy's implementation contract and delegatecalls the new implementation
with some given data. Useful for atomic upgrade-and-initialize calls.


```solidity
function upgradeAndCall(address payable _proxy, address _implementation, bytes memory _data)
    external
    payable
    onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`address payable`|         Address of the proxy to upgrade.|
|`_implementation`|`address`|Address of the new implementation address.|
|`_data`|`bytes`|          Data to trigger the new implementation with.|


## Enums
### ProxyType
The proxy types that the ProxyAdmin can manage.


```solidity
enum ProxyType {
    ERC1967,
    CHUGSPLASH,
    RESOLVED
}
```

