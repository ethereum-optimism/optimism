# L1CrossDomainMessenger
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/L1CrossDomainMessenger.sol)

**Inherits:**
[CrossDomainMessenger](/contracts/universal/CrossDomainMessenger.sol/abstract.CrossDomainMessenger.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L1CrossDomainMessenger is a message passing interface between L1 and L2 responsible
for sending and receiving data on the L1 side. Users are encouraged to use this
interface instead of interacting with lower-level contracts directly.


## State Variables
### PORTAL
Address of the OptimismPortal.


```solidity
OptimismPortal public immutable PORTAL;
```


## Functions
### constructor


```solidity
constructor(OptimismPortal _portal) Semver(1, 4, 0) CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_portal`|`OptimismPortal`|Address of the OptimismPortal contract on this network.|


### initialize

Initializer.


```solidity
function initialize() public initializer;
```

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


