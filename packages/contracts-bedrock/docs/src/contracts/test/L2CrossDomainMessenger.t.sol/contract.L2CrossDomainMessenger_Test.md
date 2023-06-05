# L2CrossDomainMessenger_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2CrossDomainMessenger.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## State Variables
### recipient

```solidity
address recipient = address(0xabbaacdc);
```


## Functions
### test_messageVersion_succeeds


```solidity
function test_messageVersion_succeeds() external;
```

### test_sendMessage_succeeds


```solidity
function test_sendMessage_succeeds() external;
```

### test_sendMessage_twice_succeeds


```solidity
function test_sendMessage_twice_succeeds() external;
```

### test_xDomainSender_senderNotSet_reverts


```solidity
function test_xDomainSender_senderNotSet_reverts() external;
```

### test_relayMessage_v2_reverts


```solidity
function test_relayMessage_v2_reverts() external;
```

### test_relayMessage_succeeds


```solidity
function test_relayMessage_succeeds() external;
```

### test_relayMessage_toSystemContract_reverts


```solidity
function test_relayMessage_toSystemContract_reverts() external;
```

### test_xDomainMessageSender_reset_succeeds


```solidity
function test_xDomainMessageSender_reset_succeeds() external;
```

### test_relayMessage_retry_succeeds


```solidity
function test_relayMessage_retry_succeeds() external;
```

