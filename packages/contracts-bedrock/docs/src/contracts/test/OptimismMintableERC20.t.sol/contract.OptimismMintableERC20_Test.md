# OptimismMintableERC20_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismMintableERC20.t.sol)

**Inherits:**
[Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.Bridge_Initializer.md)


## Functions
### test_semver_succeeds


```solidity
function test_semver_succeeds() external;
```

### test_remoteToken_succeeds


```solidity
function test_remoteToken_succeeds() external;
```

### test_bridge_succeeds


```solidity
function test_bridge_succeeds() external;
```

### test_l1Token_succeeds


```solidity
function test_l1Token_succeeds() external;
```

### test_l2Bridge_succeeds


```solidity
function test_l2Bridge_succeeds() external;
```

### test_legacy_succeeds


```solidity
function test_legacy_succeeds() external;
```

### test_mint_succeeds


```solidity
function test_mint_succeeds() external;
```

### test_mint_notBridge_reverts


```solidity
function test_mint_notBridge_reverts() external;
```

### test_burn_succeeds


```solidity
function test_burn_succeeds() external;
```

### test_burn_notBridge_reverts


```solidity
function test_burn_notBridge_reverts() external;
```

### test_erc165_supportsInterface_succeeds


```solidity
function test_erc165_supportsInterface_succeeds() external;
```

## Events
### Mint

```solidity
event Mint(address indexed account, uint256 amount);
```

### Burn

```solidity
event Burn(address indexed account, uint256 amount);
```

