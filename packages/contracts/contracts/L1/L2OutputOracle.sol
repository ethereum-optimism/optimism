//SPDX-License-Identifier: MIT
pragma solidity >=0.8.10;

import { Ownable } from "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";

/**
 * @title L2OutputOracle
 */
contract L2OutputOracle is Ownable {
    event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp);

    // The interval in seconds at which checkpoints must be submitted.
    uint256 public immutable submissionInterval;

    // The time between blocks on L2.
    uint256 public immutable l2BlockTime;

    // The number of blocks in the chain before the first block in this contract.
    uint256 public immutable historicalTotalBlocks;

    // The timestamp of the first L2 block recorded in this contract.
    uint256 public immutable startingBlockTimestamp;

    // The timestamp of the most recent L2 block recorded in this contract.
    uint256 public latestBlockTimestamp;

    // A mapping from L2 timestamps to the output root for the block with that timestamp
    mapping(uint256 => bytes32) public l2Outputs;

    /**
     * Initialize the L2OutputOracle contract.
     * @param _submissionInterval The desired interval in seconds at which
     *        checkpoints must be submitted.
     * @param _l2BlockTime The desired L2 inter-block time in seconds.
     * @param _genesisL2Output The initial L2 output of the L2 chain.
     * @param _historicalTotalBlocks The number of blocks that preceding the
     *        initialization of the L2 chain.
     */
    constructor(
        uint256 _submissionInterval,
        uint256 _l2BlockTime,
        bytes32 _genesisL2Output,
        uint256 _historicalTotalBlocks,
        address sequencer
    ) {
        submissionInterval = _submissionInterval;
        l2BlockTime = _l2BlockTime;
        l2Outputs[block.timestamp] = _genesisL2Output; // solhint-disable not-rely-on-time
        historicalTotalBlocks = _historicalTotalBlocks;
        latestBlockTimestamp = block.timestamp; // solhint-disable not-rely-on-time
        startingBlockTimestamp = block.timestamp; // solhint-disable not-rely-on-time

        _transferOwnership(sequencer);
    }

    /**
     * Accepts an L2 output checkpoint and the timestamp of the corresponding L2
     * block. The timestamp must be equal to the current value returned by
     * `nextTimestamp()` in order to be accepted.
     * @param _l2Output The L2 output of the checkpoint block.
     * @param _l2timestamp The L2 block timestamp that resulted in _l2Output.
     * @param _blockhash A block hash which must be included in the current chain.
     * @param _blocknumber The block number with the specified block hash.
     */
    function appendL2Output(
        bytes32 _l2Output,
        uint256 _l2timestamp,
        bytes32 _blockhash,
        uint256 _blocknumber
    ) external payable onlyOwner {
        require(_l2timestamp < block.timestamp, "Cannot append L2 output in future");
        require(_l2timestamp == nextTimestamp(), "Timestamp not equal to next expected timestamp");
        require(_l2Output != bytes32(0), "Cannot submit empty L2 output");

        if (_blockhash != bytes32(0)) {
            // This check allows the sequencer to append an output based on a given L1 block,
            // without fear that it will be reorged out.
            // It will also revert if the blockheight provided is more than 256 blocks behind the
            // chain tip (as the hash will return as zero). This does open the door to a grieffing
            // attack in which the sequencer's submission is censored until the block is no longer
            // retrievable, if the sequencer is experiencing this attack it can simply leave out the
            // blockhash value, and delay submission until it is confident that the L1 block is
            // finalized.
            require(
                blockhash(_blocknumber) == _blockhash,
                "Blockhash does not match the hash at the expected height."
            );
        }

        l2Outputs[_l2timestamp] = _l2Output;
        latestBlockTimestamp = _l2timestamp;

        emit l2OutputAppended(_l2Output, _l2timestamp);
    }

    /**
     * Computes the timestamp of the next L2 block that needs to be
     * checkpointed.
     */
    function nextTimestamp() public view returns (uint256) {
        return latestBlockTimestamp + submissionInterval;
    }

    /**
     * Computes the L2 block number given a target L2 block timestamp.
     * @param _l2timestamp The L2 block timestamp of the target block.
     */
    function computeL2BlockNumber(uint256 _l2timestamp) external view returns (uint256) {
        require(
            _l2timestamp >= startingBlockTimestamp,
            "Timestamp prior to startingBlockTimestamp"
        );
        // If _l2timestamp == startingBlockTimestamp, then the L2BlockNumber should be
        // historicalTotalBlocks + 1
        return historicalTotalBlocks + 1 + (_l2timestamp - startingBlockTimestamp) / l2BlockTime;
    }
}
