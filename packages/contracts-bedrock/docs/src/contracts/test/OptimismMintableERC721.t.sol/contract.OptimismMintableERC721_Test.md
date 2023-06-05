# OptimismMintableERC721_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismMintableERC721.t.sol)

**Inherits:**
[ERC721Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.ERC721Bridge_Initializer.md)


## State Variables
### L1Token

```solidity
ERC721 internal L1Token;
```


### L2Token

```solidity
OptimismMintableERC721 internal L2Token;
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

### test_supportsInterfaces_succeeds

Ensure that the contract supports the expected interfaces.


```solidity
function test_supportsInterfaces_succeeds() external;
```

### test_safeMint_succeeds


```solidity
function test_safeMint_succeeds() external;
```

### test_safeMint_notBridge_reverts


```solidity
function test_safeMint_notBridge_reverts() external;
```

### test_burn_succeeds


```solidity
function test_burn_succeeds() external;
```

### test_burn_notBridge_reverts


```solidity
function test_burn_notBridge_reverts() external;
```

### test_tokenURI_succeeds


```solidity
function test_tokenURI_succeeds() external;
```

## Events
### Transfer

```solidity
event Transfer(address indexed from, address indexed to, uint256 indexed tokenId);
```

### Mint

```solidity
event Mint(address indexed account, uint256 tokenId);
```

### Burn

```solidity
event Burn(address indexed account, uint256 tokenId);
```

