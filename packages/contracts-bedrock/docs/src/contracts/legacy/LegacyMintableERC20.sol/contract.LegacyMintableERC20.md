# LegacyMintableERC20
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/LegacyMintableERC20.sol)

**Inherits:**
[ILegacyMintableERC20](/contracts/universal/IOptimismMintableERC20.sol/interface.ILegacyMintableERC20.md), ERC20

The legacy implementation of the OptimismMintableERC20. This
contract is deprecated and should no longer be used.


## State Variables
### l1Token
The token on the remote domain.


```solidity
address public l1Token;
```


### l2Bridge
The local bridge.


```solidity
address public l2Bridge;
```


## Functions
### constructor


```solidity
constructor(address _l2Bridge, address _l1Token, string memory _name, string memory _symbol) ERC20(_name, _symbol);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2Bridge`|`address`|Address of the L2 standard bridge.|
|`_l1Token`|`address`|Address of the corresponding L1 token.|
|`_name`|`string`|ERC20 name.|
|`_symbol`|`string`|ERC20 symbol.|


### onlyL2Bridge

Modifier that requires the contract was called by the bridge.


```solidity
modifier onlyL2Bridge();
```

### supportsInterface

EIP165 implementation.


```solidity
function supportsInterface(bytes4 _interfaceId) public pure returns (bool);
```

### mint

Only the bridge can mint tokens.


```solidity
function mint(address _to, uint256 _amount) public virtual onlyL2Bridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|    The account receiving tokens.|
|`_amount`|`uint256`|The amount of tokens to receive.|


### burn

Only the bridge can burn tokens.


```solidity
function burn(address _from, uint256 _amount) public virtual onlyL2Bridge;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_from`|`address`|  The account having tokens burnt.|
|`_amount`|`uint256`|The amount of tokens being burnt.|


## Events
### Mint
Emitted when the token is minted by the bridge.


```solidity
event Mint(address indexed _account, uint256 _amount);
```

### Burn
Emitted when a token is burned by the bridge.


```solidity
event Burn(address indexed _account, uint256 _amount);
```

