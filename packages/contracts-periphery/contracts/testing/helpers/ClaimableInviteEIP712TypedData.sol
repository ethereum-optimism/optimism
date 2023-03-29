//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { OptimistInviter } from "../../universal/op-nft/OptimistInviter.sol";

/**
 * Helper library that generates the EIP712 digest for a ClaimableInvite.
 */
library ClaimableInviteEIP712TypedData {
    bytes32 public constant CLAIMABLE_INVITE_TYPEHASH =
        keccak256("ClaimableInvite(address issuer,bytes32 nonce)");

    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256(
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
        );

    function _getStructHash(OptimistInviter.ClaimableInvite memory _claimableInvite)
        internal
        pure
        returns (bytes32)
    {
        return
            keccak256(
                abi.encode(
                    CLAIMABLE_INVITE_TYPEHASH,
                    _claimableInvite.issuer,
                    _claimableInvite.nonce
                )
            );
    }

    /**
     * @notice Computes the EIP712 digest with the given domain parameters.
     *         Used for testing that different domain parameters fail.
     */
    function getDigest(
        OptimistInviter.ClaimableInvite calldata _claimableInvite,
        bytes memory _name,
        bytes memory _version,
        uint256 _chainid,
        address _verifyingContract
    ) public pure returns (bytes32) {
        bytes32 domainSeparator = keccak256(
            abi.encode(
                EIP712_DOMAIN_TYPEHASH,
                keccak256(_name),
                keccak256(_version),
                _chainid,
                _verifyingContract
            )
        );
        return ECDSA.toTypedDataHash(domainSeparator, _getStructHash(_claimableInvite));
    }
}
