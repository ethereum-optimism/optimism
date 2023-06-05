# ArtifactResourceMetering_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ResourceMetering.t.sol)

**Inherits:**
Test

A table test that sets the state of the ResourceParams and then requests
various amounts of gas. This test ensures that a wide range of values
can safely be used with the `ResourceMetering` contract.
It also writes a CSV file to disk that includes useful information
about how much gas is used and how expensive it is in USD terms to
purchase the deposit gas.


## State Variables
### minimumBaseFee

```solidity
uint128 internal minimumBaseFee;
```


### maximumBaseFee

```solidity
uint128 internal maximumBaseFee;
```


### maxResourceLimit

```solidity
uint64 internal maxResourceLimit;
```


### targetResourceLimit

```solidity
uint64 internal targetResourceLimit;
```


### outfile

```solidity
string internal outfile;
```


### cannotBuyMoreGas

```solidity
bytes32 internal cannotBuyMoreGas = 0x84edc668cfd5e050b8999f43ff87a1faaa93e5f935b20bc1dd4d3ff157ccf429;
```


### overflowErr

```solidity
bytes32 internal overflowErr = 0x1ca389f2c8264faa4377de9ce8e14d6263ef29c68044a9272d405761bab2db27;
```


### emptyReturnData

```solidity
bytes32 internal emptyReturnData = 0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470;
```


## Functions
### setUp

Sets up the tests by getting constants from the ResourceMetering
contract.


```solidity
function setUp() public;
```

### test_meter_generateArtifact_succeeds

Generate a CSV file. The call to `meter` should be called with at
most the L1 block gas limit. Without specifying the amount of
gas, it can take very long to execute.


```solidity
function test_meter_generateArtifact_succeeds() external;
```

