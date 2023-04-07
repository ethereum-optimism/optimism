# Challenge
[Git Source](https://github.com/ethereum-optimism/cannon-v2-contracts/blob/896a9e7a2e7769b1273deb0b0a9ed4c533f56f75/src/Challenge.sol)

**Inherits:**
[Clone](/src/util/Clone.sol/contract.Clone.md), [Initializable](/src/util/Initializable.sol/contract.Initializable.md)

The Challenge contract (desc: TODO)


## State Variables
### ROOT
The starting generic index is always 1, which represents the root node
of the trie. The root is a gindex starting with 1, other nodes are a bitlength
matching depth + bitpath of the route to the node.


```solidity
Types.Gindex internal constant ROOT = Types.Gindex.wrap(1);
```


### claims
Maps a unique GindexClaim to a Claim.


```solidity
mapping(Types.GindexClaim => Types.Claim) public claims;
```


### bonds
Maps a unique GindexClaim to a Bond.


```solidity
mapping(Types.GindexClaim => Types.Bond) public bonds;
```


### startingTimestamps
Maps a unique GindexClaim to its starting timestamp.


```solidity
mapping(Types.GindexClaim => Types.Timestamp) public startingTimestamps;
```


### clocks
Maps a unique GindexClaim its chess clock.


```solidity
mapping(Types.GindexClaim => Types.Duration) public clocks;
```


### gIndicies
Maps a unique GindexClaim to its Gindex.


```solidity
mapping(Types.GindexClaim => Types.Gindex) public gIndicies;
```


### parents
Maps a unique GindexClaim to its parent.


```solidity
mapping(Types.GindexClaim => Types.GindexClaim) public parents;
```


## Functions
### init

Initializes the challenge contract.


```solidity
function init() external initializer;
```

### subClaim

Allows anyone to go into the tree


```solidity
function subClaim(Types.GindexClaim disagreement, Types.Claim pivot) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`disagreement`|`GindexClaim.Types`|Disagreed upon gindex|
|`pivot`|`Claim.Types`|Allows others to go deeper into the tree|


### getRootClaim

Fetches the root claim from the calldata that the CWIA proxy appends
to the end when delegatecalling this contract.


```solidity
function getRootClaim() public view returns (Types.Claim rootClaim);
```

