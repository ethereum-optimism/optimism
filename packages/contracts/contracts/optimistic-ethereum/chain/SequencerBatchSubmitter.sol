pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import { CanonicalTransactionChain } from "./CanonicalTransactionChain.sol";
import { StateCommitmentChain } from "./StateCommitmentChain.sol";

/**
 * @title SequencerBatchSubmitter
 * @notice Helper contract that allows the sequencer to submit both a state
 *         commitment batch and tx batch in a single transaction. This ensures
 *         that # state roots == # of txs, preventing other users from
 *         submitting state batches to the state chain.
 */
contract SequencerBatchSubmitter {
    /*
     * Contract Variables
     */

    CanonicalTransactionChain canonicalTransactionChain;
    StateCommitmentChain stateCommitmentChain;
    address public sequencer;

    /*
     * Modifiers
     */

    modifier onlySequencer () {
        require(
            msg.sender == sequencer,
            "Only the sequencer may perform this action"
        );
        _;
    }


    /*
    * Constructor
    */

    constructor(address _sequencer) public {
        sequencer = _sequencer;
    }


    /*
    * Public Functions
    */

    function initialize(
        address _canonicalTransactionChain,
        address _stateCommitmentChain
    ) public onlySequencer {
        canonicalTransactionChain = CanonicalTransactionChain(_canonicalTransactionChain);
        stateCommitmentChain = StateCommitmentChain(_stateCommitmentChain);
    }

    /**
     * @notice Append equal sized batches of transactions and state roots to
     *         their respective chains.
     * @param _txBatch An array of transactions.
     * @param _txBatchTimestamp The timestamp that will be submitted with the
     *                          tx batch - this timestamp will likely lag
     *                          behind the actual time by a few minutes.
     * @param _stateBatch An array of 32 byte state roots
     */
    function appendTransitionBatch(
        bytes[] memory _txBatch,
        uint _txBatchTimestamp,
        bytes[] memory _stateBatch
    ) public onlySequencer {
        require(
            _stateBatch.length == _txBatch.length,
            "Must append the same number of state roots and transactions"
        );

        canonicalTransactionChain.appendSequencerBatch(_txBatch, _txBatchTimestamp);
        stateCommitmentChain.appendStateBatch(_stateBatch);
    }
}
