# TeleportrDisburser



> TeleportrDisburser





## Methods

### disburse

```solidity
function disburse(uint256 _nextDepositId, TeleportrDisburser.Disbursement[] _disbursements) external payable
```

Accepts a list of Disbursements and forwards the amount paid to the contract to each recipient. The method reverts if there are zero disbursements, the total amount to forward differs from the amount sent in the transaction, or the _nextDepositId is unexpected. Failed disbursements will not cause the method to revert, but will instead be held by the contract and available for the owner to withdraw.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _nextDepositId | uint256 | The depositId of the first Dispursement.
| _disbursements | TeleportrDisburser.Disbursement[] | A list of Disbursements to process.

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


### totalDisbursements

```solidity
function totalDisbursements() external view returns (uint256)
```

The total number of disbursements processed.




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

### DisbursementFailed

```solidity
event DisbursementFailed(uint256 indexed depositId, address indexed to, uint256 amount)
```

Emitted any time a disbursement fails to send.



#### Parameters

| Name | Type | Description |
|---|---|---|
| depositId `indexed` | uint256 | The unique sequence number identifying the deposit. |
| to `indexed` | address | The intended recipient of the disbursement. |
| amount  | uint256 | The amount intended to be sent to the recipient. |

### DisbursementSuccess

```solidity
event DisbursementSuccess(uint256 indexed depositId, address indexed to, uint256 amount)
```

Emitted any time a disbursement is successfuly sent.



#### Parameters

| Name | Type | Description |
|---|---|---|
| depositId `indexed` | uint256 | The unique sequence number identifying the deposit. |
| to `indexed` | address | The recipient of the disbursement. |
| amount  | uint256 | The amount sent to the recipient. |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |



