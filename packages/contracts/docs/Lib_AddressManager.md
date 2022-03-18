# Lib_AddressManager



> Lib_AddressManager





## Methods

### getAddress

```solidity
function getAddress(string _name) external view returns (address)
```

Retrieves the address associated with a given name.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _name | string | Name to retrieve an address for.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address associated with the given name.

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### setAddress

```solidity
function setAddress(string _name, address _address) external nonpayable
```

Changes the address associated with a particular name.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _name | string | String name to associate an address with.
| _address | address | Address to associate with the name.

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined



## Events

### AddressSet

```solidity
event AddressSet(string indexed _name, address _newAddress, address _oldAddress)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _name `indexed` | string | undefined |
| _newAddress  | address | undefined |
| _oldAddress  | address | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |



