# OptimismMintableERC721
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/OptimismMintableERC721.sol)

**Inherits:**
ERC721Enumerable, [IOptimismMintableERC721](/contracts/universal/IOptimismMintableERC721.sol/interface.IOptimismMintableERC721.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

This contract is the remote representation for some token that lives on another network,
typically an Optimism representation of an Ethereum-based token. Standard reference
implementation that can be extended or modified according to your needs.


## State Variables
### REMOTE_CHAIN_ID
Chain ID of the chain where the remote token is deployed.


```solidity
uint256 public immutable REMOTE_CHAIN_ID;
```


### REMOTE_TOKEN
Address of the token on the remote domain.


```solidity
address public immutable REMOTE_TOKEN;
```


### BRIDGE
Address of the ERC721 bridge on this network.


```solidity
address public immutable BRIDGE;
```


### baseTokenURI
Base token URI for this token.


```solidity
string public baseTokenURI;
```


## Functions
### onlyBridge

Modifier that prevents callers other than the bridge from calling the function.


```solidity
modifier onlyBridge();
```

### constructor


```solidity
constructor(address _bridge, uint256 _remoteChainId, address _remoteToken, string memory _name, string memory _symbol)
    ERC721(_name, _symbol)
    Semver(1, 1, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bridge`|`address`|       Address of the bridge on this network.|
|`_remoteChainId`|`uint256`|Chain ID where the remote token is deployed.|
|`_remoteToken`|`address`|  Address of the corresponding token on the other network.|
|`_name`|`string`|         ERC721 name.|
|`_symbol`|`string`|       ERC721 symbol.|


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

### safeMint

Mints some token ID for a user, checking first that contract recipients
are aware of the ERC721 protocol to prevent tokens from being forever locked.


```solidity
function safeMint(address _to, uint256 _tokenId) external virtual onlyBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|     Address of the user to mint the token for.|
|`_tokenId`|`uint256`|Token ID to mint.|


### burn

Burns a token ID from a user.


```solidity
function burn(address _from, uint256 _tokenId) external virtual onlyBridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|   Address of the user to burn the token from.|
|`_tokenId`|`uint256`|Token ID to burn.|


### supportsInterface

Checks if a given interface ID is supported by this contract.


```solidity
function supportsInterface(bytes4 _interfaceId) public view override(ERC721Enumerable, IERC165) returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_interfaceId`|`bytes4`|The interface ID to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|True if the interface ID is supported, false otherwise.|


### _baseURI

Returns the base token URI.


```solidity
function _baseURI() internal view virtual override returns (string memory);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`string`|Base token URI.|


