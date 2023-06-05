# CrossDomainOwnable2_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CrossDomainOwnable2.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## State Variables
### setter

```solidity
XDomainSetter2 setter;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_onlyOwner_notMessenger_reverts


```solidity
function test_onlyOwner_notMessenger_reverts() external;
```

### test_onlyOwner_notOwner_reverts


```solidity
function test_onlyOwner_notOwner_reverts() external;
```

### test_onlyOwner_notOwner2_reverts


```solidity
function test_onlyOwner_notOwner2_reverts() external;
```

### test_onlyOwner_succeeds


```solidity
function test_onlyOwner_succeeds() external;
```

