// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title LibGameArgs
/// @notice Library for decoding the game arguments used in dispute games.
library LibGameArgs {
    /// @notice Decodes the game arguments from a bytes array.
    /// @param _gameArgs The bytes array containing the encoded game arguments.
    function decode(bytes memory _gameArgs)
        internal
        pure
        returns (
            bytes32 absolutePrestate_,
            address vm_,
            address asr_,
            address weth_,
            uint256 l2ChainId_,
            address proposer_,
            address challenger_
        )
    {
        uint256 len = _gameArgs.length;
        require(len == 164 || len == 124, "GameArgs: decode with invalid length");
        assembly {
            let d := add(_gameArgs, 32)
            absolutePrestate_ := mload(d)
            vm_ := shr(96, mload(add(d, 32)))
            asr_ := shr(96, mload(add(d, 52)))
            weth_ := shr(96, mload(add(d, 72)))
            l2ChainId_ := mload(add(d, 92))
        }

        if (len == 164) {
            assembly {
                let d := add(_gameArgs, 32)
                proposer_ := shr(96, mload(add(d, 124)))
                challenger_ := shr(96, mload(add(d, 144)))
            }
        }
    }
}
