// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { Hash, Duration } from "src/dispute/lib/LibUDT.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

// Interfaces
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";

/// @title OPContractsManagerMigrationValidator
/// @notice Validates the configuration of L1 contracts after an interop migration. Separated from
///         OPContractsManagerStandardValidator due to EIP-170 contract size limits.
contract OPContractsManagerMigrationValidator is ISemver {
    /// @notice The semantic version of the OPContractsManagerMigrationValidator contract.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Struct containing the input parameters for post-migration validation.
    struct MigrationValidationInput {
        IDisputeGameFactory dgf;
        ISystemConfig[] chainSystemConfigs;
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        address proposer;
        address challenger;
    }

    constructor() { }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    function validateMigration(
        MigrationValidationInput memory _input,
        bool _allowFailure
    )
        external
        view
        returns (string memory)
    {
        return validateMigrationWithOverrides(
            _input,
            _allowFailure,
            IOPContractsManagerStandardValidator.ValidationOverrides({
                l1PAOMultisig: address(0),
                challenger: address(0)
            })
        );
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    ///         Supports overrides of certain storage values denoted in the ValidationOverrides struct.
    function validateMigrationWithOverrides(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides
    )
        public
        view
        returns (string memory)
    {
        // Overrides accepted but unused for now.
        (_overrides);

        string memory _errors = "";

        _errors = assertValidSharedDGFShape(_errors, _input.dgf);
        _errors = assertValidSharedSuperGame(
            _errors, _input.dgf, _input.cannonPrestate, true, _input.proposer, _input.challenger, "MIG-SPDG"
        );
        _errors = assertValidSharedSuperGame(
            _errors, _input.dgf, _input.cannonKonaPrestate, false, address(0), address(0), "MIG-SCKDG"
        );
        _errors = assertValidPerChainMigration(_errors, _input.dgf, _input.chainSystemConfigs);

        if (bytes(_errors).length > 0 && !_allowFailure) {
            revert(string.concat("OPContractsManagerMigrationValidator: ", _errors));
        }

        return _errors;
    }

    /// @notice Validates the shape of the shared DGF — correct game types registered/unregistered.
    function assertValidSharedDGFShape(
        string memory _errors,
        IDisputeGameFactory _dgf
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            address(_dgf.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)) != address(0), "MIG-DGF-10", _errors
        );
        _errors =
            internalRequire(address(_dgf.gameImpls(GameTypes.SUPER_CANNON_KONA)) != address(0), "MIG-DGF-20", _errors);
        _errors = internalRequire(address(_dgf.gameImpls(GameTypes.CANNON)) == address(0), "MIG-DGF-30", _errors);
        _errors =
            internalRequire(address(_dgf.gameImpls(GameTypes.PERMISSIONED_CANNON)) == address(0), "MIG-DGF-40", _errors);
        _errors = internalRequire(address(_dgf.gameImpls(GameTypes.CANNON_KONA)) == address(0), "MIG-DGF-50", _errors);
        _errors = internalRequire(address(_dgf.gameImpls(GameTypes.SUPER_CANNON)) == address(0), "MIG-DGF-60", _errors);
        return _errors;
    }

    /// @notice Validates a single super game type's configuration on the shared DGF.
    function assertValidSharedSuperGame(
        string memory _errors,
        IDisputeGameFactory _dgf,
        bytes32 _expectedPrestate,
        bool _isPermissioned,
        address _proposer,
        address _challenger,
        string memory _prefix
    )
        internal
        view
        returns (string memory)
    {
        GameType gameType = _isPermissioned ? GameTypes.SUPER_PERMISSIONED_CANNON : GameTypes.SUPER_CANNON_KONA;

        // If game impl is address(0), skip — already caught by shape checks.
        address gameImpl = address(_dgf.gameImpls(gameType));
        if (gameImpl == address(0)) return _errors;

        // Validate game args length and decode.
        LibGameArgs.GameArgs memory gameArgs;
        {
            bytes memory gameArgsBytes = _dgf.gameArgs(gameType);
            bool argsOk = _isPermissioned
                ? LibGameArgs.isValidPermissionedArgs(gameArgsBytes)
                : LibGameArgs.isValidPermissionlessArgs(gameArgsBytes);
            _errors = internalRequire(argsOk, string.concat(_prefix, "-GARGS-10"), _errors);
            if (!argsOk) return _errors;
            gameArgs = LibGameArgs.decode(gameArgsBytes);
        }

        // Validate game args fields.
        _errors = internalRequire(gameArgs.l2ChainId == 0, string.concat(_prefix, "-10"), _errors);
        _errors =
            internalRequire(gameArgs.absolutePrestate == _expectedPrestate, string.concat(_prefix, "-20"), _errors);

        // Validate game impl params.
        {
            IPermissionedDisputeGame game = IPermissionedDisputeGame(gameImpl);
            _errors = internalRequire(game.maxGameDepth() == 73, string.concat(_prefix, "-30"), _errors);
            _errors = internalRequire(game.splitDepth() == 30, string.concat(_prefix, "-40"), _errors);
            _errors =
                internalRequire(Duration.unwrap(game.clockExtension()) == 10800, string.concat(_prefix, "-50"), _errors);
            _errors = internalRequire(
                Duration.unwrap(game.maxClockDuration()) == 302400, string.concat(_prefix, "-60"), _errors
            );
            _errors = internalRequire(game.l2SequenceNumber() == 0, string.concat(_prefix, "-70"), _errors);
        }

        // Validate anchor root is non-zero (from ASR in game args).
        {
            (Hash anchorRoot,) = IAnchorStateRegistry(gameArgs.anchorStateRegistry).getAnchorRoot();
            _errors = internalRequire(Hash.unwrap(anchorRoot) != bytes32(0), string.concat(_prefix, "-80"), _errors);
        }

        // Validate proposer and challenger for permissioned games.
        if (_isPermissioned) {
            _errors = internalRequire(gameArgs.proposer == _proposer, string.concat(_prefix, "-90"), _errors);
            _errors = internalRequire(gameArgs.challenger == _challenger, string.concat(_prefix, "-100"), _errors);
        }

        return _errors;
    }

    /// @notice Validates per-chain migration state: portal ASR, per-chain DGF cleared, lockbox auth.
    function assertValidPerChainMigration(
        string memory _errors,
        IDisputeGameFactory _sharedDGF,
        ISystemConfig[] memory _chainSystemConfigs
    )
        internal
        view
        returns (string memory)
    {
        if (_chainSystemConfigs.length == 0) {
            return internalRequire(false, "MIG-CHAIN-EMPTY", _errors);
        }

        // Derive shared ASR from SUPER_PERMISSIONED_CANNON game args on shared DGF.
        address sharedASR;
        {
            address spdgImpl = address(_sharedDGF.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON));
            if (spdgImpl == address(0)) {
                // Can't derive shared ASR — already caught by MIG-DGF-10. Skip per-chain checks.
                return _errors;
            }
            bytes memory spdgArgs = _sharedDGF.gameArgs(GameTypes.SUPER_PERMISSIONED_CANNON);
            if (!LibGameArgs.isValidPermissionedArgs(spdgArgs)) {
                // Can't decode — already caught by MIG-SPDG-GARGS-10. Skip per-chain checks.
                return _errors;
            }
            LibGameArgs.GameArgs memory args = LibGameArgs.decode(spdgArgs);
            sharedASR = args.anchorStateRegistry;
        }

        // Derive shared lockbox from first chain's portal.
        IETHLockbox sharedLockbox;
        {
            IOptimismPortal2 firstPortal = IOptimismPortal2(payable(_chainSystemConfigs[0].optimismPortal()));
            sharedLockbox = firstPortal.ethLockbox();
        }

        for (uint256 i = 0; i < _chainSystemConfigs.length; i++) {
            string memory idx = LibString.toString(i);

            IOptimismPortal2 portal = IOptimismPortal2(payable(_chainSystemConfigs[i].optimismPortal()));

            // Portal's ASR should point to the shared ASR.
            _errors = internalRequire(
                address(portal.anchorStateRegistry()) == sharedASR, string.concat("MIG-CHAIN-", idx, "-10"), _errors
            );

            // Per-chain DGF should have all game types cleared.
            IDisputeGameFactory perChainDGF = IDisputeGameFactory(_chainSystemConfigs[i].disputeGameFactory());
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.CANNON)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-20"),
                _errors
            );
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.PERMISSIONED_CANNON)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-30"),
                _errors
            );
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.CANNON_KONA)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-40"),
                _errors
            );
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.SUPER_CANNON)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-50"),
                _errors
            );
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-60"),
                _errors
            );
            _errors = internalRequire(
                address(perChainDGF.gameImpls(GameTypes.SUPER_CANNON_KONA)) == address(0),
                string.concat("MIG-CHAIN-", idx, "-70"),
                _errors
            );

            // Portal should be authorized in the shared lockbox.
            _errors = internalRequire(
                sharedLockbox.authorizedPortals(portal), string.concat("MIG-CHAIN-", idx, "-80"), _errors
            );

            // Portal's lockbox should match the shared lockbox.
            _errors = internalRequire(
                address(portal.ethLockbox()) == address(sharedLockbox), string.concat("MIG-CHAIN-", idx, "-90"), _errors
            );
        }

        return _errors;
    }

    /// @notice Internal function to require a condition to be true, otherwise append an error message.
    function internalRequire(
        bool _condition,
        string memory _message,
        string memory _errors
    )
        internal
        pure
        returns (string memory)
    {
        if (_condition) {
            return _errors;
        }
        if (bytes(_errors).length == 0) {
            _errors = _message;
        } else {
            _errors = string.concat(_errors, ",", _message);
        }
        return _errors;
    }
}
