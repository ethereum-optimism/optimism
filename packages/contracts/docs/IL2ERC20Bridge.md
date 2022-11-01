# IL2ERC20Bridge



> IL2ERC20Bridge





## Methods

### finalizeDeposit

```solidity
function finalizeDeposit(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) external nonpayable
```



*Complete a deposit from L1 to L2, and credits funds to the recipient&#39;s balance of this L2 token. This call will fail if it did not originate from a corresponding deposit in L1StandardTokenBridge.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address for the l1 token this is called with
| _l2Token | address | Address for the l2 token this is called with
| _from | address | Account to pull the deposit from on L2.
| _to | address | Address to receive the withdrawal at
| _amount | uint256 | Amount of the token to withdraw
| _data | bytes | Data provider by the sender on L1. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### l1TokenBridge

```solidity
function l1TokenBridge() external nonpayable returns (address)
```



*get the address of the corresponding L1 bridge contract.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address of the corresponding L1 bridge contract.

### withdraw

```solidity
function withdraw(address _l2Token, uint256 _amount, uint32 _l1Gas, bytes _data) external nonpayable
```



*initiate a withdraw of some tokens to the caller&#39;s account on L1*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | Address of L2 token where withdrawal was initiated.
| _amount | uint256 | Amount of the token to withdraw.
| _l1Gas | uint32 | Unused, but included for potential forward compatibility considerations.
| _data | bytes | Optional data to forward to L1. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### withdrawTo

```solidity
function withdrawTo(address _l2Token, address _to, uint256 _amount, uint32 _l1Gas, bytes _data) external nonpayable
```



*initiate a withdraw of some token to a recipient&#39;s account on L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | Address of L2 token where withdrawal is initiated.
| _to | address | L1 adress to credit the withdrawal to.
| _amount | uint256 | Amount of the token to withdraw.
| _l1Gas | uint32 | Unused, but included for potential forward compatibility considerations.
| _data | bytes | Optional data to forward to L1. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.



## Events

### DepositFailed

```solidity
event DepositFailed(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### DepositFinalized

```solidity
event DepositFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### WithdrawalInitiated

```solidity
event WithdrawalInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token `indexed` | address | undefined |
| _l2Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |



