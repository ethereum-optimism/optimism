// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { IBondManager } from "../universal/IBondManager.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { Semver } from "../universal/Semver.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2OutputOracleV2
 * @notice The L2OutputOracleV2 contains a map of L2 block numbers to their associated
 *         L2 state outputs, where each output is a commitment to the state of the L2 chain.
 *         Other contracts like the OptimismPortal use these outputs to verify information
 *         about the state of L2.
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
     * @notice Highest L2 block number that has been proposed.
     */
    uint256 public highestL2BlockNumber;

    /**
     * @notice Map of L2 output proposals.
     */
    mapping(uint256 => Types.OutputProposal) internal l2Outputs;

    /**
     * @notice Map of bonds that can be claimed once finalized.
     * @notice If the address is address(0), the bond has been seized.
     */
    mapping(uint256 => address) internal bonds;

    /**
     * @notice Linked list of previous highest block numbers.
     * @dev This maps a block number to the previous highest L2 block number.
     */
    mapping(uint256 => uint256) internal previousBlockNumber;

    /**
     * @notice Linked list of next highest block numbers.
     * @dev This maps a block number to the next highest L2 block number.
     */
    mapping(uint256 => uint256) internal nextBlockNumber;

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
        require(_l2BlockTime > 0, "L2OutputOracleV2: L2 block time must be greater than 0");

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
            "L2OutputOracleV2: starting L2 timestamp must be less than current time"
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
            msg.sender == CHALLENGER,
            "L2OutputOracleV2: only the challenger address can delete an output"
        );

        // Do not allow deleting any outputs that have already been finalized.
        require(
            block.timestamp - l2Outputs[_l2BlockNumber].timestamp < FINALIZATION_PERIOD_SECONDS,
            "L2OutputOracleV2: cannot delete outputs that have already been finalized"
        );

        // Join the linked list
        uint256 nextL2Block = nextBlockNumber[_l2BlockNumber];
        uint256 prevL2Block = previousBlockNumber[_l2BlockNumber];
        previousBlockNumber[nextL2Block] = prevL2Block;
        nextBlockNumber[prevL2Block] = nextL2Block;
        delete nextBlockNumber[_l2BlockNumber];
        delete previousBlockNumber[_l2BlockNumber];

        // Retrieve the previous high L2 block number
        if (_l2BlockNumber == highestL2BlockNumber) {
            highestL2BlockNumber = prevL2Block;
        }

        // Remove the associated output state
        delete l2Outputs[_l2BlockNumber];
        uint256 amount = BOND_MANAGER.call(keccak256(abi.encode(_l2BlockNumber)), msg.sender);
        delete bonds[_l2BlockNumber];

        emit OutputsDeleted(nextL2Block, _l2BlockNumber);
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
            msg.value >= BOND_MANAGER.next(),
            "L2OutputOracleV2: minimum proposal cost not provided"
        );

        require(
            l2Outputs[_l2BlockNumber].timestamp == 0,
            "L2OutputOracleV2: output already proposed"
        );

        require(
            bonds[_l2BlockNumber] == address(0),
            "L2OutputOracleV2: bond already posted for this output"
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

        // Get the L2 block number for which we have an output proposal just before this block
        // Need to walk backwards in the doubly linked list
        uint256 prevBlockNum = highestL2BlockNumber;
        while (prevBlockNum > _l2BlockNumber) {
            prevBlockNum = previousBlockNumber[prevBlockNum];
        }

        // Insert the block by placing in the doubly linked list
        uint256 updatePrevNext = nextBlockNumber[prevBlockNum];
        if (updatePrevNext != 0) {
            previousBlockNumber[updatePrevNext] = _l2BlockNumber;
        }
        nextBlockNumber[prevBlockNum] = _l2BlockNumber;
        nextBlockNumber[_l2BlockNumber] = updatePrevNext;
        previousBlockNumber[_l2BlockNumber] = prevBlockNum;

        // Update the highest L2 block number if necessary.
        if (_l2BlockNumber > highestL2BlockNumber) {
            highestL2BlockNumber = _l2BlockNumber;
        }

        // Post the bond to the bond manager
        BOND_MANAGER.post{ value: msg.value }(keccak256(abi.encode(_l2BlockNumber)));
        bonds[_l2BlockNumber] = msg.sender;

        // Set the output
        l2Outputs[_l2BlockNumber] = Types.OutputProposal({
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
     * @notice Returns the proposer for a given L2 block number.
     *
     * @param _l2BlockNumber L2 block number of the output.
     *
     * @return The address that proposed the given output at an L2 block number.
     */
    function getProposer(uint256 _l2BlockNumber) external view returns (address) {
        return bonds[_l2BlockNumber];
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
