# ILegacyMintableERC20
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/IOptimismMintableERC20.sol)

**Inherits:**
IERC165

This interface was available on the legacy L2StandardERC20 contract. It remains available
on the OptimismMintableERC20 contract for backwards compatibility.


## Functions
### l1Token


```solidity
function l1Token() external view returns (address);
```

### mint


```solidity
function mint(address _to, uint256 _amount) external;
```

### burn


```solidity
function burn(address _from, uint256 _amount) external;
```

