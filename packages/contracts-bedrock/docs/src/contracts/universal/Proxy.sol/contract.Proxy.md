# Proxy
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/Proxy.sol)

Proxy is a transparent proxy that passes through the call if the caller is the owner or
if the caller is address(0), meaning that the call originated from an off-chain
simulation.


## State Variables
### IMPLEMENTATION_KEY
The storage slot that holds the address of the implementation.
bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)


```solidity
bytes32 internal constant IMPLEMENTATION_KEY = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
```


### OWNER_KEY
The storage slot that holds the address of the owner.
bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)


```solidity
bytes32 internal constant OWNER_KEY = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;
```


## Functions
### proxyCallIfNotAdmin

A modifier that reverts if not called by the owner or by address(0) to allow
eth_call to interact with this proxy without needing to use low-level storage
inspection. We assume that nobody is able to trigger calls from address(0) during
normal EVM execution.


```solidity
modifier proxyCallIfNotAdmin();
```

### constructor

Sets the initial admin during contract deployment. Admin address is stored at the
EIP-1967 admin storage slot so that accidental storage collision with the
implementation is not possible.


```solidity
constructor(address _admin);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_admin`|`address`|Address of the initial contract admin. Admin as the ability to access the transparent proxy interface.|


### receive


```solidity
receive() external payable;
```

### fallback


```solidity
fallback() external payable;
```

### upgradeTo

Set the implementation contract address. The code at the given address will execute
when this contract is called.


```solidity
function upgradeTo(address _implementation) public virtual proxyCallIfNotAdmin;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_implementation`|`address`|Address of the implementation contract.|


### upgradeToAndCall

Set the implementation and call a function in a single transaction. Useful to ensure
atomic execution of initialization-based upgrades.


```solidity
function upgradeToAndCall(address _implementation, bytes calldata _data)
    public
    payable
    virtual
    proxyCallIfNotAdmin
    returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_implementation`|`address`|Address of the implementation contract.|
|`_data`|`bytes`|          Calldata to delegatecall the new implementation with.|


### changeAdmin

Changes the owner of the proxy contract. Only callable by the owner.


```solidity
function changeAdmin(address _admin) public virtual proxyCallIfNotAdmin;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_admin`|`address`|New owner of the proxy contract.|


### admin

Gets the owner of the proxy contract.


```solidity
function admin() public virtual proxyCallIfNotAdmin returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Owner address.|


### implementation

Queries the implementation address.


```solidity
function implementation() public virtual proxyCallIfNotAdmin returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Implementation address.|


### _setImplementation

Sets the implementation address.


```solidity
function _setImplementation(address _implementation) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_implementation`|`address`|New implementation address.|


### _changeAdmin

Changes the owner of the proxy contract.


```solidity
function _changeAdmin(address _admin) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_admin`|`address`|New owner of the proxy contract.|


### _doProxyCall

Performs the proxy call via a delegatecall.


```solidity
function _doProxyCall() internal;
```

### _getImplementation

Queries the implementation address.


```solidity
function _getImplementation() internal view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Implementation address.|


### _getAdmin

Queries the owner of the proxy contract.


```solidity
function _getAdmin() internal view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Owner address.|


## Events
### Upgraded
An event that is emitted each time the implementation is changed. This event is part
of the EIP-1967 specification.


```solidity
event Upgraded(address indexed implementation);
```

### AdminChanged
An event that is emitted each time the owner is upgraded. This event is part of the
EIP-1967 specification.


```solidity
event AdminChanged(address previousAdmin, address newAdmin);
```

