// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { EIP712 } from "@solady/utils/EIP712.sol";
import { ECDSA } from "@solady/utils/ECDSA.sol";

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { Hash } from "../libraries/DisputeTypes.sol";
import { Claim } from "../libraries/DisputeTypes.sol";
import { GameType } from "../libraries/DisputeTypes.sol";
import { Timestamp } from "../libraries/DisputeTypes.sol";
import { GameStatus } from "../libraries/DisputeTypes.sol";

import { InvalidSignature } from "../libraries/DisputeErrors.sol";
import { AlreadyChallenged } from "../libraries/DisputeErrors.sol";

import { IAttestationDisputeGame } from "./IAttestationDisputeGame.sol";
import { IBondManager } from "./IBondManager.sol";
import { Clone } from "../libraries/Clone.sol";

import { ISystemConfig } from "./ISystemConfig.sol";
import { IL2OutputOracle } from "./IL2OutputOracle.sol";

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
    IBondManager public immutable bondManager;

    /**
     * @notice The L1's SystemConfig contract.
     */
    ISystemConfig public immutable systemConfig;

    /**
     * @notice The L2OutputOracle contract.
     */
    IL2OutputOracle public immutable l2OutputOracle;

    /**
     * @notice The timestamp that the DisputeGame contract was created at.
     */
    Timestamp public createdAt;

    /**
     * @notice The current status of the game.
     */
    GameStatus public status;

    /**
     * @notice An array of addresses that have submitted positive attestations for the `rootClaim`.
     */
    address[] public attestationSubmitters;

    /**
     * @notice A set of signer addresses allowed to participate in this game.
     */
    address[] public frozenSignerSet;

    /**
     * @notice The number of authorized signatures required to successfully support the `rootClaim`.
     */
    uint256 public frozenSignatureThreshold;

    /**
     * @notice A mapping of addresses from the `signerSet` to booleans signifying whether
     *         or not they support the `rootClaim` being the valid output for `l2BlockNumber`.
     */
    mapping(address => bool) public challenges;

    /**
     * @notice Initialize the implementation upon deployment.
     * @param _bondmanager The BondManager contract that is used to manage the bonds for this game.
     * @param _systemConfig The L1's SystemConfig contract.
     * @param _l2OutputOracle The L2OutputOracle contract.
     */
    constructor(
        IBondManager _bondmanager,
        ISystemConfig _systemConfig,
        IL2OutputOracle _l2OutputOracle
    ) EIP712() {
        bondManager = _bondmanager;
        systemConfig = _systemConfig;
        l2OutputOracle = _l2OutputOracle;
    }

    /**
     * @notice The signer set consists of authorized public keys that may challenge the `rootClaim`.
     * @param addr The address to check if it is part of the signer set.
     * @return _isAuthorized Whether or not the `addr` is part of the signer set.
     */
    function signerSet(address addr) external view override returns (bool _isAuthorized) {
        for (uint256 i = 0; i < frozenSignerSet.length; i++) {
            if (frozenSignerSet[i] == addr) {
                _isAuthorized = true;
            }
        }
    }

    /**
     * @notice Challenge the `rootClaim`.
     * @dev - If the `ecrecover`ed address that created the signature is not a part of the
     *      signer set returned by `signerSet`, this function should revert.
     *      - If the signature provided is the signature that breaches
     *      the signature threshold, the function should call the `resolve`
     *      function to resolve the game as `CHALLENGER_WINS`.
     *      - When the game resolves, the bond attached to the root claim
     *      should be distributed among the signers who participated in
     *      challenging the invalid claim.
     * @param signature An EIP-712 signature committing to the `rootClaim`
     *        and `l2BlockNumber` (within the `extraData`) from a key that
     *        exists within the `signerSet`.
     */
    function challenge(bytes calldata signature) external {
        // Attempt to recover the signature provided. Solady's ECDSA library
        // will revert if the signer cannot be recovered from the given signature.
        address recovered = ECDSA.recoverCalldata(Hash.unwrap(getTypedDataHash()), signature);

        // Check that the recovered address is part of the `signerSet`.
        // if (!systemConfig.signerSet(recovered)) {
        if (false) {
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
     *         `Dispute` struct. This hash is signed by members of the `signerSet` to
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

            disputeStructHash := keccak256(ptr, 0x60)
        }

        _typedDataHash = Hash.wrap(_hashTypedData(Hash.unwrap(disputeStructHash)));
    }

    /**
     * @notice Initializes the contract.
     */
    function initialize() external initializer {
        createdAt = Timestamp.wrap(uint64(block.timestamp));
        frozenSignerSet = systemConfig.signerSet();
        frozenSignatureThreshold = systemConfig.signatureThreshold();
    }

    /**
     * @notice Returns the semantic version of the DisputeGame contract.
     * @dev Current version: 0.0.1
     * @return The semantic version of the DisputeGame contract.
     */
    function version() external pure override returns (string memory) {
        assembly {
            // Store the pointer to the string
            mstore(returndatasize(), 0x20)
            // Store the version ("0.0.1")
            // len |   "0.0.1"
            // 0x05|302E302E31
            mstore(0x25, 0x05302E302E31)
            // Return the semantic version of the contract
            return(returndatasize(), 0x60)
        }
    }

    /**
     * @notice If all necessary information has been gathered, this function should mark the game
     *         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
     *         the resolved game. It is at this stage that the bonds should be awarded to the
     *         necessary parties.
     * @dev May only be called if the `status` is `IN_PROGRESS`.
     * @return _status The status of the resolved game.
     */
    function resolve() public returns (GameStatus _status) {
        // Set the status as `CHALLENGER_WINS`.
        status = GameStatus.CHALLENGER_WINS;

        // Delete all outputs from [l2BlockNumber, currentL2BlockNumber]
        l2OutputOracle.deleteL2Outputs(l2BlockNumber());

        // Request the `BondManager` to distribute the faulty output bond to the attestors.
        uint256 _l2BlockNumber = l2BlockNumber();
        bondManager.seizeAndSplit(keccak256(abi.encode(_l2BlockNumber)), attestationSubmitters);

        return status;
    }

    /**
     * @notice Returns the type of proof system being used for the AttestationDisputeGame.
     * @dev The reference impl should be entirely different depending on the type (fault, validity)
     *      i.e. The game type should indicate the security model.
     * @return _gameType The type of proof system being used.
     */
    function gameType() public pure override returns (GameType) {
        return GameType.ATTESTATION;
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
        _version = "0.0.1";
    }

    /**
     * @notice Returns the extra data supplied to the dispute game contract by the creator.
     *         This is just the L2 block number that the root claim commits to.
     * @dev `clones-with-immutable-args` argument #3
     * @return _extraData Any extra data supplied to the dispute game contract by the creator.
     */
    function extraData() external pure returns (bytes memory _extraData) {
        _extraData = _getArgDynBytes(0x20, 0x20);
    }

    /**
     * @notice Fetches the root claim from the calldata appended by the CWIA proxy.
     * @dev `clones-with-immutable-args` argument #2
     * @return _rootClaim The root claim of the DisputeGame.
     */
    function rootClaim() public pure returns (Claim _rootClaim) {
        _rootClaim = Claim.wrap(_getArgFixedBytes(0x00));
    }

    /**
     * @notice Fetches the L2 block number that the `rootClaim` commits to.
     *         Exists within the `extraData`.
     * @return _l2BlockNumber The L2 block number that the `rootClaim` commits to.
     */
    function l2BlockNumber() public pure returns (uint256 _l2BlockNumber) {
        _l2BlockNumber = _getArgUint256(0x20);
    }
}
