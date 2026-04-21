// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title ZKRejectingVerifier
/// @notice A mock ZK verifier that always reverts. Test only.
contract ZKRejectingVerifier is IZKVerifier {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Returns the verifier type identifier.
    function verifierType() external pure returns (string memory) {
        return "ZKRejectingVerifier";
    }

    /// @notice Always reverts.
    function verify(bytes32, bytes calldata, bytes calldata) external pure {
        revert("ZKRejectingVerifier: invalid proof");
    }
}
