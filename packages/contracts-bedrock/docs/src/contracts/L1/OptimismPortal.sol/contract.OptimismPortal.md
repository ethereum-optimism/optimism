# OptimismPortal
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/OptimismPortal.sol)

**Inherits:**
Initializable, [ResourceMetering](/contracts/L1/ResourceMetering.sol/abstract.ResourceMetering.md), [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The OptimismPortal is a low-level contract responsible for passing messages between L1
and L2. Messages sent directly to the OptimismPortal have no form of replayability.
Users are encouraged to use the L1CrossDomainMessenger for a higher-level interface.


## State Variables
### DEPOSIT_VERSION
Version of the deposit event.


```solidity
uint256 internal constant DEPOSIT_VERSION = 0;
```


### RECEIVE_DEFAULT_GAS_LIMIT
The L2 gas limit set when eth is deposited using the receive() function.


```solidity
uint64 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 100_000;
```


### L2_ORACLE
Address of the L2OutputOracle contract.


```solidity
L2OutputOracle public immutable L2_ORACLE;
```


### SYSTEM_CONFIG
Address of the SystemConfig contract.


```solidity
SystemConfig public immutable SYSTEM_CONFIG;
```


### GUARDIAN
Address that has the ability to pause and unpause withdrawals.


```solidity
address public immutable GUARDIAN;
```


### l2Sender
Address of the L2 account which initiated a withdrawal in this transaction. If the
of this variable is the default L2 sender address, then we are NOT inside of a call
to finalizeWithdrawalTransaction.


```solidity
address public l2Sender;
```


### finalizedWithdrawals
A list of withdrawal hashes which have been successfully finalized.


```solidity
mapping(bytes32 => bool) public finalizedWithdrawals;
```


### provenWithdrawals
A mapping of withdrawal hashes to `ProvenWithdrawal` data.


```solidity
mapping(bytes32 => ProvenWithdrawal) public provenWithdrawals;
```


### paused
Determines if cross domain messaging is paused. When set to true,
withdrawals are paused. This may be removed in the future.


```solidity
bool public paused;
```


## Functions
### whenNotPaused

Reverts when paused.


```solidity
modifier whenNotPaused();
```

### constructor


```solidity
constructor(L2OutputOracle _l2Oracle, address _guardian, bool _paused, SystemConfig _config) Semver(1, 6, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2Oracle`|`L2OutputOracle`|                 Address of the L2OutputOracle contract.|
|`_guardian`|`address`|                 Address that can pause deposits and withdrawals.|
|`_paused`|`bool`|                   Sets the contract's pausability state.|
|`_config`|`SystemConfig`|                   Address of the SystemConfig contract.|


### initialize

Initializer.


```solidity
function initialize(bool _paused) public initializer;
```

### pause

Pause deposits and withdrawals.


```solidity
function pause() external;
```

### unpause

Unpause deposits and withdrawals.


```solidity
function unpause() external;
```

### minimumGasLimit

Computes the minimum gas limit for a deposit. The minimum gas limit
linearly increases based on the size of the calldata. This is to prevent
users from creating L2 resource usage without paying for it. This function
can be used when interacting with the portal to ensure forwards compatibility.


```solidity
function minimumGasLimit(uint64 _byteCount) public pure returns (uint64);
```

### receive

Accepts value so that users can send ETH directly to this contract and have the
funds be deposited to their address on L2. This is intended as a convenience
function for EOAs. Contracts should call the depositTransaction() function directly
otherwise any deposited funds will be lost due to address aliasing.


```solidity
receive() external payable;
```

### donateETH

Accepts ETH value without triggering a deposit to L2. This function mainly exists
for the sake of the migration between the legacy Optimism system and Bedrock.


```solidity
function donateETH() external payable;
```

### _resourceConfig

Getter for the resource config. Used internally by the ResourceMetering
contract. The SystemConfig is the source of truth for the resource config.


```solidity
function _resourceConfig() internal view override returns (ResourceMetering.ResourceConfig memory);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`ResourceConfig.ResourceMetering`|ResourceMetering.ResourceConfig|


### proveWithdrawalTransaction

Proves a withdrawal transaction.


```solidity
function proveWithdrawalTransaction(
    Types.WithdrawalTransaction memory _tx,
    uint256 _l2OutputIndex,
    Types.OutputRootProof calldata _outputRootProof,
    bytes[] calldata _withdrawalProof
) external whenNotPaused;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_tx`|`WithdrawalTransaction.Types`|             Withdrawal transaction to finalize.|
|`_l2OutputIndex`|`uint256`|  L2 output index to prove against.|
|`_outputRootProof`|`OutputRootProof.Types`|Inclusion proof of the L2ToL1MessagePasser contract's storage root.|
|`_withdrawalProof`|`bytes[]`|Inclusion proof of the withdrawal in L2ToL1MessagePasser contract.|


### finalizeWithdrawalTransaction

Finalizes a withdrawal transaction.


```solidity
function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external whenNotPaused;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_tx`|`WithdrawalTransaction.Types`|Withdrawal transaction to finalize.|


### depositTransaction

Accepts deposits of ETH and data, and emits a TransactionDeposited event for use in
deriving deposit transactions. Note that if a deposit is made by a contract, its
address will be aliased when retrieved using `tx.origin` or `msg.sender`. Consider
using the CrossDomainMessenger contracts for a simpler developer experience.


```solidity
function depositTransaction(address _to, uint256 _value, uint64 _gasLimit, bool _isCreation, bytes memory _data)
    public
    payable
    metered(_gasLimit);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|        Target address on L2.|
|`_value`|`uint256`|     ETH value to send to the recipient.|
|`_gasLimit`|`uint64`|  Minimum L2 gas limit (can be greater than or equal to this value).|
|`_isCreation`|`bool`|Whether or not the transaction is a contract creation.|
|`_data`|`bytes`|      Data to trigger the recipient with.|


### isOutputFinalized

Determine if a given output is finalized. Reverts if the call to
L2_ORACLE.getL2Output reverts. Returns a boolean otherwise.


```solidity
function isOutputFinalized(uint256 _l2OutputIndex) external view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2OutputIndex`|`uint256`|Index of the L2 output to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the output is finalized.|


### _isFinalizationPeriodElapsed

Determines whether the finalization period has elapsed w/r/t a given timestamp.


```solidity
function _isFinalizationPeriodElapsed(uint256 _timestamp) internal view returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_timestamp`|`uint256`|Timestamp to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the finalization period has elapsed.|


## Events
### TransactionDeposited
Emitted when a transaction is deposited from L1 to L2. The parameters of this event
are read by the rollup node and used to derive deposit transactions on L2.


```solidity
event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);
```

### WithdrawalProven
Emitted when a withdrawal transaction is proven.


```solidity
event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);
```

### WithdrawalFinalized
Emitted when a withdrawal transaction is finalized.


```solidity
event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);
```

### Paused
Emitted when the pause is triggered.


```solidity
event Paused(address account);
```

### Unpaused
Emitted when the pause is lifted.


```solidity
event Unpaused(address account);
```

## Structs
### ProvenWithdrawal
Represents a proven withdrawal.


```solidity
struct ProvenWithdrawal {
    bytes32 outputRoot;
    uint128 timestamp;
    uint128 l2OutputIndex;
}
```

