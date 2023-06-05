# SystemConfig
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/SystemConfig.sol)

**Inherits:**
OwnableUpgradeable, [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The SystemConfig contract is used to manage configuration of an Optimism network. All
configuration is stored on L1 and picked up by L2 as part of the derviation of the L2
chain.


## State Variables
### VERSION
Version identifier, used for upgrades.


```solidity
uint256 public constant VERSION = 0;
```


### UNSAFE_BLOCK_SIGNER_SLOT
Storage slot that the unsafe block signer is stored at. Storing it at this
deterministic storage slot allows for decoupling the storage layout from the way
that `solc` lays out storage. The `op-node` uses a storage proof to fetch this value.


```solidity
bytes32 public constant UNSAFE_BLOCK_SIGNER_SLOT = keccak256("systemconfig.unsafeblocksigner");
```


### overhead
Fixed L2 gas overhead. Used as part of the L2 fee calculation.


```solidity
uint256 public overhead;
```


### scalar
Dynamic L2 gas overhead. Used as part of the L2 fee calculation.


```solidity
uint256 public scalar;
```


### batcherHash
Identifier for the batcher. For version 1 of this configuration, this is represented
as an address left-padded with zeros to 32 bytes.


```solidity
bytes32 public batcherHash;
```


### gasLimit
L2 block gas limit.


```solidity
uint64 public gasLimit;
```


### _resourceConfig
The configuration for the deposit fee market. Used by the OptimismPortal
to meter the cost of buying L2 gas on L1. Set as internal and wrapped with a getter
so that the struct is returned instead of a tuple.


```solidity
ResourceMetering.ResourceConfig internal _resourceConfig;
```


## Functions
### constructor


```solidity
constructor(
    address _owner,
    uint256 _overhead,
    uint256 _scalar,
    bytes32 _batcherHash,
    uint64 _gasLimit,
    address _unsafeBlockSigner,
    ResourceMetering.ResourceConfig memory _config
) Semver(1, 3, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|            Initial owner of the contract.|
|`_overhead`|`uint256`|         Initial overhead value.|
|`_scalar`|`uint256`|           Initial scalar value.|
|`_batcherHash`|`bytes32`|      Initial batcher hash.|
|`_gasLimit`|`uint64`|         Initial gas limit.|
|`_unsafeBlockSigner`|`address`|Initial unsafe block signer address.|
|`_config`|`ResourceConfig.ResourceMetering`|           Initial resource config.|


### initialize

Initializer. The resource config must be set before the
require check.


```solidity
function initialize(
    address _owner,
    uint256 _overhead,
    uint256 _scalar,
    bytes32 _batcherHash,
    uint64 _gasLimit,
    address _unsafeBlockSigner,
    ResourceMetering.ResourceConfig memory _config
) public initializer;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_owner`|`address`|            Initial owner of the contract.|
|`_overhead`|`uint256`|         Initial overhead value.|
|`_scalar`|`uint256`|           Initial scalar value.|
|`_batcherHash`|`bytes32`|      Initial batcher hash.|
|`_gasLimit`|`uint64`|         Initial gas limit.|
|`_unsafeBlockSigner`|`address`|Initial unsafe block signer address.|
|`_config`|`ResourceConfig.ResourceMetering`|           Initial ResourceConfig.|


### minimumGasLimit

Returns the minimum L2 gas limit that can be safely set for the system to
operate. The L2 gas limit must be larger than or equal to the amount of
gas that is allocated for deposits per block plus the amount of gas that
is allocated for the system transaction.
This function is used to determine if changes to parameters are safe.


```solidity
function minimumGasLimit() public view returns (uint64);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint64`|uint64|


### unsafeBlockSigner

High level getter for the unsafe block signer address. Unsafe blocks can be
propagated across the p2p network if they are signed by the key corresponding to
this address.


```solidity
function unsafeBlockSigner() external view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the unsafe block signer.|


### setUnsafeBlockSigner

Updates the unsafe block signer address.


```solidity
function setUnsafeBlockSigner(address _unsafeBlockSigner) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_unsafeBlockSigner`|`address`|New unsafe block signer address.|


### setBatcherHash

Updates the batcher hash.


```solidity
function setBatcherHash(bytes32 _batcherHash) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_batcherHash`|`bytes32`|New batcher hash.|


### setGasConfig

Updates gas config.


```solidity
function setGasConfig(uint256 _overhead, uint256 _scalar) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_overhead`|`uint256`|New overhead value.|
|`_scalar`|`uint256`|  New scalar value.|


### setGasLimit

Updates the L2 gas limit.


```solidity
function setGasLimit(uint64 _gasLimit) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_gasLimit`|`uint64`|New gas limit.|


### _setUnsafeBlockSigner

Low level setter for the unsafe block signer address. This function exists to
deduplicate code around storing the unsafeBlockSigner address in storage.


```solidity
function _setUnsafeBlockSigner(address _unsafeBlockSigner) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_unsafeBlockSigner`|`address`|New unsafeBlockSigner value.|


### resourceConfig

A getter for the resource config. Ensures that the struct is
returned instead of a tuple.


```solidity
function resourceConfig() external view returns (ResourceMetering.ResourceConfig memory);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`ResourceConfig.ResourceMetering`|ResourceConfig|


### setResourceConfig

An external setter for the resource config. In the future, this
method may emit an event that the `op-node` picks up for when the
resource config is changed.


```solidity
function setResourceConfig(ResourceMetering.ResourceConfig memory _config) external onlyOwner;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_config`|`ResourceConfig.ResourceMetering`|The new resource config values.|


### _setResourceConfig

An internal setter for the resource config. Ensures that the
config is sane before storing it by checking for invariants.


```solidity
function _setResourceConfig(ResourceMetering.ResourceConfig memory _config) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_config`|`ResourceConfig.ResourceMetering`|The new resource config.|


## Events
### ConfigUpdate
Emitted when configuration is updated


```solidity
event ConfigUpdate(uint256 indexed version, UpdateType indexed updateType, bytes data);
```

## Enums
### UpdateType
Enum representing different types of updates.


```solidity
enum UpdateType {
    BATCHER,
    GAS_CONFIG,
    GAS_LIMIT,
    UNSAFE_BLOCK_SIGNER
}
```

