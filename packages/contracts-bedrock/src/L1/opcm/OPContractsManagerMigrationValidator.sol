// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { Features } from "src/libraries/Features.sol";
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
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import {
    EXPECTED_MAX_GAME_DEPTH,
    EXPECTED_SPLIT_DEPTH,
    EXPECTED_CLOCK_EXTENSION,
    EXPECTED_MAX_CLOCK_DURATION
} from "src/L1/opcm/StandardValidatorUtils.sol";

/// @title OPContractsManagerMigrationValidator
/// @notice Validates the configuration of L1 contracts after an interop migration. Separated from
///         OPContractsManagerStandardValidator due to EIP-170 contract size limits.
///         This validator checks migration-specific state: shared DGF game type registration,
///         shared DGF proxy/impl/owner, super game parameters (prestates, depths, clocks, anchor
///         roots, roles, VM, WETH), shared ASR configuration, shared lockbox proxy/impl, shared
///         DelayedWETH configuration, per-chain portal ASR migration, per-chain DGF clearing,
///         and lockbox authorization.
contract OPContractsManagerMigrationValidator {
    /// @notice Reference addresses and config for shared infrastructure validation.
    struct SharedImplementations {
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address ethLockboxImpl;
        address delayedWETHImpl;
        address mipsImpl;
        address l1PAOMultisig;
        uint256 withdrawalDelaySeconds;
    }

    /// @notice Discovered shared contracts from on-chain state.
    struct SharedContracts {
        IProxyAdmin proxyAdmin;
        address asr;
        address weth;
        IETHLockbox lockbox;
    }

    /// @notice Struct containing the input parameters for post-migration validation.
    struct MigrationValidationInput {
        IDisputeGameFactory dgf;
        ISystemConfig[] chainSystemConfigs;
        bytes32 cannonPrestate;
        bytes32 cannonKonaPrestate;
        address proposer;
        address challenger;
    }

    /// @notice Parameters for a single super game type validation.
    struct SuperGameParams {
        IDisputeGameFactory dgf;
        bytes32 expectedPrestate;
        bool isPermissioned;
        address proposer;
        address challenger;
        string prefix;
        address mipsImpl;
        address discoveredWeth;
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    function validateMigration(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        SharedImplementations memory _refs
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
            _refs
        );
    }

    /// @notice Validates the configuration of all L1 contracts after an interop migration.
    ///         Supports overrides of certain storage values denoted in the ValidationOverrides struct.
    function validateMigrationWithOverrides(
        MigrationValidationInput memory _input,
        bool _allowFailure,
        IOPContractsManagerStandardValidator.ValidationOverrides memory _overrides,
        SharedImplementations memory _refs
    )
        public
        view
        returns (string memory)
    {
        if (_overrides.l1PAOMultisig != address(0)) {
            _refs.l1PAOMultisig = _overrides.l1PAOMultisig;
        }
        if (_overrides.challenger != address(0)) {
            _input.challenger = _overrides.challenger;
        }

        string memory _errors = "";

        // 1. Check DGF shape (game type registration).
        _errors = assertValidSharedDGFShape(_errors, _input.dgf);

        // 2. Discover shared contracts from on-chain state.
        SharedContracts memory _sharedContracts = _getSharedContracts(_input.dgf, _input.chainSystemConfigs);
        bool _foundSharedContracts = address(_sharedContracts.proxyAdmin) != address(0);

        // 3. If discovery succeeded, run shared DGF proxy/impl/owner checks.
        if (_foundSharedContracts) {
            _errors = assertValidSharedDGF(_errors, _input.dgf, _sharedContracts.proxyAdmin, _refs);
        }

        // 4. Super game checks (always run — they handle missing impls/args internally).
        _errors = assertValidSharedSuperGame(
            _errors,
            SuperGameParams({
                dgf: _input.dgf,
                expectedPrestate: _input.cannonPrestate,
                isPermissioned: true,
                proposer: _input.proposer,
                challenger: _input.challenger,
                prefix: "MIG-SPDG",
                mipsImpl: _refs.mipsImpl,
                discoveredWeth: _sharedContracts.weth
            })
        );
        _errors = assertValidSharedSuperGame(
            _errors,
            SuperGameParams({
                dgf: _input.dgf,
                expectedPrestate: _input.cannonKonaPrestate,
                isPermissioned: false,
                proposer: address(0),
                challenger: address(0),
                prefix: "MIG-SCKDG",
                mipsImpl: _refs.mipsImpl,
                discoveredWeth: _sharedContracts.weth
            })
        );

        // 5. If discovery succeeded, run remaining shared infra checks.
        if (_foundSharedContracts) {
            // Shared ASR checks (includes SCKDG cross-validation).
            _errors =
                assertValidSharedASR(_errors, _sharedContracts.asr, _input.dgf, _sharedContracts.proxyAdmin, _refs);

            // Shared lockbox checks (skip if lockbox is address(0) — caught by MIG-LOCKBOX-MISSING downstream).
            if (address(_sharedContracts.lockbox) != address(0)) {
                _errors =
                    assertValidSharedLockbox(_errors, _sharedContracts.lockbox, _sharedContracts.proxyAdmin, _refs);
            }

            // Shared DelayedWETH checks (skip if no chains — weth can't be discovered).
            if (_sharedContracts.weth != address(0)) {
                _errors =
                    assertValidSharedDelayedWETH(_errors, _sharedContracts.weth, _sharedContracts.proxyAdmin, _refs);
            }
        }

        // Per-chain migration checks.
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
    ///         registered; all legacy game types (CANNON, PERMISSIONED_CANNON, CANNON_KONA)
    ///         unregistered. SUPER_CANNON status pending #20030.
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
        // TODO(#20030): Once SUPER_CANNON is disabled in migrator, re-add check that SUPER_CANNON == address(0).
        // _errors =
        //     internalRequire(address(_dgf.gameImpls(GameTypes.SUPER_CANNON)) == address(0), "MIG-DGF-60", _errors);
        return _errors;
    }

    /// @notice Validates the shared DGF proxy implementation, version, owner, and ProxyAdmin.
    function assertValidSharedDGF(
        string memory _errors,
        IDisputeGameFactory _dgf,
        IProxyAdmin _proxyAdmin,
        SharedImplementations memory _refs
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(address(_dgf)).version(), ISemver(_refs.disputeGameFactoryImpl).version()),
            "MIG-SDGF-10",
            _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(address(_dgf)) == _refs.disputeGameFactoryImpl, "MIG-SDGF-20", _errors
        );
        _errors = internalRequire(_dgf.owner() == _refs.l1PAOMultisig, "MIG-SDGF-30", _errors);
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(address(_dgf)).proxyAdmin()) == address(_proxyAdmin), "MIG-SDGF-40", _errors
        );
        return _errors;
    }

    /// @notice Validates a single super game type's configuration on the shared DGF.
    function assertValidSharedSuperGame(
        string memory _errors,
        SuperGameParams memory _p
    )
        internal
        view
        returns (string memory)
    {
        GameType gameType = _p.isPermissioned ? GameTypes.SUPER_PERMISSIONED_CANNON : GameTypes.SUPER_CANNON_KONA;

        // If game impl is address(0), skip — already caught by shape checks.
        address gameImpl = address(_p.dgf.gameImpls(gameType));
        if (gameImpl == address(0)) return _errors;

        // Validate game args length and decode.
        LibGameArgs.GameArgs memory gameArgs;
        {
            bytes memory gameArgsBytes = _p.dgf.gameArgs(gameType);
            bool argsOk = _p.isPermissioned
                ? LibGameArgs.isValidPermissionedArgs(gameArgsBytes)
                : LibGameArgs.isValidPermissionlessArgs(gameArgsBytes);
            _errors = internalRequire(argsOk, string.concat(_p.prefix, "-GARGS-10"), _errors);
            if (!argsOk) return _errors;
            gameArgs = LibGameArgs.decode(gameArgsBytes);
        }

        // Validate game args fields.
        _errors = internalRequire(gameArgs.l2ChainId == 0, string.concat(_p.prefix, "-10"), _errors);
        _errors =
            internalRequire(gameArgs.absolutePrestate == _p.expectedPrestate, string.concat(_p.prefix, "-20"), _errors);
        _errors = internalRequire(gameArgs.vm == _p.mipsImpl, string.concat(_p.prefix, "-GARGS-20"), _errors);
        // Skip weth cross-check if discoveredWeth is address(0) (no chains provided, can't discover).
        if (_p.discoveredWeth != address(0)) {
            _errors =
                internalRequire(gameArgs.weth == _p.discoveredWeth, string.concat(_p.prefix, "-GARGS-30"), _errors);
        }

        // Validate game impl params.
        {
            IFaultDisputeGame game = IFaultDisputeGame(gameImpl);
            _errors = internalRequire(
                game.maxGameDepth() == EXPECTED_MAX_GAME_DEPTH, string.concat(_p.prefix, "-30"), _errors
            );
            _errors =
                internalRequire(game.splitDepth() == EXPECTED_SPLIT_DEPTH, string.concat(_p.prefix, "-40"), _errors);
            _errors = internalRequire(
                Duration.unwrap(game.clockExtension()) == EXPECTED_CLOCK_EXTENSION,
                string.concat(_p.prefix, "-50"),
                _errors
            );
            _errors = internalRequire(
                Duration.unwrap(game.maxClockDuration()) == EXPECTED_MAX_CLOCK_DURATION,
                string.concat(_p.prefix, "-60"),
                _errors
            );
            _errors = internalRequire(game.l2SequenceNumber() == 0, string.concat(_p.prefix, "-70"), _errors);
        }

        // Validate anchor root is non-zero (from ASR in game args).
        {
            (Hash anchorRoot,) = IAnchorStateRegistry(gameArgs.anchorStateRegistry).getAnchorRoot();
            _errors = internalRequire(Hash.unwrap(anchorRoot) != bytes32(0), string.concat(_p.prefix, "-80"), _errors);
        }

        // Validate proposer and challenger for permissioned games.
        if (_p.isPermissioned) {
            _errors = internalRequire(gameArgs.proposer == _p.proposer, string.concat(_p.prefix, "-90"), _errors);
            _errors = internalRequire(gameArgs.challenger == _p.challenger, string.concat(_p.prefix, "-100"), _errors);
        }

        return _errors;
    }

    /// @notice Validates the shared ASR: version, proxy impl, DGF reference, ProxyAdmin,
    ///         retirement timestamp, respected game type, and SCKDG cross-validation.
    function assertValidSharedASR(
        string memory _errors,
        address _asr,
        IDisputeGameFactory _dgf,
        IProxyAdmin _proxyAdmin,
        SharedImplementations memory _refs
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(_asr).version(), ISemver(_refs.anchorStateRegistryImpl).version()),
            "MIG-SASR-10",
            _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(_asr) == _refs.anchorStateRegistryImpl, "MIG-SASR-20", _errors
        );
        _errors = internalRequire(
            address(IAnchorStateRegistry(_asr).disputeGameFactory()) == address(_dgf), "MIG-SASR-30", _errors
        );
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(_asr).proxyAdmin()) == address(_proxyAdmin), "MIG-SASR-40", _errors
        );
        _errors = internalRequire(IAnchorStateRegistry(_asr).retirementTimestamp() > 0, "MIG-SASR-50", _errors);
        _errors = internalRequire(
            GameTypes.isSuperGame(IAnchorStateRegistry(_asr).respectedGameType()), "MIG-SASR-60", _errors
        );

        // Cross-validate: ASR in each super game type's args should match portal-discovered ASR.
        {
            address sckdgImpl = address(_dgf.gameImpls(GameTypes.SUPER_CANNON_KONA));
            if (sckdgImpl != address(0)) {
                bytes memory sckdgArgs = _dgf.gameArgs(GameTypes.SUPER_CANNON_KONA);
                if (LibGameArgs.isValidPermissionlessArgs(sckdgArgs)) {
                    LibGameArgs.GameArgs memory args = LibGameArgs.decode(sckdgArgs);
                    _errors = internalRequire(args.anchorStateRegistry == _asr, "MIG-SASR-70", _errors);
                }
            }
        }
        {
            address spdgImpl = address(_dgf.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON));
            if (spdgImpl != address(0)) {
                bytes memory spdgArgs = _dgf.gameArgs(GameTypes.SUPER_PERMISSIONED_CANNON);
                if (LibGameArgs.isValidPermissionedArgs(spdgArgs)) {
                    LibGameArgs.GameArgs memory args = LibGameArgs.decode(spdgArgs);
                    _errors = internalRequire(args.anchorStateRegistry == _asr, "MIG-SASR-80", _errors);
                }
            }
        }

        return _errors;
    }

    /// @notice Validates the shared ETHLockbox: version, proxy impl, ProxyAdmin.
    function assertValidSharedLockbox(
        string memory _errors,
        IETHLockbox _lockbox,
        IProxyAdmin _proxyAdmin,
        SharedImplementations memory _refs
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(address(_lockbox)).version(), ISemver(_refs.ethLockboxImpl).version()),
            "MIG-SLOCKBOX-10",
            _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(address(_lockbox)) == _refs.ethLockboxImpl, "MIG-SLOCKBOX-20", _errors
        );
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(address(_lockbox)).proxyAdmin()) == address(_proxyAdmin),
            "MIG-SLOCKBOX-30",
            _errors
        );
        return _errors;
    }

    /// @notice Validates the shared DelayedWETH: version, delay, proxyAdminOwner, proxy impl, ProxyAdmin.
    function assertValidSharedDelayedWETH(
        string memory _errors,
        address _weth,
        IProxyAdmin _proxyAdmin,
        SharedImplementations memory _refs
    )
        internal
        view
        returns (string memory)
    {
        _errors = internalRequire(
            LibString.eq(ISemver(_weth).version(), ISemver(_refs.delayedWETHImpl).version()), "MIG-SDWETH-10", _errors
        );
        _errors = internalRequire(
            IDelayedWETH(payable(_weth)).delay() == _refs.withdrawalDelaySeconds, "MIG-SDWETH-20", _errors
        );
        _errors = internalRequire(
            IDelayedWETH(payable(_weth)).proxyAdminOwner() == _refs.l1PAOMultisig, "MIG-SDWETH-30", _errors
        );
        _errors = internalRequire(
            _proxyAdmin.getProxyImplementation(_weth) == _refs.delayedWETHImpl, "MIG-SDWETH-40", _errors
        );
        _errors = internalRequire(
            address(IProxyAdminOwnedBase(_weth).proxyAdmin()) == address(_proxyAdmin), "MIG-SDWETH-50", _errors
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
        // After migration, systemConfig.disputeGameFactory() on the first chain is authoritative
        // for the shared DGF — per-chain migration already wired portals to the shared ASR/DGF.
        IOptimismPortal2 firstPortal = IOptimismPortal2(payable(_chainSystemConfigs[0].optimismPortal()));
        address sharedASR = address(firstPortal.anchorStateRegistry());
        IETHLockbox sharedLockbox = firstPortal.ethLockbox();
        IDisputeGameFactory sharedDGF = IDisputeGameFactory(_chainSystemConfigs[0].disputeGameFactory());

        // Guard against missing lockbox — would revert on authorizedPortals call.
        if (address(sharedLockbox) == address(0)) {
            return internalRequire(false, "MIG-LOCKBOX-MISSING", _errors);
        }

        for (uint256 i = 0; i < _chainSystemConfigs.length; i++) {
            string memory idx = LibString.toString(i);

            IOptimismPortal2 portal = IOptimismPortal2(payable(_chainSystemConfigs[i].optimismPortal()));

            // Portal's ASR should point to the shared ASR.
            _errors = internalRequire(
                address(portal.anchorStateRegistry()) == sharedASR, string.concat("MIG-CHAIN-", idx, "-10"), _errors
            );

            // Per-chain DGF should have all game types cleared.
            // After migration, systemConfig.disputeGameFactory() resolves through
            // portal → shared ASR → shared DGF. When the per-chain DGF IS the shared DGF,
            // skip these checks — the shared DGF is already validated by the shape checks above.
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

            // Portal should be authorized in the shared lockbox.
            _errors = internalRequire(
                sharedLockbox.authorizedPortals(portal), string.concat("MIG-CHAIN-", idx, "-80"), _errors
            );

            // Portal's lockbox should match the shared lockbox.
            _errors = internalRequire(
                address(portal.ethLockbox()) == address(sharedLockbox), string.concat("MIG-CHAIN-", idx, "-90"), _errors
            );

            // Feature flags must be enabled on each chain's SystemConfig.
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
