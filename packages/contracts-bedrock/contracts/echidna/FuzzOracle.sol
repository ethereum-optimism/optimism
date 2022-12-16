pragma solidity 0.8.15;

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { Types } from "../libraries/Types.sol";

/**
 * @dev Modified version of `L2OutputOracle` that contains a less restrictive
 * version `proposeL2Output` for use with echidna.
 */
contract ModL2OutputOracle is L2OutputOracle {
    constructor(
        uint256 _submissionInterval,
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp,
        address _proposer,
        address _challenger
    )
        L2OutputOracle(
            _submissionInterval,
            _l2BlockTime,
            _startingBlockNumber,
            _startingTimestamp,
            _proposer,
            _challenger
        )
    {
        // ...
    }

    /**
     * @dev This function is a modified version of `proposeL2Output` that removes
     * several checks which limit echidna.
     *
     * Because the purpose of this test is to check that the block number of the
     * proposed outputs must monotonically increase, these checks are safe to remove.
     */
    function proposeLessChecks(bytes32 _outputRoot, uint256 _l2BlockNumber) external {
        require(
            _l2BlockNumber == nextBlockNumber(),
            "L2OutputOracle: block number must be equal to next expected block number"
        );

        emit OutputProposed(_outputRoot, nextOutputIndex(), block.timestamp, _l2BlockNumber);

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );
    }
}

contract EchidnaFuzzL2OutputOracle {
    // ENV
    uint256 internal constant submissionInterval = 1800;
    uint256 internal constant l2BlockTime = 2;
    uint256 internal constant startingBlockNumber = 200;
    uint256 internal constant startingTimestamp = 1000;
    ModL2OutputOracle public immutable oracle;

    // STATE
    bool internal blockNumberMonotonicallyIncreases = true;

    constructor() {
        // Create a new modified L2 Output Oracle
        oracle = new ModL2OutputOracle(
            submissionInterval,
            l2BlockTime,
            startingBlockNumber,
            startingTimestamp,
            address(this),
            address(this)
        );
    }

    /**
     * @dev Propose an l2 output root using `ModL2OutputOracle`'s less restrictive
     * `proposeLessChecks` function.
     */
    function propose(bytes32 _outputRoot, uint256 _l2BlockNumber) external {
        // Grab the previous latest block number
        uint256 previousLatestBlock = oracle.latestBlockNumber();

        // Attempt to propose a new output
        oracle.proposeLessChecks(_outputRoot, _l2BlockNumber);

        // Ensure that the block number monotonically increased
        if (oracle.latestBlockNumber() < previousLatestBlock) {
            blockNumberMonotonicallyIncreases = false;
        }
    }

    /**
     * @dev Delete one or multiple L2 outputs from the L2 Output Oracle.
     */
    function deleteL2Output(uint256 _index) external {
        oracle.deleteL2Outputs(_index);
    }

    /**
     * Invariant: The block number of output root proposals should monotonically increase.
     */
    function echidna_blockNumberMonotonicallyIncreases() public view returns (bool) {
        return blockNumberMonotonicallyIncreases;
    }
}
