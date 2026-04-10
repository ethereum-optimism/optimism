// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IRiscZeroVerifier } from "interfaces/vendor/IRiscZeroVerifier.sol";

/// @title MockRiscZeroRejectingVerifier
/// @notice Mock RISC Zero verifier that always reverts. Test only.
contract MockRiscZeroRejectingVerifier is IRiscZeroVerifier {
    /// @notice Returns the mock version string.
    function VERSION() external pure returns (string memory) {
        return "1.2.0";
    }

    /// @notice Always reverts.
    function verify(bytes calldata, bytes32, bytes32) external pure {
        revert("RiscZeroVerifier: invalid proof");
    }
}
