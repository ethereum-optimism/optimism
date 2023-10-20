// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SignatureDecoder } from "safe-contracts/common/SignatureDecoder.sol";

abstract contract GetSigners is SignatureDecoder {
    /// @notice Extract the signers from a set of signatures.
    /// @param dataHash Hash of the data.
    /// @param signatures Signature data for identifying signers.
    /// @param requiredSignatures Amount of required valid signatures.
    function _getNSigners(
        bytes32 dataHash,
        bytes memory signatures,
        uint256 requiredSignatures
    )
        internal
        pure
        returns (address[] memory _owners)
    {
        _owners = new address[](requiredSignatures);

        /// The following code is extracted from the Safe.checkNSignatures() method. It removes the signature
        /// validation code, and keeps only the parsing code necessary to extract the owner addresses from the
        /// signatures. We do not double check if the owner derived from a signature is valid. As this is handled
        /// in the final require statement of Safe.checkNSignatures().
        address currentOwner;
        uint8 v;
        bytes32 r;
        bytes32 s;
        uint256 i;
        for (i = 0; i < requiredSignatures; i++) {
            (v, r, s) = signatureSplit(signatures, i);
            if (v == 0) {
                // If v is 0 then it is a contract signature
                // When handling contract signatures the address of the contract is encoded into r
                currentOwner = address(uint160(uint256(r)));
            } else if (v == 1) {
                // If v is 1 then it is an approved hash
                // When handling approved hashes the address of the approver is encoded into r
                currentOwner = address(uint160(uint256(r)));
            } else if (v > 30) {
                // If v > 30 then default va (27,28) has been adjusted for eth_sign flow
                // To support eth_sign and similar we adjust v and hash the messageHash with the Ethereum message prefix
                // before applying ecrecover
                currentOwner =
                    ecrecover(keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", dataHash)), v - 4, r, s);
            } else {
                // Default is the ecrecover flow with the provided data hash
                // Use ecrecover with the messageHash for EOA signatures
                currentOwner = ecrecover(dataHash, v, r, s);
            }
            _owners[i] = currentOwner;
        }
    }
}
