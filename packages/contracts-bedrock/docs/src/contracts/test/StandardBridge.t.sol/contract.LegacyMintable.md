# LegacyMintable
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/StandardBridge.t.sol)

**Inherits:**
ERC20, [ILegacyMintableERC20](/contracts/universal/IOptimismMintableERC20.sol/interface.ILegacyMintableERC20.md)

Simple implementation of the legacy OptimismMintableERC20.


## Functions
### constructor


```solidity
constructor(string memory _name, string memory _ticker) ERC20(_name, _ticker);
```

### l1Token


```solidity
function l1Token() external pure returns (address);
```

### mint


```solidity
function mint(address _to, uint256 _amount) external pure;
```

### burn


```solidity
function burn(address _from, uint256 _amount) external pure;
```

### supportsInterface

Implements ERC165. This implementation should not be changed as
it is how the actual legacy optimism mintable token does the
check. Allows for testing against code that is has been deployed,
assuming different compiler version is no problem.


```solidity
function supportsInterface(bytes4 _interfaceId) external pure returns (bool);
```

