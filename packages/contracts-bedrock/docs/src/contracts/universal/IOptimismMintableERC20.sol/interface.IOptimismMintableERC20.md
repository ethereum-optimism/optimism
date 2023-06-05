# IOptimismMintableERC20
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/IOptimismMintableERC20.sol)

**Inherits:**
IERC165

This interface is available on the OptimismMintableERC20 contract. We declare it as a
separate interface so that it can be used in custom implementations of
OptimismMintableERC20.


## Functions
### remoteToken


```solidity
function remoteToken() external view returns (address);
```

### bridge


```solidity
function bridge() external returns (address);
```

### mint


```solidity
function mint(address _to, uint256 _amount) external;
```

### burn


```solidity
function burn(address _from, uint256 _amount) external;
```

