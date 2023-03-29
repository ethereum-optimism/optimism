//SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";

import { OptimistInviter } from "../../universal/op-nft/OptimistInviter.sol";

/**
 * Simple helper contract that helps with testing flow and signature for OptimistInviter contract.
 * Made this a separate contract instead of including in OptimistInviter.t.sol for reusability.
 */
contract OptimistInviterHelper {
    bytes32 public constant CLAIMABLE_INVITE_TYPEHASH =
        keccak256("ClaimableInvite(address issuer,bytes32 nonce)");

    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256(
            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
        );

    OptimistInviter public optimistInviter;
    string public name;
    uint256 public currentNonce;

    constructor(OptimistInviter _optimistInviter, string memory _name) {
        optimistInviter = _optimistInviter;
        name = _name;
        currentNonce = 0;
    }

    /**
     * @notice Returns the hash of the struct ClaimableInvite
     */
    function getClaimableInviteStructHash(OptimistInviter.ClaimableInvite memory _claimableInvite)
        public
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
     * @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
     *         pseudorandom nonces.
     */
    function consumeNonce() public returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /**
     * @notice Returns a ClaimableInvite with the issuer and current nonce
     */
    function getClaimableInviteWithNewNonce(address _issuer)
        public
        returns (OptimistInviter.ClaimableInvite memory)
    {
        return OptimistInviter.ClaimableInvite(_issuer, consumeNonce());
    }

    /**
     * @notice Computes the EIP712 digest with default correct parameters.
     */
    function getDigest(OptimistInviter.ClaimableInvite calldata _claimableInvite)
        public
        view
        returns (bytes32)
    {
        return
            getDigestWithEIP712Domain(
                _claimableInvite,
                bytes(name),
                bytes(optimistInviter.EIP712_VERSION()),
                block.chainid,
                address(optimistInviter)
            );
    }

    /**
     * @notice Computes the EIP712 digest with the given domain parameters.
     *         Used for testing that different domain parameters fail.
     */
    function getDigestWithEIP712Domain(
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
        return
            ECDSA.toTypedDataHash(domainSeparator, getClaimableInviteStructHash(_claimableInvite));
    }
}
