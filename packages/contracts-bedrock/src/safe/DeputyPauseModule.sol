// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Contracts
import { EIP712 } from "@openzeppelin/contracts-v5/utils/cryptography/EIP712.sol";

// Libraries
import { ECDSA } from "@openzeppelin/contracts-v5/utils/cryptography/ECDSA.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @title DeputyPauseModule
/// @notice Safe Module designed to be installed into the Guardian Safe which allows a specific
///         deputy address to act as the Guardian Safe for the sake of triggering a pause within
///         the SuperchainConfig contract.
contract DeputyPauseModule is ISemver, EIP712 {
    /// @notice Error message for deputy being invalid.
    error DeputyPauseModule_InvalidDeputy();

    /// @notice Error message for unauthorized calls.
    error DeputyPauseModule_Unauthorized();

    /// @notice Error message for nonce reuse.
    error DeputyPauseModule_NonceAlreadyUsed();

    /// @notice Error message for failed transaction execution.
    error DeputyPauseModule_ExecutionFailed(string);

    /// @notice Error message for the SuperchainConfig not being paused.
    error DeputyPauseModule_SuperchainNotPaused();

    /// @notice Error message for the call not being from the Foundation Safe.
    error DeputyPauseModule_NotFromSafe();

    /// @notice Struct for the Pause action.
    /// @custom:field nonce Signature nonce.
    /// @custom:field identifier Identifier to pause.
    struct PauseMessage {
        bytes32 nonce;
        address identifier;
    }

    /// @notice Struct for the DeputyAuth action.
    /// @custom:field deputy Address of the deputy account.
    struct DeputyAuthMessage {
        address deputy;
    }

    /// @notice Event emitted when the deputy address is set.
    event DeputySet(address indexed deputy);

    /// @notice Event emitted when the pause is triggered.
    event PauseTriggered(address indexed deputy, bytes32 nonce, address identifier);

    /// @notice Guardian Safe.
    Safe internal immutable GUARDIAN_SAFE;

    /// @notice Foundation Safe.
    Safe internal immutable FOUNDATION_SAFE;

    /// @notice SuperchainConfig contract.
    ISuperchainConfig internal immutable SUPERCHAIN_CONFIG;

    /// @notice Typehash for the Pause action.
    bytes32 internal constant PAUSE_MESSAGE_TYPEHASH = keccak256("PauseMessage(bytes32 nonce,address identifier)");

    /// @notice Typehash for the DeputyAuth message.
    bytes32 internal constant DEPUTY_AUTH_MESSAGE_TYPEHASH = keccak256("DeputyAuthMessage(address deputy)");

    /// @notice Address of the Deputy account.
    address public deputy;

    /// @notice Used nonces.
    mapping(bytes32 => bool) public usedNonces;

    /// @notice Semantic version.
    /// @custom:semver 2.1.0
    string public constant version = "2.1.0";

    /// @param _guardianSafe Address of the Guardian Safe.
    /// @param _foundationSafe Address of the Foundation Safe.
    /// @param _superchainConfig Address of the SuperchainConfig contract.
    /// @param _deputy Address of the deputy account.
    /// @param _deputySignature Signature from the deputy verifying that the account is an EOA.
    constructor(
        Safe _guardianSafe,
        Safe _foundationSafe,
        ISuperchainConfig _superchainConfig,
        address _deputy,
        bytes memory _deputySignature
    )
        EIP712("DeputyPauseModule", "1")
    {
        _setDeputy(_deputy, _deputySignature);
        GUARDIAN_SAFE = _guardianSafe;
        FOUNDATION_SAFE = _foundationSafe;
        SUPERCHAIN_CONFIG = _superchainConfig;
    }

    /// @notice Getter function for the Guardian Safe address.
    /// @return guardianSafe_ Guardian Safe address.
    function guardianSafe() public view returns (Safe guardianSafe_) {
        guardianSafe_ = GUARDIAN_SAFE;
    }

    /// @notice Getter function for the Foundation Safe address.
    /// @return foundationSafe_ Foundation Safe address.
    function foundationSafe() public view returns (Safe foundationSafe_) {
        foundationSafe_ = FOUNDATION_SAFE;
    }

    /// @notice Getter function for the SuperchainConfig address.
    /// @return superchainConfig_ SuperchainConfig address.
    function superchainConfig() public view returns (ISuperchainConfig superchainConfig_) {
        superchainConfig_ = SUPERCHAIN_CONFIG;
    }

    /// @notice Getter function for the Pause message typehash.
    /// @return pauseMessageTypehash_ Pause message typehash.
    function pauseMessageTypehash() public pure returns (bytes32 pauseMessageTypehash_) {
        pauseMessageTypehash_ = PAUSE_MESSAGE_TYPEHASH;
    }

    /// @notice Getter function for the DeputyAuth message typehash.
    /// @return deputyAuthMessageTypehash_ DeputyAuth message typehash.
    function deputyAuthMessageTypehash() public pure returns (bytes32 deputyAuthMessageTypehash_) {
        deputyAuthMessageTypehash_ = DEPUTY_AUTH_MESSAGE_TYPEHASH;
    }

    /// @notice Sets the deputy address.
    /// @param _deputy Deputy address.
    /// @param _deputySignature Deputy signature.
    function setDeputy(address _deputy, bytes memory _deputySignature) external {
        // Can only be called by the Foundation Safe itself.
        if (msg.sender != address(FOUNDATION_SAFE)) {
            revert DeputyPauseModule_NotFromSafe();
        }

        // Set the deputy address.
        _setDeputy(_deputy, _deputySignature);
    }

    /// @notice Calls the Guardian Safe's `execTransactionFromModuleReturnData()` function with
    ///         the arguments necessary to call `pause()` on the SuperchainConfig.
    ///         Front-running this function is completely safe, it'll pause either way.
    /// @param _nonce Signature nonce.
    /// @param _identifier The identifier to pause in the SuperchainConfig.
    /// @param _signature ECDSA signature.
    function pause(bytes32 _nonce, address _identifier, bytes memory _signature) external {
        // Verify the signature.
        bytes32 digest = _hashTypedDataV4(keccak256(abi.encode(PAUSE_MESSAGE_TYPEHASH, _nonce, _identifier)));
        if (ECDSA.recover(digest, _signature) != deputy) {
            revert DeputyPauseModule_Unauthorized();
        }

        // Make sure the nonce hasn't been used yet.
        if (usedNonces[_nonce]) {
            revert DeputyPauseModule_NonceAlreadyUsed();
        }

        // Mark the nonce as used.
        usedNonces[_nonce] = true;

        // Attempt to trigger the call.
        (bool success, bytes memory returnData) = GUARDIAN_SAFE.execTransactionFromModuleReturnData(
            address(SUPERCHAIN_CONFIG), 0, abi.encodeCall(ISuperchainConfig.pause, (_identifier)), Enum.Operation.Call
        );

        // If the call fails, revert.
        if (!success) {
            revert DeputyPauseModule_ExecutionFailed(string(returnData));
        }

        // Verify that the SuperchainConfig is now paused.
        if (!SUPERCHAIN_CONFIG.paused(_identifier)) {
            revert DeputyPauseModule_SuperchainNotPaused();
        }

        // Emit that the pause was triggered.
        emit PauseTriggered(deputy, _nonce, _identifier);
    }

    /// @notice Internal function to set the deputy address.
    /// @param _deputy Deputy address.
    /// @param _deputySignature Deputy signature.
    function _setDeputy(address _deputy, bytes memory _deputySignature) internal {
        // Check that the deputy signature is valid.
        bytes32 digest = _hashTypedDataV4(keccak256(abi.encode(DEPUTY_AUTH_MESSAGE_TYPEHASH, _deputy)));
        if (ECDSA.recover(digest, _deputySignature) != _deputy) {
            revert DeputyPauseModule_InvalidDeputy();
        }

        // Set the deputy address.
        deputy = _deputy;

        // Emit the DeputySet event.
        emit DeputySet(_deputy);
    }
}
