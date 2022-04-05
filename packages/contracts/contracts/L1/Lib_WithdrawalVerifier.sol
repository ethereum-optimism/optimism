//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { L2OutputOracle } from "./L2OutputOracle.sol";
import {
    Lib_SecureMerkleTrie
} from "../../lib/optimism/packages/contracts/contracts/libraries/trie/Lib_SecureMerkleTrie.sol";

/**
 * @title WithdrawalVerifier
 */
library WithdrawalVerifier {
    struct OutputRootProof {
        uint256 timestamp;
        bytes32 version;
        bytes32 stateRoot;
        bytes32 withdrawerStorageRoot;
        bytes32 latestBlockhash;
    }

    function _verifyWithdrawerStorageRoot(
        bytes32 _outputRoot,
        OutputRootProof calldata _outputRootProof
    ) internal pure returns (bool) {
        return
            _outputRoot ==
            keccak256(
                abi.encode(
                    _outputRootProof.version,
                    _outputRootProof.stateRoot,
                    _outputRootProof.withdrawerStorageRoot,
                    _outputRootProof.latestBlockhash
                )
            );
    }

    function _verifyWithdrawalInclusion(
        bytes32 _withdrawalHash,
        bytes32 _withdrawerStorageRoot,
        bytes calldata _withdrawalProof
    ) internal pure returns (bool) {
        bytes32 storageKey = keccak256(
            abi.encode(
                _withdrawalHash,
                uint256(1) // The withdrawals mapping is at the second slot in the layout
            )
        );

        return
            Lib_SecureMerkleTrie.verifyInclusionProof(
                abi.encodePacked(storageKey),
                hex"01",
                _withdrawalProof,
                _withdrawerStorageRoot
            );
    }
}
