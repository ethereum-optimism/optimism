// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {Semver} from "../universal/Semver.sol";
import {Types} from "../libraries/Types.sol";
import {BitcoinSPV} from "../btc/BitcoinSPV.sol";
import {BTCUtils} from "../btc/BTCUtils.sol";

/**
 * @custom:proxied
 * @title L2OutputOracle
 * @notice The L2OutputOracle contains an array of L2 state outputs, where each output is a
 *         commitment to the state of the L2 chain. Other contracts like the OptimismPortal use
 *         these outputs to verify information about the state of L2.
 */
contract L2OutputOracle is Semver, BitcoinSPVSimple {
    uint256 public immutable L2_BLOCK_TIME;

    uint256 public startingBlockNumber;
    uint256 public startingTimestamp;

    Types.OutputProposal[] internal l2Outputs;

    event OutputProposed(
        bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp
    );

    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    constructor(uint256 _l2BlockTime, uint256 _startingBlockNumber, uint256 _startingTimestamp) Semver(0, 1, 0) {
        require(_l2BlockTime > 0, "L2OutputOracle: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;

        initialize(_startingBlockNumber, _startingTimestamp);
    }

    function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp) public initializer {
        require(
            _startingTimestamp <= block.timestamp,
            "L2OutputOracle: starting L2 timestamp must be less than current time"
        );

        startingTimestamp = _startingTimestamp;
        startingBlockNumber = _startingBlockNumber;
    }

    // TODO: Either improve Bitcoin SPV client or make `addHeaders` permissioned
    function addHeaders(bytes calldata _headers) external {
        _addHeaders(_header);
    }

    function pushL2Output(
        bytes32 _outputRoot,
        uint256 _l2BlockNumber,
        // BTC TX proof
        uint256 _btcTxRootIndex,
        bytes[] calldata _txMerkleProof,
        bytes calldata _coinbaseTx,
        bytes[] calldata _txWitnessMerkleProof,
        uint256 _txIndex,
        bytes calldata _postTx,
        bytes32 _witnessRoot,
        bytes32 _addedWitnessValue
    ) external payable {
        require(
            _l2BlockNumber == nextBlockNumber(),
            "L2OutputOracle: block number must be equal to next expected block number"
        );

        require(
            computeL2Timestamp(_l2BlockNumber) < block.timestamp,
            "L2OutputOracle: cannot propose L2 output in the future"
        );

        bytes32 witnessCommitment = BTCUtils.getWitnessRootFromCoinbase(_coinbaseTx);

        // TODO: Verify that `_coinbaseTx` is actually a coinbase transaction, malicious actor could
        // mine block where coinbase TX is not the first and attests to data that is not present
        require(
            BTCUtils.isValidMerkleCoinbase(txRoots[_btcTxRootIndex], sha256(sha256(_coinbaseTx)), _txMerkleProof)
                && sha256(sha256(abi.encode(_witnessRoot, _addedWitnessValue))) == witnessCommitment
                && BTCUtils.isValidMerkle(_witnessRoot, sha256(sha256(_postTx)), _txIndex, _txWitnessMerkleProof),
            "L2OutputOracle: Invalid BTC proof"
        );

        emit OutputProposed(_outputRoot, nextOutputIndex(), _l2BlockNumber, block.timestamp);

        bytes memory witnessScript = BTCUtils.getWitnessScript(_postTx);
        // TODO: Extract root and store

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_l2BlockNumber)
            })
        );
    }

    function getL2Output(uint256 _l2OutputIndex) external view returns (Types.OutputProposal memory) {
        return l2Outputs[_l2OutputIndex];
    }

    function latestOutputIndex() external view returns (uint256) {
        return l2Outputs.length - 1;
    }

    function nextOutputIndex() public view returns (uint256) {
        return l2Outputs.length;
    }

    function latestBlockNumber() public view returns (uint256) {
        return l2Outputs.length == 0 ? startingBlockNumber : l2Outputs[l2Outputs.length - 1].l2BlockNumber;
    }

    function computeL2Timestamp(uint256 _l2BlockNumber) public view returns (uint256) {
        return startingTimestamp + ((_l2BlockNumber - startingBlockNumber) * L2_BLOCK_TIME);
    }
}
