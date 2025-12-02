// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ECDSAUpgradeable } from "@openzeppelin/contracts-upgradeable/utils/cryptography/ECDSAUpgradeable.sol";
import { AdminFaucetAuthModule } from "src/periphery/faucet/authmodules/AdminFaucetAuthModule.sol";

/// @notice Simple helper contract that helps with testing the Faucet contract.
contract FaucetHelper {
    /// @notice EIP712 typehash for the Proof type.
    bytes32 public constant PROOF_TYPEHASH = keccak256("Proof(address recipient,bytes32 nonce,bytes32 id)");

    /// @notice EIP712 typehash for the EIP712Domain type that is included as part of the signature.
    bytes32 public constant EIP712_DOMAIN_TYPEHASH =
        keccak256("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)");

    /// @notice Keeps track of current nonce to generate new nonces for each drip.
    uint256 public currentNonce;

    /// @notice Returns a bytes32 nonce that should change everytime. In practice, people should use
    ///         pseudorandom nonces.
    /// @return Nonce that should be used as part of drip parameters.
    function consumeNonce() public returns (bytes32) {
        return bytes32(keccak256(abi.encode(currentNonce++)));
    }

    /// @notice Returns the hash of the struct Proof.
    /// @param _proof Proof struct to hash.
    /// @return EIP-712 typed struct hash.
    function getProofStructHash(AdminFaucetAuthModule.Proof memory _proof) public pure returns (bytes32) {
        return keccak256(abi.encode(PROOF_TYPEHASH, _proof.recipient, _proof.nonce, _proof.id));
    }

    /// @notice Computes the EIP712 digest with the given domain parameters.
    ///         Used for testing that different domain parameters fail.
    /// @param _proof             Proof struct to hash.
    /// @param _name              Contract name to use in the EIP712 domain.
    /// @param _version           Contract version to use in the EIP712 domain.
    /// @param _chainid           Chain ID to use in the EIP712 domain.
    /// @param _verifyingContract Address to use in the EIP712 domain.
    /// @param _verifyingContract Address to use in the EIP712 domain.
    /// @param _verifyingContract Address to use in the EIP712 domain.
    /// @return EIP-712 compatible digest.
    function getDigestWithEIP712Domain(
        AdminFaucetAuthModule.Proof memory _proof,
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
        return ECDSAUpgradeable.toTypedDataHash(domainSeparator, getProofStructHash(_proof));
    }
}
