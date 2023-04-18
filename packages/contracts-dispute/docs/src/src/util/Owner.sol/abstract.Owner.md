# Owner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/util/Owner.sol)

**Author:**
Adapted from Solmate (https://github.com/transmissions11/solmate/blob/main/src/auth/Owned.sol)

Simple single owner contract.


## State Variables
### _owner

```solidity
address internal _owner;
```


## Functions
### onlyOwner


```solidity
modifier onlyOwner() virtual;
```

### constructor


```solidity
constructor(address newOwner);
```

### transferOwnership


```solidity
function transferOwnership(address newOwner) public virtual onlyOwner;
```

## Events
### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed user, address indexed newOwner);
```

