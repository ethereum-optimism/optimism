# LegacyMessagePasser
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/LegacyMessagePasser.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The LegacyMessagePasser was the low-level mechanism used to send messages from L2 to L1
before the Bedrock upgrade. It is now deprecated in favor of the new MessagePasser.


## State Variables
### sentMessages
Mapping of sent message hashes to boolean status.


```solidity
mapping(bytes32 => bool) public sentMessages;
```


## Functions
### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### passMessageToL1

Passes a message to L1.


```solidity
function passMessageToL1(bytes memory _message) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_message`|`bytes`|Message to pass to L1.|


