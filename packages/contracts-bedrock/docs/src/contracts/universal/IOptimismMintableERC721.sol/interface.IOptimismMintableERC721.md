# IOptimismMintableERC721
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/IOptimismMintableERC721.sol)

**Inherits:**
IERC721Enumerable

Interface for contracts that are compatible with the OptimismMintableERC721 standard.
Tokens that follow this standard can be easily transferred across the ERC721 bridge.


## Functions
### safeMint

Mints some token ID for a user, checking first that contract recipients
are aware of the ERC721 protocol to prevent tokens from being forever locked.


```solidity
function safeMint(address _to, uint256 _tokenId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|     Address of the user to mint the token for.|
|`_tokenId`|`uint256`|Token ID to mint.|


### burn

Burns a token ID from a user.


```solidity
function burn(address _from, uint256 _tokenId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|   Address of the user to burn the token from.|
|`_tokenId`|`uint256`|Token ID to burn.|


### REMOTE_CHAIN_ID

Chain ID of the chain where the remote token is deployed.


```solidity
function REMOTE_CHAIN_ID() external view returns (uint256);
```

### REMOTE_TOKEN

Address of the token on the remote domain.


```solidity
function REMOTE_TOKEN() external view returns (address);
```

### BRIDGE

Address of the ERC721 bridge on this network.


```solidity
function BRIDGE() external view returns (address);
```

### remoteChainId

Chain ID of the chain where the remote token is deployed.


```solidity
function remoteChainId() external view returns (uint256);
```

### remoteToken

Address of the token on the remote domain.


```solidity
function remoteToken() external view returns (address);
```

### bridge

Address of the ERC721 bridge on this network.


```solidity
function bridge() external view returns (address);
```

## Events
### Mint
Emitted when a token is minted.


```solidity
event Mint(address indexed account, uint256 tokenId);
```

### Burn
Emitted when a token is burned.


```solidity
event Burn(address indexed account, uint256 tokenId);
```

