// SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2 state is committed to in this contract
 *         The payable keyword is used on appendL2Output to save gas on the msg.value check.
 *         This contract should be deployed behind an upgradable proxy
 */
// slither-disable-next-line locked-ether
contract L2OutputOracle is OwnableUpgradeable, Semver {
    /**
     * @notice OutputProposal represents a commitment to the L2 state.
     *         The timestamp is the L1 timestamp that the output root is posted.
     *         This timestamp is used to verify that the finalization period
     *         has passed since the output root was submitted.
     */
    struct OutputProposal {
        bytes32 outputRoot;
        uint256 timestamp;
    }

    /**
     * @notice Emitted when an output is appended.
     *
     * @param l2Output      The output root.
     * @param l1Timestamp   The L1 timestamp when appended.
     * @param l2BlockNumber The L2 block number of the output root.
     */
    event L2OutputAppended(
        bytes32 indexed l2Output,
        uint256 indexed l1Timestamp,
        uint256 indexed l2BlockNumber
    );

    /**
     * @notice Emitted when an output is deleted.
     *
     * @param l2Output      The output root.
     * @param l1Timestamp   The L1 timestamp when appended.
     * @param l2BlockNumber The L2 block number of the output root.
     */
    event L2OutputDeleted(
        bytes32 indexed l2Output,
        uint256 indexed l1Timestamp,
        uint256 indexed l2BlockNumber
    );

    /**
     * @notice Emitted when the sequencer address is changed.
     *
     * @param previousSequencer The previous sequencer address.
     * @param newSequencer      The new sequencer address.
     */
    event SequencerChanged(address indexed previousSequencer, address indexed newSequencer);

    /**
     * @notice The interval in L2 blocks at which checkpoints must be submitted.
     */
    uint256 public immutable SUBMISSION_INTERVAL;

    /**
     * @notice The number of blocks in the chain before the first block in this contract.
     */
    uint256 public immutable HISTORICAL_TOTAL_BLOCKS;

    /**
     * @notice The number of the first L2 block recorded in this contract.
     */
    uint256 public immutable STARTING_BLOCK_NUMBER;

    /**
     * @notice The timestamp of the first L2 block recorded in this contract.
     */
    uint256 public immutable STARTING_TIMESTAMP;

    /**
     * @notice The time between L2 blocks in seconds.
     */
    uint256 public immutable L2_BLOCK_TIME;

    /**
     * @notice The address of the sequencer;
     */
    address public sequencer;

    /**
     * @notice The number of the most recent L2 block recorded in this contract.
     */
    uint256 public latestBlockNumber;

    /**
     * @notice A mapping from L2 block numbers to the respective output root. Note that these
     *         outputs should not be considered finalized until the finalization period (as defined
     *         in the Optimism Portal) has passed.
     */
    mapping(uint256 => OutputProposal) internal l2Outputs;

    /**
     * @notice Reverts if called by any account other than the sequencer.
     */
    modifier onlySequencer() {
        require(sequencer == msg.sender, "OutputOracle: caller is not the sequencer");
        _;
    }

    /**
     * @custom:semver 0.0.1
     *
     * @param _submissionInterval    Interval in blocks at which checkpoints must be submitted.
     * @param _genesisL2Output       The initial L2 output of the L2 chain.
     * @param _historicalTotalBlocks Number of blocks preceding this L2 chain.
     * @param _startingBlockNumber   The number of the first L2 block.
     * @param _startingTimestamp     The timestamp of the first L2 block.
     * @param _l2BlockTime           The timestamp of the first L2 block.
     * @param _sequencer             The address of the sequencer.
     * @param _owner                 The address of the owner.
     */
    constructor(
        uint256 _submissionInterval,
        bytes32 _genesisL2Output,
        uint256 _historicalTotalBlocks,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        uint256 _l2BlockTime,
        address _sequencer,
        address _owner
    ) Semver(0, 0, 1) {
        require(
            _l2BlockTime < block.timestamp,
            "Output Oracle: Initial L2 block time must be less than current time"
        );

        SUBMISSION_INTERVAL = _submissionInterval;
        HISTORICAL_TOTAL_BLOCKS = _historicalTotalBlocks;
        STARTING_BLOCK_NUMBER = _startingBlockNumber;
        STARTING_TIMESTAMP = _startingTimestamp;
        L2_BLOCK_TIME = _l2BlockTime;

        initialize(_genesisL2Output, _startingBlockNumber, _sequencer, _owner);
    }

    /**
     * @notice Initializer.
     *
     * @param _genesisL2Output     The initial L2 output of the L2 chain.
     * @param _startingBlockNumber The timestamp to start L2 block at.
     * @param _sequencer           The address of the sequencer.
     * @param _owner               The address of the owner.
     */
    function initialize(
        bytes32 _genesisL2Output,
        uint256 _startingBlockNumber,
        address _sequencer,
        address _owner
    ) public initializer {
        l2Outputs[_startingBlockNumber] = OutputProposal(_genesisL2Output, block.timestamp);
        latestBlockNumber = _startingBlockNumber;
        __Ownable_init();
        changeSequencer(_sequencer);
        _transferOwnership(_owner);
    }

    /**
     * @notice Accepts an L2 outputRoot and the timestamp of the corresponding L2 block. The
     *         timestamp must be equal to the current value returned by `nextTimestamp()` in order
     *         to be accepted. This function may only be called by the Sequencer.
     *
     * @param _l2Output      The L2 output of the checkpoint block.
     * @param _l2BlockNumber The L2 block number that resulted in _l2Output.
     * @param _l1Blockhash   A block hash which must be included in the current chain.
     * @param _l1BlockNumber The block number with the specified block hash.
     */
    function appendL2Output(
        bytes32 _l2Output,
        uint256 _l2BlockNumber,
        bytes32 _l1Blockhash,
        uint256 _l1BlockNumber
    ) external payable onlySequencer {
        require(
            _l2BlockNumber == nextBlockNumber(),
            "OutputOracle: Block number must be equal to next expected block number."
        );
        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "OutputOracle: Cannot append L2 output in future."
        );
        require(_l2Output != bytes32(0), "OutputOracle: Cannot submit empty L2 output.");

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
                "OutputOracle: Blockhash does not match the hash at the expected height."
            );
        }

        l2Outputs[_l2BlockNumber] = OutputProposal(_l2Output, block.timestamp);
        latestBlockNumber = _l2BlockNumber;

        emit L2OutputAppended(_l2Output, block.timestamp, _l2BlockNumber);
    }

    /**
     * @notice Deletes the most recent output. This is used to remove the most recent output in the
     *         event that an erreneous output is submitted. It can only be called by the contract's
     *         owner, not the sequencer. Longer term, this should be replaced with a more robust
     *         mechanism which will allow deletion of proposals shown to be invalid by a fault
     *         proof.
     *
     * @param _proposal Represents the output proposal to delete
     */
    function deleteL2Output(OutputProposal memory _proposal) external onlyOwner {
        OutputProposal memory outputToDelete = l2Outputs[latestBlockNumber];

        require(
            _proposal.outputRoot == outputToDelete.outputRoot,
            "OutputOracle: The output root to delete does not match the latest output proposal."
        );
        require(
            _proposal.timestamp == outputToDelete.timestamp,
            "OutputOracle: The timestamp to delete does not match the latest output proposal."
        );

        emit L2OutputDeleted(
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
     *         Returns a null output proposal if none is found.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function getL2Output(uint256 _l2BlockNumber) external view returns (OutputProposal memory) {
        return l2Outputs[_l2BlockNumber];
    }

    /**
     * @notice Returns the L2 timestamp corresponding to a given L2 block number.
     *         Returns a null output proposal if none is found.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function computeL2Timestamp(uint256 _l2BlockNumber) public view returns (uint256) {
        require(
            _l2BlockNumber >= STARTING_BLOCK_NUMBER,
            "OutputOracle: Block number must be greater than or equal to the starting block number."
        );

        return STARTING_TIMESTAMP + ((_l2BlockNumber - STARTING_BLOCK_NUMBER) * L2_BLOCK_TIME);
    }

    /**
     * @notice Transfers the sequencer role to a new account (`newSequencer`).
     *         Can only be called by the current owner.
     */
    function changeSequencer(address _newSequencer) public onlyOwner {
        require(_newSequencer != address(0), "OutputOracle: new sequencer is the zero address");
        require(_newSequencer != owner(), "OutputOracle: sequencer cannot be same as the owner");
        emit SequencerChanged(sequencer, _newSequencer);
        sequencer = _newSequencer;
    }
}
