# CrossDomainMessenger_BaseGas_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CrossDomainMessenger.t.sol)

**Inherits:**
[Messenger_Initializer](/contracts/test/CommonTest.t.sol/contract.Messenger_Initializer.md)


## Functions
### test_baseGas_succeeds


```solidity
function test_baseGas_succeeds() external view;
```

### testFuzz_baseGas_succeeds


```solidity
function testFuzz_baseGas_succeeds(uint32 _minGasLimit) external view;
```

### testFuzz_baseGas_portalMinGasLimit_succeeds

The baseGas function should always return a value greater than
or equal to the minimum gas limit value on the OptimismPortal.
This guarantees that the messengers will always pass sufficient
gas to the OptimismPortal.


```solidity
function testFuzz_baseGas_portalMinGasLimit_succeeds(bytes memory _data, uint32 _minGasLimit) external;
```

