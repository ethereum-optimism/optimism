# WETH9









## Methods

### allowance

```solidity
function allowance(address, address) external view returns (uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### approve

```solidity
function approve(address guy, uint256 wad) external nonpayable returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| guy | address | undefined
| wad | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### balanceOf

```solidity
function balanceOf(address) external view returns (uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### decimals

```solidity
function decimals() external view returns (uint8)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint8 | undefined

### deposit

```solidity
function deposit() external payable
```






### name

```solidity
function name() external view returns (string)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | string | undefined

### symbol

```solidity
function symbol() external view returns (string)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | string | undefined

### totalSupply

```solidity
function totalSupply() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### transfer

```solidity
function transfer(address dst, uint256 wad) external nonpayable returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| dst | address | undefined
| wad | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### transferFrom

```solidity
function transferFrom(address src, address dst, uint256 wad) external nonpayable returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| src | address | undefined
| dst | address | undefined
| wad | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### withdraw

```solidity
function withdraw(uint256 wad) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| wad | uint256 | undefined



## Events

### Approval

```solidity
event Approval(address indexed src, address indexed guy, uint256 wad)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| src `indexed` | address | undefined |
| guy `indexed` | address | undefined |
| wad  | uint256 | undefined |

### Deposit

```solidity
event Deposit(address indexed dst, uint256 wad)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| dst `indexed` | address | undefined |
| wad  | uint256 | undefined |

### Transfer

```solidity
event Transfer(address indexed src, address indexed dst, uint256 wad)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| src `indexed` | address | undefined |
| dst `indexed` | address | undefined |
| wad  | uint256 | undefined |

### Withdrawal

```solidity
event Withdrawal(address indexed src, uint256 wad)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| src `indexed` | address | undefined |
| wad  | uint256 | undefined |



