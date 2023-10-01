// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard } from "safe-contracts/base/GuardManager.sol";
import { SignatureDecoder } from "safe-contracts/common/SignatureDecoder.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

contract LivenessGuard is SignatureDecoder, BaseGuard {
    Safe public safe;
    mapping(address => uint256) public lastSigned;

    constructor(Safe _safe) {
        safe = _safe;
    }

    /// @notice We just need to satisfy the BaseGuard interfae, but we don't actually need to use this method.
    function checkAfterExecution(bytes32, bool) external pure {
        return;
    }

    /// @notice Records the most recent time which any owner has signed a transaction.
    function checkTransaction(
        address to,
        uint256 value,
        bytes memory data,
        Enum.Operation operation,
        uint256 safeTxGas,
        uint256 baseGas,
        uint256 gasPrice,
        address gasToken,
        address payable refundReceiver,
        bytes memory signatures,
        address
    )
        external
    {
        require(msg.sender == address(safe), "LivenessGuard: only Safe can call this function");
        if (to == address(safe) && data[0:4] == bytes4(keccak256("setGuard(address)"))) {
            // We can't allow the guard to be disabled, or else the upgrade delay can be bypassed.
            // TODO(Maurelian): Figure out how to best address this.
        }

        // This is a bit of a hack, maybe just replicate the functionality here rather than calling home
        bytes32 txHash = Safe(payable(msg.sender)).getTransactionHash(
            // Transaction info
            to,
            value,
            data,
            operation,
            safeTxGas,
            // Payment info
            baseGas,
            gasPrice,
            gasToken,
            refundReceiver,
            // Signature info
        );
        address[] memory signers = _getNSigners(txHash, signatures);
        for (uint256 i = 0; i < signers.length; i++) {
            lastSigned[signers[i]] = block.timestamp;
        }
    }

    /// @notice Exctract the signers from a set of signatures.
    function _getNSigners(bytes32 dataHash, bytes memory signatures) internal pure returns (address[] memory _owners) {
        uint256 numSignatures = signatures.length / 65;
        _owners = new address[](numSignatures);

        /// The following code is extracted from the Safe.checkNSignatures() method. It removes the signature
        /// validation code, and keeps only the parsing code necessary to extract the owner addresses from the
        /// signatures. We do not double check if the owner derived from a signature is valid. As this is handled
        /// in the final require statement of Safe.checkNSignatures().
        address currentOwner;
        uint8 v;
        bytes32 r;
        bytes32 s;
        uint256 i;
        for (i = 0; i < numSignatures; i++) {
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
