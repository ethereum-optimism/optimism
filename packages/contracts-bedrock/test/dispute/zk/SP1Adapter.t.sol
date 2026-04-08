// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @notice Minimal interface to interact with SP1Adapter deployed via vm.deployCode.
interface ISP1Adapter is IZKVerifier {
    function sp1Verifier() external view returns (address);
}

/// @notice Mock SP1 verifier that always succeeds.
contract MockSP1Verifier is ISP1Verifier {
    function VERSION() external pure returns (string memory) {
        return "v6.0.0";
    }

    function verifyProof(bytes32, bytes calldata, bytes calldata) external pure { }
}

/// @notice Mock SP1 verifier that always reverts.
contract MockSP1RejectingVerifier is ISP1Verifier {
    function VERSION() external pure returns (string memory) {
        return "v6.0.0";
    }

    function verifyProof(bytes32, bytes calldata, bytes calldata) external pure {
        revert("SP1Verifier: invalid proof");
    }
}

contract SP1Adapter_Init is Test {
    ISP1Adapter internal adapter;
    MockSP1Verifier internal mockVerifier;

    function setUp() public virtual {
        mockVerifier = new MockSP1Verifier();
        adapter = ISP1Adapter(vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(mockVerifier))));
    }
}

contract SP1Adapter_Constructor_Test is SP1Adapter_Init {
    /// @notice Tests that the sp1Verifier is set correctly.
    function test_sp1Verifier_succeeds() external view {
        assertEq(adapter.sp1Verifier(), address(mockVerifier));
    }
}

contract SP1Adapter_Version_Test is SP1Adapter_Init {
    /// @notice Tests that the version is correct.
    function test_version_succeeds() external view {
        assertEq(adapter.version(), "1.0.0");
    }
}

contract SP1Adapter_VerifierType_Test is SP1Adapter_Init {
    /// @notice Tests that verifierType returns the expected string.
    function test_verifierType_succeeds() external view {
        assertEq(adapter.verifierType(), "SP1-v6.0.0");
    }
}

contract SP1Adapter_Verify_Test is SP1Adapter_Init {
    /// @notice Tests that verify succeeds when the underlying verifier succeeds.
    function test_verify_succeeds() external view {
        adapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }

    /// @notice Tests that verify forwards the correct arguments.
    function test_verify_forwardsArgs_succeeds() external {
        bytes32 programId = bytes32(uint256(42));
        bytes memory publicValues = hex"1234";
        bytes memory proof = hex"5678";

        vm.expectCall(
            address(mockVerifier),
            abi.encodeCall(ISP1Verifier.verifyProof, (programId, publicValues, proof))
        );
        adapter.verify(programId, publicValues, proof);
    }
}

contract SP1Adapter_Verify_TestFail is Test {
    ISP1Adapter internal adapter;

    function setUp() public {
        MockSP1RejectingVerifier rejectingVerifier = new MockSP1RejectingVerifier();
        adapter = ISP1Adapter(vm.deployCode("SP1Adapter.sol:SP1Adapter", abi.encode(address(rejectingVerifier))));
    }

    /// @notice Tests that verify reverts when the underlying verifier reverts.
    function test_verify_invalidProof_reverts() external {
        vm.expectRevert("SP1Verifier: invalid proof");
        adapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }
}
