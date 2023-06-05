# GasBenchMark_OptimismPortal
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/BenchmarkTest.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)


## State Variables
### _defaultTx

```solidity
Types.WithdrawalTransaction _defaultTx;
```


### _proposedOutputIndex

```solidity
uint256 _proposedOutputIndex;
```


### _proposedBlockNumber

```solidity
uint256 _proposedBlockNumber;
```


### _withdrawalProof

```solidity
bytes[] _withdrawalProof;
```


### _outputRootProof

```solidity
Types.OutputRootProof internal _outputRootProof;
```


### _outputRoot

```solidity
bytes32 _outputRoot;
```


## Functions
### constructor


```solidity
constructor();
```

### setUp


```solidity
function setUp() public virtual override;
```

### test_depositTransaction_benchmark


```solidity
function test_depositTransaction_benchmark() external;
```

### test_depositTransaction_benchmark_1


```solidity
function test_depositTransaction_benchmark_1() external;
```

### test_proveWithdrawalTransaction_benchmark


```solidity
function test_proveWithdrawalTransaction_benchmark() external;
```

