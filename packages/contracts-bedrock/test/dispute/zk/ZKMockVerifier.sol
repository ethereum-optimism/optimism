// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IZKVerifier } from "src/dispute/zk/IZKVerifier.sol";

/// @title ZKMockVerifier
/// @notice A mock ZK verifier that always succeeds. Test only.
contract ZKMockVerifier is IZKVerifier {
    /// @notice Always succeeds (no-op).
    function verify(bytes calldata, bytes calldata) external pure { }
}
