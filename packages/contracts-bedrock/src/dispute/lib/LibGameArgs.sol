// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title LibGameArgs
/// @notice Library for decoding the game arguments used in dispute games.
library LibGameArgs {
    /// @notice Struct representing the game arguments.
    struct GameArgs {
        bytes32 absolutePrestate;
        address vm;
        address anchorStateRegistry;
        address weth;
        uint256 l2ChainId;
        address proposer;
        address challenger;
    }

    /// @notice Decodes the game arguments from a bytes array.
    /// @param _gameArgs The bytes array containing the encoded game arguments.
    function decode(bytes memory _gameArgs) internal pure returns (GameArgs memory) {
        uint256 len = _gameArgs.length;
        require(len == 164 || len == 124, "GameArgs: decode with invalid length");

        bytes32 absolutePrestate;
        address vm;
        address asr;
        address weth;
        uint256 l2ChainId;
        address proposer;
        address challenger;

        assembly {
            // skip length prefix
            let d := add(_gameArgs, 32)
            absolutePrestate := mload(d)
            vm := shr(96, mload(add(d, 32)))
            asr := shr(96, mload(add(d, 52)))
            weth := shr(96, mload(add(d, 72)))
            l2ChainId := mload(add(d, 92))
        }

        if (len == 164) {
            assembly {
                // skip length prefix
                let d := add(_gameArgs, 32)
                proposer := shr(96, mload(add(d, 124)))
                challenger := shr(96, mload(add(d, 144)))
            }
        }
        return GameArgs({
            absolutePrestate: absolutePrestate,
            vm: vm,
            anchorStateRegistry: asr,
            weth: weth,
            l2ChainId: l2ChainId,
            proposer: proposer,
            challenger: challenger
        });
    }

    function isValidPermissionlessArgs(bytes memory _args) internal pure returns (bool) {
        return _args.length == 124;
    }

    function isValidPermissionedArgs(bytes memory _args) internal pure returns (bool) {
        return _args.length == 164;
    }
}
