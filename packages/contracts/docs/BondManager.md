# BondManager



> BondManager



*This contract is, for now, a stub of the &quot;real&quot; BondManager that does nothing but allow the &quot;OVM_Proposer&quot; to submit state root batches.*

## Methods

### c_0xaff0b0c8

```solidity
function c_0xaff0b0c8(bytes32 c__0xaff0b0c8) external pure
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| c__0xaff0b0c8 | bytes32 | undefined

### c_0xb9519e7a

```solidity
function c_0xb9519e7a(bytes32 c__0xb9519e7a) external pure
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| c__0xb9519e7a | bytes32 | undefined

### isCollateralized

```solidity
function isCollateralized(address _who) external view returns (bool)
```

Checks whether a given address is properly collateralized and can perform actions within the system.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _who | address | Address to check.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | true if the address is properly collateralized, false otherwise.

### libAddressManager

```solidity
function libAddressManager() external view returns (contract Lib_AddressManager)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract Lib_AddressManager | undefined

### resolve

```solidity
function resolve(string _name) external view returns (address)
```

Resolves the address associated with a given name.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _name | string | Name to resolve an address for.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address associated with the given name.




