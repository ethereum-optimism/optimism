# L2OutputOracle_Proposer
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/L2OutputOracle.t.sol)


## State Variables
### oracle

```solidity
L2OutputOracle internal oracle;
```


### vm

```solidity
Vm internal vm;
```


## Functions
### constructor


```solidity
constructor(L2OutputOracle _oracle, Vm _vm);
```

### proposeL2Output

*Allows the actor to propose an L2 output to the `L2OutputOracle`*


```solidity
function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1BlockHash, uint256 _l1BlockNumber)
    external;
```

