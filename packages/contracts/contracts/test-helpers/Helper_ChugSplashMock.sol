// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_MerkleTree } from "../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";

contract Helper_ChugSplashMock {
    enum ActionType {
        SET_CODE,
        SET_STORAGE
    }

    struct ChugSplashAction {
        ActionType actionType;
        address target;
        bytes data;
    }

    struct ChugSplashActionProof {
        uint256 actionIndex;
        bytes32[] siblings;
    }

    function validateAction(
        bytes32 _bundleHash,
        uint256 _bundleSize,
        ChugSplashAction memory _action,
        ChugSplashActionProof memory _proof
    )
        public
    {
        require(
            Lib_MerkleTree.verify(
                _bundleHash,
                keccak256(
                    abi.encode(
                        _action.actionType,
                        _action.target,
                        _action.data
                    )
                ),
                _proof.actionIndex,
                _proof.siblings,
                _bundleSize
            ),
            "ChugSplashDeployer: invalid action proof"
        );
    }
}
