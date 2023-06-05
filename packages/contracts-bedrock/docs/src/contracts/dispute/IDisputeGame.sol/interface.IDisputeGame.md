# IDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/dispute/IDisputeGame.sol)

**Inherits:**
[IInitializable](/contracts/dispute/IInitializable.sol/interface.IInitializable.md), [IVersioned](/contracts/dispute/IVersioned.sol/interface.IVersioned.md)

The generic interface for a DisputeGame contract.


## Functions
### createdAt

Returns the timestamp that the DisputeGame contract was created at.

Returns the timestamp that the DisputeGame contract was created at.


```solidity
function createdAt() external view returns (Timestamp _createdAt);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_createdAt`|`Timestamp`|The timestamp that the DisputeGame contract was created at.|


### status

Returns the current status of the game.


```solidity
function status() external view returns (GameStatus _status);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_status`|`GameStatus`|The current status of the game.|


### gameType

Getter for the game type.

*`clones-with-immutable-args` argument #1*

*The reference impl should be entirely different depending on the type (fault, validity)
i.e. The game type should indicate the security model.*


```solidity
function gameType() external view returns (GameType _gameType);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_gameType`|`GameType`|The type of proof system being used.|


### rootClaim

Getter for the root claim.

*`clones-with-immutable-args` argument #2*


```solidity
function rootClaim() external view returns (Claim _rootClaim);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rootClaim`|`Claim`|The root claim of the DisputeGame.|


### extraData

Getter for the extra data.

*`clones-with-immutable-args` argument #3*


```solidity
function extraData() external view returns (bytes memory _extraData);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_extraData`|`bytes`|Any extra data supplied to the dispute game contract by the creator.|


### bondManager

Returns the address of the `BondManager` used.


```solidity
function bondManager() external view returns (IBondManager _bondManager);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_bondManager`|`IBondManager`|The address of the `BondManager` used.|


### resolve

If all necessary information has been gathered, this function should mark the game
status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
the resolved game. It is at this stage that the bonds should be awarded to the
necessary parties.

*May only be called if the `status` is `IN_PROGRESS`.*


```solidity
function resolve() external returns (GameStatus _status);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_status`|`GameStatus`|The status of the game after resolution.|


## Events
### Resolved
Emitted when the game is resolved.


```solidity
event Resolved(GameStatus indexed status);
```

