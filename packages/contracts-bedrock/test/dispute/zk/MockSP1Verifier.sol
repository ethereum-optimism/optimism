// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title MockSP1Verifier
/// @notice Mock SP1 verifier that always succeeds. Test only.
contract MockSP1Verifier is ISP1Verifier {
    /// @notice Returns the mock version string.
    function VERSION() external pure returns (string memory) {
        return "v6.0.0";
    }

    /// @notice Always succeeds (no-op).
    function verifyProof(bytes32, bytes calldata, bytes calldata) external pure { }
}
