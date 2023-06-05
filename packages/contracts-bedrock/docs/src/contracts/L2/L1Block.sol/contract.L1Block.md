# L1Block
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/L1Block.sol)

**Inherits:**
[Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L1Block predeploy gives users access to information about the last known L1 block.
Values within this contract are updated once per epoch (every L1 block) and can only be
set by the "depositor" account, a special system address. Depositor account transactions
are created by the protocol whenever we move to a new epoch.


## State Variables
### DEPOSITOR_ACCOUNT
Address of the special depositor account.


```solidity
address public constant DEPOSITOR_ACCOUNT = 0xDeaDDEaDDeAdDeAdDEAdDEaddeAddEAdDEAd0001;
```


### number
The latest L1 block number known by the L2 system.


```solidity
uint64 public number;
```


### timestamp
The latest L1 timestamp known by the L2 system.


```solidity
uint64 public timestamp;
```


### basefee
The latest L1 basefee.


```solidity
uint256 public basefee;
```


### hash
The latest L1 blockhash.


```solidity
bytes32 public hash;
```


### sequenceNumber
The number of L2 blocks in the same epoch.


```solidity
uint64 public sequenceNumber;
```


### batcherHash
The versioned hash to authenticate the batcher by.


```solidity
bytes32 public batcherHash;
```


### l1FeeOverhead
The overhead value applied to the L1 portion of the transaction
fee.


```solidity
uint256 public l1FeeOverhead;
```


### l1FeeScalar
The scalar value applied to the L1 portion of the transaction fee.


```solidity
uint256 public l1FeeScalar;
```


## Functions
### constructor


```solidity
constructor() Semver(1, 0, 0);
```

### setL1BlockValues

Updates the L1 block values.


```solidity
function setL1BlockValues(
    uint64 _number,
    uint64 _timestamp,
    uint256 _basefee,
    bytes32 _hash,
    uint64 _sequenceNumber,
    bytes32 _batcherHash,
    uint256 _l1FeeOverhead,
    uint256 _l1FeeScalar
) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_number`|`uint64`|        L1 blocknumber.|
|`_timestamp`|`uint64`|     L1 timestamp.|
|`_basefee`|`uint256`|       L1 basefee.|
|`_hash`|`bytes32`|          L1 blockhash.|
|`_sequenceNumber`|`uint64`|Number of L2 blocks since epoch start.|
|`_batcherHash`|`bytes32`|   Versioned hash to authenticate batcher by.|
|`_l1FeeOverhead`|`uint256`| L1 fee overhead.|
|`_l1FeeScalar`|`uint256`|   L1 fee scalar.|


