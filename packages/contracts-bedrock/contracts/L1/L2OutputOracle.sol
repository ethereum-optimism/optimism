// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { Semver } from "../universal/Semver.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2 state is committed to in this contract
 *         The payable keyword is used on proposeL2Output to save gas on the msg.value check.
 *         This contract should be deployed behind an upgradable proxy
 */
// slither-disable-next-line locked-ether
contract L2OutputOracle is Initializable, Semver {
    /**
     * @notice The interval in L2 blocks at which checkpoints must be submitted.
     */
    uint256 public immutable SUBMISSION_INTERVAL;

    /**
     * @notice The time between L2 blocks in seconds.
     */
    uint256 public immutable L2_BLOCK_TIME;

    /**
     * @notice The address of the challenger. Can be updated via upgrade.
     */
    address public immutable CHALLENGER;

    /**
     * @notice The address of the proposer. Can be updated via upgrade.
     */
    address public immutable PROPOSER;

    /**
     * @notice The number of the first L2 block recorded in this contract.
     */
    uint256 public startingBlockNumber;

    /**
     * @notice The timestamp of the first L2 block recorded in this contract.
     */
    uint256 public startingTimestamp;

    /**
     * @notice The number of the most recent L2 block recorded in this contract.
     */
    uint256 public latestBlockNumber;

    /**
     * @notice A mapping from L2 block numbers to the respective output root. Note that these
     *         outputs should not be considered finalized until the finalization period (as defined
     *         in the Optimism Portal) has passed.
     */
    mapping(uint256 => Types.OutputProposal) internal l2Outputs;

    /**
     * @notice Emitted when an output is proposed.
     *
     * @param outputRoot    The output root.
     * @param l1Timestamp   The L1 timestamp when proposed.
     * @param l2BlockNumber The L2 block number of the output root.
     */
    event OutputProposed(
        bytes32 indexed outputRoot,
        uint256 indexed l1Timestamp,
        uint256 indexed l2BlockNumber
    );

    /**
     * @notice Emitted when outputs are deleted.
     *
     * @param l2BlockNumber First L2 block number deleted.
     */
    event OutputsDeleted(uint256 indexed l2BlockNumber);

    /**
     * @custom:semver 0.0.1
     *
     * @param _submissionInterval    Interval in blocks at which checkpoints must be submitted.
     * @param _l2BlockTime           The time per L2 block, in seconds.
     * @param _startingBlockNumber   The number of the first L2 block.
     * @param _startingTimestamp     The timestamp of the first L2 block.
     * @param _proposer              The address of the proposer.
     * @param _challenger            The address of the challenger.
     */
    constructor(
        uint256 _submissionInterval,
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _proposer,
        address _challenger
    ) Semver(0, 0, 1) {
        SUBMISSION_INTERVAL = _submissionInterval;
        L2_BLOCK_TIME = _l2BlockTime;
        PROPOSER = _proposer;
        CHALLENGER = _challenger;

        initialize(_startingBlockNumber, _startingTimestamp);
    }

    /**
     * @notice Initializer.
     *
     * @param _startingBlockNumber Block number for the first recoded L2 block.
     * @param _startingTimestamp   Timestamp for the first recoded L2 block.
     */
    function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp)
        public
        initializer
    {
        require(
            _startingTimestamp <= block.timestamp,
            "L2OutputOracle: starting L2 timestamp must be less than current time"
        );

        startingTimestamp = _startingTimestamp;
        startingBlockNumber = _startingBlockNumber;
        latestBlockNumber = _startingBlockNumber;
    }

    /**
     * @notice Deletes all output proposals after and including the proposal that corresponds to
     *         the given block number. Only the challenger address can delete outputs.
     *
     * @param _l2BlockNumber L2 block number of the first output root to delete.
     */
    // solhint-disable-next-line ordering
    function deleteL2Outputs(uint256 _l2BlockNumber) external {
        require(
            msg.sender == CHALLENGER,
            "L2OutputOracle: only the challenger address can delete outputs"
        );

        // Simple check that accomplishes two things:
        //   1. Prevents deleting anything before (and including) the starting block.
        //   2. Prevents deleting anything other than a checkpoint block.
        require(
            l2Outputs[_l2BlockNumber].outputRoot != bytes32(0),
            "L2OutputOracle: cannot delete a non-existent output"
        );

        // Prevent deleting beyond latest block number. Above check will miss this case if we
        // already deleted an output and then the user tries to delete a later output.
        require(
            _l2BlockNumber <= latestBlockNumber,
            "L2OutputOracle: cannot delete outputs after the latest block number"
        );

        // We're setting the latest block number back to the checkpoint block before the given L2
        // block number. Next proposal will overwrite the deleted output and following proposals
        // will delete any outputs after that.
        latestBlockNumber = _l2BlockNumber - SUBMISSION_INTERVAL;

        emit OutputsDeleted(_l2BlockNumber);
    }

    /**
     * @notice Accepts an outputRoot and the timestamp of the corresponding L2 block. The
     *         timestamp must be equal to the current value returned by `nextTimestamp()` in order
     *         to be accepted. This function may only be called by the Proposer.
     *
     * @param _outputRoot    The L2 output of the checkpoint block.
     * @param _l2BlockNumber The L2 block number that resulted in _outputRoot.
     * @param _l1Blockhash   A block hash which must be included in the current chain.
     * @param _l1BlockNumber The block number with the specified block hash.
     */
    function proposeL2Output(
        bytes32 _outputRoot,
        uint256 _l2BlockNumber,
        bytes32 _l1Blockhash,
        uint256 _l1BlockNumber
    ) external payable {
        require(
            msg.sender == PROPOSER,
            "L2OutputOracle: only the proposer address can propose new outputs"
        );

        require(
            _l2BlockNumber == nextBlockNumber(),
            "L2OutputOracle: block number must be equal to next expected block number"
        );

        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2OutputOracle: cannot propose L2 output in the future"
        );

        require(
            _outputRoot != bytes32(0),
            "L2OutputOracle: L2 output proposal cannot be the zero hash"
        );

        if (_l1Blockhash != bytes32(0)) {
            // This check allows the proposer to propose an output based on a given L1 block,
            // without fear that it will be reorged out.
            // It will also revert if the blockheight provided is more than 256 blocks behind the
            // chain tip (as the hash will return as zero). This does open the door to a griefing
            // attack in which the proposer's submission is censored until the block is no longer
            // retrievable, if the proposer is experiencing this attack it can simply leave out the
            // blockhash value, and delay submission until it is confident that the L1 block is
            // finalized.
            require(
                blockhash(_l1BlockNumber) == _l1Blockhash,
                "L2OutputOracle: blockhash does not match the hash at the expected height"
            );
        }

        emit OutputProposed(_outputRoot, block.timestamp, _l2BlockNumber);

        l2Outputs[_l2BlockNumber] = Types.OutputProposal(_outputRoot, block.timestamp);
        latestBlockNumber = _l2BlockNumber;
    }

    /**
     * @notice Returns the L2 output proposal associated with a target L2 block number. If the
     *         L2 block number provided is between checkpoints, this function will rerutn the next
     *         proposal for the next checkpoint.
     *         Reverts if the output proposal is either not found, or predates
     *         the startingBlockNumber.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function getL2Output(uint256 _l2BlockNumber)
        external
        view
        returns (Types.OutputProposal memory)
    {
        require(
            _l2BlockNumber <= latestBlockNumber,
            "L2OutputOracle: block number cannot be greater than the latest block number"
        );

        // Find the distance between _l2BlockNumber, and the checkpoint block before it.
        uint256 offset = (_l2BlockNumber - startingBlockNumber) % SUBMISSION_INTERVAL;

        // If the offset is zero, then the _l2BlockNumber should be checkpointed.
        // Otherwise, we'll look up the next block that will be checkpointed.
        uint256 lookupBlockNumber = offset == 0
            ? _l2BlockNumber
            : _l2BlockNumber + (SUBMISSION_INTERVAL - offset);

        Types.OutputProposal memory output = l2Outputs[lookupBlockNumber];
        require(
            output.outputRoot != bytes32(0),
            "L2OutputOracle: no output found for the given block number"
        );

        return output;
    }

    /**
     * @notice Computes the block number of the next L2 block that needs to be checkpointed.
     */
    function nextBlockNumber() public view returns (uint256) {
        return latestBlockNumber + SUBMISSION_INTERVAL;
    }

    /**
     * @notice Returns the L2 timestamp corresponding to a given L2 block number.
     *         If the L2 block number provided is between checkpoints, this function will return the
     *         timestamp of the previous checkpoint.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function computeL2Timestamp(uint256 _l2BlockNumber) public view returns (uint256) {
        return startingTimestamp + ((_l2BlockNumber - startingBlockNumber) * L2_BLOCK_TIME);
    }
}
