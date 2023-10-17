// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { GetSigners } from "src/Safe/GetSigners.sol";
import "test/safe-tools/SafeTestTools.sol";

import { SignatureDecoder } from "safe-contracts/common/SignatureDecoder.sol";

contract GetSigners_Test is Test, SafeTestTools, GetSigners {
    /// @dev Test that for a given set of signatures:
    ///      1. safe.checkNSignatures() succeeds
    ///      2. the getSigners() method returns the expected signers
    ///      3. the expected signers are all owners of the safe.
    ///      Demonstrating these three properties is sufficient to prove that the getSigners() method
    ///      returns the same signatures as those recovered by safe.checkNSignatures().
    /// todo(maurelian): include tests for EIP1271 signatures, and contract signatures.
    function testDiff_getSignaturesVsCheckSignatures_succeeds(uint256 _numSigs, bytes32 _digest) external {
        uint256 numSigs = bound(_numSigs, 1, 100);
        (, uint256[] memory keys) = makeAddrsAndKeys(numSigs);
        SafeInstance memory safeInstance = SafeTestTools._setupSafe(keys, numSigs, 0);

        bytes memory signatures;
        for (uint256 i; i < numSigs; i++) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(keys[i], _digest);

            // Safe signatures are encoded as r, s, v, not v, r, s.
            signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
        }

        // Signature checking on the Safe should succeed.
        safeInstance.safe.checkNSignatures(_digest, hex"", signatures, numSigs);

        // Recover the signatures using the getSigners() method.
        address[] memory gotSigners = _getNSigners(_digest, signatures);

        // Compare the recovered signers to the expected signers.
        assertEq(gotSigners.length, numSigs);
        assertEq(gotSigners.length, safeInstance.owners.length);
        for (uint256 i; i < numSigs; i++) {
            assertEq(safeInstance.owners[i], gotSigners[i]);
        }
    }
}
