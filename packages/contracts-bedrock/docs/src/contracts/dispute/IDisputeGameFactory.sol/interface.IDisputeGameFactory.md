# IDisputeGameFactory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/dispute/IDisputeGameFactory.sol)

The interface for a DisputeGameFactory contract.


## Functions
### games

`games` queries an internal a mapping that maps the hash of
`gameType ++ rootClaim ++ extraData` to the deployed `DisputeGame` clone.

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

`gameImpls` is a mapping that maps `GameType`s to their respective
`IDisputeGame` implementations.


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

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`proxy`|`IDisputeGame`|The address of the created DisputeGame proxy.|


### setImplementation

Sets the implementation contract for a specific `GameType`.

*May only be called by the `owner`.*


```solidity
function setImplementation(GameType gameType, IDisputeGame impl) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the DisputeGame.|
|`impl`|`IDisputeGame`|The implementation contract for the given `GameType`.|


## Events
### DisputeGameCreated
Emitted when a new dispute game is created


```solidity
event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
```

### ImplementationSet
Emitted when a new game implementation added to the factory


```solidity
event ImplementationSet(address indexed impl, GameType indexed gameType);
```

