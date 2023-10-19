// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Safe } from "safe-contracts/Safe.sol";
import { GetSigners } from "src/Safe/GetSigners.sol";
import "test/safe-tools/SafeTestTools.sol";

import { SignatureDecoder } from "safe-contracts/common/SignatureDecoder.sol";

contract GetSigners_Test is Test, SafeTestTools, GetSigners {
    struct SigTypeCount {
        uint256 numEoaSigs;
        uint256 numEthSignSigs;
        uint256 numApprovedHashSigs;
        uint256 numContractSigs;
    }

    enum SigType {
        Eoa,
        EthSign,
        ApprovedHash,
        Contract
    }

    mapping(uint256 => SigType) public sigTypes;

    /// @dev Test that for a given set of signatures:
    ///      1. safe.checkNSignatures() succeeds
    ///      2. the getSigners() method returns the expected signers
    ///      3. the expected signers are all owners of the safe.
    ///      Demonstrating these three properties is sufficient to prove that the getSigners() method
    ///      returns the same signatures as those recovered by safe.checkNSignatures().
    /// todo(maurelian): include tests for EIP1271 signatures, and contract signatures.
    function testDiff_getSignaturesVsCheckSignatures_succeeds(bytes32 _digest, SigTypeCount memory _split) external {
        // Limit the number of each signature type to 25
        uint256 numEoaSigs = bound(_split.numEoaSigs, 1, 25);
        uint256 numEthSignSigs = bound(_split.numEthSignSigs, 1, 25);
        // uint256 numContractSigs = bound(_split.numContractSigs, 1, 25);
        // uint256 numApprovedHashSigs = bound(_split.numApprovedHashSigs, 1, 25);

        // uint256 numSigs = numEoaSigs + numApprovedHashSigs + numContractSigs + numEthSignSigs;
        uint256 numSigs = numEoaSigs + numEthSignSigs;

        (, uint256[] memory keys) = makeAddrsAndKeys(numSigs);

        // record the signature types for each key
        for (uint256 i; i < numSigs; i++) {
            // Generate EOA keys for both EOA and ETH Sign signatures
            if (i < numEoaSigs) {
                sigTypes[keys[i]] = SigType.Eoa;
            } else if (i < numEoaSigs + numEthSignSigs) {
                sigTypes[keys[i]] = SigType.EthSign;
            } else {
                // Generate approved hash signatures
                // Generate eth_sign signatures
                revert("not implemented");
            }
        }

        // Now sort the keys array. By doing this after assigning a signature type to each key,
        // we ensure that the signature types are randomly ordered. It probably doesn't matter either
        // way, but this is more realistic.
        keys = sortPKsByComputedAddress(keys);

        // Create a new safeInstance with M=N, so that it requires a signature from each key.
        SafeInstance memory safeInstance = SafeTestTools._setupSafe(keys, numSigs, 0);

        // Create an empty array of signature data
        bytes memory signatures;

        // Populate the signatures by iterating over the safeInstance owners list.
        // is a requirement for the ordering of signatures in the Safe contract.
        for (uint256 i; i < keys.length; i++) {
            if (sigTypes[keys[i]] == SigType.Eoa) {
                (uint8 v, bytes32 r, bytes32 s) = vm.sign(keys[i], _digest);
                // Safe signatures are encoded as r, s, v, not v, r, s.
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            } else if (sigTypes[keys[i]] == SigType.EthSign) {
                (uint8 v, bytes32 r, bytes32 s) =
                    vm.sign(keys[i], keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", _digest)));
                v += 4;
                signatures = bytes.concat(signatures, abi.encodePacked(r, s, v));
            }
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
