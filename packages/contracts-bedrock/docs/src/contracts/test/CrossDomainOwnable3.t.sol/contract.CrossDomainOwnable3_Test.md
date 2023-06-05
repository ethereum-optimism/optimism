# CrossDomainOwnable3_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CrossDomainOwnable3.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## State Variables
### setter

```solidity
XDomainSetter3 setter;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() public;
```

### test_localOnlyOwner_notOwner_reverts


```solidity
function test_localOnlyOwner_notOwner_reverts() public;
```

### test_transferOwnership_notOwner_reverts


```solidity
function test_transferOwnership_notOwner_reverts() public;
```

### test_crossDomainOnlyOwner_notOwner_reverts


```solidity
function test_crossDomainOnlyOwner_notOwner_reverts() public;
```

### test_crossDomainOnlyOwner_notOwner2_reverts


```solidity
function test_crossDomainOnlyOwner_notOwner2_reverts() public;
```

### test_crossDomainOnlyOwner_notMessenger_reverts


```solidity
function test_crossDomainOnlyOwner_notMessenger_reverts() public;
```

### test_transferOwnership_zeroAddress_reverts


```solidity
function test_transferOwnership_zeroAddress_reverts() public;
```

### test_transferOwnership_noLocalZeroAddress_reverts


```solidity
function test_transferOwnership_noLocalZeroAddress_reverts() public;
```

### test_localOnlyOwner_succeeds


```solidity
function test_localOnlyOwner_succeeds() public;
```

### test_localTransferOwnership_succeeds


```solidity
function test_localTransferOwnership_succeeds() public;
```

### test_transferOwnershipNoLocal_succeeds

The existing transferOwnership(address) method
still exists on the contract


```solidity
function test_transferOwnershipNoLocal_succeeds() public;
```

### test_crossDomainTransferOwnership_succeeds


```solidity
function test_crossDomainTransferOwnership_succeeds() public;
```

## Events
### OwnershipTransferred
OpenZeppelin Ownable.sol transferOwnership event


```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);
```

### OwnershipTransferred
CrossDomainOwnable3.sol transferOwnership event


```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner, bool isLocal);
```

