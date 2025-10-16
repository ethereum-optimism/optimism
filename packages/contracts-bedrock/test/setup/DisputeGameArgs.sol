// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {ByteUtils} from "./ByteUtils.sol";

library DisputeGameArgs {
    using ByteUtils for bytes;

    function writeAbsolutePrestate(bytes memory _gameArgs, bytes32 _newPrestateValue) internal pure {
        _gameArgs.overwriteAtOffset(0, abi.encode(_newPrestateValue));
    }
}
