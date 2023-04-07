# DisputeGameFactory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/DisputeGameFactory.sol)

**Inherits:**
[IDisputeGameFactory](/src/interfaces/IDisputeGameFactory.sol/interface.IDisputeGameFactory.md), [Owner](/src/util/Owner.sol/abstract.Owner.md)

**Author:**
refcell <https://github.com/refcell>

A factory contract for creating [`DisputeGame`] contracts.


## State Variables
### disputeGames
Mapping of GameType to the `DisputeGame` proxy contract.

*The GameType id is computed as the hash of `gameType . rootClaim . extraData`.*


```solidity
mapping(GameType => IDisputeGame) internal disputeGames;
```


## Functions
### constructor

Constructs a new DisputeGameFactory contract.


```solidity
constructor(address _owner) Owner(_owner);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|The owner of the contract.|


### games

Retrieves the hash of `gameType . rootClaim . extraData` to the deployed `DisputeGame` clone.

*Note: `.` denotes concatenation.*


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


### owner

The owner of the contract.

The owner can update the implementation contracts for a given GameType.


```solidity
function owner() external view returns (address _owner);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|The owner of the contract.|


### getGameID

Returns a game id for the given dispute game parameters.


```solidity
function getGameID(GameType gameType, Claim rootClaim, bytes calldata extraData) public pure returns (bytes32);
```

### getImplementation

Gets the `IDisputeGame` for a given `GameType`.

*Notice, we can just use the `games` mapping to get the implementation.*

*This works since clones are mapped using a hash of `gameType . rootClaim . extraData`.*


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


### create

Creates a new DisputeGame proxy contract.

If a dispute game with the given parameters already exists, it will be returned.


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
|`proxy`|`IDisputeGame`|The clone of the `DisputeGame` created with the given parameters.|


### setImplementation

Sets the implementation contract for a specific `GameType`


```solidity
function setImplementation(GameType gameType, IDisputeGame impl) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gameType`|`GameType`|The type of the DisputeGame|
|`impl`|`IDisputeGame`|The implementation contract for the given `GameType`|


