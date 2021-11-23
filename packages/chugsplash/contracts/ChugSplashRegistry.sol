// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { ChugSplashManager } from "./ChugSplashManager.sol";

contract ChugSplashRegistry {
    mapping(string => ChugSplashManager) public registry;

    function register(string memory _name, address _owner) public {
        require(
            address(registry[_name]) == address(0),
            "ChugSplashRegistry: name already registered"
        );

        registry[_name] = new ChugSplashManager(_owner);
    }
}
