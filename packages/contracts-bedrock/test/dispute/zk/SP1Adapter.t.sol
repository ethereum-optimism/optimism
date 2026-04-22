// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "test/setup/Test.sol";
import { ISP1Adapter } from "interfaces/dispute/zk/ISP1Adapter.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";
import { MockSP1Verifier } from "test/dispute/zk/MockSP1Verifier.sol";
import { MockSP1RejectingVerifier } from "test/dispute/zk/MockSP1RejectingVerifier.sol";

contract SP1Adapter_TestInit is Test {
    ISP1Adapter internal adapter;
    MockSP1Verifier internal mockVerifier;

    function setUp() public virtual {
        mockVerifier = new MockSP1Verifier();
        adapter =
            ISP1Adapter(vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(mockVerifier), string("PLONK"))));
    }
}

contract SP1Adapter_Constructor_Test is SP1Adapter_TestInit {
    /// @notice Tests that the sp1Verifier is set correctly.
    function test_sp1Verifier_succeeds() external view {
        assertEq(address(adapter.sp1Verifier()), address(mockVerifier));
    }

    /// @notice Tests that the proofSystem is set correctly.
    function test_proofSystem_succeeds() external view {
        assertEq(adapter.proofSystem(), "PLONK");
    }

    /// @notice Tests that construction reverts when the verifier address has no code.
    function test_constructor_verifierHasNoCode_reverts() external {
        address emptyVerifier = makeAddr("emptyVerifier");
        assertEq(emptyVerifier.code.length, 0);

        vm.expectRevert(ISP1Adapter.SP1Adapter_InvalidVerifier.selector);
        vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(emptyVerifier, string("PLONK")));
    }

    /// @notice Tests that construction reverts when the proof system identifier is empty.
    function test_constructor_emptyProofSystem_reverts() external {
        vm.expectRevert(ISP1Adapter.SP1Adapter_InvalidProofSystem.selector);
        vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(mockVerifier), string("")));
    }
}

contract SP1Adapter_Version_Test is SP1Adapter_TestInit {
    /// @notice Tests that the version is correct.
    function test_version_succeeds() external view {
        assertEq(adapter.version(), "1.0.0");
    }
}

contract SP1Adapter_VerifierType_Test is SP1Adapter_TestInit {
    /// @notice Tests that verifierType returns the expected string for a PLONK-configured adapter.
    function test_verifierType_plonk_succeeds() external view {
        assertEq(adapter.verifierType(), "SP1-PLONK-v6.0.0");
    }

    /// @notice Tests that verifierType reflects the proof system supplied at construction.
    function test_verifierType_groth16_succeeds() external {
        ISP1Adapter groth16Adapter = ISP1Adapter(
            vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(mockVerifier), string("GROTH16")))
        );
        assertEq(groth16Adapter.verifierType(), "SP1-GROTH16-v6.0.0");
    }
}

contract SP1Adapter_Verify_Test is SP1Adapter_TestInit {
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
        ISP1Adapter rejectingAdapter = ISP1Adapter(
            vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(rejectingVerifier), string("PLONK")))
        );

        vm.expectRevert("SP1Verifier: invalid proof");
        rejectingAdapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }
}
