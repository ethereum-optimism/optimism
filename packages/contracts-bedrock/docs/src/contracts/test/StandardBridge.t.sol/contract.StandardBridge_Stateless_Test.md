# StandardBridge_Stateless_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/StandardBridge.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)

Tests internal functions that require no existing state or contract
interactions with the messenger.


## State Variables
### bridge

```solidity
StandardBridgeTester internal bridge;
```


### mintable

```solidity
OptimismMintableERC20 internal mintable;
```


### erc20

```solidity
ERC20 internal erc20;
```


### legacy

```solidity
LegacyMintable internal legacy;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_isOptimismMintableERC20_succeeds

Test coverage for identifying OptimismMintableERC20 tokens.
This function should return true for both modern and legacy
OptimismMintableERC20 tokens and false for any accounts that
do not implement the interface.


```solidity
function test_isOptimismMintableERC20_succeeds() external;
```

### test_isCorrectTokenPair_succeeds

Test coverage of isCorrectTokenPair under different types of
tokens.


```solidity
function test_isCorrectTokenPair_succeeds() external;
```

