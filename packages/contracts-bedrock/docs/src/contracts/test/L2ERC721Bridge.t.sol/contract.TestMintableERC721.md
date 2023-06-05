# TestMintableERC721
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2ERC721Bridge.t.sol)

**Inherits:**
[OptimismMintableERC721](/contracts/universal/OptimismMintableERC721.sol/contract.OptimismMintableERC721.md)


## Functions
### constructor


```solidity
constructor(address _bridge, address _remoteToken) OptimismMintableERC721(_bridge, 1, _remoteToken, "Test", "TST");
```

### mint


```solidity
function mint(address to, uint256 tokenId) public;
```

