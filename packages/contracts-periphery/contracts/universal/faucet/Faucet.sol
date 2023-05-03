// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "@eth-optimism/contracts-bedrock/contracts/universal/Semver.sol";
import { SignatureChecker } from "@openzeppelin/contracts/utils/cryptography/SignatureChecker.sol";
import {
    EIP712Upgradeable
} from "@openzeppelin/contracts-upgradeable/utils/cryptography/draft-EIP712Upgradeable.sol";

/**
 * @title  SafeSend
 * @notice Sends ETH to a recipient account without triggering any code.
 */
contract SafeSend {
    /**
     * @param recipient Account to send ETH to.
     */
    constructor(
        address payable recipient
    )
        payable
    {
        selfdestruct(recipient);
    }
}

/**
 * @title  FaucetAuthModule
 * @notice Interface for faucet authentication modules.
 */
interface FaucetAuthModule {
    /**
     * @notice Verifies that the given drip parameters are valid.
     *
     * @param params Drip parameters to verify.
     * @param id     Authentication ID to verify.
     * @param proof  Authentication proof to verify.
     */
    function verify(
        Faucet.DripParameters memory params,
        bytes memory id,
        bytes memory proof
    )
        external
        view
        returns (
            bool
        );
}

/**
 * @title  AdminFAM
 * @notice FaucetAuthModule that allows an admin to sign off on a given faucet drip. Takes an admin
 *         as the constructor argument.
 */
contract AdminFAM is FaucetAuthModule, Semver, EIP712Upgradeable {
    /**
     * @notice Admin address that can sign off on drips.
     */
    address public immutable ADMIN;

    /**
     * @notice EIP712 typehash for the ClaimableInvite type.
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
     * @param admin     Admin address that can sign off on drips.
     */
    constructor(
        address admin
    ) Semver(1, 0, 0) {
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
     * @inheritdoc FaucetAuthModule
     */
    function verify(
        Faucet.DripParameters memory params,
        bytes memory id,
        bytes memory proof
    ) external view returns (bool) {
        // Generate a EIP712 typed data hash to compare against the proof.
        bytes32 digest = _hashTypedDataV4(
            keccak256(
                abi.encode(
                    PROOF_TYPEHASH,
                    params.recipient,
                    params.nonce,
                    id
                )
            )
        );
        return SignatureChecker.isValidSignatureNow(ADMIN, digest, proof);
    }
}

/**
 * @title  Faucet
 * @notice Faucet contract that drips ETH to users.
 */
contract Faucet {
    /**
     * @notice Parameters for a drip.
     */
    struct DripParameters {
        address payable recipient;
        bytes32 nonce;
    }

    /**
     * @notice Parameters for authentication.
     */
    struct AuthParameters {
        FaucetAuthModule module;
        bytes id;
        bytes proof;
    }

    /**
     * @notice Configuration for an authentication module.
     */
    struct ModuleConfig {
        bool enabled;
        uint256 ttl;
        uint256 amount;
    }

    /**
     * @notice Admin address that can configure the faucet.
     */
    address public immutable ADMIN;

    /**
     * @notice Mapping of authentication modules to their configurations.
     */
    mapping (FaucetAuthModule => ModuleConfig) public modules;

    /**
     * @notice Mapping of authentication IDs to the next timestamp at which they can be used.
     */
    mapping (FaucetAuthModule => mapping (bytes => uint256)) public timeouts;

    /**
     * @notice Maps from id to nonces to whether or not they have been used.
     */
    mapping(bytes => mapping(bytes32 => bool)) public usedNonces;

	/**
     * @notice Modifier that makes a function admin priviledged.
     */
    modifier priviledged() {
        require(
			msg.sender == ADMIN,
			"Faucet: function can only be called by admin"
        );
		_;
    }

    /**
     * @param admin Admin address that can configure the faucet.
     */
    constructor(
        address admin
    ) {
        ADMIN = admin;
    }

    /**
     * @notice Allows users to donate ETH to this contract.
     */
    receive() external payable {
	    // Thank you!
	}

   /**
     * @notice Allows the admin to withdraw funds.
     *
     * @param recipient Address to receive the funds.
     * @param amount    Amount of ETH in wei to withdraw.
     */
    function withdraw(
        address payable recipient,
        uint256 amount
    ) public priviledged {
		new SafeSend{value: amount}(recipient);
	}

    /**
     * @notice Allows the admin to configure an authentication module.
     *
     * @param module Authentication module to configure.
     * @param config Configuration to set for the module.
     */
    function configure(
        FaucetAuthModule module,
        ModuleConfig memory config
    ) public priviledged {
        modules[module] = config;
    }

    /**
     * @notice Drips ETH to a recipient account.
     *
     * @param params Drip parameters.
     * @param auth   Authentication parameters.
     */
    function drip(
        DripParameters memory params,
        AuthParameters memory auth
    ) public {
	    // Grab the module config once.
		ModuleConfig memory config = modules[auth.module];

        // Make sure we're using a supported security module.
        require(
            config.enabled,
            "Faucet: provided auth module is not supported by this faucet"
        );

        // The issuer's signature commits to a nonce to prevent replay attacks.
        // This checks that the nonce has not been used for this issuer before. The nonces are
        // scoped to the issuer address, so the same nonce can be used by different issuers without
        // clashing.
        require(
            usedNonces[auth.id][params.nonce] == false,
            "Faucet: nonce has already been used"
        );

        // Make sure the timeout has elapsed.
        require(
            timeouts[auth.module][auth.id] < block.timestamp,
            "Faucet: auth cannot be used yet because timeout has not elapsed"
        );

        // Verify the proof.
        require(
            auth.module.verify(params, auth.id, auth.proof),
            "Faucet: drip parameters could not be verified by security module"
        );

        // Set the next timestamp at which this auth id can be used.
        timeouts[auth.module][auth.id] = block.timestamp + config.ttl;

        // Execute a safe transfer of ETH to the recipient account.
        new SafeSend{value: config.amount}(params.recipient);
    }
}
