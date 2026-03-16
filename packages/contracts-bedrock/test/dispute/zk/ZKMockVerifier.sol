// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title ZKMockVerifier
/// @notice A mock ZK verifier that always succeeds. Test only.
contract ZKMockVerifier is IZKVerifier {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Always succeeds (no-op).
    function verify(bytes32, bytes calldata, bytes calldata) external pure { }
}
