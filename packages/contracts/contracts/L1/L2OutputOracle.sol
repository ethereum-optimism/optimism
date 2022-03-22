//SPDX-License-Identifier: MIT
pragma solidity >=0.8.10;

import { Ownable } from "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";

/**
 * @title L2OutputOracle
 */
contract L2OutputOracle is Ownable {

    event l2OutputAppended(
        bytes32 indexed _l2Output,
        uint256 indexed _timestamp
    );

    uint256 public submissionInterval;
    uint256 public l2BlockTime;
    mapping(uint256 => bytes32) public l2Outputs;
    uint256 public historicalTotalBlocks;
    uint256 public latestBlockTimestamp;
    uint256 public startingBlockTimestamp;

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
     * @param _timestamp The L2 block timestamp that resulted in _l2Output.
     */
    function appendL2Output(bytes32 _l2Output, uint256 _timestamp) external payable onlyOwner {
        // todo: separate owner and sequencer roles
        require(block.timestamp > _timestamp, "Cannot append L2 output in future");
        require(_l2Output != bytes32(0), "Cannot submit empty L2 output");
        require(_timestamp == nextTimestamp(), "Timestamp not equal to next expected timestamp");
        // todo: add require statement to ensure a specific prev-hash exists on the current chain
        l2Outputs[_timestamp] = _l2Output;
        latestBlockTimestamp = _timestamp;

        emit l2OutputAppended(_l2Output, _timestamp);
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
     * @param _timestamp The L2 block timestamp of the target block.
     */
    function computeL2BlockNumber(uint256 _timestamp) external view returns (uint256) {
        require(_timestamp >= startingBlockTimestamp, "timestamp prior to startingBlockTimestamp");
        return historicalTotalBlocks + (_timestamp - startingBlockTimestamp) / l2BlockTime;
    }
}
