# OptimismMintableERC721Factory_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismMintableERC721Factory.t.sol)

**Inherits:**
[ERC721Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.ERC721Bridge_Initializer.md)


## State Variables
### factory

```solidity
OptimismMintableERC721Factory internal factory;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_createOptimismMintableERC721_succeeds


```solidity
function test_createOptimismMintableERC721_succeeds() external;
```

### test_createOptimismMintableERC721_zeroRemoteToken_reverts


```solidity
function test_createOptimismMintableERC721_zeroRemoteToken_reverts() external;
```

## Events
### OptimismMintableERC721Created

```solidity
event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);
```

