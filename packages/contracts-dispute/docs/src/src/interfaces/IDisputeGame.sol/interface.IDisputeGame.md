# IDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IDisputeGame.sol)

**Inherits:**
[Initializable](/src/interfaces/Initializable.sol/interface.Initializable.md), [Versioned](/src/interfaces/Versioned.sol/interface.Versioned.md)

**Authors:**
clabby <https://github.com/clabby>, refcell <https://github.com/refcell>

The generic interface for a DisputeGame contract.


## Functions
### status

Returns the current status of the game.


```solidity
function status() external view returns (GameStatus _status);
```

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

Returns the address of the `BondManager` used to handle in-game bonds.


```solidity
function bondManager() external view returns (IBondManager _bondManager);
```

### resolve

If all necessary information has been gathered, this function should mark the game
status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
the resolved game. It is at this stage that the bonds should be awarded to the
necessary parties.

*May only be called if the `status` is `IN_PROGRESS`.*


```solidity
function resolve() external returns (GameStatus _status);
```

## Events
### Resolved
Emitted when the game is resolved.
TODO: Define the semantics of this event.


```solidity
event Resolved();
```

