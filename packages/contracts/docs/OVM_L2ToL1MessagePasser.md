# OVM_L2ToL1MessagePasser



> OVM_L2ToL1MessagePasser



*The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the of a message on L2. The L1 Cross Domain Messenger performs this proof in its _verifyStorageProof function, which verifies the existence of the transaction hash in this contract&#39;s `sentMessages` mapping.*

## Methods

### passMessageToL1

```solidity
function passMessageToL1(bytes _message) external nonpayable
```

Passes a message to L1.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _message | bytes | Message to pass to L1.

### sentMessages

```solidity
function sentMessages(bytes32) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined



## Events

### L2ToL1Message

```solidity
event L2ToL1Message(uint256 _nonce, address _sender, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _nonce  | uint256 | undefined |
| _sender  | address | undefined |
| _data  | bytes | undefined |



