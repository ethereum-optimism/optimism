# FaultDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/FaultDisputeGame.sol)

**Inherits:**
[IFaultDisputeGame](/src/interfaces/IFaultDisputeGame.sol/interface.IFaultDisputeGame.md), [Clone](/src/util/Clone.sol/contract.Clone.md), [Initializable](/src/util/Initializable.sol/abstract.Initializable.md)

**Authors:**
clabby <https://github.com/clabby>, protolambda <https://github.com/protolambda>

An implementation of the `IFaultDisputeGame` interface.


## State Variables
### MAX_GAME_DEPTH
The max depth of the game.

*TODO: Update this to the value that we will use in prod.*


```solidity
uint256 internal constant MAX_GAME_DEPTH = 4;
```


### GAME_DURATION
The duration of the game.

*TODO: Account for resolution buffer.*


```solidity
Duration internal constant GAME_DURATION = Duration.wrap(7 days);
```


### ROOT_POSITION
The root claim's position is always at depth 0; index 0.


```solidity
Position internal constant ROOT_POSITION = Position.wrap(0);
```


### gameStart
The starting timestamp of the game


```solidity
Timestamp public gameStart;
```


### bondManager
The DisputeGame's bond manager.


```solidity
IBondManager public bondManager;
```


### leftMostPosition
The left most, deepest position found during the resolution phase.

*Defaults to the position of the root claim, but will be set during the resolution
phase to the left most, deepest position found (if any qualify.)*


```solidity
Position public leftMostPosition;
```


### claims
Maps a unique ClaimHash to a Claim.


```solidity
mapping(ClaimHash => Claim) public claims;
```


### parents
Maps a unique ClaimHash to its parent.


```solidity
mapping(ClaimHash => ClaimHash) public parents;
```


### positions
Maps a unique ClaimHash to its position in the game tree.


```solidity
mapping(ClaimHash => Position) public positions;
```


### bonds
Maps a unique ClaimHash to a Bond.


```solidity
mapping(ClaimHash => Bond) public bonds;
```


### clocks
Maps a unique ClaimHash its chess clock.


```solidity
mapping(ClaimHash => Clock) public clocks;
```


### rc
Maps a unique ClaimHash to its reference counter.


```solidity
mapping(ClaimHash => uint64) public rc;
```


### countered
Tracks whether or not a unique ClaimHash has been countered.


```solidity
mapping(ClaimHash => bool) public countered;
```


## Functions
### attack

Attack a disagreed upon ClaimHash.


```solidity
function attack(ClaimHash disagreement, Claim pivot) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`disagreement`|`ClaimHash`|Disagreed upon ClaimHash|
|`pivot`|`Claim`|The supplied pivot to the disagreement.|


### defend


```solidity
function defend(ClaimHash agreement, Claim pivot) external;
```

### step

Performs a VM step


```solidity
function step(ClaimHash disagreement) public;
```

### _move

Performs a VM step via an on-chain fault proof processor

Internal move function, used by both `attack` and `defend`.

*This function should point to a fault proof processor in order to execute
a step in the fault proof program on-chain. The interface of the fault proof processor
contract should be generic enough such that we can use different fault proof VMs (MIPS, RiscV5, etc.)*


```solidity
function _move(ClaimHash claimHash, Claim pivot, bool isAttack) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The claim hash that the move is being made against.|
|`pivot`|`Claim`|The pivot point claim provided in response to `claimHash`.|
|`isAttack`|`bool`|Whether or not the move is an attack or defense.|


### initialize

Initializes the `DisputeGame_Fault` contract.


```solidity
function initialize() external initializer;
```

### version

Returns the semantic version of the DisputeGame contract.

*Current version: 0.0.1*


```solidity
function version() external pure override returns (string memory);
```

### gameType

Fetches the game type from the calldata appended by the CWIA proxy.

*`clones-with-immutable-args` argument #1*

*The reference impl should be entirely different depending on the type (fault, validity)
i.e. The game type should indicate the security model.*


```solidity
function gameType() public pure override returns (GameType _gameType);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_gameType`|`GameType`|The type of proof system being used.|


### rootClaim

Fetches the root claim from the calldata appended by the CWIA proxy.

*`clones-with-immutable-args` argument #2*


```solidity
function rootClaim() public pure returns (Claim _rootClaim);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rootClaim`|`Claim`|The root claim of the DisputeGame.|


### createdAt

Returns the timestamp that the DisputeGame contract was created at.


```solidity
function createdAt() external view returns (Timestamp _createdAt);
```

### status

Returns the current status of the game.


```solidity
function status() external pure returns (GameStatus _status);
```

### extraData

Getter for the extra data.

*`clones-with-immutable-args` argument #3*


```solidity
function extraData() external pure returns (bytes memory _extraData);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_extraData`|`bytes`|Any extra data supplied to the dispute game contract by the creator.|


### resolve

If all necessary information has been gathered, this function should mark the game
status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
the resolved game. It is at this stage that the bonds should be awarded to the
necessary parties.

*May only be called if the `status` is `IN_PROGRESS`.*


```solidity
function resolve() external pure returns (GameStatus _status);
```

