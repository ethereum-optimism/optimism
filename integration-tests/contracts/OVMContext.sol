// SPDX-License-Identifier: MIT

/*

MIT License

Copyright (c) 2018 Maker Foundation

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

*/

pragma solidity ^0.8.9;

// Can't do this until the package is published.
//import { iOVM_L1BlockNumber } from "@eth-optimism/contracts/iOVM_L1BlockNumber";

interface iOVM_L1BlockNumber {
    function getL1BlockNumber() external view returns (uint256);
}

/// @title OVMContext - Helper Functions
contract OVMContext {
    function getCurrentBlockTimestamp() public view returns (uint256 timestamp) {
        timestamp = block.timestamp;
    }

    function getCurrentL1BlockNumber() public view returns (uint256) {
        return iOVM_L1BlockNumber(
            0x4200000000000000000000000000000000000013
        ).getL1BlockNumber();
    }

    function getCurrentBlockNumber() public view returns (uint256) {
        return block.number;
    }

    function getChainID() external view returns (uint256) {
        uint256 id;
        assembly {
            id := chainid()
        }
        return id;
    }
}
