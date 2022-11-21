// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    OwnableUpgradeable
} from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";
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
contract L2OutputOracle is OwnableUpgradeable, Semver {
    /**
     * @notice The interval in L2 blocks at which checkpoints must be submitted.
     */
    uint256 public immutable SUBMISSION_INTERVAL;

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
     * @notice The address of the proposer;
     */
    address public proposer;

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
     * @notice Emitted when the proposer address is changed.
     *
     * @param previousProposer The previous proposer address.
     * @param newProposer      The new proposer address.
     */
    event ProposerChanged(address indexed previousProposer, address indexed newProposer);

    /**
     * @notice Reverts if called by any account other than the proposer.
     */
    modifier onlyProposer() {
        require(proposer == msg.sender, "L2OutputOracle: function can only be called by proposer");
        _;
    }

    /**
     * @custom:semver 0.0.1
     *
     * @param _submissionInterval    Interval in blocks at which checkpoints must be submitted.
     * @param _genesisL2Output       The initial L2 output of the L2 chain.
     * @param _startingBlockNumber   The number of the first L2 block.
     * @param _startingTimestamp     The timestamp of the first L2 block.
     * @param _l2BlockTime           The time per L2 block, in seconds.
     * @param _proposer              The address of the proposer.
     * @param _owner                 The address of the owner.
     */
    constructor(
        uint256 _submissionInterval,
        bytes32 _genesisL2Output,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        uint256 _l2BlockTime,
        address _proposer,
        address _owner
    ) Semver(0, 0, 1) {
        require(
            _startingTimestamp <= block.timestamp,
            "L2OutputOracle: starting L2 timestamp must be less than current time"
        );

        SUBMISSION_INTERVAL = _submissionInterval;
        STARTING_BLOCK_NUMBER = _startingBlockNumber;
        STARTING_TIMESTAMP = _startingTimestamp;
        L2_BLOCK_TIME = _l2BlockTime;

        initialize(_genesisL2Output, _proposer, _owner);
    }

    /**
     * @notice Initializer.
     *
     * @param _genesisL2Output     The initial L2 output of the L2 chain.
     * @param _proposer            The address of the proposer.
     * @param _owner               The address of the owner.
     */
    function initialize(
        bytes32 _genesisL2Output,
        address _proposer,
        address _owner
    ) public initializer {
        require(_proposer != _owner, "L2OutputOracle: proposer cannot be the same as the owner");
        l2Outputs[STARTING_BLOCK_NUMBER] = Types.OutputProposal(_genesisL2Output, block.timestamp);
        latestBlockNumber = STARTING_BLOCK_NUMBER;
        __Ownable_init();
        changeProposer(_proposer);
        _transferOwnership(_owner);
    }

    /**
     * @notice Deletes all output proposals after and including the proposal that corresponds to
     *         the given block number. Can only be called by the owner, but will be replaced with
     *         a mechanism that allows a challenger contract to delete proposals.
     *
     * @param _l2BlockNumber L2 block number of the first output root to delete.
     */
    // solhint-disable-next-line ordering
    function deleteL2Outputs(uint256 _l2BlockNumber) external onlyOwner {
        // Simple check that accomplishes two things:
        //   1. Prevents deleting anything from before the genesis block.
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
    ) external payable onlyProposer {
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

        l2Outputs[_l2BlockNumber] = Types.OutputProposal(_outputRoot, block.timestamp);
        latestBlockNumber = _l2BlockNumber;

        emit OutputProposed(_outputRoot, block.timestamp, _l2BlockNumber);
    }

    /**
     * @notice Returns the L2 output proposal associated with a target L2 block number. If the
     *         L2 block number provided is between checkpoints, this function will rerutn the next
     *         proposal for the next checkpoint.
     *         Reverts if the output proposal is either not found, or predates
     *         the STARTING_BLOCK_NUMBER.
     *
     * @param _l2BlockNumber The L2 block number of the target block.
     */
    function getL2Output(uint256 _l2BlockNumber)
        external
        view
        returns (Types.OutputProposal memory)
    {
        require(
            _l2BlockNumber >= STARTING_BLOCK_NUMBER,
            "L2OutputOracle: block number cannot be less than the starting block number"
        );

        require(
            _l2BlockNumber <= latestBlockNumber,
            "L2OutputOracle: block number cannot be greater than the latest block number"
        );

        // Find the distance between _l2BlockNumber, and the checkpoint block before it.
        uint256 offset = (_l2BlockNumber - STARTING_BLOCK_NUMBER) % SUBMISSION_INTERVAL;

        // If the offset is zero, then the _l2BlockNumber should be checkpointed.
        // Otherwise, we'll look up the next block that will be checkpointed.
        uint256 lookupBlockNumber = offset == 0
            ? _l2BlockNumber
            : _l2BlockNumber + (SUBMISSION_INTERVAL - offset);

        Types.OutputProposal memory output = l2Outputs[lookupBlockNumber];
        require(
            output.outputRoot != bytes32(0),
            "L2OutputOracle: no output found for that block number"
        );

        return output;
    }

    /**
     * @notice Overrides the standard implementation of transferOwnership
     *         to add the requirement that the owner and proposer are distinct.
     *         Can only be called by the current owner.
     */
    function transferOwnership(address _newOwner) public override onlyOwner {
        require(_newOwner != proposer, "L2OutputOracle: owner cannot be the same as the proposer");
        super.transferOwnership(_newOwner);
    }

    /**
     * @notice Transfers the proposer role to a new account (`newProposer`).
     *         Can only be called by the current owner.
     */
    function changeProposer(address _newProposer) public onlyOwner {
        require(
            _newProposer != address(0),
            "L2OutputOracle: new proposer cannot be the zero address"
        );

        require(
            _newProposer != owner(),
            "L2OutputOracle: proposer cannot be the same as the owner"
        );

        emit ProposerChanged(proposer, _newProposer);
        proposer = _newProposer;
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
        require(
            _l2BlockNumber >= STARTING_BLOCK_NUMBER,
            "L2OutputOracle: block number must be greater than or equal to starting block number"
        );

        return STARTING_TIMESTAMP + ((_l2BlockNumber - STARTING_BLOCK_NUMBER) * L2_BLOCK_TIME);
    }
}
