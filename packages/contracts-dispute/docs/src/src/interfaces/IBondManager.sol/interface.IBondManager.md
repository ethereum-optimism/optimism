# IBondManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IBondManager.sol)

The Bond Manager holds ether posted as a bond for a bond id.


## Functions
### post

Post a bond for a given id.


```solidity
function post(bytes32 id) external payable;
```

### call

Calls a bond for a given bond id.

Only the address that posted the bond may claim it.


```solidity
function call(bytes32 id, address to) external returns (uint256);
```

### next

Returns the next minimum bond amount.


```solidity
function next() external returns (uint256);
```

