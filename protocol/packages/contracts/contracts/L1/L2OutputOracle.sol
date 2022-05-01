//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

/* Library Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title L2OutputOracle
 * @notice
 */
// The payable keyword is used on appendL2Output to save gas on the msg.value check.
// slither-disable-next-line locked-ether
contract L2OutputOracle is Ownable {
    /**********
     * Events *
     **********/

    /// @notice Emitted when an output is appended.
    event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp);

    /// @notice Emitted when an output is deleted.
    event l2OutputDeleted(bytes32 indexed _l2Output, uint256 indexed _l2timestamp);

    /**********************
     * Contract Variables *
     **********************/

    /// @notice The interval in seconds at which checkpoints must be submitted.
    uint256 public immutable SUBMISSION_INTERVAL;

    /// @notice The time between blocks on L2.
    uint256 public immutable L2_BLOCK_TIME;

    /// @notice The number of blocks in the chain before the first block in this contract.
    uint256 public immutable HISTORICAL_TOTAL_BLOCKS;

    /// @notice The timestamp of the first L2 block recorded in this contract.
    uint256 public immutable STARTING_BLOCK_TIMESTAMP;

    /// @notice The timestamp of the most recent L2 block recorded in this contract.
    uint256 public latestBlockTimestamp;

    /// @notice A mapping from L2 timestamps to the output root for the block with that timestamp.
    mapping(uint256 => bytes32) internal l2Outputs;

    /***************
     * Constructor *
     ***************/

    /**
     * @notice Initialize the L2OutputOracle contract.
     * @param _submissionInterval The desired interval in seconds at which
     *        checkpoints must be submitted.
     * @param _l2BlockTime The desired L2 inter-block time in seconds.
     * @param _genesisL2Output The initial L2 output of the L2 chain.
     * @param _historicalTotalBlocks The number of blocks that preceding the
     *        initialization of the L2 chain.
     * @param _startingBlockTimestamp The timestamp to start L2 block at.
     */
    constructor(
        uint256 _submissionInterval,
        uint256 _l2BlockTime,
        bytes32 _genesisL2Output,
        uint256 _historicalTotalBlocks,
        uint256 _startingBlockTimestamp,
        address sequencer
    ) {
        require(
            _submissionInterval % _l2BlockTime == 0,
            "Submission Interval must be a multiple of L2 Block Time"
        );

        SUBMISSION_INTERVAL = _submissionInterval;
        L2_BLOCK_TIME = _l2BlockTime;
        l2Outputs[_startingBlockTimestamp] = _genesisL2Output; // solhint-disable not-rely-on-time
        HISTORICAL_TOTAL_BLOCKS = _historicalTotalBlocks;
        latestBlockTimestamp = _startingBlockTimestamp; // solhint-disable not-rely-on-time
        STARTING_BLOCK_TIMESTAMP = _startingBlockTimestamp; // solhint-disable not-rely-on-time

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
     * @param _l2timestamp The L2 block timestamp that resulted in _l2Output.
     * @param _l1Blockhash A block hash which must be included in the current chain.
     * @param _l1Blocknumber The block number with the specified block hash.
     */
    function appendL2Output(
        bytes32 _l2Output,
        uint256 _l2timestamp,
        bytes32 _l1Blockhash,
        uint256 _l1Blocknumber
    ) external payable onlyOwner {
        require(_l2timestamp < block.timestamp, "Cannot append L2 output in future");
        require(_l2timestamp == nextTimestamp(), "Timestamp not equal to next expected timestamp");
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
                blockhash(_l1Blocknumber) == _l1Blockhash,
                "Blockhash does not match the hash at the expected height."
            );
        }

        l2Outputs[_l2timestamp] = _l2Output;
        latestBlockTimestamp = _l2timestamp;

        emit l2OutputAppended(_l2Output, _l2timestamp);
    }

    /**
     * @notice Deletes the most recent output.
     * @param _l2Output The value of the most recent output. Used to prevent erroneously deleting
     *  the wrong root
     */
    function deleteL2Output(bytes32 _l2Output) external onlyOwner {
        bytes32 outputToDelete = l2Outputs[latestBlockTimestamp];
        require(_l2Output == outputToDelete, "Can only delete the most recent output.");
        emit l2OutputDeleted(outputToDelete, latestBlockTimestamp);

        delete l2Outputs[latestBlockTimestamp];
        latestBlockTimestamp = latestBlockTimestamp - SUBMISSION_INTERVAL;
    }

    /**
     * @notice Computes the timestamp of the next L2 block that needs to be checkpointed.
     */
    function nextTimestamp() public view returns (uint256) {
        return latestBlockTimestamp + SUBMISSION_INTERVAL;
    }

    /**
     * @notice Returns the L2 output root given a target L2 block timestamp. Returns 0 if none is
     * found.
     * @param _l2Timestamp The L2 block timestamp of the target block.
     */
    function getL2Output(uint256 _l2Timestamp) external view returns (bytes32) {
        return l2Outputs[_l2Timestamp];
    }

    /**
     * @notice Computes the L2 block number given a target L2 block timestamp.
     * @param _l2timestamp The L2 block timestamp of the target block.
     */
    function computeL2BlockNumber(uint256 _l2timestamp) external view returns (uint256) {
        require(
            _l2timestamp >= STARTING_BLOCK_TIMESTAMP,
            "Timestamp prior to startingBlockTimestamp"
        );
        // For the first block recorded (ie. _l2timestamp = STARTING_BLOCK_TIMESTAMP), the
        // L2BlockNumber should be HISTORICAL_TOTAL_BLOCKS + 1.
        unchecked {
            return
                HISTORICAL_TOTAL_BLOCKS +
                1 +
                ((_l2timestamp - STARTING_BLOCK_TIMESTAMP) / L2_BLOCK_TIME);
        }
    }
}
