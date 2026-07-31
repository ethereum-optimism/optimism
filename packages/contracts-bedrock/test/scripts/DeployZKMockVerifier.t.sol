// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";

// Scripts
import { DeployZKMockVerifier } from "scripts/deploy/DeployZKMockVerifier.s.sol";

/// @title DeployZKMockVerifierTest
/// @notice Tests the ZK mock verifier deployment script.
contract DeployZKMockVerifierTest is Test {
    /// @notice Tests that the deployment script returns a deployed mock verifier.
    function test_run_succeeds() public {
        ZKMockVerifier verifier = new DeployZKMockVerifier().run();

        assertGt(address(verifier).code.length, 0);
        assertEq(verifier.verifierType(), "ZKMockVerifier");
    }
}
