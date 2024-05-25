// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { OptimistInviter } from "src/periphery/op-nft/OptimistInviter.sol";
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

/// @notice Simple helper contract that helps with testing flow and signature for
///         OptimistInviter contract. Made this a separate contract instead of including
///         in OptimistInviter.t.sol for reusability.
contract OptimistInviterHelper {
    /// @notice EIP712 typehash for the ClaimableInvite type.
    bytes32 public constant CLAIMABLE_INVITE_TYPEHASH = keccak256("ClaimableInvite(address issuer,bytes32 nonce)");

    /// @notice EIP712 typehash for the EIP712Domain type that is included as part of the signature.
    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");

    /// @notice Address of OptimistInviter contract we are testing.
    OptimistInviter public optimistInviter;

    /// @notice OptimistInviter contract name. Used to construct the EIP-712 domain.
    string public name;

    /// @notice Keeps track of current nonce to generate new nonces for each invite.
    uint256 public currentNonce;

    constructor(OptimistInviter _optimistInviter, string memory _name) {
        optimistInviter = _optimistInviter;
        name = _name;
    }

    /// @notice Returns the hash of the struct ClaimableInvite.
    /// @param _claimableInvite ClaimableInvite struct to hash.
    /// @return EIP-712 typed struct hash.
    function getClaimableInviteStructHash(OptimistInviter.ClaimableInvite memory _claimableInvite)
        public
        pure
        returns (bytes32)
    {
        return keccak256(abi.encode(CLAIMABLE_INVITE_TYPEHASH, _claimableInvite.issuer, _claimableInvite.nonce));
    }

    /// @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
    ///         pseudorandom nonces.
    /// @return Nonce that should be used as part of ClaimableInvite.
    function consumeNonce() public returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /// @notice Returns a ClaimableInvite with the issuer and current nonce.
    /// @param _issuer Issuer to include in the ClaimableInvite.
    /// @return ClaimableInvite that can be hashed & signed.
    function getClaimableInviteWithNewNonce(address _issuer) public returns (OptimistInviter.ClaimableInvite memory) {
        return OptimistInviter.ClaimableInvite(_issuer, consumeNonce());
    }

    /// @notice Computes the EIP712 digest with default correct parameters.
    /// @param _claimableInvite ClaimableInvite struct to hash.
    /// @return EIP-712 compatible digest.
    function getDigest(OptimistInviter.ClaimableInvite calldata _claimableInvite) public view returns (bytes32) {
        return getDigestWithEIP712Domain(
            _claimableInvite,
            bytes(name),
            bytes(optimistInviter.EIP712_VERSION()),
            block.chainid,
            address(optimistInviter)
        );
    }

    /// @notice Computes the EIP712 digest with the given domain parameters.
    ///         Used for testing that different domain parameters fail.
    /// @param _claimableInvite   ClaimableInvite struct to hash.
    /// @param _name              Contract name to use in the EIP712 domain.
    /// @param _version           Contract version to use in the EIP712 domain.
    /// @param _chainid           Chain ID to use in the EIP712 domain.
    /// @param _verifyingContract Address to use in the EIP712 domain.
    /// @return EIP-712 compatible digest.
    function getDigestWithEIP712Domain(
        OptimistInviter.ClaimableInvite calldata _claimableInvite,
        bytes memory _name,
        bytes memory _version,
        uint256 _chainid,
        address _verifyingContract
    )
        public
        pure
        returns (bytes32)
    {
        bytes32 domainSeparator = keccak256(
            abi.encode(EIP712_DOMAIN_TYPEHASH, keccak256(_name), keccak256(_version), _chainid, _verifyingContract)
        );
        return ECDSA.toTypedDataHash(domainSeparator, getClaimableInviteStructHash(_claimableInvite));
    }
}
