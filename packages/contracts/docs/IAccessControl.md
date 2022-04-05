# IAccessControl







*External interface of AccessControl declared to support ERC165 detection.*

## Methods

### getRoleAdmin

```solidity
function getRoleAdmin(bytes32 role) external view returns (bytes32)
```



*Returns the admin role that controls `role`. See {grantRole} and {revokeRole}. To change a role&#39;s admin, use {AccessControl-_setRoleAdmin}.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

### grantRole

```solidity
function grantRole(bytes32 role, address account) external nonpayable
```



*Grants `role` to `account`. If `account` had not been already granted `role`, emits a {RoleGranted} event. Requirements: - the caller must have ``role``&#39;s admin role.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role | bytes32 | undefined
| account | address | undefined

### hasRole

```solidity
function hasRole(bytes32 role, address account) external view returns (bool)
```



*Returns `true` if `account` has been granted `role`.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role | bytes32 | undefined
| account | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### renounceRole

```solidity
function renounceRole(bytes32 role, address account) external nonpayable
```



*Revokes `role` from the calling account. Roles are often managed via {grantRole} and {revokeRole}: this function&#39;s purpose is to provide a mechanism for accounts to lose their privileges if they are compromised (such as when a trusted device is misplaced). If the calling account had been granted `role`, emits a {RoleRevoked} event. Requirements: - the caller must be `account`.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role | bytes32 | undefined
| account | address | undefined

### revokeRole

```solidity
function revokeRole(bytes32 role, address account) external nonpayable
```



*Revokes `role` from `account`. If `account` had been granted `role`, emits a {RoleRevoked} event. Requirements: - the caller must have ``role``&#39;s admin role.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role | bytes32 | undefined
| account | address | undefined



## Events

### RoleAdminChanged

```solidity
event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
```



*Emitted when `newAdminRole` is set as ``role``&#39;s admin role, replacing `previousAdminRole` `DEFAULT_ADMIN_ROLE` is the starting admin for all roles, despite {RoleAdminChanged} not being emitted signaling this. _Available since v3.1._*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role `indexed` | bytes32 | undefined |
| previousAdminRole `indexed` | bytes32 | undefined |
| newAdminRole `indexed` | bytes32 | undefined |

### RoleGranted

```solidity
event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
```



*Emitted when `account` is granted `role`. `sender` is the account that originated the contract call, an admin role bearer except when using {AccessControl-_setupRole}.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role `indexed` | bytes32 | undefined |
| account `indexed` | address | undefined |
| sender `indexed` | address | undefined |

### RoleRevoked

```solidity
event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
```



*Emitted when `account` is revoked `role`. `sender` is the account that originated the contract call:   - if using `revokeRole`, it is the admin role bearer   - if using `renounceRole`, it is the role bearer (i.e. `account`)*

#### Parameters

| Name | Type | Description |
|---|---|---|
| role `indexed` | bytes32 | undefined |
| account `indexed` | address | undefined |
| sender `indexed` | address | undefined |



