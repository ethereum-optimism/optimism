# ExternalRelay
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CrossDomainMessenger.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)

A mock external contract called via the SafeCall inside
the CrossDomainMessenger's `relayMessage` function.


## State Variables
### op

```solidity
address internal op;
```


### fuzzedSender

```solidity
address internal fuzzedSender;
```


### L1Messenger

```solidity
L1CrossDomainMessenger internal L1Messenger;
```


## Functions
### constructor


```solidity
constructor(L1CrossDomainMessenger _l1Messenger, address _op);
```

### _internalRelay

Internal helper function to relay a message and perform assertions.


```solidity
function _internalRelay(address _innerSender) internal;
```

### externalCallWithMinGas

externalCallWithMinGas is called by the CrossDomainMessenger.


```solidity
function externalCallWithMinGas() external payable;
```

### getCallData

Helper function to get the callData for an `externalCallWithMinGas


```solidity
function getCallData() public pure returns (bytes memory);
```

### setFuzzedSender

Helper function to set the fuzzed sender


```solidity
function setFuzzedSender(address _fuzzedSender) public;
```

## Events
### FailedRelayedMessage

```solidity
event FailedRelayedMessage(bytes32 indexed msgHash);
```

