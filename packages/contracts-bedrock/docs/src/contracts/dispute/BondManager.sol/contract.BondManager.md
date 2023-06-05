# BondManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/dispute/BondManager.sol)

The Bond Manager serves as an escrow for permissionless output proposal bonds.


## State Variables
### bonds
Mapping from bondId to bond.


```solidity
mapping(bytes32 => Bond) public bonds;
```


### DISPUTE_GAME_FACTORY
The permissioned dispute game factory.

*Used to verify the status of bonds.*


```solidity
IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;
```


### TRANSFER_GAS
Amount of gas used to transfer ether when splitting the bond.
This is a reasonable amount of gas for a transfer, even to a smart contract.
The number of participants is bound of by the block gas limit.


```solidity
uint256 private constant TRANSFER_GAS = 30_000;
```


## Functions
### constructor

Instantiates the bond maanger with the registered dispute game factory.


```solidity
constructor(IDisputeGameFactory _disputeGameFactory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_disputeGameFactory`|`IDisputeGameFactory`|is the dispute game factory.|


### post

Post a bond with a given id and owner.

*This function will revert if the provided bondId is already in use.*


```solidity
function post(bytes32 _bondId, address _bondOwner, uint256 _minClaimHold) external payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|
|`_bondOwner`|`address`|is the address that owns the bond.|
|`_minClaimHold`|`uint256`|is the minimum amount of time the owner must wait before reclaiming their bond.|


### seize

Seizes the bond with the given id.

*This function will revert if there is no bond at the given id.*


```solidity
function seize(bytes32 _bondId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|


### seizeAndSplit

Seizes the bond with the given id and distributes it to recipients.

*This function will revert if there is no bond at the given id.*


```solidity
function seizeAndSplit(bytes32 _bondId, address[] calldata _claimRecipients) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|
|`_claimRecipients`|`address[]`|is a set of addresses to split the bond amongst.|


### reclaim

Reclaims the bond of the bond owner.

*This function will revert if there is no bond at the given id.*


```solidity
function reclaim(bytes32 _bondId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|


## Events
### BondPosted
BondPosted is emitted when a bond is posted.


```solidity
event BondPosted(bytes32 bondId, address owner, uint256 expiration, uint256 amount);
```

### BondSeized
BondSeized is emitted when a bond is seized.


```solidity
event BondSeized(bytes32 bondId, address owner, address seizer, uint256 amount);
```

### BondReclaimed
BondReclaimed is emitted when a bond is reclaimed by the owner.


```solidity
event BondReclaimed(bytes32 bondId, address claiment, uint256 amount);
```

## Structs
### Bond
The Bond Type


```solidity
struct Bond {
    address owner;
    uint256 expiration;
    bytes32 id;
    uint256 amount;
}
```

