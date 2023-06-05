# RelayActor
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/CrossDomainMessenger.t.sol)

**Inherits:**
StdUtils


## State Variables
### senderSlotIndex

```solidity
uint256 constant senderSlotIndex = 50;
```


### numHashes

```solidity
uint256 public numHashes;
```


### hashes

```solidity
bytes32[] public hashes;
```


### reverted

```solidity
bool public reverted = false;
```


### op

```solidity
OptimismPortal op;
```


### xdm

```solidity
L1CrossDomainMessenger xdm;
```


### vm

```solidity
Vm vm;
```


### doFail

```solidity
bool doFail;
```


## Functions
### constructor


```solidity
constructor(OptimismPortal _op, L1CrossDomainMessenger _xdm, Vm _vm, bool _doFail);
```

### relay

Relays a message to the `L1CrossDomainMessenger` with a random `version`, and `_message`.


```solidity
function relay(uint8 _version, uint8 _value, bytes memory _message) external;
```

