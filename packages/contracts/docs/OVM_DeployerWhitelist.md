# OVM_DeployerWhitelist



> OVM_DeployerWhitelist



*The Deployer Whitelist is a temporary predeploy used to provide additional safety during the initial phases of our mainnet roll out. It is owned by the Optimism team, and defines accounts which are allowed to deploy contracts on Layer2. The Execution Manager will only allow an ovmCREATE or ovmCREATE2 operation to proceed if the deployer&#39;s address whitelisted.*

## Methods

### enableArbitraryContractDeployment

```solidity
function enableArbitraryContractDeployment() external nonpayable
```

Permanently enables arbitrary contract deployment and deletes the owner.




### isDeployerAllowed

```solidity
function isDeployerAllowed(address _deployer) external view returns (bool)
```

Checks whether an address is allowed to deploy contracts.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _deployer | address | Address to check.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | _allowed Whether or not the address can deploy contracts.

### owner

```solidity
function owner() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### setOwner

```solidity
function setOwner(address _owner) external nonpayable
```

Updates the owner of this contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _owner | address | Address of the new owner.

### setWhitelistedDeployer

```solidity
function setWhitelistedDeployer(address _deployer, bool _isWhitelisted) external nonpayable
```

Adds or removes an address from the deployment whitelist.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _deployer | address | Address to update permissions for.
| _isWhitelisted | bool | Whether or not the address is whitelisted.

### whitelist

```solidity
function whitelist(address) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined



## Events

### OwnerChanged

```solidity
event OwnerChanged(address oldOwner, address newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| oldOwner  | address | undefined |
| newOwner  | address | undefined |

### WhitelistDisabled

```solidity
event WhitelistDisabled(address oldOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| oldOwner  | address | undefined |

### WhitelistStatusChanged

```solidity
event WhitelistStatusChanged(address deployer, bool whitelisted)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| deployer  | address | undefined |
| whitelisted  | bool | undefined |



