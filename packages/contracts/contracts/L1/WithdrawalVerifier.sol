//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle } from "./L2OutputOracle.sol";
import {
    Lib_SecureMerkleTrie
} from "../../lib/optimism/packages/contracts/contracts/libraries/trie/Lib_SecureMerkleTrie.sol";


/**
 * @title WithdrawalVerifier
 */
contract WithdrawalVerifier {
    L2OutputOracle public immutable l2Oracle;
    address public immutable withdrawalsPredeploy;
    uint256 public immutable finalizationWindow;

    struct OutputRootProof {
        uint256 timestamp;
        bytes32 version;
        bytes32 stateRoot;
        bytes32 withdrawerStorageRoot;
        bytes32 latestBlockhash;
    }

    event WithdrawalVerified(
        uint256 indexed messageNonce,
        address indexed sender,
        address indexed target,
        uint256 value,
        uint256 gasLimit,
        bytes data
    );

    constructor(
        L2OutputOracle _l2Oracle,
        address _withdrawalsPredeploy,
        uint256 _finalizationWindow
    ) {
        l2Oracle = _l2Oracle;
        withdrawalsPredeploy = _withdrawalsPredeploy;
        finalizationWindow = _finalizationWindow;
    }

    function verifyWithdrawal(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes calldata _data,
        OutputRootProof calldata _outputRootProof,
        bytes calldata _withdrawalProof
    ) external returns (bool) {
        // check that the timestamp is 7 days old
        // hash _outputRootProof and compare with the outputOracle's value
        // how do I get the withdrawal root itself?
        require(_outputRootProof.timestamp <= block.timestamp - finalizationWindow, "Too soon");

        // Add a block scope to avoid stack-too-deep
        {
            bytes32 outputRoot = l2Oracle.getL2Output(_outputRootProof.timestamp);
            require(
                outputRoot ==
                    keccak256(
                        abi.encode(
                            _outputRootProof.version,
                            _outputRootProof.stateRoot,
                            _outputRootProof.withdrawerStorageRoot,
                            _outputRootProof.latestBlockhash
                        )
                    ),
                "Calculated output root does not match expected value"
            );
        }
        bytes32 withdrawalHash = keccak256(
            abi.encode(_nonce, _sender, _target, _value, _gasLimit, _data)
        );

        bytes32 storageKey = keccak256(
            abi.encode(
                withdrawalHash,
                uint256(1) // The withdrawals mapping is at the second slot in the layout
            )
        );

        bool verified = Lib_SecureMerkleTrie.verifyInclusionProof(
            abi.encodePacked(storageKey),
            hex"01",
            _withdrawalProof,
            _outputRootProof.withdrawerStorageRoot
        );

        emit WithdrawalVerified(_nonce, _sender, _target, _value, _gasLimit, _data);

        return verified;
    }
}
