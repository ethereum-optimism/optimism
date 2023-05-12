// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { Types } from "../libraries/Types.sol";
import { Semver } from "../universal/Semver.sol";

import { GameType } from "../libraries/DisputeTypes.sol";
import { GameStatus } from "../libraries/DisputeTypes.sol";

import { IDisputeGame } from "../dispute/IDisputeGame.sol";
import { IBondManager } from "../dispute/IBondManager.sol";
import { IDisputeGameFactory } from "../dispute/IDisputeGameFactory.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2OutputOracle contains an array of L2 state outputs, where each output is a
 *         commitment to the state of the L2 chain. Other contracts like the OptimismPortal use
 *         these outputs to verify information about the state of L2.
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
     * @notice Minimum time (in seconds) that must elapse before a withdrawal can be finalized.
     */
    uint256 public immutable FINALIZATION_PERIOD_SECONDS;

    /**
     * @notice The number of the first L2 block recorded in this contract.
     */
    uint256 public startingBlockNumber;

    /**
     * @notice The timestamp of the first L2 block recorded in this contract.
     */
    uint256 public startingTimestamp;

    /**
     * @notice The dispute game factory.
     */
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    /**
     * @notice The bond manager.
     */
    IBondManager public immutable BOND_MANAGER;

    /**
     * @notice Array of L2 output proposals.
     */
    Types.OutputProposal[] internal l2Outputs;


    /**
     * @notice Internal Mapping of l2BlockNumber to index in l2Outputs + 1.
     *
     * @dev We need to add 1 to the index in order to allow 0 to identify to be set.
     */
    mapping(uint256 => uint256) internal l2OutputIndices;

    /**
     * @notice Emitted when an output is proposed.
     *
     * @param outputRoot    The output root.
     * @param l2OutputIndex The index of the output in the l2Outputs array.
     * @param l2BlockNumber The L2 block number of the output root.
     * @param l1Timestamp   The L1 timestamp when proposed.
     */
    event OutputProposed(
        bytes32 indexed outputRoot,
        uint256 indexed l2OutputIndex,
        uint256 indexed l2BlockNumber,
        uint256 l1Timestamp
    );

    /**
     * @notice Emitted when outputs are deleted.
     *
     * @param prevNextOutputIndex Next L2 output index before the deletion.
     * @param newNextOutputIndex  Next L2 output index after the deletion.
     */
    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    /**
     * @custom:semver 2.0.0
     *
     * @param _l2BlockTime                  The time per L2 block, in seconds.
     * @param _startingBlockNumber          The number of the first L2 block.
     * @param _startingTimestamp            The timestamp of the first L2 block.
     * @param _finalizationPeriodSeconds    The time until an output finalizes.
     * @param _bondManager                  The bond manager to handle output proposals.
     * @param _disputeGameFactory           The dispute game factory to validate dispute game calls.
     */
    constructor(
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        uint256 _finalizationPeriodSeconds,
        IBondManager _bondManager,
        IDisputeGameFactory _disputeGameFactory
    ) Semver(2, 0, 0) {
        require(_l2BlockTime > 0, "L2OutputOracle: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;
        FINALIZATION_PERIOD_SECONDS = _finalizationPeriodSeconds;
        BOND_MANAGER = _bondManager;
        DISPUTE_GAME_FACTORY = _disputeGameFactory;

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
     * @notice Deletes an output proposal at the given output l2BlockNumber.
     *         This function should be called by the dispute game after the game is resolved.
     *
     * @param _l2BlockNumber L2 Block Number whose output to delete.
     */
    function deleteL2Outputs(uint256 _l2BlockNumber) external {
        // Validate the caller dispute game is complete
        IDisputeGame caller = IDisputeGame(msg.sender);
        IDisputeGame game = IDisputeGame(
            address(
                DISPUTE_GAME_FACTORY.games(
                    caller.gameType(),
                    caller.rootClaim(),
                    caller.extraData()
                )
            )
        );

        require(msg.sender == address(game), "L2OutputOracle: Unauthorized output deletion.");

        require(
            uint8(game.status()) == uint8(GameStatus.CHALLENGER_WINS),
            "L2OutputOracle: Game incomplete."
        );

        uint256 index = l2OutputIndices[_l2BlockNumber];
        require(index != 0, "L2OutputOracle: No output exists for the given L2 block number");

        // Do not allow deleting any outputs that have already been finalized.
        require(
            block.timestamp - l2Outputs[index - 1].timestamp < FINALIZATION_PERIOD_SECONDS,
            "L2OutputOracle: cannot delete outputs that have already been finalized"
        );

        // Delete the output root
        delete l2Outputs[index - 1];

        uint256 l2OutputsLength = l2Outputs.length;
        emit OutputsDeleted(l2OutputsLength, l2OutputsLength);
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
        require(index == 0, "L2OutputOracle: Output already exists at the given block number");

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
        BOND_MANAGER.post{ value: msg.value }({
           _bondId: bytes32(abi.encode(_l2BlockNumber)),
           _bondOwner: msg.sender,
           _minClaimHold: FINALIZATION_PERIOD_SECONDS
        });

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );

        // The index is 1-based, so we set it as the length of the array.
        l2OutputIndices[_l2BlockNumber] = l2Outputs.length;

        emit OutputProposed(_outputRoot, l2Outputs.length - 1, _l2BlockNumber, block.timestamp);
    }

    /**
     * @notice Returns an output by index.
     *
     * @param _l2OutputIndex Index of the output to return.
     *
     * @return The output at the given index.
     */
    function getL2Output(uint256 _l2OutputIndex)
        external
        view
        returns (Types.OutputProposal memory)
    {
        return l2Outputs[_l2OutputIndex];
    }

    /**
     * @notice Returns an output by block number.
     *
     * @param _l2BlockNumber L2 Block Number of the output to return.
     *
     * @return The output for the given block number.
     */
    function getL2OutputByNumber(uint256 _l2BlockNumber)
        external
        view
        returns (Types.OutputProposal memory)
    {
        uint256 index = l2OutputIndices[_l2BlockNumber];
        require(index != 0, "L2OutputOracle: No output exists for the given L2 block number.");
        return l2Outputs[index - 1];
    }

    /**
     * @notice Returns the index of the L2 output that checkpoints a given L2 block number. Uses a
     *         binary search to find the first output greater than or equal to the given block.
     *
     * @param _l2BlockNumber L2 block number to find a checkpoint for.
     *
     * @return Index of the first checkpoint that commits to the given L2 block number.
     */
    function getL2OutputIndexAfter(uint256 _l2BlockNumber) public view returns (uint256) {
        // Make sure an output for this block number has actually been proposed.
        require(
            _l2BlockNumber <= latestBlockNumber(),
            "L2OutputOracle: cannot get output for a block that has not been proposed"
        );

        // Make sure there's at least one output proposed.
        require(
            l2Outputs.length > 0,
            "L2OutputOracle: cannot get output as no outputs have been proposed yet"
        );

        // Find the output via binary search, guaranteed to exist.
        uint256 lo = 0;
        uint256 hi = l2Outputs.length;
        while (lo < hi) {
            uint256 mid = (lo + hi) / 2;
            if (l2Outputs[mid].l2BlockNumber < _l2BlockNumber) {
                lo = mid + 1;
            } else {
                hi = mid;
            }
        }

        return lo;
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
        return l2Outputs[getL2OutputIndexAfter(_l2BlockNumber)];
    }

    /**
     * @notice Returns the index of the most recent output proposal.
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
     * @notice Computes the block number of the next L2 block that needs to be checkpointed.
     *
     * @return Next L2 block number.
     */
    function nextBlockNumber() public view returns (uint256) {
        return latestBlockNumber() + 1;
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
