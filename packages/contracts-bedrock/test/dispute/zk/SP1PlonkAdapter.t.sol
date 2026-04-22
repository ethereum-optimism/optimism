// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "test/setup/Test.sol";
import { ISP1PlonkAdapter } from "interfaces/dispute/zk/ISP1PlonkAdapter.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";
import { MockSP1Verifier } from "test/dispute/zk/MockSP1Verifier.sol";
import { MockSP1RejectingVerifier } from "test/dispute/zk/MockSP1RejectingVerifier.sol";

contract SP1PlonkAdapter_TestInit is Test {
    ISP1PlonkAdapter internal adapter;
    MockSP1Verifier internal mockVerifier;

    function setUp() public virtual {
        mockVerifier = new MockSP1Verifier();
        adapter =
            ISP1PlonkAdapter(vm.deployCode("SP1PlonkAdapter.sol:SP1PlonkAdapter", abi.encode(address(mockVerifier))));
    }
}

contract SP1PlonkAdapter_Constructor_Test is SP1PlonkAdapter_TestInit {
    /// @notice Tests that the sp1Verifier is set correctly.
    function test_sp1Verifier_succeeds() external view {
        assertEq(address(adapter.sp1Verifier()), address(mockVerifier));
    }

    /// @notice Tests that construction reverts when the verifier address has no code.
    function test_constructor_verifierHasNoCode_reverts() external {
        address emptyVerifier = makeAddr("emptyVerifier");
        assertEq(emptyVerifier.code.length, 0);

        vm.expectRevert(ISP1PlonkAdapter.SP1PlonkAdapter_InvalidVerifier.selector);
        vm.deployCode("SP1PlonkAdapter.sol:SP1PlonkAdapter", abi.encode(emptyVerifier));
    }
}

contract SP1PlonkAdapter_Version_Test is SP1PlonkAdapter_TestInit {
    /// @notice Tests that the version is correct.
    function test_version_succeeds() external view {
        assertEq(adapter.version(), "1.0.0");
    }
}

contract SP1PlonkAdapter_VerifierType_Test is SP1PlonkAdapter_TestInit {
    /// @notice Tests that verifierType returns the expected string.
    function test_verifierType_succeeds() external view {
        assertEq(adapter.verifierType(), "SP1-PLONK-v6.0.0");
    }
}

contract SP1PlonkAdapter_Verify_Test is SP1PlonkAdapter_TestInit {
    /// @notice Tests that verify succeeds when the underlying verifier succeeds.
    function test_verify_succeeds() external view {
        adapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }

    /// @notice Tests that verify forwards the correct arguments.
    function test_verify_forwardsArgs_succeeds() external {
        bytes32 programId = bytes32(uint256(42));
        bytes memory publicValues = hex"1234";
        bytes memory proof = hex"5678";

        vm.expectCall(address(mockVerifier), abi.encodeCall(ISP1Verifier.verifyProof, (programId, publicValues, proof)));
        adapter.verify(programId, publicValues, proof);
    }

    /// @notice Tests that verify reverts when the underlying verifier reverts.
    function test_verify_invalidProof_reverts() external {
        MockSP1RejectingVerifier rejectingVerifier = new MockSP1RejectingVerifier();
        ISP1PlonkAdapter rejectingAdapter = ISP1PlonkAdapter(
            vm.deployCode("SP1PlonkAdapter.sol:SP1PlonkAdapter", abi.encode(address(rejectingVerifier)))
        );

        vm.expectRevert("SP1Verifier: invalid proof");
        rejectingAdapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }
}
