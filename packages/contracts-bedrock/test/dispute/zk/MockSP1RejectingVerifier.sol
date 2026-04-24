// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title MockSP1RejectingVerifier
/// @notice Mock SP1 verifier that always reverts. Test only.
contract MockSP1RejectingVerifier is ISP1Verifier {
    /// @notice Returns the mock version string.
    function VERSION() external pure returns (string memory) {
        return "v6.0.0";
    }

    /// @notice Always reverts.
    function verifyProof(bytes32, bytes calldata, bytes calldata) external pure {
        revert("SP1Verifier: invalid proof");
    }
}
