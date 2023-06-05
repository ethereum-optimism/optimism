# GovernanceToken
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/governance/GovernanceToken.sol)

**Inherits:**
ERC20Burnable, ERC20Votes, Ownable

The Optimism token used in governance and supporting voting and delegation. Implements
EIP 2612 allowing signed approvals. Contract is "owned" by a `MintManager` instance with
permission to the `mint` function only, for the purposes of enforcing the token inflation
schedule.


## Functions
### constructor


```solidity
constructor() ERC20("Optimism", "OP") ERC20Permit("Optimism");
```

### mint

Allows the owner to mint tokens.


```solidity
function mint(address _account, uint256 _amount) public onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_account`|`address`|The account receiving minted tokens.|
|`_amount`|`uint256`| The amount of tokens to mint.|


### _afterTokenTransfer

Callback called after a token transfer.


```solidity
function _afterTokenTransfer(address from, address to, uint256 amount) internal override(ERC20, ERC20Votes);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`from`|`address`|  The account sending tokens.|
|`to`|`address`|    The account receiving tokens.|
|`amount`|`uint256`|The amount of tokens being transfered.|


### _mint

Internal mint function.


```solidity
function _mint(address to, uint256 amount) internal override(ERC20, ERC20Votes);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`to`|`address`|    The account receiving minted tokens.|
|`amount`|`uint256`|The amount of tokens to mint.|


### _burn

Internal burn function.


```solidity
function _burn(address account, uint256 amount) internal override(ERC20, ERC20Votes);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`account`|`address`|The account that tokens will be burned from.|
|`amount`|`uint256`| The amount of tokens that will be burned.|


