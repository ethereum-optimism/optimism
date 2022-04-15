# Vm









## Methods

### accesses

```solidity
function accesses(address) external nonpayable returns (bytes32[] reads, bytes32[] writes)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| reads | bytes32[] | undefined
| writes | bytes32[] | undefined

### addr

```solidity
function addr(uint256) external nonpayable returns (address)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### assume

```solidity
function assume(bool) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### clearMockedCalls

```solidity
function clearMockedCalls() external nonpayable
```






### deal

```solidity
function deal(address, uint256) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | uint256 | undefined

### etch

```solidity
function etch(address, bytes) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | bytes | undefined

### expectCall

```solidity
function expectCall(address, bytes) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | bytes | undefined

### expectEmit

```solidity
function expectEmit(bool, bool, bool, bool) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined
| _1 | bool | undefined
| _2 | bool | undefined
| _3 | bool | undefined

### expectRevert

```solidity
function expectRevert() external nonpayable
```






### fee

```solidity
function fee(uint256) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### ffi

```solidity
function ffi(string[]) external nonpayable returns (bytes)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | string[] | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes | undefined

### getCode

```solidity
function getCode(string) external nonpayable returns (bytes)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | string | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes | undefined

### label

```solidity
function label(address, string) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | string | undefined

### load

```solidity
function load(address, bytes32) external nonpayable returns (bytes32)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

### mockCall

```solidity
function mockCall(address, bytes, bytes) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | bytes | undefined
| _2 | bytes | undefined

### prank

```solidity
function prank(address) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### record

```solidity
function record() external nonpayable
```






### roll

```solidity
function roll(uint256) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### sign

```solidity
function sign(uint256, bytes32) external nonpayable returns (uint8, bytes32, bytes32)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined
| _1 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint8 | undefined
| _1 | bytes32 | undefined
| _2 | bytes32 | undefined

### startPrank

```solidity
function startPrank(address, address) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | address | undefined

### stopPrank

```solidity
function stopPrank() external nonpayable
```






### store

```solidity
function store(address, bytes32, bytes32) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined
| _1 | bytes32 | undefined
| _2 | bytes32 | undefined

### warp

```solidity
function warp(uint256) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined




