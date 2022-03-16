# L2NativeERC20Bridge



> L2NativeERC20Bridge





## Methods

### depositERC20

```solidity
function depositERC20(address _l2Token, address _l1Token, uint256 _amount, uint32 _l1Gas, bytes _data) external nonpayable
```



*deposit an amount of the ERC20 to the caller&#39;s balance on L1.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | Address of the L2 ERC20 we are depositing
| _l1Token | address | Address of the L2 respective L1 ERC20
| _amount | uint256 | Amount of the ERC20 to deposit
| _l1Gas | uint32 | Gas limit required to complete the deposit on L1.
| _data | bytes | Optional data to forward to L1. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### depositERC20To

```solidity
function depositERC20To(address _l2Token, address _l1Token, address _to, uint256 _amount, uint32 _l1Gas, bytes _data) external nonpayable
```



*deposit an amount of ERC20 to a recipient&#39;s balance on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | Address of the L2 ERC20 we are depositing
| _l1Token | address | Address of the L1 respective L2 ERC20
| _to | address | L2 address to credit the withdrawal to.
| _amount | uint256 | Amount of the ERC20 to deposit.
| _l1Gas | uint32 | Gas limit required to complete the deposit on L1.
| _data | bytes | Optional data to forward to L1. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### deposits

```solidity
function deposits(address, address) external view returns (uint256)
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

### finalizeERC20Withdrawal

```solidity
function finalizeERC20Withdrawal(address _l2Token, address _l1Token, address _from, address _to, uint256 _amount, bytes _data) external nonpayable
```



*Complete a withdrawal from L1 to L2, and credit funds to the recipient&#39;s balance of the L2 ERC20 token. This call will fail if the initialized withdrawal from L1 has not been finalized.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token | address | Address of L2 token to finalizeWithdrawal for.
| _l1Token | address | Address of L1 token where withdrawal was initiated.
| _from | address | L1 address initiating the transfer.
| _to | address | L2 address to credit the withdrawal to.
| _amount | uint256 | Amount of the ERC20 to deposit.
| _data | bytes | Data provided by the sender on L1. This data is provided   solely as a convenience for external contracts. Aside from enforcing a maximum   length, these contracts provide no guarantees about its content.

### initialize

```solidity
function initialize(address _l2messenger, address _l1TokenBridge) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2messenger | address | L2 Messenger address being used for cross-chain communications.
| _l1TokenBridge | address | L1 standard bridge address.

### l1TokenBridge

```solidity
function l1TokenBridge() external view returns (address)
```



*get the address of the corresponding L1 native bridge contract.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address of the corresponding L1 native bridge contract.

### messenger

```solidity
function messenger() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined



## Events

### NativeERC20DepositInitiated

```solidity
event NativeERC20DepositInitiated(address indexed _l2Token, address indexed _l1Token, address indexed _from, address _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Token `indexed` | address | undefined |
| _l1Token `indexed` | address | undefined |
| _from `indexed` | address | undefined |
| _to  | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### NativeERC20WithdrawalFinalized

```solidity
event NativeERC20WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
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



