pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";
import {RollupMerkleUtils} from "./RollupMerkleUtils.sol";
import {CanonicalTransactionChain} from "./CanonicalTransactionChain.sol";
import {StateCommitmentChain} from "./StateCommitmentChain.sol";

contract SequencerBatchSubmitter {
  CanonicalTransactionChain canonicalTransactionChain;
  StateCommitmentChain stateCommitmentChain;
  address public sequencer;

  constructor(address _sequencer) public {
    sequencer = _sequencer;
  }

  function initialize(
    address _canonicalTransactionChain,
    address _stateCommitmentChain
  ) public onlySequencer {
    canonicalTransactionChain = CanonicalTransactionChain(_canonicalTransactionChain);
    stateCommitmentChain = StateCommitmentChain(_stateCommitmentChain);
  }

  function appendTransitionBatch(
    bytes[] memory _txBatch,
    uint _txBatchTimestamp,
    bytes[] memory _stateBatch
  ) public onlySequencer {
    require(_stateBatch.length == _txBatch.length,
      "Must append the same number of state roots and transactions");
    canonicalTransactionChain.appendTransactionBatch(_txBatch, _txBatchTimestamp);
    stateCommitmentChain.appendStateBatch(_stateBatch);
  }

  modifier onlySequencer () {
    require(msg.sender == sequencer, "Only the sequencer may perform this action");
    _;
  }
}
