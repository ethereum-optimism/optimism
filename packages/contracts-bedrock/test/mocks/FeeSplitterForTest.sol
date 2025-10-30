// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Contracts
import { FeeSplitter } from "src/L2/FeeSplitter.sol";

/// @title FeeSplitterForTest
/// @notice Test contract for the FeeSplitter contract.
/// @dev Makes the setTransientDisbursingAddress function public for testing purposes.
contract FeeSplitterForTest is FeeSplitter {
    function setTransientDisbursingAddress(address _allowedCaller) external {
        _setTransientDisbursingAddress(_allowedCaller);
    }
}
