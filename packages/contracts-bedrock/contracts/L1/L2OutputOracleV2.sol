// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { Semver } from "../universal/Semver.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2OutputOracleV2
 * @notice The L2OutputOracleV2 contains a map of L2 block numbers to their associated L2 state outputs,
 *         where each output is a commitment to the state of the L2 chain. Other contracts like the
 *         OptimismPortal use these outputs to verify information about the state of L2.
 */
contract L2OutputOracleV2 is Initializable, Semver {
    /**
     * @notice The time between L2 blocks in seconds. Once set, this value MUST NOT be modified.
     */
    uint256 public immutable L2_BLOCK_TIME;

    /**
     * @notice The address of the challenger. Can be updated via upgrade.
     */
    address public immutable CHALLENGER;

    /**
     * @notice Minimum time (in seconds) that must elapse before a withdrawal can be finalized.
     */
    uint256 public immutable FINALIZATION_PERIOD_SECONDS;

    /**
     * @notice Minimum cost of submitting an output proposal.
     */
    uint256 public immutable MINIMUM_OUTPUT_PROPOSAL_COST;

    /**
     * @notice The number of the first L2 block recorded in this contract.
     */
    uint256 public startingBlockNumber;

    /**
     * @notice The timestamp of the first L2 block recorded in this contract.
     */
    uint256 public startingTimestamp;

    /**
     * @notice Highest L2 block number that has been proposed.
     */
    uint256 public highestL2BlockNumber;

    /**
     * @notice Array of L2 output proposals.
     */
    mapping (uint256 => Types.OutputProposal) internal l2Outputs;

    /**
     * @notice Emitted when an output is proposed.
     *
     * @param outputRoot    The output root.
     * @param l2BlockNumber The L2 block number of the output root.
     * @param l1Timestamp   The L1 timestamp when proposed.
     */
    event OutputProposed(
        bytes32 indexed outputRoot,
        uint256 indexed l2BlockNumber,
        uint256 l1Timestamp
    );

    /**
     * @notice Emitted when an output are deleted.
     *
     * @param l2BlockNumber Block number of the proposal that was deleted.
     */
    event OutputDeleted(uint256 indexed l2BlockNumber);

    /**
     * @custom:semver 2.0.0
     *
     * @param _l2BlockTime               The time per L2 block, in seconds.
     * @param _startingBlockNumber       The number of the first L2 block.
     * @param _startingTimestamp         The timestamp of the first L2 block.
     * @param _challenger                The address of the challenger.
     * @param _finalizationPeriodSeconds The time before an output can be finalized.
     * @param _minimumOutputProposalCost The amount that must be paid to post an output.
     */
    constructor(
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _challenger,
        uint256 _finalizationPeriodSeconds,
        uint256 _minimumOutputProposalCost
    ) Semver(2, 0, 0) {
        require(_l2BlockTime > 0, "L2OutputOracleV2: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;
        CHALLENGER = _challenger;
        FINALIZATION_PERIOD_SECONDS = _finalizationPeriodSeconds;
        MINIMUM_OUTPUT_PROPOSAL_COST = _minimumOutputProposalCost;

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
            "L2OutputOracleV2: starting L2 timestamp must be less than current time"
        );

        startingTimestamp = _startingTimestamp;
        startingBlockNumber = _startingBlockNumber;
    }

    /**
     * @notice Deletes an output proposal based on the given L2 block number.
     *
     * @param _l2BlockNumber The L2 block number of the output to delete.
     * @param _setPreviousHigh The L2 block number to (potentially) update the highestL2BlockNumber to.
     */
    // solhint-disable-next-line ordering
    function deleteL2Output(uint256 _l2BlockNumber, uint256 _setPreviousHigh) external {
        require(
            msg.sender == CHALLENGER,
            "L2OutputOracleV2: only the challenger address can delete an output"
        );

        // Do not allow deleting any outputs that have already been finalized.
        require(
            block.timestamp - l2Outputs[_l2BlockNumber].timestamp < FINALIZATION_PERIOD_SECONDS,
            "L2OutputOracleV2: cannot delete outputs that have already been finalized"
        );

        // TODO: This introduces a nasty case whereby if the challenger feeds in a bad value this could remove all outputs
        // TODO: This might be reason to switch back to an array over a mapping
        if (_l2BlockNumber == highestL2BlockNumber) {
            highestL2BlockNumber = _setPreviousHigh;
        }

        delete l2Outputs[_l2BlockNumber];

        emit OutputDeleted(_l2BlockNumber);
    }

    /**
     * @notice Accepts an outputRoot and the timestamp of the corresponding L2 block.
     *
     * @param _outputRoot    The L2 output of the checkpoint block.
     * @param _l2BlockNumber The L2 block number that resulted in _outputRoot.
     * @param _l1BlockHash   A block hash which must be included in the current chain.
     * @param _l1BlockNumber The block number with the specified block hash.
     */
    function proposeL2Output(
        bytes32 _outputRoot,
        uint256 _l2BlockNumber,
        bytes32 _l1BlockHash,
        uint256 _l1BlockNumber
    ) external payable {
        require(
            msg.value >= MINIMUM_OUTPUT_PROPOSAL_COST,
            "L2OutputOracleV2: minimum proposal cost not provided"
        );

        require(
            l2Outputs[_l2BlockNumber].timestamp == 0,
            "L2OutputOracleV2: output already proposed"
        );

        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2OutputOracleV2: cannot propose L2 output in the future"
        );

        require(
            _outputRoot != bytes32(0),
            "L2OutputOracleV2: L2 output proposal cannot be the zero hash"
        );

        if (_l1BlockHash != bytes32(0)) {
            // This check allows the proposer to propose an output based on a given L1 block,
            // without fear that it will be reorged out.
            // It will also revert if the blockheight provided is more than 256 blocks behind the
            // chain tip (as the hash will return as zero). This does open the door to a griefing
            // attack in which the proposer's submission is censored until the block is no longer
            // retrievable, if the proposer is experiencing this attack it can simply leave out the
            // blockhash value, and delay submission until it is confident that the L1 block is
            // finalized.
            require(
                blockhash(_l1BlockNumber) == _l1BlockHash,
                "L2OutputOracleV2: block hash does not match the hash at the expected height"
            );
        }

        // Update the highest L2 block number if necessary.
        if (_l2BlockNumber > highestL2BlockNumber) {
            highestL2BlockNumber = _l2BlockNumber;
        }

        l2Outputs[_l2BlockNumber] =
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            });

        emit OutputProposed(_outputRoot, _l2BlockNumber, block.timestamp);
    }

    /**
     * @notice Returns an output by block number.
     *
     * @param _l2BlockNumber L2 block number of the output to return.
     *
     * @return The output for the given block number.
     */
    function getL2Output(uint256 _l2BlockNumber)
        external
        view
        returns (Types.OutputProposal memory)
    {
        return l2Outputs[_l2BlockNumber];
    }

    /**
     * @notice Returns the L2 output proposal that checkpoints a given L2 block number. Uses a
     *         binary search to find the first output greater than or equal to the given block.
     *
     * @param _l2BlockNumber L2 block number to find a checkpoint for.
     *
     * @return First checkpoint that commits to the given L2 block number.
     */
    function getL2OutputAfter(uint256 _l2BlockNumber)
        external
        view
        returns (Types.OutputProposal memory)
    {
        require(
            _l2BlockNumber <= highestL2BlockNumber,
            "L2OutputOracleV2: cannot get output for block number that has not been proposed"
        );

        uint256 l2BlockNumber = _l2BlockNumber;
        Types.OutputProposal memory proposal = l2Outputs[l2BlockNumber];
        while (proposal.timestamp == 0) {
            l2BlockNumber++;
            proposal = l2Outputs[l2BlockNumber];
            require(
                l2BlockNumber <= highestL2BlockNumber,
                "L2OutputOracleV2: cannot get output for block number that has not been proposed"
            );
        }

        return l2Outputs[l2BlockNumber];
    }

    /**
     * @notice Returns the L2 timestamp corresponding to a given L2 block number.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     *
     * @return L2 timestamp of the given block.
     */
    function computeL2Timestamp(uint256 _l2BlockNumber) public view returns (uint256) {
        return startingTimestamp + ((_l2BlockNumber - startingBlockNumber) * L2_BLOCK_TIME);
    }
}
