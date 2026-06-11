// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { Features } from "src/libraries/Features.sol";
import { GameType, GameTypes, Claim } from "src/dispute/lib/Types.sol";
import { LibGameArgs } from "src/dispute/lib/LibGameArgs.sol";

// Interfaces
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import {
    DisputeGameImplementation,
    DisputeGameValidationArgs,
    DisputeGameImpls,
    DisputeGameConfig,
    SuperPermissionedDisputeGameImplementation,
    SuperPermissionedDisputeGameImpls,
    SuperPermissionedDisputeGameValidationArgs
} from "src/L1/opcm/StandardValidatorUtils.sol";
import { IOPContractsManagerMigrationValidator } from "interfaces/L1/opcm/IOPContractsManagerMigrationValidator.sol";

/// @title OPContractsManagerMigrationValidator
/// @notice Validates the configuration of L1 contracts after an interop migration. Separated from
///         OPContractsManagerStandardValidator due to EIP-170 contract size limits.
///         This validator checks migration-specific state: shared DGF game type registration,
///         shared DGF proxy/impl/owner, super game parameters (delegated to StandardValidatorUtils
///         for game-specific validation), shared
///         lockbox proxy/impl, per-chain portal ASR migration, per-chain DGF clearing, and
///         lockbox authorization.
contract OPContractsManagerMigrationValidator {
    /// @notice Discovered shared contracts from on-chain state.
    struct SharedContracts {
        IProxyAdmin proxyAdmin;
        address asr;
        address weth;
        IETHLockbox lockbox;
    }

    /// @notice Parameters for shared super permissioned dispute game validation.
    struct SuperPermissionedGameParams {
        IDisputeGameFactory dgf;
        ISystemConfig sysCfg;
        IProxyAdmin proxyAdmin;
        address proposer;
        string prefix;
        address expectedGameImpl;
    }

    /// @notice Parameters for shared super permissionless dispute game validation.
    struct SuperPermissionlessGameParams {
        IDisputeGameFactory dgf;
        ISystemConfig sysCfg;
        IProxyAdmin proxyAdmin;
        bytes32 expectedPrestate;
        string prefix;
        address discoveredWeth;
        address expectedGameImpl;
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    function validateMigration(
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls,
        IOPContractsManagerMigrationValidator.SharedConfig memory _cfg
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
            }),
            _impls,
            _cfg
        );
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    ///         Supports overrides of certain storage values denoted in the ValidationOverrides struct.
    function validateMigrationWithOverrides(
        IOPContractsManagerMigrationValidator.MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls,
        IOPContractsManagerMigrationValidator.SharedConfig memory _cfg
    )
        public
        view
        returns (string memory)
    {
        if (_overrides.l1PAOMultisig != address(0)) {
            _cfg.l1PAOMultisig = _overrides.l1PAOMultisig;
        }
        if (_overrides.challenger != address(0)) {
            _cfg.challenger = _overrides.challenger;
        }

        string memory _errors = "";

        // Game type registration shape on the shared DGF.
        _errors = assertValidSharedDGFShape(_errors, _input.dgf);

        // Discover the shared contracts (ProxyAdmin / ASR / WETH / lockbox) from chain[0]'s state.
        SharedContracts memory _sharedContracts = _getSharedContracts(_input.dgf, _input.chainSystemConfigs);
        bool _foundSharedContracts = address(_sharedContracts.proxyAdmin) != address(0);

        // Shared DGF proxy/impl/owner. Plus one ASR invariant the super-game drill-down
        // does not cover: respectedGameType must be a super game type post-migration.
        if (_foundSharedContracts) {
            _errors = assertValidSharedDGF(_errors, _input.dgf, _sharedContracts.proxyAdmin, _impls, _cfg);
            _errors = internalRequire(
                GameTypes.isSuperGame(IAnchorStateRegistry(_sharedContracts.asr).respectedGameType()),
                "MIG-SASR-RGT",
                _errors
            );
        }

        ISystemConfig firstCfg =
            _input.chainSystemConfigs.length > 0 ? _input.chainSystemConfigs[0] : ISystemConfig(address(0));

        // Super game checks. Always run — the validators handle missing
        // impls/args internally so a partially-broken setup still surfaces all errors.
        _errors = assertValidSharedSuperPermissionedGame(
            _errors,
            SuperPermissionedGameParams({
                dgf: _input.dgf,
                sysCfg: firstCfg,
                proxyAdmin: _sharedContracts.proxyAdmin,
                proposer: _input.proposer,
                prefix: "MIG-SPDG",
                expectedGameImpl: _impls.superPermissionedDisputeGameImpl
            }),
            _impls
        );
        _errors = assertValidSharedSuperPermissionlessGame(
            _errors,
            SuperPermissionlessGameParams({
                dgf: _input.dgf,
                sysCfg: firstCfg,
                proxyAdmin: _sharedContracts.proxyAdmin,
                expectedPrestate: _input.cannonKonaPrestate,
                prefix: "MIG-SCKDG",
                discoveredWeth: _sharedContracts.weth,
                expectedGameImpl: _impls.superFaultDisputeGameImpl
            }),
            _impls,
            _cfg
        );

        // Shared lockbox proxy/impl/admin (only if discovery surfaced one).
        if (_foundSharedContracts && address(_sharedContracts.lockbox) != address(0)) {
            _errors = assertValidSharedLockbox(_errors, _sharedContracts.lockbox, _sharedContracts.proxyAdmin, _impls);
        }

        // Per-chain invariants (portal points at shared ASR/lockbox, legacy game types cleared).
        _errors = assertValidPerChainMigration(_errors, _input.chainSystemConfigs);

        if (bytes(_errors).length > 0 && !_allowFailure) {
            revert(string.concat("OPContractsManagerMigrationValidator: ", _errors));
        }

        return _errors;
    }

    /// @notice Discovers shared contracts from on-chain state.
    ///         Returns zero-initialized struct if no chains are provided.
    function _getSharedContracts(
        IDisputeGameFactory _dgf,
        ISystemConfig[] memory _chainSystemConfigs
    )
        internal
        view
        returns (SharedContracts memory)
    {
        if (_chainSystemConfigs.length == 0) {
            return SharedContracts(IProxyAdmin(address(0)), address(0), address(0), IETHLockbox(address(0)));
        }

        IProxyAdmin proxyAdmin = IProxyAdminOwnedBase(address(_dgf)).proxyAdmin();
        IOptimismPortal2 firstPortal = IOptimismPortal2(payable(_chainSystemConfigs[0].optimismPortal()));

        return SharedContracts({
            proxyAdmin: proxyAdmin,
            asr: address(firstPortal.anchorStateRegistry()),
            weth: _chainSystemConfigs[0].delayedWETH(),
            lockbox: firstPortal.ethLockbox()
        });
    }

    /// @notice Validates the shape of the shared DGF — correct game types registered/unregistered.
    ///         Post-migration interop requires SUPER_PERMISSIONED_CANNON and SUPER_CANNON_KONA
    ///         registered; all legacy game types (CANNON, PERMISSIONED_CANNON, CANNON_KONA,
    ///         SUPER_CANNON) unregistered.
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

    /// @notice Validates the shared DGF proxy implementation, version, owner, and ProxyAdmin.
    function assertValidSharedDGF(
        string memory _errors,
        IDisputeGameFactory _dgf,
        IProxyAdmin _proxyAdmin,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls,
        IOPContractsManagerMigrationValidator.SharedConfig memory _cfg
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(address(_dgf)).version(), ISemver(_impls.disputeGameFactoryImpl).version()),
            "MIG-SDGF-10",
            _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(address(_dgf)) == _impls.disputeGameFactoryImpl, "MIG-SDGF-20", _errors
        );
        _errors = internalRequire(_dgf.owner() == _cfg.l1PAOMultisig, "MIG-SDGF-30", _errors);
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(address(_dgf)).proxyAdmin()) == address(_proxyAdmin), "MIG-SDGF-40", _errors
        );
        return _errors;
    }

    /// @notice Validates the shared super permissioned dispute game.
    function assertValidSharedSuperPermissionedGame(
        string memory _errors,
        SuperPermissionedGameParams memory _p,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls
    )
        internal
        view
        returns (string memory)
    {
        // If game impl is address(0), skip — already caught by shape checks.
        address gameImplAddr = address(_p.dgf.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON));
        if (gameImplAddr == address(0)) return _errors;

        bytes memory gameArgsBytes = _p.dgf.gameArgs(GameTypes.SUPER_PERMISSIONED_CANNON);
        bool argsOk = LibGameArgs.isValidSuperPermissionedArgs(gameArgsBytes);
        _errors = internalRequire(argsOk, string.concat(_p.prefix, "-GARGS-10"), _errors);
        if (!argsOk) return _errors;

        LibGameArgs.SuperPermissionedGameArgs memory gameArgs = LibGameArgs.decodeSuperPermissioned(gameArgsBytes);

        // Delegate full fault dispute game validation (impls, drill-downs) to the shared utility.
        // Skip when we have no sysCfg (empty chain list) — nothing meaningful to validate against.
        if (address(_p.sysCfg) != address(0)) {
            _errors = _delegateSuperPermissionedGameValidation(_errors, _p, gameImplAddr, gameArgs, _impls);
        }

        return _errors;
    }

    /// @notice Validates the shared super permissionless dispute game.
    function assertValidSharedSuperPermissionlessGame(
        string memory _errors,
        SuperPermissionlessGameParams memory _p,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls,
        IOPContractsManagerMigrationValidator.SharedConfig memory _cfg
    )
        internal
        view
        returns (string memory)
    {
        // If game impl is address(0), skip — already caught by shape checks.
        address gameImplAddr = address(_p.dgf.gameImpls(GameTypes.SUPER_CANNON_KONA));
        if (gameImplAddr == address(0)) return _errors;

        bytes memory gameArgsBytes = _p.dgf.gameArgs(GameTypes.SUPER_CANNON_KONA);
        bool argsOk = LibGameArgs.isValidPermissionlessArgs(gameArgsBytes);
        _errors = internalRequire(argsOk, string.concat(_p.prefix, "-GARGS-10"), _errors);
        if (!argsOk) return _errors;

        LibGameArgs.GameArgs memory gameArgs = LibGameArgs.decode(gameArgsBytes);

        // Migration-unique cross-check: super game's weth must match the first chain's weth.
        if (_p.discoveredWeth != address(0)) {
            _errors =
                internalRequire(gameArgs.weth == _p.discoveredWeth, string.concat(_p.prefix, "-GARGS-30"), _errors);
        }

        // Delegate full fault dispute game validation (impls, drill-downs) to the shared utility.
        // Skip when we have no sysCfg (empty chain list) — nothing meaningful to validate against.
        if (address(_p.sysCfg) != address(0)) {
            _errors = _delegateSuperPermissionlessGameValidation(_errors, _p, gameImplAddr, gameArgs, _impls, _cfg);
        }

        return _errors;
    }

    /// @notice Calls the simplified super permissioned dispute game validator.
    function _delegateSuperPermissionedGameValidation(
        string memory _errors,
        SuperPermissionedGameParams memory _p,
        address _gameImplAddr,
        LibGameArgs.SuperPermissionedGameArgs memory _gameArgs,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls
    )
        private
        view
        returns (string memory)
    {
        return _impls.standardValidatorUtils.assertValidSuperPermissionedDisputeGame(
            SuperPermissionedDisputeGameValidationArgs({
                errors: _errors,
                sysCfg: _p.sysCfg,
                game: SuperPermissionedDisputeGameImplementation({
                    gameAddress: _gameImplAddr,
                    asr: IAnchorStateRegistry(_gameArgs.anchorStateRegistry),
                    proposer: _gameArgs.proposer
                }),
                admin: _p.proxyAdmin,
                expectedProposer: _p.proposer,
                errorPrefix: _p.prefix
            }),
            SuperPermissionedDisputeGameImpls({
                expectedGameImpl: _p.expectedGameImpl,
                anchorStateRegistryImpl: _impls.anchorStateRegistryImpl
            })
        );
    }

    /// @notice Builds the struct payloads and calls `standardValidatorUtils.assertValidDisputeGame`.
    function _delegateSuperPermissionlessGameValidation(
        string memory _errors,
        SuperPermissionlessGameParams memory _p,
        address _gameImplAddr,
        LibGameArgs.GameArgs memory _gameArgs,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls,
        IOPContractsManagerMigrationValidator.SharedConfig memory _cfg
    )
        private
        view
        returns (string memory)
    {
        return _impls.standardValidatorUtils.assertValidDisputeGame(
            DisputeGameValidationArgs({
                errors: _errors,
                sysCfg: _p.sysCfg,
                game: _readSharedSuperGameImpl(GameTypes.SUPER_CANNON_KONA, _gameImplAddr, _gameArgs),
                absolutePrestate: _p.expectedPrestate,
                l2ChainID: 0,
                admin: _p.proxyAdmin,
                gameType: GameTypes.SUPER_CANNON_KONA,
                errorPrefix: _p.prefix
            }),
            DisputeGameImpls({
                expectedGameImpl: _p.expectedGameImpl,
                mipsImpl: _impls.mipsImpl,
                delayedWETHImpl: _impls.delayedWETHImpl,
                anchorStateRegistryImpl: _impls.anchorStateRegistryImpl
            }),
            DisputeGameConfig({ l1PAOMultisig: _cfg.l1PAOMultisig, withdrawalDelaySeconds: _cfg.withdrawalDelaySeconds })
        );
    }

    /// @notice Reads the on-chain fields of a super game impl into a `DisputeGameImplementation`.
    function _readSharedSuperGameImpl(
        GameType _gameType,
        address _gameImplAddr,
        LibGameArgs.GameArgs memory _gameArgs
    )
        private
        view
        returns (DisputeGameImplementation memory gameImpl_)
    {
        IFaultDisputeGame game = IFaultDisputeGame(_gameImplAddr);
        gameImpl_ = DisputeGameImplementation({
            gameAddress: _gameImplAddr,
            maxGameDepth: game.maxGameDepth(),
            splitDepth: game.splitDepth(),
            maxClockDuration: game.maxClockDuration(),
            clockExtension: game.clockExtension(),
            gameType: _gameType,
            l2SequenceNumber: game.l2SequenceNumber(),
            absolutePrestate: Claim.wrap(_gameArgs.absolutePrestate),
            vm: IBigStepper(_gameArgs.vm),
            asr: IAnchorStateRegistry(_gameArgs.anchorStateRegistry),
            weth: IDelayedWETH(payable(_gameArgs.weth)),
            l2ChainId: _gameArgs.l2ChainId,
            challenger: _gameArgs.challenger,
            proposer: _gameArgs.proposer
        });
    }

    /// @notice Validates the shared ETHLockbox: version, proxy impl, ProxyAdmin.
    function assertValidSharedLockbox(
        string memory _errors,
        IETHLockbox _lockbox,
        IProxyAdmin _proxyAdmin,
        IOPContractsManagerMigrationValidator.SharedImplementations memory _impls
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(address(_lockbox)).version(), ISemver(_impls.ethLockboxImpl).version()),
            "MIG-SLOCKBOX-10",
            _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(address(_lockbox)) == _impls.ethLockboxImpl, "MIG-SLOCKBOX-20", _errors
        );
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(address(_lockbox)).proxyAdmin()) == address(_proxyAdmin),
            "MIG-SLOCKBOX-30",
            _errors
        );
        return _errors;
    }

    /// @notice Validates per-chain migration state: portal ASR, per-chain DGF cleared, lockbox auth.
    function assertValidPerChainMigration(
        string memory _errors,
        ISystemConfig[] memory _chainSystemConfigs
    )
        internal
        view
        returns (string memory)
    {
        if (_chainSystemConfigs.length == 0) {
            return internalRequire(false, "MIG-CHAIN-EMPTY", _errors);
        }

        // Derive shared ASR, DGF, and lockbox from first chain.
        IOptimismPortal2 firstPortal = IOptimismPortal2(payable(_chainSystemConfigs[0].optimismPortal()));
        address sharedASR = address(firstPortal.anchorStateRegistry());
        IETHLockbox sharedLockbox = firstPortal.ethLockbox();
        IDisputeGameFactory sharedDGF = IDisputeGameFactory(_chainSystemConfigs[0].disputeGameFactory());
        address sharedWETH = _chainSystemConfigs[0].delayedWETH();

        // Guard against missing lockbox — would revert on authorizedPortals call.
        if (address(sharedLockbox) == address(0)) {
            return internalRequire(false, "MIG-LOCKBOX-MISSING", _errors);
        }

        for (uint256 i = 0; i < _chainSystemConfigs.length; i++) {
            string memory idx = LibString.toString(i);

            IOptimismPortal2 portal = IOptimismPortal2(payable(_chainSystemConfigs[i].optimismPortal()));

            _errors = internalRequire(
                address(portal.anchorStateRegistry()) == sharedASR, string.concat("MIG-CHAIN-", idx, "-10"), _errors
            );

            IDisputeGameFactory perChainDGF = IDisputeGameFactory(_chainSystemConfigs[i].disputeGameFactory());
            if (address(perChainDGF) != address(sharedDGF)) {
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
            }

            _errors = internalRequire(
                sharedLockbox.authorizedPortals(portal), string.concat("MIG-CHAIN-", idx, "-80"), _errors
            );

            _errors = internalRequire(
                address(portal.ethLockbox()) == address(sharedLockbox), string.concat("MIG-CHAIN-", idx, "-90"), _errors
            );

            _errors = internalRequire(
                _chainSystemConfigs[i].isFeatureEnabled(Features.INTEROP),
                string.concat("MIG-CHAIN-", idx, "-100"),
                _errors
            );
            _errors = internalRequire(
                _chainSystemConfigs[i].isFeatureEnabled(Features.ETH_LOCKBOX),
                string.concat("MIG-CHAIN-", idx, "-110"),
                _errors
            );

            _errors = internalRequire(
                _chainSystemConfigs[i].delayedWETH() == sharedWETH, string.concat("MIG-CHAIN-", idx, "-120"), _errors
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
        returns (string memory errors_)
    {
        if (_condition) {
            return _errors;
        }
        if (bytes(_errors).length == 0) {
            errors_ = _message;
        } else {
            errors_ = string.concat(_errors, ",", _message);
        }
    }
}
