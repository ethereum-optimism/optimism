# OptimismPortal_Invariant_Harness
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/OptimismPortal.t.sol)

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


### _stateRoot

```solidity
bytes32 _stateRoot;
```


### _storageRoot

```solidity
bytes32 _storageRoot;
```


### _outputRoot

```solidity
bytes32 _outputRoot;
```


### _withdrawalHash

```solidity
bytes32 _withdrawalHash;
```


### _withdrawalProof

```solidity
bytes[] _withdrawalProof;
```


### _outputRootProof

```solidity
Types.OutputRootProof internal _outputRootProof;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

