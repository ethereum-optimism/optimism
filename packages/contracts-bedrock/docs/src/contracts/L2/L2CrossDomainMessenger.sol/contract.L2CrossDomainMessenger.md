# L2CrossDomainMessenger
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/L2CrossDomainMessenger.sol)

**Inherits:**
[CrossDomainMessenger](/contracts/universal/CrossDomainMessenger.sol/abstract.CrossDomainMessenger.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L2CrossDomainMessenger is a high-level interface for message passing between L1 and
L2 on the L2 side. Users are generally encouraged to use this contract instead of lower
level message passing contracts.


## Functions
### constructor


```solidity
constructor(address _l1CrossDomainMessenger) Semver(1, 4, 0) CrossDomainMessenger(_l1CrossDomainMessenger);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1CrossDomainMessenger`|`address`|Address of the L1CrossDomainMessenger contract.|


### initialize

Initializer.


```solidity
function initialize() public initializer;
```

### l1CrossDomainMessenger

Legacy getter for the remote messenger. Use otherMessenger going forward.


```solidity
function l1CrossDomainMessenger() public view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the L1CrossDomainMessenger contract.|


### _sendMessage

Sends a low-level message to the other messenger. Needs to be implemented by child
contracts because the logic for this depends on the network where the messenger is
being deployed.


```solidity
function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal override;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|      Recipient of the message on the other chain.|
|`_gasLimit`|`uint64`|Minimum gas limit the message can be executed with.|
|`_value`|`uint256`|   Amount of ETH to send with the message.|
|`_data`|`bytes`|    Message data.|


### _isOtherMessenger

Checks whether the message is coming from the other messenger. Implemented by child
contracts because the logic for this depends on the network where the messenger is
being deployed.


```solidity
function _isOtherMessenger() internal view override returns (bool);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether the message is coming from the other messenger.|


### _isUnsafeTarget

Checks whether a given call target is a system address that could cause the
messenger to peform an unsafe action. This is NOT a mechanism for blocking user
addresses. This is ONLY used to prevent the execution of messages to specific
system addresses that could cause security issues, e.g., having the
CrossDomainMessenger send messages to itself.


```solidity
function _isUnsafeTarget(address _target) internal view override returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|Address of the contract to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the address is an unsafe system address.|


