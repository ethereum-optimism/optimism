// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../libraries/DisputeTypes.sol";
import "../libraries/DisputeErrors.sol";

import { Clone } from "../libraries/Clone.sol";
import { EIP712 } from "@solady/utils/EIP712.sol";
import { ECDSA } from "@solady/utils/ECDSA.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { IAttestationDisputeGame } from "./interfaces/IAttestationDisputeGame.sol";
import { IDisputeGame } from "./interfaces/IDisputeGame.sol";
import { IInitializable } from "./interfaces/IInitializable.sol";
import { IVersioned } from "./interfaces/IVersioned.sol";
import { IBondManager } from "./interfaces/IBondManager.sol";

import { SystemConfig } from "../L1/SystemConfig.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";

/**
 * @title AttetationDisputeGame
 * @notice A contract for disputing the validity of a claim via permissioned attestations.
 */
contract AttestationDisputeGame is Initializable, IAttestationDisputeGame, Clone, EIP712 {
    /**
     * @notice The EIP-712 type hash for the `Dispute` struct.
     */
    Hash public constant DISPUTE_TYPE_HASH =
        Hash.wrap(keccak256("Dispute(bytes32 outputRoot,uint256 l2BlockNumber)"));

    /**
     * @notice The BondManager contract that is used to manage the bonds for this game.
     */
    IBondManager public immutable BOND_MANAGER;

    /**
     * @notice The L1's SystemConfig contract.
     */
    SystemConfig public immutable SYSTEM_CONFIG;

    /**
     * @notice The L2OutputOracle contract.
     */
    L2OutputOracle public immutable L2_OUTPUT_ORACLE;

    /**
     * @notice The current version of the `AttestationDisputeGame` contract.
     */
    string internal constant VERSION = "0.0.1";

    /**
     * @inheritdoc IDisputeGame
     */
    Timestamp public createdAt;

    /**
     * @inheritdoc IDisputeGame
     */
    GameStatus public status;

    /**
     * @notice An array of addresses that have submitted positive attestations for the `rootClaim`.
     */
    address[] public attestationSubmitters;

    /**
     * @notice A set of signer addresses allowed to participate in this game.
     */
    address[] public frozenAttestorSet;

    /**
     * @inheritdoc IAttestationDisputeGame
     */
    uint256 public frozenSignatureThreshold;

    /**
     * @inheritdoc IAttestationDisputeGame
     */
    mapping(address => bool) public challenges;

    /**
     * @notice Initialize the implementation upon deployment.
     * @param _bondManager The BondManager contract that is used to manage the bonds for this game.
     * @param _systemConfig The L1's SystemConfig contract.
     * @param _l2OutputOracle The L2OutputOracle contract.
     */
    constructor(
        IBondManager _bondManager,
        SystemConfig _systemConfig,
        L2OutputOracle _l2OutputOracle
    ) EIP712() {
        BOND_MANAGER = _bondManager;
        SYSTEM_CONFIG = _systemConfig;
        L2_OUTPUT_ORACLE = _l2OutputOracle;
    }

    /**
     * @inheritdoc IAttestationDisputeGame
     */
    function attestorSet(address addr) public view override returns (bool _isAuthorized) {
        for (uint256 i = 0; i < frozenAttestorSet.length; i++) {
            if (frozenAttestorSet[i] == addr) {
                _isAuthorized = true;
            }
        }
    }

    /**
     * @inheritdoc IAttestationDisputeGame
     */
    function challenge(bytes calldata signature) external {
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // Attempt to recover the signature provided. Solady's ECDSA library
        // will revert if the signer cannot be recovered from the given signature.
        address recovered = ECDSA.recoverCalldata(Hash.unwrap(getTypedDataHash()), signature);

        // Check that the recovered address is part of the `attestorSet`.
        if (!attestorSet(recovered)) {
            revert InvalidSignature();
        }

        // If the `recovered` address has already issued a positive
        // attestation for the `rootClaim`, revert.
        if (challenges[recovered]) {
            revert AlreadyChallenged();
        }

        // Mark that the authorized signer has issued a positive attestation for the `rootClaim`.
        challenges[recovered] = true;

        // Increment the number of positive attestations that have been issued for the `rootClaim`.
        attestationSubmitters.push(msg.sender);

        // If the provided signature breaches the signature threshold, resolve the game.
        if (attestationSubmitters.length == frozenSignatureThreshold) {
            resolve();
        }
    }

    /**
     * @notice Returns an Ethereum Signed Typed Data hash, as defined in EIP-712, for the
     *         `Dispute` struct. This hash is signed by members of the `attestorSet` to
     *         issue a positive attestation for the `rootClaim`.
     * @return _typedDataHash The EIP-712 hash of the `Dispute` struct.
     */
    function getTypedDataHash() public view returns (Hash _typedDataHash) {
        // Copy the `DISPUTE_TYPE_HASH` onto the stack.
        Hash disputeTypeHash = DISPUTE_TYPE_HASH;
        // Grab the root claim of the `AttestationDisputeGame`.
        Claim _rootClaim = rootClaim();
        // Grab the L2 block number that the `rootClaim` commits to.
        uint256 _l2BlockNumber = l2BlockNumber();

        // Hash the `Dispute` struct.
        Hash disputeStructHash;
        assembly {
            // Grab the location of some free memory.
            let ptr := mload(0x40)

            // Store the `DISPUTE_TYPE_HASH`
            mstore(ptr, disputeTypeHash)
            // Store the `rootClaim` of the `AttestationDisputeGame`.
            mstore(add(ptr, 0x20), _rootClaim)
            // Store the L2 block number that the `rootClaim` commits to.
            mstore(add(ptr, 0x40), _l2BlockNumber)

            // Hash the `Dispute` struct.
            disputeStructHash := keccak256(ptr, 0x60)

            // Update the free memory pointer
            mstore(0x40, and(add(ptr, 0x7F), not(0x1F)))
        }

        _typedDataHash = Hash.wrap(_hashTypedData(Hash.unwrap(disputeStructHash)));
    }

    /**
     * @inheritdoc IInitializable
     */
    function initialize() external initializer {
        createdAt = Timestamp.wrap(uint64(block.timestamp));
        frozenAttestorSet = SYSTEM_CONFIG.attestorSet();
        frozenSignatureThreshold = SYSTEM_CONFIG.attestationThreshold();
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function bondManager() external view returns (IBondManager _bondManager) {
        _bondManager = BOND_MANAGER;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function resolve() public returns (GameStatus _status) {
        if (status != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // Set the status as `CHALLENGER_WINS`.
        status = GameStatus.CHALLENGER_WINS;

        // Fetch the L2 block number that the `rootClaim` commits to.
        uint256 _l2BlockNumber = l2BlockNumber();

        // Delete all outputs from [l2BlockNumber, currentL2BlockNumber]
        L2_OUTPUT_ORACLE.deleteL2Outputs(_l2BlockNumber);

        // Request the `BondManager` to distribute the faulty output bond to the attestors.
        BOND_MANAGER.seizeAndSplit(keccak256(abi.encode(_l2BlockNumber)), attestationSubmitters);

        return status;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function gameType() public pure override returns (GameType) {
        return GameType.ATTESTATION;
    }

    /**
     * @inheritdoc IVersioned
     */
    function version() external pure override returns (string memory _version) {
        _version = VERSION;
    }

    /**
     * @notice Overrides the EIP712 domain information
     * @return _name The name of the domain.
     * @return _version The version of the domain.
     */
    function _domainNameAndVersion()
        internal
        pure
        override
        returns (string memory _name, string memory _version)
    {
        _name = "AttestationDisputeGame";
        _version = VERSION;
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function extraData() external pure returns (bytes memory _extraData) {
        _extraData = _getArgDynBytes(0x20, 0x20);
    }

    /**
     * @inheritdoc IDisputeGame
     */
    function rootClaim() public pure returns (Claim _rootClaim) {
        _rootClaim = Claim.wrap(_getArgFixedBytes(0x00));
    }

    /**
     * @inheritdoc IAttestationDisputeGame
     */
    function l2BlockNumber() public pure returns (uint256 _l2BlockNumber) {
        _l2BlockNumber = _getArgUint256(0x20);
    }
}
