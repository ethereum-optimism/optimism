# OVM_GasPriceOracle



> OVM_GasPriceOracle



*This contract exposes the current l2 gas price, a measure of how congested the network currently is. This measure is used by the Sequencer to determine what fee to charge for transactions. When the system is more congested, the l2 gas price will increase and fees will also increase as a result. All public variables are set while generating the initial L2 state. The constructor doesn&#39;t run in practice as the L2 state generation script uses the deployed bytecode instead of running the initcode.*

## Methods

### decimals

```solidity
function decimals() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### gasPrice

```solidity
function gasPrice() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### getL1Fee

```solidity
function getL1Fee(bytes _data) external view returns (uint256)
```

Computes the L1 portion of the fee based on the size of the RLP encoded tx and the current l1BaseFee



#### Parameters

| Name | Type | Description |
|---|---|---|
| _data | bytes | Unsigned RLP encoded tx, 6 elements

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | L1 fee that should be paid for the tx

### getL1GasUsed

```solidity
function getL1GasUsed(bytes _data) external view returns (uint256)
```

Computes the amount of L1 gas used for a transaction The overhead represents the per batch gas overhead of posting both transaction and state roots to L1 given larger batch sizes. 4 gas for 0 byte https://github.com/ethereum/go-ethereum/blob/9ada4a2e2c415e6b0b51c50e901336872e028872/params/protocol_params.go#L33 16 gas for non zero byte https://github.com/ethereum/go-ethereum/blob/9ada4a2e2c415e6b0b51c50e901336872e028872/params/protocol_params.go#L87 This will need to be updated if calldata gas prices change Account for the transaction being unsigned Padding is added to account for lack of signature on transaction 1 byte for RLP V prefix 1 byte for V 1 byte for RLP R prefix 32 bytes for R 1 byte for RLP S prefix 32 bytes for S Total: 68 bytes of padding



#### Parameters

| Name | Type | Description |
|---|---|---|
| _data | bytes | Unsigned RLP encoded tx, 6 elements

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | Amount of L1 gas used for a transaction

### l1BaseFee

```solidity
function l1BaseFee() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### overhead

```solidity
function overhead() external view returns (uint256)
```






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


### scalar

```solidity
function scalar() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### setDecimals

```solidity
function setDecimals(uint256 _decimals) external nonpayable
```

Allows the owner to modify the decimals.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _decimals | uint256 | New decimals

### setGasPrice

```solidity
function setGasPrice(uint256 _gasPrice) external nonpayable
```

Allows the owner to modify the l2 gas price.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _gasPrice | uint256 | New l2 gas price.

### setL1BaseFee

```solidity
function setL1BaseFee(uint256 _baseFee) external nonpayable
```

Allows the owner to modify the l1 base fee.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _baseFee | uint256 | New l1 base fee

### setOverhead

```solidity
function setOverhead(uint256 _overhead) external nonpayable
```

Allows the owner to modify the overhead.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _overhead | uint256 | New overhead

### setScalar

```solidity
function setScalar(uint256 _scalar) external nonpayable
```

Allows the owner to modify the scalar.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _scalar | uint256 | New scalar

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined



## Events

### DecimalsUpdated

```solidity
event DecimalsUpdated(uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0  | uint256 | undefined |

### GasPriceUpdated

```solidity
event GasPriceUpdated(uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0  | uint256 | undefined |

### L1BaseFeeUpdated

```solidity
event L1BaseFeeUpdated(uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0  | uint256 | undefined |

### OverheadUpdated

```solidity
event OverheadUpdated(uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0  | uint256 | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### ScalarUpdated

```solidity
event ScalarUpdated(uint256)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0  | uint256 | undefined |



