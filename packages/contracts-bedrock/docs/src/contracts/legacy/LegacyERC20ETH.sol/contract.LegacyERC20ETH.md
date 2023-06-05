# LegacyERC20ETH
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/LegacyERC20ETH.sol)

**Inherits:**
[OptimismMintableERC20](/contracts/universal/OptimismMintableERC20.sol/contract.OptimismMintableERC20.md)

LegacyERC20ETH is a legacy contract that held ETH balances before the Bedrock upgrade.
All ETH balances held within this contract were migrated to the state trie as part of
the Bedrock upgrade. Functions within this contract that mutate state were already
disabled as part of the EVM equivalence upgrade.


## Functions
### constructor

Initializes the contract as an Optimism Mintable ERC20.


```solidity
constructor() OptimismMintableERC20(Predeploys.L2_STANDARD_BRIDGE, address(0), "Ether", "ETH");
```

### balanceOf

Returns the ETH balance of the target account. Overrides the base behavior of the
contract to preserve the invariant that the balance within this contract always
matches the balance in the state trie.


```solidity
function balanceOf(address _who) public view virtual override returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_who`|`address`|Address of the account to query.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|The ETH balance of the target account.|


### mint

Mints some amount of ETH.


```solidity
function mint(address, uint256) public virtual override;
```

### burn

Burns some amount of ETH.


```solidity
function burn(address, uint256) public virtual override;
```

### transfer

Transfers some amount of ETH.


```solidity
function transfer(address, uint256) public virtual override returns (bool);
```

### approve

Approves a spender to spend some amount of ETH.


```solidity
function approve(address, uint256) public virtual override returns (bool);
```

### transferFrom

Transfers funds from some sender account.


```solidity
function transferFrom(address, address, uint256) public virtual override returns (bool);
```

### increaseAllowance

Increases the allowance of a spender.


```solidity
function increaseAllowance(address, uint256) public virtual override returns (bool);
```

### decreaseAllowance

Decreases the allowance of a spender.


```solidity
function decreaseAllowance(address, uint256) public virtual override returns (bool);
```

