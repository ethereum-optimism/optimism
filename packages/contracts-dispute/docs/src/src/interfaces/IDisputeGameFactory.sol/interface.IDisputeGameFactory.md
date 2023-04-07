# IDisputeGameFactory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IDisputeGameFactory.sol)

**Author:**
clabby <https://github.com/clabby>

The interface for a DisputeGameFactory contract.


## Functions
### games

`games` is a mapping that maps the hash of `gameType ++ rootClaim ++ extraData` to the deployed
`DisputeGame` clone.

*`++` equates to concatenation.*


```solidity
function games(GameType gameType, Claim rootClaim, bytes calldata extraData)
    external
    view
    returns (IDisputeGame _proxy);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the DisputeGame - used to decide the proxy implementation|
|`rootClaim`|`Claim`|The root claim of the DisputeGame.|
|`extraData`|`bytes`|Any extra data that should be provided to the created dispute game.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`IDisputeGame`|The clone of the `DisputeGame` created with the given parameters. address(0) if nonexistent.|


### getImplementation

Gets the `IDisputeGame` for a given `GameType`.


```solidity
function getImplementation(GameType gameType) external view returns (IDisputeGame _impl);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the dispute game.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_impl`|`IDisputeGame`|The address of the implementation of the game type. Will be cloned on creation.|


### owner

The owner of the contract.

*Owner Permissions:
- Update the implementation contracts for a given game type.*


```solidity
function owner() external view returns (address _owner);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|The owner of the contract.|


### create

Creates a new DisputeGame proxy contract.


```solidity
function create(GameType gameType, Claim rootClaim, bytes calldata extraData) external returns (IDisputeGame proxy);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the DisputeGame - used to decide the proxy implementation|
|`rootClaim`|`Claim`|The root claim of the DisputeGame.|
|`extraData`|`bytes`|Any extra data that should be provided to the created dispute game.|


### setImplementation

Sets the implementation contract for a specific `GameType`


```solidity
function setImplementation(GameType gameType, IDisputeGame impl) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the DisputeGame|
|`impl`|`IDisputeGame`|The implementation contract for the given `GameType`|


## Events
### DisputeGameCreated
Emitted when a new dispute game is created


```solidity
event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
```

