// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "test/setup/Test.sol";
import { IRiscZeroAdapter } from "interfaces/dispute/zk/IRiscZeroAdapter.sol";
import { IRiscZeroVerifier } from "interfaces/vendor/IRiscZeroVerifier.sol";
import { MockRiscZeroVerifier } from "test/dispute/zk/MockRiscZeroVerifier.sol";
import { MockRiscZeroRejectingVerifier } from "test/dispute/zk/MockRiscZeroRejectingVerifier.sol";

contract RiscZeroAdapter_TestInit is Test {
    IRiscZeroAdapter internal adapter;
    MockRiscZeroVerifier internal mockVerifier;

    function setUp() public virtual {
        mockVerifier = new MockRiscZeroVerifier();
        adapter =
            IRiscZeroAdapter(vm.deployCode("RiscZeroAdapter.sol:RiscZeroAdapter", abi.encode(address(mockVerifier))));
    }
}

contract RiscZeroAdapter_Constructor_Test is RiscZeroAdapter_TestInit {
    /// @notice Tests that the riscZeroVerifier is set correctly.
    function test_riscZeroVerifier_succeeds() external view {
        assertEq(address(adapter.riscZeroVerifier()), address(mockVerifier));
    }
}

contract RiscZeroAdapter_Version_Test is RiscZeroAdapter_TestInit {
    /// @notice Tests that the version is correct.
    function test_version_succeeds() external view {
        assertEq(adapter.version(), "1.0.0");
    }
}

contract RiscZeroAdapter_VerifierType_Test is RiscZeroAdapter_TestInit {
    /// @notice Tests that verifierType returns the expected string.
    function test_verifierType_succeeds() external view {
        assertEq(adapter.verifierType(), "RISC_ZERO-1.2.0");
    }
}

contract RiscZeroAdapter_Verify_Test is RiscZeroAdapter_TestInit {
    /// @notice Tests that verify succeeds when the underlying verifier succeeds.
    function test_verify_succeeds() external view {
        adapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }

    /// @notice Fuzz test: verify hashes publicValues using sha256 as expected and reorders arguments correctly.
    /// @param _programId The program ID to pass to the adapter.
    /// @param _publicValues The arbitrary public values to fuzz.
    /// @param _proof The arbitrary proof bytes to fuzz.
    function testFuzz_verify_forwardsArgs_succeeds(
        bytes32 _programId,
        bytes memory _publicValues,
        bytes memory _proof
    )
        external
    {
        bytes32 expectedJournalDigest = sha256(_publicValues);

        vm.expectCall(
            address(mockVerifier), abi.encodeCall(IRiscZeroVerifier.verify, (_proof, _programId, expectedJournalDigest))
        );
        adapter.verify(_programId, _publicValues, _proof);
    }

    /// @notice Tests that verify reverts when the underlying verifier reverts.
    function test_verify_invalidProof_reverts() external {
        MockRiscZeroRejectingVerifier rejectingVerifier = new MockRiscZeroRejectingVerifier();
        IRiscZeroAdapter rejectingAdapter = IRiscZeroAdapter(
            vm.deployCode("RiscZeroAdapter.sol:RiscZeroAdapter", abi.encode(address(rejectingVerifier)))
        );

        vm.expectRevert("RiscZeroVerifier: invalid proof");
        rejectingAdapter.verify(bytes32(uint256(1)), hex"aabb", hex"ccdd");
    }
}
