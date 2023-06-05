# L1ChugSplashProxy
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/L1ChugSplashProxy.sol)

Basic ChugSplash proxy contract for L1. Very close to being a normal proxy but has added
functions `setCode` and `setStorage` for changing the code or storage of the contract.
Note for future developers: do NOT make anything in this contract 'public' unless you
know what you're doing. Anything public can potentially have a function signature that
conflicts with a signature attached to the implementation contract. Public functions
SHOULD always have the `proxyCallIfNotOwner` modifier unless there's some *really* good
reason not to have that modifier. And there almost certainly is not a good reason to not
have that modifier. Beware!


## State Variables
### DEPLOY_CODE_PREFIX
"Magic" prefix. When prepended to some arbitrary bytecode and used to create a
contract, the appended bytecode will be deployed as given.


```solidity
bytes13 internal constant DEPLOY_CODE_PREFIX = 0x600D380380600D6000396000f3;
```


### IMPLEMENTATION_KEY
bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)


```solidity
bytes32 internal constant IMPLEMENTATION_KEY = 0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;
```


### OWNER_KEY
bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)


```solidity
bytes32 internal constant OWNER_KEY = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;
```


## Functions
### onlyWhenNotPaused

Blocks a function from being called when the parent signals that the system should
be paused via an isUpgrading function.


```solidity
modifier onlyWhenNotPaused();
```

### proxyCallIfNotOwner

Makes a proxy call instead of triggering the given function when the caller is
either the owner or the zero address. Caller can only ever be the zero address if
this function is being called off-chain via eth_call, which is totally fine and can
be convenient for client-side tooling. Avoids situations where the proxy and
implementation share a sighash and the proxy function ends up being called instead
of the implementation one.
Note: msg.sender == address(0) can ONLY be triggered off-chain via eth_call. If
there's a way for someone to send a transaction with msg.sender == address(0) in any
real context then we have much bigger problems. Primary reason to include this
additional allowed sender is because the owner address can be changed dynamically
and we do not want clients to have to keep track of the current owner in order to
make an eth_call that doesn't trigger the proxied contract.


```solidity
modifier proxyCallIfNotOwner();
```

### constructor


```solidity
constructor(address _owner);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|Address of the initial contract owner.|


### receive


```solidity
receive() external payable;
```

### fallback


```solidity
fallback() external payable;
```

### setCode

Sets the code that should be running behind this proxy.
Note: This scheme is a bit different from the standard proxy scheme where one would
typically deploy the code separately and then set the implementation address. We're
doing it this way because it gives us a lot more freedom on the client side. Can
only be triggered by the contract owner.


```solidity
function setCode(bytes memory _code) external proxyCallIfNotOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_code`|`bytes`|New contract code to run inside this contract.|


### setStorage

Modifies some storage slot within the proxy contract. Gives us a lot of power to
perform upgrades in a more transparent way. Only callable by the owner.


```solidity
function setStorage(bytes32 _key, bytes32 _value) external proxyCallIfNotOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_key`|`bytes32`|  Storage key to modify.|
|`_value`|`bytes32`|New value for the storage key.|


### setOwner

Changes the owner of the proxy contract. Only callable by the owner.


```solidity
function setOwner(address _owner) external proxyCallIfNotOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|New owner of the proxy contract.|


### getOwner

Queries the owner of the proxy contract. Can only be called by the owner OR by
making an eth_call and setting the "from" address to address(0).


```solidity
function getOwner() external proxyCallIfNotOwner returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Owner address.|


### getImplementation

Queries the implementation address. Can only be called by the owner OR by making an
eth_call and setting the "from" address to address(0).


```solidity
function getImplementation() external proxyCallIfNotOwner returns (address);
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


### _setOwner

Changes the owner of the proxy contract.


```solidity
function _setOwner(address _owner) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|New owner of the proxy contract.|


### _doProxyCall

Performs the proxy call via a delegatecall.


```solidity
function _doProxyCall() internal onlyWhenNotPaused;
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


### _getOwner

Queries the owner of the proxy contract.


```solidity
function _getOwner() internal view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Owner address.|


### _getAccountCodeHash

Gets the code hash for a given account.


```solidity
function _getAccountCodeHash(address _account) internal view returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_account`|`address`|Address of the account to get a code hash for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Code hash for the account.|


