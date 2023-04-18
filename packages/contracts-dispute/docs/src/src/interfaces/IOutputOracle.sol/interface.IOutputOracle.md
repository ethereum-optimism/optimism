# IOutputOracle
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IOutputOracle.sol)

An interface for the L2OutputOracle contract.


## Functions
### deleteL2Outputs

Deletes the L2 output for the given parameter.


```solidity
function deleteL2Outputs(uint256) external;
```

### getL2Output

Returns the L2 output for the given parameter.


```solidity
function getL2Output(uint256 _l2OutputIndex) external view returns (OutputProposal memory);
```

## Structs
### OutputProposal
OutputProposal represents a commitment to the L2 state. The timestamp is the L1
timestamp that the output root is posted. This timestamp is used to verify that the
finalization period has passed since the output root was submitted.


```solidity
struct OutputProposal {
    bytes32 outputRoot;
    uint128 timestamp;
    uint128 l2BlockNumber;
}
```

