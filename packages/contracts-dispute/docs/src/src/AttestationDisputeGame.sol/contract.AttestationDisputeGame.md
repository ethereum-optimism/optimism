# AttestationDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/AttestationDisputeGame.sol)

**Inherits:**
[IDisputeGame](/src/interfaces/IDisputeGame.sol/interface.IDisputeGame.md), [Owner](/src/util/Owner.sol/abstract.Owner.md), [Initialize](/src/util/Initialize.sol/contract.Initialize.md)

**Author:**
refcell <https://github.com/refcell>

The attestation dispute game allows a permissioned set of challengers to dispute an output.

The contract owner should be the `L2OutputOracle`.

Whereas the provided challengerSet is intended to be a multisig responsible for resolving the dispute.


## State Variables
### gameStart
The starting timestamp of the game


```solidity
Timestamp public gameStart;
```


### l2BlockNumber
The l2 block number for which the output to dispute


```solidity
uint256 public l2BlockNumber;
```


### challengeSet
The set of challengers that can challenge the output.

*This should be a multisig that can resolve the dispute.*

*The multisig must reach a quorum before calling `resolve`.*


```solidity
address public challengeSet;
```


### gameStatus
The game status.


```solidity
GameStatus internal gameStatus;
```


## Functions
### constructor

Instantiates a new AttestationDisputeGame contract.


```solidity
constructor(address _owner, uint256 _blockNum, address _challengeSet) Owner(_owner);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|The owner of the contract.|
|`_blockNum`|`uint256`|The l2 block number for which the output to dispute.|
|`_challengeSet`|`address`|The set of challengers that can challenge the output.|


### initialize

Initializes the challenge contract.


```solidity
function initialize() external initializer;
```

### version

Returns the semantic version.


```solidity
function version() external pure override returns (string memory);
```

### status

Returns the current status of the game.


```solidity
function status() external view override returns (GameStatus _status);
```

### gameType

Returns the dispute game type.


```solidity
function gameType() external pure override returns (GameType _gameType);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_gameType`|`GameType`|The type of proof system being used.|


### extraData

Getter for the extra data.

*`clones-with-immutable-args` argument #3*


```solidity
function extraData() external pure override returns (bytes memory _extraData);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_extraData`|`bytes`|Any extra data supplied to the dispute game contract by the creator.|


### bondManager

Attestation games do not have bond managers.

This will return an invalid IBondManager at address 0x0.


```solidity
function bondManager() external pure returns (IBondManager _bondManager);
```

### rootClaim

Returns the output that is being disputed.


```solidity
function rootClaim() external view override returns (Claim _rootClaim);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rootClaim`|`Claim`|The root claim of the DisputeGame.|


### resolve

If all necessary information has been gathered, this function should mark the game
status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
the resolved game. It is at this stage that the bonds should be awarded to the
necessary parties.

*May only be called if the `status` is `IN_PROGRESS`.*


```solidity
function resolve() external override returns (GameStatus _status);
```

