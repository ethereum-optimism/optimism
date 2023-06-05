# SystemConfig_Setters_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/SystemConfig.t.sol)

**Inherits:**
[SystemConfig_Init](/contracts/test/SystemConfig.t.sol/contract.SystemConfig_Init.md)


## Functions
### testFuzz_setBatcherHash_succeeds


```solidity
function testFuzz_setBatcherHash_succeeds(bytes32 newBatcherHash) external;
```

### testFuzz_setGasConfig_succeeds


```solidity
function testFuzz_setGasConfig_succeeds(uint256 newOverhead, uint256 newScalar) external;
```

### testFuzz_setGasLimit_succeeds


```solidity
function testFuzz_setGasLimit_succeeds(uint64 newGasLimit) external;
```

### testFuzz_setUnsafeBlockSigner_succeeds


```solidity
function testFuzz_setUnsafeBlockSigner_succeeds(address newUnsafeSigner) external;
```

## Events
### ConfigUpdate

```solidity
event ConfigUpdate(uint256 indexed version, SystemConfig.UpdateType indexed updateType, bytes data);
```

