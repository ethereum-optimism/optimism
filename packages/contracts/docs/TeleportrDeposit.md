# TeleportrDeposit



> TeleportrDeposit Shout out to 0xclem for providing the inspiration for this contract: https://github.com/0xclem/teleportr/blob/main/contracts/BridgeDeposit.sol





## Methods

### maxBalance

```solidity
function maxBalance() external view returns (uint256)
```

The maximum balance the contract can hold after a receive.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### maxDepositAmount

```solidity
function maxDepositAmount() external view returns (uint256)
```

The maximum amount that be deposited in a receive.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### minDepositAmount

```solidity
function minDepositAmount() external view returns (uint256)
```

The minimum amount that be deposited in a receive.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### setMaxAmount

```solidity
function setMaxAmount(uint256 _maxDepositAmount) external nonpayable
```

Sets the maximum amount that can be deposited in a receive.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _maxDepositAmount | uint256 | The new maximum deposit amount.

### setMaxBalance

```solidity
function setMaxBalance(uint256 _maxBalance) external nonpayable
```

Sets the maximum balance the contract can hold after a receive.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _maxBalance | uint256 | The new maximum contract balance.

### setMinAmount

```solidity
function setMinAmount(uint256 _minDepositAmount) external nonpayable
```

Sets the minimum amount that can be deposited in a receive.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _minDepositAmount | uint256 | The new minimum deposit amount.

### totalDeposits

```solidity
function totalDeposits() external view returns (uint256)
```

The total number of successful deposits received.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined

### withdrawBalance

```solidity
function withdrawBalance() external nonpayable
```

Sends the contract&#39;s current balance to the owner.






## Events

### BalanceWithdrawn

```solidity
event BalanceWithdrawn(address indexed owner, uint256 balance)
```

Emitted any time the balance is withdrawn by the owner.



#### Parameters

| Name | Type | Description |
|---|---|---|
| owner `indexed` | address | The current owner and recipient of the funds. |
| balance  | uint256 | The current contract balance paid to the owner. |

### EtherReceived

```solidity
event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount)
```

Emitted any time a successful deposit is received.



#### Parameters

| Name | Type | Description |
|---|---|---|
| depositId `indexed` | uint256 | A unique sequencer number identifying the deposit. |
| emitter `indexed` | address | The sending address of the payer. |
| amount `indexed` | uint256 | The amount deposited by the payer. |

### MaxBalanceSet

```solidity
event MaxBalanceSet(uint256 previousBalance, uint256 newBalance)
```

Emitted any time the contract maximum balance is set.



#### Parameters

| Name | Type | Description |
|---|---|---|
| previousBalance  | uint256 | The previous maximum contract balance. |
| newBalance  | uint256 | The new maximum contract balance. |

### MaxDepositAmountSet

```solidity
event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount)
```

Emitted any time the maximum deposit amount is set.



#### Parameters

| Name | Type | Description |
|---|---|---|
| previousAmount  | uint256 | The previous maximum deposit amount. |
| newAmount  | uint256 | The new maximum deposit amount. |

### MinDepositAmountSet

```solidity
event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount)
```

Emitted any time the minimum deposit amount is set.



#### Parameters

| Name | Type | Description |
|---|---|---|
| previousAmount  | uint256 | The previous minimum deposit amount. |
| newAmount  | uint256 | The new minimum deposit amount. |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |



