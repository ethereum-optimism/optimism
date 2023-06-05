# OptimismMintableERC721Factory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/OptimismMintableERC721Factory.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

Factory contract for creating OptimismMintableERC721 contracts.


## State Variables
### BRIDGE
Address of the ERC721 bridge on this network.


```solidity
address public immutable BRIDGE;
```


### REMOTE_CHAIN_ID
Chain ID for the remote network.


```solidity
uint256 public immutable REMOTE_CHAIN_ID;
```


### isOptimismMintableERC721
Tracks addresses created by this factory.


```solidity
mapping(address => bool) public isOptimismMintableERC721;
```


## Functions
### constructor

The semver MUST be bumped any time that there is a change in
the OptimismMintableERC721 token contract since this contract
is responsible for deploying OptimismMintableERC721 contracts.


```solidity
constructor(address _bridge, uint256 _remoteChainId) Semver(1, 2, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bridge`|`address`|Address of the ERC721 bridge on this network.|
|`_remoteChainId`|`uint256`|Chain ID for the remote network.|


### createOptimismMintableERC721

Creates an instance of the standard ERC721.


```solidity
function createOptimismMintableERC721(address _remoteToken, string memory _name, string memory _symbol)
    external
    returns (address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_remoteToken`|`address`|Address of the corresponding token on the other domain.|
|`_name`|`string`|       ERC721 name.|
|`_symbol`|`string`|     ERC721 symbol.|


## Events
### OptimismMintableERC721Created
Emitted whenever a new OptimismMintableERC721 contract is created.


```solidity
event OptimismMintableERC721Created(address indexed localToken, address indexed remoteToken, address deployer);
```

