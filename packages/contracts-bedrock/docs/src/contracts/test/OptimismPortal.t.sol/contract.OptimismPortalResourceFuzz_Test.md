# OptimismPortalResourceFuzz_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismPortal.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)

*Test various values of the resource metering config to ensure that deposits cannot be
broken by changing the config.*


## State Variables
### MAX_GAS_LIMIT
*The max gas limit observed throughout this test. Setting this too high can cause
the test to take too long to run.*


```solidity
uint256 constant MAX_GAS_LIMIT = 30_000_000;
```


## Functions
### testFuzz_systemConfigDeposit_succeeds

*Test that various values of the resource metering config will not break deposits.*


```solidity
function testFuzz_systemConfigDeposit_succeeds(
    uint32 _maxResourceLimit,
    uint8 _elasticityMultiplier,
    uint8 _baseFeeMaxChangeDenominator,
    uint32 _minimumBaseFee,
    uint32 _systemTxMaxGas,
    uint128 _maximumBaseFee,
    uint64 _gasLimit,
    uint64 _prevBoughtGas,
    uint128 _prevBaseFee,
    uint8 _blockDiff
) external;
```

