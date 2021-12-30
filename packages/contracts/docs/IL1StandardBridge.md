# IL1StandardBridge



> IL1StandardBridge





## Methods

### depositERC20

```solidity
function depositERC20(address _l1Token, address _l2Token, uint256 _amount, uint32 _l2Gas, bytes _data) external nonpayable
```



*deposit an amount of the ERC20 to the caller&#39;s balance on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the L1 ERC20 we are depositing
| _l2Token | address | Address of the L1 respective L2 ERC20
| _amount | uint256 | Amount of the ERC20 to deposit
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### depositERC20To

```solidity
function depositERC20To(address _l1Token, address _l2Token, address _to, uint256 _amount, uint32 _l2Gas, bytes _data) external nonpayable
```



*deposit an amount of ERC20 to a recipient&#39;s balance on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of the L1 ERC20 we are depositing
| _l2Token | address | Address of the L1 respective L2 ERC20
| _to | address | L2 address to credit the withdrawal to.
| _amount | uint256 | Amount of the ERC20 to deposit.
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### depositETH

```solidity
function depositETH(uint32 _l2Gas, bytes _data) external payable
```



*Deposit an amount of the ETH to the caller&#39;s balance on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### depositETHTo

```solidity
function depositETHTo(address _to, uint32 _l2Gas, bytes _data) external payable
```



*Deposit an amount of ETH to a recipient&#39;s balance on L2.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _to | address | L2 address to credit the withdrawal to.
| _l2Gas | uint32 | Gas limit required to complete the deposit on L2.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### finalizeERC20Withdrawal

```solidity
function finalizeERC20Withdrawal(address _l1Token, address _l2Token, address _from, address _to, uint256 _amount, bytes _data) external nonpayable
```



*Complete a withdrawal from L2 to L1, and credit funds to the recipient&#39;s balance of the L1 ERC20 token. This call will fail if the initialized withdrawal from L2 has not been finalized.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1Token | address | Address of L1 token to finalizeWithdrawal for.
| _l2Token | address | Address of L2 token where withdrawal was initiated.
| _from | address | L2 address initiating the transfer.
| _to | address | L1 address to credit the withdrawal to.
| _amount | uint256 | Amount of the ERC20 to deposit.
| _data | bytes | Data provided by the sender on L2. This data is provided   solely as a convenience for external contracts. Aside from enforcing a maximum   length, these contracts provide no guarantees about its content.

### finalizeETHWithdrawal

```solidity
function finalizeETHWithdrawal(address _from, address _to, uint256 _amount, bytes _data) external nonpayable
```



*Complete a withdrawal from L2 to L1, and credit funds to the recipient&#39;s balance of the L1 ETH token. Since only the xDomainMessenger can call this function, it will never be called before the withdrawal is finalized.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| _from | address | L2 address initiating the transfer.
| _to | address | L1 address to credit the withdrawal to.
| _amount | uint256 | Amount of the ERC20 to deposit.
| _data | bytes | Optional data to forward to L2. This data is provided        solely as a convenience for external contracts. Aside from enforcing a maximum        length, these contracts provide no guarantees about its content.

### l2TokenBridge

```solidity
function l2TokenBridge() external nonpayable returns (address)
```



*get the address of the corresponding L2 bridge contract.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address of the corresponding L2 bridge contract.



## Events

### ERC20DepositInitiated

```solidity
event ERC20DepositInitiated(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
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

### ERC20WithdrawalFinalized

```solidity
event ERC20WithdrawalFinalized(address indexed _l1Token, address indexed _l2Token, address indexed _from, address _to, uint256 _amount, bytes _data)
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

### ETHDepositInitiated

```solidity
event ETHDepositInitiated(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from `indexed` | address | undefined |
| _to `indexed` | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |

### ETHWithdrawalFinalized

```solidity
event ETHWithdrawalFinalized(address indexed _from, address indexed _to, uint256 _amount, bytes _data)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _from `indexed` | address | undefined |
| _to `indexed` | address | undefined |
| _amount  | uint256 | undefined |
| _data  | bytes | undefined |



