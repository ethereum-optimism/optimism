// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// @notice StorageLibrary is an example library used for integration testing.
library StorageLibrary {

    function addData(uint256 _data) internal pure returns (uint256) {
        return _data + 123;
    }

}

