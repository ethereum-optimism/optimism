# IBondManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/interfaces/IBondManager.sol)

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

