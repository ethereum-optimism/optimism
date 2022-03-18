# BondManager



> BondManager



*This contract is, for now, a stub of the &quot;real&quot; BondManager that does nothing but allow the &quot;OVM_Proposer&quot; to submit state root batches.*

## Methods

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




