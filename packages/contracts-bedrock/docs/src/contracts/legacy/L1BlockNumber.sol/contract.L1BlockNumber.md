# L1BlockNumber
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/L1BlockNumber.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

L1BlockNumber is a legacy contract that fills the roll of the OVM_L1BlockNumber contract
in the old version of the Optimism system. Only necessary for backwards compatibility.
If you want to access the L1 block number going forward, you should use the L1Block
contract instead.


## Functions
### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### receive

Returns the L1 block number.


```solidity
receive() external payable;
```

### fallback

Returns the L1 block number.


```solidity
fallback() external payable;
```

### getL1BlockNumber

Retrieves the latest L1 block number.


```solidity
function getL1BlockNumber() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Latest L1 block number.|


