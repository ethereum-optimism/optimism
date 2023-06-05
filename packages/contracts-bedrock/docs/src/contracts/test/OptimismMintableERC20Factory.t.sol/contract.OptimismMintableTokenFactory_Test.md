# OptimismMintableTokenFactory_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismMintableERC20Factory.t.sol)

**Inherits:**
[Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.Bridge_Initializer.md)


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_bridge_succeeds


```solidity
function test_bridge_succeeds() external;
```

### test_createStandardL2Token_succeeds


```solidity
function test_createStandardL2Token_succeeds() external;
```

### test_createStandardL2Token_sameTwice_succeeds


```solidity
function test_createStandardL2Token_sameTwice_succeeds() external;
```

### test_createStandardL2Token_remoteIsZero_succeeds


```solidity
function test_createStandardL2Token_remoteIsZero_succeeds() external;
```

## Events
### StandardL2TokenCreated

```solidity
event StandardL2TokenCreated(address indexed remoteToken, address indexed localToken);
```

### OptimismMintableERC20Created

```solidity
event OptimismMintableERC20Created(address indexed localToken, address indexed remoteToken, address deployer);
```

