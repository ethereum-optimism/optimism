# DeployerWhitelist
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/legacy/DeployerWhitelist.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

DeployerWhitelist is a legacy contract that was originally used to act as a whitelist of
addresses allowed to the Optimism network. The DeployerWhitelist has since been
disabled, but the code is kept in state for the sake of full backwards compatibility.
As of the Bedrock upgrade, the DeployerWhitelist is completely unused by the Optimism
system and could, in theory, be removed entirely.


## State Variables
### owner
Address of the owner of this contract. Note that when this address is set to
address(0), the whitelist is disabled.


```solidity
address public owner;
```


### whitelist
Mapping of deployer addresses to boolean whitelist status.


```solidity
mapping(address => bool) public whitelist;
```


## Functions
### onlyOwner

Blocks functions to anyone except the contract owner.


```solidity
modifier onlyOwner();
```

### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### setWhitelistedDeployer

Adds or removes an address from the deployment whitelist.


```solidity
function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_deployer`|`address`|     Address to update permissions for.|
|`_isWhitelisted`|`bool`|Whether or not the address is whitelisted.|


### setOwner

Updates the owner of this contract.


```solidity
function setOwner(address _owner) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|Address of the new owner.|


### enableArbitraryContractDeployment

Permanently enables arbitrary contract deployment and deletes the owner.


```solidity
function enableArbitraryContractDeployment() external onlyOwner;
```

### isDeployerAllowed

Checks whether an address is allowed to deploy contracts.


```solidity
function isDeployerAllowed(address _deployer) external view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_deployer`|`address`|Address to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the address can deploy contracts.|


## Events
### OwnerChanged
Emitted when the owner of this contract changes.


```solidity
event OwnerChanged(address oldOwner, address newOwner);
```

### WhitelistStatusChanged
Emitted when the whitelist status of a deployer changes.


```solidity
event WhitelistStatusChanged(address deployer, bool whitelisted);
```

### WhitelistDisabled
Emitted when the whitelist is disabled.


```solidity
event WhitelistDisabled(address oldOwner);
```

