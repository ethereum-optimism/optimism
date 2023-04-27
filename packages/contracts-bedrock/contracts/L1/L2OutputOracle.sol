// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { IBondManager } from "@dispute/interfaces/IBondManager.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { Semver } from "../universal/Semver.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2OutputOracle contains a map of L2 block numbers to their associated
 *         L2 state outputs, where each output is a commitment to the state of the L2 chain.
 *         Other contracts like the OptimismPortal use these outputs to verify information
 *         about the state of L2.
 */
contract L2OutputOracle is Initializable, Semver {
    /**
     * @notice The amount that must be posted as a bond for proposing an output.
     */
    uint256 public constant OUTPUT_BOND_COST = 1 ether;

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
     * @notice The Oracle Bond Manager.
     */
    IBondManager public immutable BOND_MANAGER;

    /**
     * @notice The number of the first L2 block recorded in this contract.
     */
    uint256 public startingBlockNumber;

    /**
     * @notice The timestamp of the first L2 block recorded in this contract.
     */
    uint256 public startingTimestamp;

    /**
     * @notice Map of L2 output proposals.
     */
    Types.OutputProposal[] internal l2Outputs;

    /**
     * @notice Map of l2 block numbers to their index.
     * @dev Note that the `Types.OutputProposal` at the given index may be zeroed out if deleted.
     */
    mapping(uint256 => uint256) internal l2OutputIndices;

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
     * @notice Emitted when outputs are deleted.
     * @notice This keeps backwards-compatibility with the L2OutputOracle.
     *
     * @param prevNextOutputIndex Next output index before the deletion.
     * @param newNextOutputIndex  Next output index after the deletion.
     */
    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    /**
     * @custom:semver 2.0.0
     *
     * @param _l2BlockTime               The time per L2 block, in seconds.
     * @param _startingBlockNumber       The number of the first L2 block.
     * @param _startingTimestamp         The timestamp of the first L2 block.
     * @param _challenger                The address of the challenger.
     * @param _finalizationPeriodSeconds The time before an output can be finalized.
     * @param _bondManager               The bond manager.
     */
    constructor(
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _challenger,
        uint256 _finalizationPeriodSeconds,
        IBondManager _bondManager
    ) Semver(2, 0, 0) {
        require(_l2BlockTime > 0, "L2OutputOracle: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;
        CHALLENGER = _challenger;
        FINALIZATION_PERIOD_SECONDS = _finalizationPeriodSeconds;
        BOND_MANAGER = _bondManager;

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
    }

    /**
     * @notice Deletes an output proposal based on the given L2 block number.
     *
     * @param _l2BlockNumber The L2 block number of the output to delete.
     */
    // solhint-disable-next-line ordering
    function deleteL2Output(uint256 _l2BlockNumber) external {
        require(
            // TODO: The challenger here should be the dispute game factory.
            // TODO: the dispute games themselves could then call through the factory to delete their rootClaim.
            msg.sender == CHALLENGER,
            "L2OutputOracle: only the challenger address can delete an output"
        );

        // Do not allow deleting any outputs that have already been finalized.
        uint256 index = l2OutputIndices[_l2BlockNumber];
        require(
            block.timestamp - l2Outputs[index].timestamp < FINALIZATION_PERIOD_SECONDS,
            "L2OutputOracle: cannot delete outputs that have already been finalized"
        );

        // Delete the output root
        delete l2Outputs[index];

        uint256 len = l2Outputs.length;
        emit OutputsDeleted(len, len);
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
            msg.value >= OUTPUT_BOND_COST,
            "L2OutputOracle: minimum proposal cost not provided"
        );

        uint256 index = l2OutputIndices[_l2BlockNumber];
        if (index < l2Outputs.length) {
            require(
                l2Outputs[index].timestamp == 0,
                "L2OutputOracle: output already proposed"
            );
        }

        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2OutputOracle: cannot propose L2 output in the future"
        );

        require(
            _outputRoot != bytes32(0),
            "L2OutputOracle: L2 output proposal cannot be the zero hash"
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
                "L2OutputOracle: block hash does not match the hash at the expected height"
            );
        }

        // Post the bond to the bond manager
        BOND_MANAGER.post{ value: msg.value }(keccak256(abi.encode(_l2BlockNumber)), msg.sender, uint256(10 days));

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );
        l2OutputIndices[_l2BlockNumber] = l2Outputs.length - 1;

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
        Types.OutputProposal memory proposal = l2Outputs[l2OutputIndices[_l2BlockNumber]];
        if (proposal.l2BlockNumber != _l2BlockNumber) {
            return Types.OutputProposal({
                outputRoot: bytes32(0),
                timestamp: 0,
                l2BlockNumber: 0
            });
        }
        return proposal;
    }

    /**
     * @notice Returns the L2 output proposal that checkpoints a given L2 block number.
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
        uint256 nextIndex = l2OutputIndices[_l2BlockNumber] + 1;
        Types.OutputProposal memory proposal = l2Outputs[nextIndex];
        require(
            proposal.l2BlockNumber > _l2BlockNumber,
            "L2OutputOracle: cannot get output for block number that has not been proposed"
        );
        return proposal;
    }

    /**
     * @notice Returns the number of outputs that have been proposed. Will revert if no outputs
     *         have been proposed yet.
     *
     * @return The number of outputs that have been proposed.
     */
    function latestOutputIndex() external view returns (uint256) {
        return l2Outputs.length - 1;
    }

    /**
     * @notice Returns the index of the next output to be proposed.
     *
     * @return The index of the next output to be proposed.
     */
    function nextOutputIndex() public view returns (uint256) {
        return l2Outputs.length;
    }

    /**
     * @notice Returns the block number of the latest submitted L2 output proposal. If no proposals
     *         been submitted yet then this function will return the starting block number.
     *
     * @return Latest submitted L2 block number.
     */
    function latestBlockNumber() public view returns (uint256) {
        return
            l2Outputs.length == 0
                ? startingBlockNumber
                : l2Outputs[l2Outputs.length - 1].l2BlockNumber;
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
