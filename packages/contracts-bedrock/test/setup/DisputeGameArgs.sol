// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ByteUtils } from "./ByteUtils.sol";

library DisputeGameArgs {
    using ByteUtils for bytes;

    function writeAbsolutePrestate(bytes memory _gameArgs, bytes32 _newValue) internal pure {
        _gameArgs.overwriteAtOffset(0, abi.encodePacked(_newValue));
    }

    function writeVM(bytes memory _gameArgs, address _newValue) internal pure {
        _gameArgs.overwriteAtOffset(32, abi.encodePacked(_newValue));
    }

    function writeASR(bytes memory _gameArgs, address _newValue) internal pure {
        _gameArgs.overwriteAtOffset(52, abi.encodePacked(_newValue));
    }

    function writeWeth(bytes memory _gameArgs, address _newValue) internal pure {
        _gameArgs.overwriteAtOffset(72, abi.encodePacked(_newValue));
    }

    function writeL2ChainId(bytes memory _gameArgs, uint256 _newValue) internal pure {
        _gameArgs.overwriteAtOffset(92, abi.encodePacked(_newValue));
    }

    function writeProposer(bytes memory _gameArgs, address _newValue) internal pure {
        _gameArgs.overwriteAtOffset(124, abi.encodePacked(_newValue));
    }

    function writeChallenger(bytes memory _gameArgs, address _newValue) internal pure {
        _gameArgs.overwriteAtOffset(144, abi.encodePacked(_newValue));
    }
}
