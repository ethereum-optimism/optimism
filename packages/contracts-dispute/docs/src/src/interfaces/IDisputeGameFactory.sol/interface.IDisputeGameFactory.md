# IDisputeGameFactory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/interfaces/IDisputeGameFactory.sol)

**Inherits:**
[IOwnable](/src/interfaces/IOwnable.sol/interface.IOwnable.md)

**Author:**
clabby <https://github.com/clabby>

The interface for a DisputeGameFactory contract.


## Functions
### games

`games` queries an internal a mapping that maps the hash of `gameType ++ rootClaim ++ extraData`
to the deployed `DisputeGame` clone.

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
|`_proxy`|`IDisputeGame`|The clone of the `DisputeGame` created with the given parameters. Returns `address(0)` if nonexistent.|


### gameImpls

`gameImpls` is a mapping that maps `GameType`s to their respective `IDisputeGame` implementations.


```solidity
function gameImpls(GameType gameType) external view returns (IDisputeGame _impl);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the dispute game.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_impl`|`IDisputeGame`|The address of the implementation of the game type. Will be cloned on creation of a new dispute game with the given `gameType`.|


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

*May only be called by the `owner`.*


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

