# DisputeGameBondManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/DisputeGameBondManager.sol)

**Inherits:**
[IBondManager](/src/interfaces/IBondManager.sol/interface.IBondManager.md)

**Author:**
refcell <github.com/refcell>


## State Variables
### MIN_BOND_AMOUNT
The amount for each (dumb) bond.


```solidity
uint256 immutable MIN_BOND_AMOUNT;
```


### bonds
The internal mapping of bond id to Bond object.


```solidity
mapping(bytes32 => Bond) internal bonds;
```


## Functions
### constructor

Instantiates a new DisputeGameBondManager.


```solidity
constructor(uint256 amount);
```

### next

Returns the next minimum bond amount.


```solidity
function next() public view returns (uint256);
```

### post

Post a bond for a given game step id.

The id is expected to be the hash of the packed sender,
l2BlockNumber, and game step, calculated like so:
id = keccak256(abi.encodePacked(msg.sender, l2BlockNumber, step));


```solidity
function post(bytes32 id) external payable;
```

### call

Calls a bond for a given game step id.

The id is expected to be the hash of the packed sender,
l2BlockNumber, and game step, calculated like so:
id = keccak256(abi.encodePacked(msg.sender, l2BlockNumber, step));


```solidity
function call(bytes32 id, address to) external returns (uint256);
```

## Events
### BondPosted
Emitted when a bond is posted.

*Neither the owner or value are indexed since they are not sparse.*


```solidity
event BondPosted(bytes32 indexed id, address owner, uint256 value);
```

### BondCalled
Emitted when a bond is called.

*Neither the owner or value are indexed since they are not sparse.*


```solidity
event BondCalled(bytes32 indexed id, address owner, uint256 value);
```

## Structs
### Bond
Bond holds the bond owner and amount.


```solidity
struct Bond {
    address owner;
    uint256 value;
}
```

