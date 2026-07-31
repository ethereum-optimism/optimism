// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Forge
import { Script } from "forge-std/Script.sol";

// Testing
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";

/// @title DeployZKMockVerifier
/// @notice Deploys the test-only ZK mock verifier.
contract DeployZKMockVerifier is Script {
    /// @notice Deploys the ZK mock verifier.
    function run() public returns (ZKMockVerifier verifier_) {
        vm.broadcast(msg.sender);
        verifier_ = new ZKMockVerifier();
    }
}
