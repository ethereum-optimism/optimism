// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title MockSP1Verifier
/// @notice Mock SP1 verifier that always succeeds. Test only.
contract MockSP1Verifier is ISP1Verifier {
    /// @notice Returns the mock version string.
    function VERSION() external pure returns (string memory) {
        return "v0.0.0-MOCK";
    }

    /// @notice Returns a fixed sentinel hash; the mock accepts any proof bytes.
    function VERIFIER_HASH() external pure returns (bytes32) {
        return 0x5a093a2fcb46394f5cadfe55c44d4d572fad9cec7aeb38026b0278322ef07fac;
    }

    /// @notice Always succeeds (no-op).
    function verifyProof(bytes32, bytes calldata, bytes calldata) external pure { }
}
