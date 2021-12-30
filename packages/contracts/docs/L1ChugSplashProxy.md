# L1ChugSplashProxy



> L1ChugSplashProxy



*Basic ChugSplash proxy contract for L1. Very close to being a normal proxy but has added functions `setCode` and `setStorage` for changing the code or storage of the contract. Nifty! Note for future developers: do NOT make anything in this contract &#39;public&#39; unless you know what you&#39;re doing. Anything public can potentially have a function signature that conflicts with a signature attached to the implementation contract. Public functions SHOULD always have the &#39;proxyCallIfNotOwner&#39; modifier unless there&#39;s some *really* good reason not to have that modifier. And there almost certainly is not a good reason to not have that modifier. Beware!*

## Methods

### getImplementation

```solidity
function getImplementation() external nonpayable returns (address)
```

Queries the implementation address. Can only be called by the owner OR by making an eth_call and setting the &quot;from&quot; address to address(0).




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Implementation address.

### getOwner

```solidity
function getOwner() external nonpayable returns (address)
```

Queries the owner of the proxy contract. Can only be called by the owner OR by making an eth_call and setting the &quot;from&quot; address to address(0).




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Owner address.

### setCode

```solidity
function setCode(bytes _code) external nonpayable
```

Sets the code that should be running behind this proxy. Note that this scheme is a bit different from the standard proxy scheme where one would typically deploy the code separately and then set the implementation address. We&#39;re doing it this way because it gives us a lot more freedom on the client side. Can only be triggered by the contract owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _code | bytes | New contract code to run inside this contract.

### setOwner

```solidity
function setOwner(address _owner) external nonpayable
```

Changes the owner of the proxy contract. Only callable by the owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _owner | address | New owner of the proxy contract.

### setStorage

```solidity
function setStorage(bytes32 _key, bytes32 _value) external nonpayable
```

Modifies some storage slot within the proxy contract. Gives us a lot of power to perform upgrades in a more transparent way. Only callable by the owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _key | bytes32 | Storage key to modify.
| _value | bytes32 | New value for the storage key.




