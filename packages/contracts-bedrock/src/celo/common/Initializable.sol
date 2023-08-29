// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

contract Initializable {
    bool public initialized;

    constructor(bool testingDeployment) {
        if (!testingDeployment) {
            initialized = true;
        }
    }

    modifier initializer() {
        require(!initialized, "contract already initialized");
        initialized = true;
        _;
    }
}
