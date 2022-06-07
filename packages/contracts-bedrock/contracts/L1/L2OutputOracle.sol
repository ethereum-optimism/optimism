//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title L2OutputOracle
 * @notice The L2 state is committed to in this contract
 * The payable keyword is used on appendL2Output to save gas on the msg.value check.
 * This contract should be deployed behind an upgradable proxy
 */
// slither-disable-next-line locked-ether
contract L2OutputOracle is Ownable {
    /*********
     * Types *
     *********/

    /// @notice OutputProposal represents a commitment to the L2 state.
    /// The timestamp is the L1 timestamp that the output root is posted.
    /// This timestamp is used to verify that the finalization period
    /// has passed since the output root was submitted.
    struct OutputProposal {
        bytes32 outputRoot;
        uint256 timestamp;
    }

    /**********
     * Events *
     **********/

    /// @notice Emitted when an output is appended.
    event l2OutputAppended(
        bytes32 indexed _l2Output,
        uint256 indexed _l1Timestamp,
        uint256 indexed _l2BlockNumber
    );

    /// @notice Emitted when an output is deleted.
    event l2OutputDeleted(
        bytes32 indexed _l2Output,
        uint256 indexed _l1Timestamp,
        uint256 indexed _l2BlockNumber
    );

    /**********************
     * Contract Variables *
     **********************/

    /// @notice The interval in L2 blocks at which checkpoints must be submitted.
    uint256 public immutable SUBMISSION_INTERVAL;

    /// @notice The number of blocks in the chain before the first block in this contract.
    uint256 public immutable HISTORICAL_TOTAL_BLOCKS;

    /// @notice The number of the first L2 block recorded in this contract.
    uint256 public immutable STARTING_BLOCK_NUMBER;

    /// @notice The number of the most recent L2 block recorded in this contract.
    uint256 public latestBlockNumber;

    /// @notice A mapping from L2 block numbers to the respective output root.
    mapping(uint256 => OutputProposal) internal l2Outputs;

    /***************
     * Constructor *
     ***************/

    /**
     * @notice Initialize the L2OutputOracle contract.
     * @param _submissionInterval The desired interval in seconds at which
     *        checkpoints must be submitted.
     * @param _genesisL2Output The initial L2 output of the L2 chain.
     * @param _historicalTotalBlocks The number of blocks that preceding the
     *        initialization of the L2 chain.
     * @param _startingBlockNumber The number to start L2 block at.
     */
    constructor(
        uint256 _submissionInterval,
        bytes32 _genesisL2Output,
        uint256 _historicalTotalBlocks,
        uint256 _startingBlockNumber,
        address sequencer
    ) {
        SUBMISSION_INTERVAL = _submissionInterval;
        l2Outputs[_startingBlockNumber] = OutputProposal(_genesisL2Output, block.timestamp);
        HISTORICAL_TOTAL_BLOCKS = _historicalTotalBlocks;
        latestBlockNumber = _startingBlockNumber;
        STARTING_BLOCK_NUMBER = _startingBlockNumber;

        _transferOwnership(sequencer);
    }

    /*********************************
     * External and Public Functions *
     *********************************/

    /**
     * @notice Accepts an L2 outputRoot and the timestamp of the corresponding L2 block. The
     * timestamp must be equal to the current value returned by `nextTimestamp()` in order to be
     * accepted.
     * This function may only be called by the Sequencer.
     * @param _l2Output The L2 output of the checkpoint block.
     * @param _l2BlockNumber The L2 block number that resulted in _l2Output.
     * @param _l1Blockhash A block hash which must be included in the current chain.
     * @param _l1BlockNumber The block number with the specified block hash.
     */
    function appendL2Output( // todo: rename to proposeL2Output?
        bytes32 _l2Output,
        uint256 _l2BlockNumber,
        bytes32 _l1Blockhash,
        uint256 _l1BlockNumber
    ) external payable onlyOwner {
        require(
            _l2BlockNumber == nextBlockNumber(),
            "Block number must be equal to next expected block number"
        );
        require(_l2Output != bytes32(0), "Cannot submit empty L2 output");

        if (_l1Blockhash != bytes32(0)) {
            // This check allows the sequencer to append an output based on a given L1 block,
            // without fear that it will be reorged out.
            // It will also revert if the blockheight provided is more than 256 blocks behind the
            // chain tip (as the hash will return as zero). This does open the door to a griefing
            // attack in which the sequencer's submission is censored until the block is no longer
            // retrievable, if the sequencer is experiencing this attack it can simply leave out the
            // blockhash value, and delay submission until it is confident that the L1 block is
            // finalized.
            require(
                blockhash(_l1BlockNumber) == _l1Blockhash,
                "Blockhash does not match the hash at the expected height."
            );
        }

        l2Outputs[_l2BlockNumber] = OutputProposal(_l2Output, block.timestamp);
        latestBlockNumber = _l2BlockNumber;

        emit l2OutputAppended(_l2Output, block.timestamp, _l2BlockNumber);
    }

    /**
     * @notice Deletes the most recent output.
     * @param _proposal Represents the output proposal to delete
     */
    function deleteL2Output(OutputProposal memory _proposal) external onlyOwner {
        OutputProposal memory outputToDelete = l2Outputs[latestBlockNumber];

        require(
            _proposal.outputRoot == outputToDelete.outputRoot,
            "The output root to delete does not match the that of the most recent output proposal."
        );
        require(
            _proposal.timestamp == outputToDelete.timestamp,
            "The timestamp to delete does not match the that of the most recent output proposal."
        );

        emit l2OutputDeleted(
            outputToDelete.outputRoot,
            outputToDelete.timestamp,
            latestBlockNumber
        );

        delete l2Outputs[latestBlockNumber];
        latestBlockNumber = latestBlockNumber - SUBMISSION_INTERVAL;
    }

    /**
     * @notice Computes the block number of the next L2 block that needs to be checkpointed.
     */
    function nextBlockNumber() public view returns (uint256) {
        return latestBlockNumber + SUBMISSION_INTERVAL;
    }

    /**
     * @notice Returns the L2 output proposal given a target L2 block number.
     * Returns a null output proposal if none is found.
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function getL2Output(uint256 _l2BlockNumber) external view returns (OutputProposal memory) {
        return l2Outputs[_l2BlockNumber];
    }
}
