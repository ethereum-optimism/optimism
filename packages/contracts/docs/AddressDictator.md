# AddressDictator



> AddressDictator



*The AddressDictator (glory to Arstotzka) is a contract that allows us to safely manipulate      many different addresses in the AddressManager without transferring ownership of the      AddressManager to a hot wallet or hardware wallet.*

## Methods

### c_0x59a12b1b

```solidity
function c_0x59a12b1b(bytes32 c__0x59a12b1b) external pure
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| c__0x59a12b1b | bytes32 | undefined

### finalOwner

```solidity
function finalOwner() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### getNamedAddresses

```solidity
function getNamedAddresses() external view returns (struct AddressDictator.NamedAddress[])
```

Returns the full namedAddresses array.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | AddressDictator.NamedAddress[] | undefined

### manager

```solidity
function manager() external view returns (contract Lib_AddressManager)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract Lib_AddressManager | undefined

### returnOwnership

```solidity
function returnOwnership() external nonpayable
```

Transfers ownership of this contract to the finalOwner. Only callable by the Final Owner, which is intended to be our multisig. This function shouldn&#39;t be necessary, but it gives a sense of reassurance that we can recover if something really surprising goes wrong.




### setAddresses

```solidity
function setAddresses() external nonpayable
```

Called to finalize the transfer, this function is callable by anyone, but will only result in an upgrade if this contract is the owner Address Manager.







