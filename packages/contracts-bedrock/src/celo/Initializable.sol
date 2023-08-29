// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity ^0.8.15;

contract Initializable {
    bool public initialized;

    modifier initializer() {
        require(!initialized, "contract already initialized");
        initialized = true;
        _;
    }

    constructor(bool testingDeployment) {
        if (!testingDeployment) {
            initialized = true;
        }
    }
}
