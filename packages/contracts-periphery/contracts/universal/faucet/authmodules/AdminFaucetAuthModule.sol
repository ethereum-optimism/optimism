// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import {
    EIP712Upgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/draft-EIP712Upgradeable.sol";
import { SignatureChecker } from "@openzeppelin/contracts/utils/cryptography/SignatureChecker.sol";
import { IFaucetAuthModule } from "./IFaucetAuthModule.sol";
import { Faucet } from "../Faucet.sol";

/**
 * @title  AdminFaucetAuthModule
 * @notice FaucetAuthModule that allows an admin to sign off on a given faucet drip. Takes an admin
 *         as the constructor argument.
 */
contract AdminFaucetAuthModule is IFaucetAuthModule, Semver, EIP712Upgradeable {
    /**
     * @notice Admin address that can sign off on drips.
     */
    address public immutable ADMIN;

    /**
     * @notice EIP712 typehash for the Proof type.
     */
    bytes32 public constant PROOF_TYPEHASH =
        keccak256("Proof(address recipient,bytes32 nonce,bytes id)");

    /**
     * @notice Struct that represents a proof that verifies the admin.
     *
     * @custom:field recipient Address that will be receiving the faucet funds.
     * @custom:field nonce     Pseudorandom nonce to prevent replay attacks.
     * @custom:field id        id for the user requesting the faucet funds.
     */
    struct Proof {
        address recipient;
        bytes32 nonce;
        bytes id;
    }

    /**
     * @param admin Admin address that can sign off on drips.
     */
    constructor(address admin) Semver(1, 0, 0) {
        ADMIN = admin;
    }

    /**
     * @notice Initializes this contract, setting the EIP712 context.
     *
     * @param _name Contract name.
     */
    function initialize(string memory _name) public initializer {
        __EIP712_init(_name, version());
    }

    /**
     * @inheritdoc IFaucetAuthModule
     */
    function verify(
        Faucet.DripParameters memory _params,
        bytes memory _id,
        bytes memory _proof
    ) external view returns (bool) {
        // Generate a EIP712 typed data hash to compare against the proof.
        bytes32 digest = _hashTypedDataV4(
            keccak256(abi.encode(PROOF_TYPEHASH, _params.recipient, _params.nonce, _id))
        );
        return SignatureChecker.isValidSignatureNow(ADMIN, digest, _proof);
    }
}
