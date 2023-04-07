# IFaultDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IFaultDisputeGame.sol)

**Inherits:**
[IDisputeGame](/src/interfaces/IDisputeGame.sol/interface.IDisputeGame.md)

The interface for a fault proof backed dispute game.


## Functions
### gameStart

State variable of the starting timestamp of the game, set on deployment.


```solidity
function gameStart() external view returns (Timestamp);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`Timestamp`|The starting timestamp of the game|


### claims

Maps a unique ClaimHash to a Claim.


```solidity
function claims(ClaimHash claimHash) external view returns (Claim claim);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`claim`|`Claim`|The Claim associated with the ClaimHash|


### parents

Maps a unique ClaimHash to its parent.


```solidity
function parents(ClaimHash claimHash) external view returns (ClaimHash parent);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`parent`|`ClaimHash`|The parent ClaimHash of the passed ClaimHash|


### positions

Maps a unique ClaimHash to its Position.


```solidity
function positions(ClaimHash claimHash) external view returns (Position position);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`position`|`Position`|The Position associated with the ClaimHash|


### bonds

Maps a unique ClaimHash to a Bond.


```solidity
function bonds(ClaimHash claimHash) external view returns (Bond bond);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`bond`|`Bond`|The Bond associated with the ClaimHash|


### clocks

Maps a unique ClaimHash its chess clock.


```solidity
function clocks(ClaimHash claimHash) external view returns (Clock clock);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`clock`|`Clock`|The chess clock associated with the ClaimHash|


### rc

Maps a unique ClaimHash to its reference counter.


```solidity
function rc(ClaimHash claimHash) external view returns (uint64 _rc);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique ClaimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rc`|`uint64`|The reference counter associated with the ClaimHash|


### countered

Maps a unique ClaimHash to a boolean indicating whether or not it has been countered.


```solidity
function countered(ClaimHash claimHash) external view returns (bool _countered);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|The unique claimHash|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_countered`|`bool`|Whether or not `claimHash` has been countered|


### attack

Disagree with a subclaim


```solidity
function attack(ClaimHash disagreement, Claim pivot) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`disagreement`|`ClaimHash`|The ClaimHash of the disagreement|
|`pivot`|`Claim`|The claimed pivot|


### defend

Agree with a subclaim


```solidity
function defend(ClaimHash agreement, Claim pivot) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`agreement`|`ClaimHash`|The ClaimHash of the agreement|
|`pivot`|`Claim`|The claimed pivot|


### step

Perform the final step via an on-chain fault proof processor

*This function should point to a fault proof processor in order to execute
a step in the fault proof program on-chain. The interface of the fault proof processor
contract should be generic enough such that we can use different fault proof VMs (MIPS, RiscV5, etc.)*


```solidity
function step(ClaimHash disagreement) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`disagreement`|`ClaimHash`|The ClaimHash of the disagreement|


## Events
### Attack
Emitted when a subclaim is disagreed upon by `claimant`

*Disagreeing with a subclaim is akin to attacking it.*


```solidity
event Attack(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);
```

### Defend
Emitted when a subclaim is agreed upon by `claimant`

*Agreeing with a subclaim is akin to defending it.*


```solidity
event Defend(ClaimHash indexed claimHash, Claim indexed pivot, address indexed claimant);
```

