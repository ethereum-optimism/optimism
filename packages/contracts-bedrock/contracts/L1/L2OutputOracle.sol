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
contract L2OutputOracle is Semver, BitcoinSPV {
    uint256 public immutable L2_BLOCK_TIME;

    uint256 public startingBlockNumber;
    uint256 public startingTimestamp;

    Types.OutputProposal[] internal l2Outputs;

    event OutputProposed(
        bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp
    );

    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    constructor(uint256 _l2BlockTime) Semver(0, 1, 0) BitcoinSPV(bytes32(0)) {
        require(_l2BlockTime > 0, "L2OutputOracle: L2 block time must be greater than 0");

        L2_BLOCK_TIME = _l2BlockTime;
    }

    function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp) external {
        require(startingBlockNumber == 0);
        startingBlockNumber = _startingBlockNumber;
        _startingTimestamp = _startingTimestamp;
    }

    // TODO: Either improve Bitcoin SPV client or make `addHeaders` permissioned
    function addHeaders(bytes calldata _headers) external {
        _addHeaders(_headers);
    }

    struct PushParams {
        bytes32 _outputRoot;
        uint256 _l2BlockNumber;
        // BTC TX proof
        uint256 _btcTxRootIndex;
        bytes32[] _txMerkleProof;
        bytes _coinbaseTx;
        bytes32[] _txWitnessMerkleProof;
        uint256 _txIndex;
        bytes _postTx;
        uint256 _witnessIndex;
        bytes32 _witnessRoot;
        bytes32 _addedWitnessValue;
    }

    function pushL2Output(PushParams calldata _pushParams) external payable {
        // require(
        //     _l2BlockNumber == nextBlockNumber(),
        //     "L2OutputOracle: block number must be equal to next expected block number"
        // );

        require(
            computeL2Timestamp(_pushParams._l2BlockNumber) < block.timestamp,
            "L2OutputOracle: cannot propose L2 output in the future"
        );

        bytes32 witnessCommitment = BTCUtils.getWitnessRootFromCoinbase(_pushParams._coinbaseTx);

        // TODO: Verify that `_coinbaseTx` is actually a coinbase transaction, malicious actor could
        // mine block where coinbase TX is not the first and attests to data that is not present
        require(
            BTCUtils.isValidMerkleCoinbase(
                txRoots[_pushParams._btcTxRootIndex],
                BTCUtils.sha256d(_pushParams._coinbaseTx),
                _pushParams._txMerkleProof
            )
                && BTCUtils.sha256d_mem(abi.encode(_pushParams._witnessRoot, _pushParams._addedWitnessValue))
                    == witnessCommitment
                && BTCUtils.isValidMerkle(
                    _pushParams._witnessRoot,
                    BTCUtils.sha256d(_pushParams._postTx),
                    _pushParams._txIndex,
                    _pushParams._txWitnessMerkleProof
                ),
            "L2OutputOracle: Invalid BTC proof"
        );

        bytes memory witnessScript = BTCUtils.getInscription(_pushParams._postTx, _pushParams._witnessIndex);
        // TODO: Extract root and store

        l2Outputs.push(
            Types.OutputProposal({
                outputRoot: _pushParams._outputRoot,
                timestamp: uint128(block.timestamp),
                l2BlockNumber: uint128(_pushParams._l2BlockNumber)
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

    function nextBlockNumber() public view returns (uint256) {
        return 0;
    }
}
