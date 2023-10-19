// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { GetSigners } from "src/Safe/GetSigners.sol";
import "test/safe-tools/SafeTestTools.sol";

import { SignatureDecoder } from "safe-contracts/common/SignatureDecoder.sol";

contract GetSigners_Test is Test, SafeTestTools, GetSigners {
    bytes4 internal constant EIP1271_MAGIC_VALUE = 0x20c13b0b;

    enum SigTypes {
        Eoa,
        EthSign,
        ApprovedHash //,
            // Contract
    }

    function sigType(uint256 _key) internal view returns (SigTypes sigType_) {
        uint256 t = _key % 3; //4;
        sigType_ = SigTypes(t);
    }

    /// @dev Test that for a given set of signatures:
    ///      1. safe.checkNSignatures() succeeds
    ///      2. the getSigners() method returns the expected signers
    ///      3. the expected signers are all owners of the safe.
    ///      Demonstrating these three properties is sufficient to prove that the getSigners() method
    ///      returns the same signatures as those recovered by safe.checkNSignatures().
    /// todo(maurelian): include tests for EIP1271 signatures, and contract signatures.
    function testDiff_getSignaturesVsCheckSignatures_succeeds(bytes32 _digest, uint256 _numSigs) external {
        // Limit the number of each signature type to 25
        uint256 numSigs = bound(_numSigs, 1, 25);

        (, uint256[] memory keys) = makeAddrsAndKeys(numSigs);
        keys = sortPKsByComputedAddress(keys);

        // Create a new safeInstance with M=N, so that it requires a signature from each key.
        SafeInstance memory safeInstance = SafeTestTools._setupSafe(keys, numSigs, 0);

        // Create an empty array of signature data
        bytes memory signatures;

        // Populate the signatures by iterating over the keys, and choosing the signature type based
        // on the key.
        uint8 v;
        bytes32 r;
        bytes32 s;
        for (uint256 i; i < keys.length; i++) {
            if (sigType(keys[i]) == SigTypes.Eoa) {
                (v, r, s) = vm.sign(keys[i], _digest);
                // Safe signatures are encoded as r, s, v, not v, r, s.
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            } else if (sigType(keys[i]) == SigTypes.EthSign) {
                (v, r, s) = vm.sign(keys[i], keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", _digest)));
                v += 4;
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            } else if (sigType(keys[i]) == SigTypes.ApprovedHash) {
                vm.prank(getAddr(keys[i]));
                safeInstance.safe.approveHash(_digest);
                v = 1;
                s; // s is not checked on approved hash signatures.
                r = bytes32(uint256(uint160(getAddr(keys[i]))));
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            } // else if (sigType(keys[i]) == SigTypes.Contract) {
                //     address addr = decodeSmartContractWalletAsAddress(keys[i]);
                //     r = bytes32(uint256(uint160(addr)));
                //     vm.mockCall(
                //         addr, abi.encodeWithSignature("isValidSignature(bytes,bytes)"),
                // abi.encode(EIP1271_MAGIC_VALUE)
                //     );
                //     v = 1;
                //     s; // s is not checked on approved hash signatures.
                //     signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
                // }
        }
        // Signature checking on the Safe should succeed.
        // temp note: the second arg is the data, which is only used in the contract signatures type.
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
