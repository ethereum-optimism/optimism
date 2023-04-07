// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { IDisputeGameFactory } from "../universal/IDisputeGameFactory.sol";
import { IBondManager } from "../universal/IBondManager.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { Semver } from "../universal/Semver.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2OutputOracle contains an array of L2 state outputs, where each output is a
 *         commitment to the state of the L2 chain. Other contracts like the OptimismPortal use
 *         these outputs to verify information about the state of L2.
 */
contract L2OutputOracle is Initializable, Semver {
    /**
     * @notice The time between L2 blocks in seconds. Once set, this value MUST NOT be modified.
     */
    uint256 public immutable L2_BLOCK_TIME;

    /**
     * @notice The address of the dispute game factory. Can be updated via upgrade.
     */
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

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
     * @notice Array of L2 output proposals.
     */
    Types.OutputProposal[] internal l2Outputs;

    /**
     * @notice Mapping of l2BlockNumber + 1 to the index of its output in l2Outputs.
     */
    mapping(uint256 => uint256) internal proposals;

    /**
     * @notice Emitted when an output is proposed.
     * @notice This is backwards-compatible with the previous L2OutputOracle.
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
     * @notice This is backwards-compatible with the previous L2OutputOracle.
     *
     * @param prevNextOutputNumber Next L2 output block number before the deletion.
     * @param newNextOutputNumber  Next L2 output block number after the deletion.
     */
    event OutputsDeleted(uint256 indexed prevNextOutputNumber, uint256 indexed newNextOutputNumber);

    /**
     * @custom:semver 2.0.0
     *
     * @param _l2BlockTime               The time per L2 block, in seconds.
     * @param _startingBlockNumber       The number of the first L2 block.
     * @param _startingTimestamp         The timestamp of the first L2 block.
     * @param _disputeGameFactory        The address of the dispute game factory.
     * @param _finalizationPeriodSeconds The time before an output can be finalized.
     * @param _bondManager               The bond manager.
     */
    constructor(
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _disputeGameFactory,
        uint256 _finalizationPeriodSeconds,
        IBondManager _bondManager
    ) Semver(2, 0, 0) {
        require(_l2BlockTime > 0, "L2OutputOracleV2: L2 block time must be greater than 0");

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
     * @notice Deletes all output proposals after and including the proposal that corresponds to
     *         the given output index. Only the challenger address can delete outputs.
     *
     * @param _l2BlockNumber The L2 block number of the output to delete.
     */
    // solhint-disable-next-line ordering
    function deleteL2Outputs(uint256 _l2BlockNumber) external {
        // Check that the caller is an authorized dispute game via the DGF.
        IDisputeGame game = IDisputeGame(msg.sender);
        GameType gameType = game.gameType();
        Claim rootClaim = game.rootClaim();
        bytes memory extraData = game.extraData();
        IDisputeGame fetched = DISPUTE_GAME_FACTORY.games(gameType, rootClaim, extraData);
        require(
            msg.sender == address(fetched),
            "L2OutputOracle: the caller is not an authorized challenger"
        );
        GameStatus status = fetched.status();
        require(
            fetched.status() == GameStatus.CHALLENGER_WINS.
            "L2OutputOracle: challenge game does not have the correct status"
        );

        uint256 _l2OutputIndex = proposals[_l2BlockNumber + 1];
        require(
            _l2OutputIndex != 0,
            "L2OutputOracle: there is no output for the given L2 block number"
        );

        // Do not allow deleting any outputs that have already been finalized.
        require(
            block.timestamp - l2Outputs[_l2OutputIndex].timestamp < FINALIZATION_PERIOD_SECONDS,
            "L2OutputOracle: cannot delete outputs that have already been finalized"
        );

        uint256 prevNextL2OutputIndex = nextOutputIndex();

        // Use assembly to delete the array elements because Solidity doesn't allow it.
        assembly {
            sstore(l2Outputs.slot, _l2OutputIndex)
        }

        // Remove the output
        proposals[_l2BlockNumber + 1] == 0,

        emit OutputsDeleted(prevNextL2OutputIndex, _l2OutputIndex);
    }

    /**
     * @notice Accepts an outputRoot and the timestamp of the corresponding L2 block. The timestamp
     *         must be equal to the current value returned by `nextTimestamp()` in order to be
     *         accepted.
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
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2OutputOracle: cannot propose L2 output in the future"
        );

        // Check that the output is not already proposed.
        require(
            proposals[_l2BlockNumber + 1] == 0,
            "L2OutputOracle: an output is already proposed for this L2 block number"
        );

        // Forward the bond to the bond manager.
        // This reverts if the msg.value does not satisfy the minimum bond amount.
        BOND_MANAGER.postBond{ value: msg.value }(keccak256(abi.encode(_l2BlockNumber)), msg.sender);

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

        uint256 outputIndex = nextOutputIndex();
        proposals[_l2BlockNumber + 1] = outputIndex;

        emit OutputProposed(_outputRoot, outputIndex, _l2BlockNumber, block.timestamp);

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );
    }

    /**
     * @notice Returns an output by index. Exists because Solidity's array access will return a
     *         tuple instead of a struct.
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
        return l2Outputs[getL2OutputIndexAfter(_l2BlockNumber)];
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
