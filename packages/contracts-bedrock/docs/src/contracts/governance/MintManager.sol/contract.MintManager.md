# MintManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/governance/MintManager.sol)

**Inherits:**
Ownable

Set as `owner` of the OP token and responsible for the token inflation schedule.
Contract acts as the token "mint manager" with permission to the `mint` function only.
Currently permitted to mint once per year of up to 2% of the total token supply.
Upgradable to allow changes in the inflation schedule.


## State Variables
### governanceToken
The GovernanceToken that the MintManager can mint tokens


```solidity
GovernanceToken public immutable governanceToken;
```


### MINT_CAP
The amount of tokens that can be minted per year. The value is a fixed
point number with 4 decimals.


```solidity
uint256 public constant MINT_CAP = 20;
```


### DENOMINATOR
The number of decimals for the MINT_CAP.


```solidity
uint256 public constant DENOMINATOR = 1000;
```


### MINT_PERIOD
The amount of time that must pass before the MINT_CAP number of tokens can
be minted again.


```solidity
uint256 public constant MINT_PERIOD = 365 days;
```


### mintPermittedAfter
Tracks the time of last mint.


```solidity
uint256 public mintPermittedAfter;
```


## Functions
### constructor


```solidity
constructor(address _upgrader, address _governanceToken);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_upgrader`|`address`|       The owner of this contract|
|`_governanceToken`|`address`|The governance token this contract can mint tokens of|


### mint

Only the token owner is allowed to mint a certain amount of OP per year.


```solidity
function mint(address _account, uint256 _amount) public onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_account`|`address`|Address to mint new tokens to.|
|`_amount`|`uint256`| Amount of tokens to be minted.|


### upgrade

Upgrade the owner of the governance token to a new MintManager.


```solidity
function upgrade(address _newMintManager) public onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_newMintManager`|`address`|The MintManager to upgrade to.|


