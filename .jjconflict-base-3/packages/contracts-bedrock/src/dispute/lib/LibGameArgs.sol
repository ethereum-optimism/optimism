// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { InvalidGameArgsLength } from "src/dispute/lib/Errors.sol";

/// @title LibGameArgs
/// @notice Library for decoding the game arguments used in dispute games.
library LibGameArgs {
    uint256 public constant PERMISSIONLESS_ARGS_LENGTH = 124;
    uint256 public constant PERMISSIONED_ARGS_LENGTH = 164;

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

    /// @notice Encodes the game arguments into a bytes array.
    function encode(GameArgs memory _args) internal pure returns (bytes memory) {
        if (_args.proposer == address(0) && _args.challenger == address(0)) {
            return abi.encodePacked(
                _args.absolutePrestate, _args.vm, _args.anchorStateRegistry, _args.weth, _args.l2ChainId
            );
        } else {
            return abi.encodePacked(
                _args.absolutePrestate,
                _args.vm,
                _args.anchorStateRegistry,
                _args.weth,
                _args.l2ChainId,
                _args.proposer,
                _args.challenger
            );
        }
    }

    /// @notice Decodes the game arguments from a bytes array.
    /// @param _gameArgs The bytes array containing the encoded game arguments.
    function decode(bytes memory _gameArgs) internal pure returns (GameArgs memory) {
        uint256 len = _gameArgs.length;
        if (len != PERMISSIONED_ARGS_LENGTH && len != PERMISSIONLESS_ARGS_LENGTH) {
            revert InvalidGameArgsLength();
        }

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

        if (len == PERMISSIONED_ARGS_LENGTH) {
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

    /// @notice Checks if the provided game arguments are valid for a permissionless game.
    function isValidPermissionlessArgs(bytes memory _args) internal pure returns (bool) {
        return _args.length == PERMISSIONLESS_ARGS_LENGTH;
    }

    /// @notice Checks if the provided game arguments are valid for a permissioned game.
    function isValidPermissionedArgs(bytes memory _args) internal pure returns (bool) {
        return _args.length == PERMISSIONED_ARGS_LENGTH;
    }
}
