# Messenger_Initializer
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)


## State Variables
### addressManager

```solidity
AddressManager internal addressManager;
```


### L1Messenger

```solidity
L1CrossDomainMessenger internal L1Messenger;
```


### L2Messenger

```solidity
L2CrossDomainMessenger internal L2Messenger = L2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

## Events
### SentMessage

```solidity
event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit);
```

### SentMessageExtension1

```solidity
event SentMessageExtension1(address indexed sender, uint256 value);
```

### MessagePassed

```solidity
event MessagePassed(
    uint256 indexed nonce,
    address indexed sender,
    address indexed target,
    uint256 value,
    uint256 gasLimit,
    bytes data,
    bytes32 withdrawalHash
);
```

### RelayedMessage

```solidity
event RelayedMessage(bytes32 indexed msgHash);
```

### FailedRelayedMessage

```solidity
event FailedRelayedMessage(bytes32 indexed msgHash);
```

### TransactionDeposited

```solidity
event TransactionDeposited(
    address indexed from, address indexed to, uint256 mint, uint256 value, uint64 gasLimit, bool isCreation, bytes data
);
```

### WhatHappened

```solidity
event WhatHappened(bool success, bytes returndata);
```

