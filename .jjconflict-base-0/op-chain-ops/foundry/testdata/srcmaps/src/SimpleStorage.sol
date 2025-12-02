// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {StorageLibrary} from "./StorageLibrary.sol";

// @notice SimpleStorage is a contract to test Go <> foundry integration.
// @dev uses a dependency, to test source-mapping with multiple sources.
contract SimpleStorage {

    // @dev example getter
    function getExampleData() public pure returns (uint256) {
        return StorageLibrary.addData(42);
    }
}
