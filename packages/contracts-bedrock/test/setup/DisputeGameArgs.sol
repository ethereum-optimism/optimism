// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { BytesMutator, BytesFactory } from "./ByteUtils.sol";

library DisputeGameArgs {
    using BytesMutator for bytes;

    function writeAbsolutePrestate(bytes memory _gameArgs, bytes32 newPrestateValue) internal pure {
        _gameArgs.overwriteAtOffset(0, BytesFactory.fromBytes32(newPrestateValue));
    }
}
