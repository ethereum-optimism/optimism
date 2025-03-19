// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IExecutePrecompile } from "interfaces/L1/IExecutePrecompile.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
//import { Semver } from "src/universal/Semver.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Types } from "src/libraries/Types.sol";

/**
 * @custom:proxied
 * @title L2NativeRollup
 * @notice The L2NativeRollup contains information required to keep 
 *         commitment to the state of the L2 chain. Other contracts like the OptimismPortal use
 *         these outputs to verify information about the state of L2.
 */
contract L2NativeRollup is Initializable, ISemver {

    address constant PRECOMPILE_ADDR = 0x0000000000000000000000000000000000000012;

    /**
     * @notice Semantic version.
     * @custom:semver 1.0.1
     */
    string public constant version = "1.0.1";

    /**
     * @notice The time between L2 blocks in seconds. Once set, this value MUST NOT be modified.
     */
    uint256 public immutable L2_BLOCK_TIME;

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
     * @custom:semver 1.3.0
     *
     * @param _l2BlockTime         The time per L2 block, in seconds.
     * @param _startingBlockNumber The number of the first L2 block.
     * @param _startingTimestamp   The timestamp of the first L2 block.
     */
    constructor(
        uint256 _l2BlockTime,
        uint256 _startingBlockNumber,
        uint256 _startingTimestamp
    ) {
        require(_l2BlockTime > 0, "L2NativeRollup: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;

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
            "L2NativeRollup: starting L2 timestamp must be less than current time"
        );

        startingTimestamp = _startingTimestamp;
        startingBlockNumber = _startingBlockNumber;
    }

    /**
     * @notice Accepts a proposed L2 block and verifies its execution.
     *
     * @param _outputRoot    The L2 output root, defined as: 
     *                       output_root = keccak256(version_byte || output_payload), where
     *                       output_payload = state_root || withdrawal_storage_root ||
     *                       latest_block_hash.
     * @param _payload       The payload contains a preimage of the output root, the L2 block
     *                       header, a list of L2 transactions, and witness data required
     *                       to replay those transactions and derive a post state root.
     * @param _preStateRoot  The state root of the L2 chain before executing the block
     * @param _postStateRoot The state root of the L2 chain after executing the block
     * @param _gasUsed       The total gas used during execution of the block
     */
    function proposeL2Blocks(
        bytes32 _outputRoot,
        uint256 _l2BlockNumber,
        bytes memory _payload,
        bytes32 _preStateRoot,
        bytes32 _postStateRoot,
        uint256 _gasUsed
    ) external payable {

        require(
            _l2BlockNumber == nextBlockNumber(),
            "L2NativeRollup: block number must be equal to next expected block number"
        );

        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2NativeRollup: cannot propose L2 output in the future"
        );

        // Prepend output root and block number to payload
        bytes memory modifiedPayload = abi.encodePacked(_outputRoot, _l2BlockNumber, _payload);

        bool result = IExecutePrecompile(PRECOMPILE_ADDR).execute(
            modifiedPayload,
            _preStateRoot,
            _postStateRoot,
            _gasUsed
        );

        require(result, "L2NativeRollup: precompile execution failed");

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );
    }

    /**
     * @notice Returns the expected block number of the next L2 proposal
     *
     * @return Next L2 block number.
     */
    function nextBlockNumber() public view returns (uint256) {
        return
            l2Outputs.length == 0
                ? startingBlockNumber + 1
                : l2Outputs[l2Outputs.length - 1].l2BlockNumber + 1;
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
