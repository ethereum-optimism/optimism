# GasPriceOracle
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/GasPriceOracle.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

This contract maintains the variables responsible for computing the L1 portion of the
total fee charged on L2. Before Bedrock, this contract held variables in state that were
read during the state transition function to compute the L1 portion of the transaction
fee. After Bedrock, this contract now simply proxies the L1Block contract, which has
the values used to compute the L1 portion of the fee in its state.
The contract exposes an API that is useful for knowing how large the L1 portion of the
transaction fee will be. The following events were deprecated with Bedrock:
- event OverheadUpdated(uint256 overhead);
- event ScalarUpdated(uint256 scalar);
- event DecimalsUpdated(uint256 decimals);


## State Variables
### DECIMALS
Number of decimals used in the scalar.


```solidity
uint256 public constant DECIMALS = 6;
```


## Functions
### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### getL1Fee

Computes the L1 portion of the fee based on the size of the rlp encoded input
transaction, the current L1 base fee, and the various dynamic parameters.


```solidity
function getL1Fee(bytes memory _data) external view returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_data`|`bytes`|Unsigned fully RLP-encoded transaction to get the L1 fee for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|L1 fee that should be paid for the tx|


### gasPrice

Retrieves the current gas price (base fee).


```solidity
function gasPrice() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Current L2 gas price (base fee).|


### baseFee

Retrieves the current base fee.


```solidity
function baseFee() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Current L2 base fee.|


### overhead

Retrieves the current fee overhead.


```solidity
function overhead() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Current fee overhead.|


### scalar

Retrieves the current fee scalar.


```solidity
function scalar() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Current fee scalar.|


### l1BaseFee

Retrieves the latest known L1 base fee.


```solidity
function l1BaseFee() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Latest known L1 base fee.|


### decimals

Retrieves the number of decimals used in the scalar.


```solidity
function decimals() public pure returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Number of decimals used in the scalar.|


### getL1GasUsed

Computes the amount of L1 gas used for a transaction. Adds the overhead which
represents the per-transaction gas overhead of posting the transaction and state
roots to L1. Adds 68 bytes of padding to account for the fact that the input does
not have a signature.


```solidity
function getL1GasUsed(bytes memory _data) public view returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_data`|`bytes`|Unsigned fully RLP-encoded transaction to get the L1 gas for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Amount of L1 gas used to publish the transaction.|


