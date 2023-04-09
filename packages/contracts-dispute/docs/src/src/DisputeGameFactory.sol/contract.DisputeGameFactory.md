# DisputeGameFactory
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/DisputeGameFactory.sol)

**Inherits:**
[IDisputeGameFactory](/src/interfaces/IDisputeGameFactory.sol/interface.IDisputeGameFactory.md), [Ownable](/src/util/Ownable.sol/abstract.Ownable.md)

**Authors:**
refcell <https://github.com/refcell>, clabby <https://github.com/clabby>

A factory contract for creating [`IDisputeGame`] contracts.


## State Variables
### gameImpls
Mapping of `GameType`s to their respective `IDisputeGame` implementations.

*Allows for the creation of clone proxies with immutable arguments.*


```solidity
mapping(GameType => IDisputeGame) public gameImpls;
```


### disputeGames
Mapping of a hash of `gameType . rootClaim . extraData` to the deployed `IDisputeGame` clone.

*Note: `.` denotes concatenation.*


```solidity
mapping(Hash => IDisputeGame) internal disputeGames;
```


## Functions
### constructor

Constructs a new DisputeGameFactory contract.


```solidity
constructor(address _owner) Ownable(_owner);
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
|`gameType`|`GameType`|The type of the DisputeGame - used to decide the implementation to clone.|
|`rootClaim`|`Claim`|The root claim of the DisputeGame.|
|`extraData`|`bytes`|Any extra data that should be provided to the created dispute game.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_proxy`|`IDisputeGame`|The clone of the `DisputeGame` created with the given parameters. `address(0)` if nonexistent.|


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


### getGameUUID

Returns a unique identifier for the given dispute game parameters.

*Hashes the concatenation of `gameType . rootClaim . extraData` without expanding memory.*


```solidity
function getGameUUID(GameType gameType, Claim rootClaim, bytes memory extraData) public pure returns (Hash _uuid);
```

