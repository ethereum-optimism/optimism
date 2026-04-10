// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IRiscZeroVerifier } from "interfaces/vendor/IRiscZeroVerifier.sol";

/// @title MockRiscZeroVerifier
/// @notice Mock RISC Zero verifier that always succeeds. Test only.
contract MockRiscZeroVerifier is IRiscZeroVerifier {
    /// @notice Returns the mock version string.
    function VERSION() external pure returns (string memory) {
        return "1.2.0";
    }

    /// @notice Always succeeds (no-op).
    function verify(bytes calldata, bytes32, bytes32) external pure { }
}
