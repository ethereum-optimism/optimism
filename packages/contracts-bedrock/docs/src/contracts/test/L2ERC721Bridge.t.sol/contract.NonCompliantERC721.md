# NonCompliantERC721
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2ERC721Bridge.t.sol)

*A non-compliant ERC721 token that does not implement the full ERC721 interface.
This is used to test that the bridge will revert if the token does not claim to support
the ERC721 interface.*


## State Variables
### owner

```solidity
address internal immutable owner;
```


## Functions
### constructor


```solidity
constructor(address _owner);
```

### ownerOf


```solidity
function ownerOf(uint256) external view returns (address);
```

### remoteToken


```solidity
function remoteToken() external pure returns (address);
```

### burn


```solidity
function burn(address, uint256) external;
```

### supportsInterface


```solidity
function supportsInterface(bytes4) external pure returns (bool);
```

