// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { GameType, Claim, GameTypes, Hash } from "src/dispute/lib/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";

uint256 constant EXPECTED_MAX_GAME_DEPTH = 73;
uint256 constant EXPECTED_SPLIT_DEPTH = 30;
uint256 constant EXPECTED_CLOCK_EXTENSION = 10800;
uint256 constant EXPECTED_MAX_CLOCK_DURATION = 302400;
string constant EXPECTED_PREIMAGE_ORACLE_VERSION = "1.1.5";
uint256 constant EXPECTED_CHALLENGE_PERIOD = 86400;
uint256 constant EXPECTED_MIN_PROPOSAL_SIZE = 126000;

/// @notice Struct containing the unified game args for a dispute game implementation.
struct DisputeGameImplementation {
    address gameAddress;
    uint256 maxGameDepth;
    uint256 splitDepth;
    Duration maxClockDuration;
    Duration clockExtension;
    GameType gameType;
    // extra args
    uint256 l2SequenceNumber;
    // dispute-game v2 game args
    Claim absolutePrestate;
    IBigStepper vm;
    IAnchorStateRegistry asr;
    IDelayedWETH weth;
    uint256 l2ChainId;
    address challenger;
    address proposer;
}

/// @notice Arguments passed to `assertValidDisputeGame`.
struct DisputeGameValidationArgs {
    string errors;
    ISystemConfig sysCfg;
    DisputeGameImplementation game;
    bytes32 absolutePrestate;
    uint256 l2ChainID;
    IProxyAdmin admin;
    GameType gameType;
    string errorPrefix;
}

/// @notice Implementation addresses used when validating a dispute game (resolved by caller).
struct DisputeGameImpls {
    address expectedGameImpl;
    address mipsImpl;
    address delayedWETHImpl;
    address anchorStateRegistryImpl;
}

/// @notice Config values used when validating a dispute game (resolved by caller).
struct DisputeGameConfig {
    address l1PAOMultisig;
    uint256 withdrawalDelaySeconds;
}

/// @title StandardValidatorUtils
/// @notice StandardValidatorUtils is a contract that provides some validation logic
/// for the OPContractsManagerStandardValidator to split the bytecode across multiple
/// contracts to meet the EIP-170 bytecode size limit
contract StandardValidatorUtils {
    /// @notice Appends `_message` to `_errors` when `_condition` is false; otherwise returns
    ///         `_errors` unchanged. Used to accumulate validation failures without reverting.
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

    /// @notice Asserts that the SuperchainConfig contract is valid.
    function assertValidSuperchainConfig(
        string memory _errors,
        ISuperchainConfig _superchainConfig
    )
        external
        view
        returns (string memory)
    {
        _errors = internalRequire(!_superchainConfig.paused(address(0)), "SPRCFG-10", _errors);
        return _errors;
    }

    /// @notice Asserts that the ProxyAdmin contract is valid.
    function assertValidProxyAdmin(
        string memory _errors,
        IProxyAdmin _admin,
        address _l1PAOMultisig
    )
        external
        view
        returns (string memory)
    {
        _errors = internalRequire(_admin.owner() == _l1PAOMultisig, "PROXYA-10", _errors);
        return _errors;
    }

    /// @notice Asserts that dispute games are correctly configured for super root mode.
    function assertValidSuperRootDisputeGames(
        string memory _errors,
        ISystemConfig _sysCfg
    )
        external
        view
        returns (string memory)
    {
        IDisputeGameFactory dgf = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors = internalRequire(address(dgf.gameImpls(GameTypes.CANNON)) == address(0), "PLDG-SHAPE", _errors);
        _errors =
            internalRequire(address(dgf.gameImpls(GameTypes.PERMISSIONED_CANNON)) == address(0), "PDDG-SHAPE", _errors);
        _errors = internalRequire(address(dgf.gameImpls(GameTypes.CANNON_KONA)) == address(0), "CKDG-SHAPE", _errors);
        // TODO(#20030): Once SUPER_CANNON is disabled in migrator, re-add check that SUPER_CANNON == address(0).
        // _errors =
        //     internalRequire(address(dgf.gameImpls(GameTypes.SUPER_CANNON)) == address(0), "SCDG-SHAPE", _errors);
        _errors = internalRequire(
            address(dgf.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)) != address(0), "SPDG-SHAPE", _errors
        );
        _errors =
            internalRequire(address(dgf.gameImpls(GameTypes.SUPER_CANNON_KONA)) != address(0), "SCKDG-SHAPE", _errors);
        return _errors;
    }

    /// @notice Asserts that super game types are NOT registered in non-super-root mode.
    function assertValidNonSuperRootDisputeGames(
        string memory _errors,
        ISystemConfig _sysCfg
    )
        external
        view
        returns (string memory)
    {
        IDisputeGameFactory dgf = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors = internalRequire(address(dgf.gameImpls(GameTypes.SUPER_CANNON)) == address(0), "SCDG-NOSHAPE", _errors);
        _errors = internalRequire(
            address(dgf.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)) == address(0), "SPDG-NOSHAPE", _errors
        );
        _errors =
            internalRequire(address(dgf.gameImpls(GameTypes.SUPER_CANNON_KONA)) == address(0), "SCKDG-NOSHAPE", _errors);
        _errors = internalRequire(
            address(dgf.gameImpls(GameTypes.PERMISSIONED_CANNON)) != address(0), "PDDG-NOSHAPE", _errors
        );
        _errors = internalRequire(address(dgf.gameImpls(GameTypes.CANNON_KONA)) != address(0), "CKDG-NOSHAPE", _errors);
        return _errors;
    }

    /// @notice Asserts that the MipsVm contract is valid.
    function assertValidMipsVm(
        string memory _errors,
        IMIPS64 _mips,
        address _mipsImpl,
        string memory _errorPrefix
    )
        public
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-VM");
        _errors = internalRequire(address(_mips) == _mipsImpl, string.concat(_errorPrefix, "-10"), _errors);
        _errors = internalRequire(
            LibString.eq(ISemver(address(_mips)).version(), ISemver(_mipsImpl).version()),
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        _errors = internalRequire(_mips.stateVersion() == 8, string.concat(_errorPrefix, "-30"), _errors);
        return _errors;
    }

    /// @notice Asserts that the OptimismMintableERC20Factory contract is valid.
    function assertValidOptimismMintableERC20Factory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl
    )
        external
        view
        returns (string memory)
    {
        IOptimismMintableERC20Factory _factory = IOptimismMintableERC20Factory(_sysCfg.optimismMintableERC20Factory());
        _errors = internalRequire(
            LibString.eq(ISemver(address(_factory)).version(), ISemver(_impl).version()), "MERC20F-10", _errors
        );
        _errors = internalRequire(_admin.getProxyImplementation(address(_factory)) == _impl, "MERC20F-20", _errors);

        IL1StandardBridge _bridge = IL1StandardBridge(payable(_sysCfg.l1StandardBridge()));
        _errors = internalRequire(_factory.BRIDGE() == address(_bridge), "MERC20F-30", _errors);
        _errors = internalRequire(_factory.bridge() == address(_bridge), "MERC20F-40", _errors);
        return _errors;
    }

    /// @notice Asserts that the DisputeGameFactory contract is valid.
    function assertValidDisputeGameFactory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl,
        address _l1PAOMultisig
    )
        external
        view
        returns (string memory)
    {
        IDisputeGameFactory _factory = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors = internalRequire(
            LibString.eq(ISemver(address(_factory)).version(), ISemver(_impl).version()), "DF-10", _errors
        );
        _errors = internalRequire(_admin.getProxyImplementation(address(_factory)) == _impl, "DF-20", _errors);
        _errors = internalRequire(_factory.owner() == _l1PAOMultisig, "DF-30", _errors);
        _errors = internalRequire(IProxyAdminOwnedBase(address(_factory)).proxyAdmin() == _admin, "DF-40", _errors);
        // At least one permissioned game must be registered — either the legacy
        // PERMISSIONED_CANNON or the super-root SUPER_PERMISSIONED_CANNON.
        _errors = internalRequire(
            address(_factory.gameImpls(GameTypes.PERMISSIONED_CANNON)) != address(0)
                || address(_factory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)) != address(0),
            "DF-50",
            _errors
        );
        return _errors;
    }

    /// @notice Asserts that the OptimismPortal contract is valid.
    function assertValidOptimismPortal(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        address _impl
    )
        external
        view
        returns (string memory)
    {
        IOptimismPortal2 _portal = IOptimismPortal2(payable(_sysCfg.optimismPortal()));

        _errors = internalRequire(
            LibString.eq(ISemver(address(_portal)).version(), ISemver(_impl).version()), "PORTAL-10", _errors
        );
        _errors = internalRequire(_admin.getProxyImplementation(address(_portal)) == _impl, "PORTAL-20", _errors);

        IDisputeGameFactory _dgf = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors = internalRequire(address(_portal.disputeGameFactory()) == address(_dgf), "PORTAL-30", _errors);
        _errors = internalRequire(address(_portal.systemConfig()) == address(_sysCfg), "PORTAL-40", _errors);
        _errors = internalRequire(_portal.l2Sender() == Constants.DEFAULT_L2_SENDER, "PORTAL-80", _errors);
        _errors = internalRequire(IProxyAdminOwnedBase(address(_portal)).proxyAdmin() == _admin, "PORTAL-90", _errors);
        return _errors;
    }

    /// @notice Asserts that the PreimageOracle contract is valid.
    function assertValidPreimageOracle(
        string memory _errors,
        IPreimageOracle _oracle,
        string memory _errorPrefix
    )
        public
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-PIMGO");
        _errors = internalRequire(
            LibString.eq(ISemver(address(_oracle)).version(), EXPECTED_PREIMAGE_ORACLE_VERSION),
            string.concat(_errorPrefix, "-10"),
            _errors
        );
        _errors = internalRequire(
            _oracle.challengePeriod() == EXPECTED_CHALLENGE_PERIOD, string.concat(_errorPrefix, "-20"), _errors
        );
        _errors = internalRequire(
            _oracle.minProposalSize() == EXPECTED_MIN_PROPOSAL_SIZE, string.concat(_errorPrefix, "-30"), _errors
        );
        return _errors;
    }

    /// @notice Asserts that the DelayedWETH contract is valid.
    function assertValidDelayedWETH(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDelayedWETH _weth,
        IProxyAdmin _admin,
        address _l1PAOMultisig,
        address _delayedWETHImpl,
        uint256 _withdrawalDelaySeconds,
        string memory _errorPrefix
    )
        public
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-DWETH");
        _errors = internalRequire(
            LibString.eq(ISemver(address(_weth)).version(), ISemver(_delayedWETHImpl).version()),
            string.concat(_errorPrefix, "-10"),
            _errors
        );
        _errors = internalRequire(
            _admin.getProxyImplementation(address(_weth)) == _delayedWETHImpl,
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        _errors =
            internalRequire(_weth.proxyAdminOwner() == _l1PAOMultisig, string.concat(_errorPrefix, "-30"), _errors);
        _errors = internalRequire(_weth.delay() == _withdrawalDelaySeconds, string.concat(_errorPrefix, "-40"), _errors);
        _errors = internalRequire(_weth.systemConfig() == _sysCfg, string.concat(_errorPrefix, "-50"), _errors);
        _errors = internalRequire(
            IProxyAdminOwnedBase(address(_weth)).proxyAdmin() == _admin, string.concat(_errorPrefix, "-60"), _errors
        );
        return _errors;
    }

    /// @notice Asserts that the AnchorStateRegistry contract is valid.
    function assertValidAnchorStateRegistry(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDisputeGameFactory _dgf,
        IAnchorStateRegistry _asr,
        IProxyAdmin _admin,
        address _anchorStateRegistryImpl,
        string memory _errorPrefix
    )
        public
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-ANCHORP");
        _errors = internalRequire(
            LibString.eq(ISemver(address(_asr)).version(), ISemver(_anchorStateRegistryImpl).version()),
            string.concat(_errorPrefix, "-10"),
            _errors
        );
        _errors = internalRequire(
            _admin.getProxyImplementation(address(_asr)) == _anchorStateRegistryImpl,
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        _errors = internalRequire(
            address(_asr.disputeGameFactory()) == address(_dgf), string.concat(_errorPrefix, "-30"), _errors
        );
        _errors = internalRequire(_asr.systemConfig() == _sysCfg, string.concat(_errorPrefix, "-40"), _errors);
        _errors = internalRequire(
            IProxyAdminOwnedBase(address(_asr)).proxyAdmin() == _admin, string.concat(_errorPrefix, "-50"), _errors
        );
        _errors = internalRequire(_asr.retirementTimestamp() > 0, string.concat(_errorPrefix, "-60"), _errors);
        return _errors;
    }

    /// @notice Asserts that a DisputeGame contract is valid. Drills into WETH, ASR, MIPS, and
    ///         PreimageOracle validation. Caller resolves storage-dependent values via the
    ///         `DisputeGameImpls` and `DisputeGameConfig` structs.
    function assertValidDisputeGame(
        DisputeGameValidationArgs memory _args,
        DisputeGameImpls memory _impls,
        DisputeGameConfig memory _cfg
    )
        public
        view
        returns (string memory errors_)
    {
        errors_ = _args.errors;
        string memory errorPrefix = _args.errorPrefix;
        DisputeGameImplementation memory game = _args.game;
        (Hash anchorRoot,) = game.asr.getAnchorRoot();
        IDisputeGameFactory dgf = IDisputeGameFactory(_args.sysCfg.disputeGameFactory());

        errors_ = internalRequire(
            LibString.eq(ISemver(game.gameAddress).version(), ISemver(_impls.expectedGameImpl).version()),
            string.concat(errorPrefix, "-20"),
            errors_
        );

        errors_ = internalRequire(
            GameType.unwrap(game.gameType) == GameType.unwrap(_args.gameType),
            string.concat(errorPrefix, "-30"),
            errors_
        );
        errors_ = internalRequire(
            Claim.unwrap(game.absolutePrestate) == _args.absolutePrestate, string.concat(errorPrefix, "-40"), errors_
        );
        // Super game types store l2ChainId=0 in game args because the chain ID is embedded
        // in the super root proof extraData, not in the game args.
        uint256 expectedL2ChainId = GameTypes.isSuperGame(_args.gameType) ? 0 : _args.l2ChainID;
        errors_ = internalRequire(game.l2ChainId == expectedL2ChainId, string.concat(errorPrefix, "-60"), errors_);
        errors_ = internalRequire(game.l2SequenceNumber == 0, string.concat(errorPrefix, "-70"), errors_);
        errors_ = internalRequire(
            Duration.unwrap(game.clockExtension) == EXPECTED_CLOCK_EXTENSION, string.concat(errorPrefix, "-80"), errors_
        );
        errors_ = internalRequire(game.splitDepth == EXPECTED_SPLIT_DEPTH, string.concat(errorPrefix, "-90"), errors_);
        errors_ =
            internalRequire(game.maxGameDepth == EXPECTED_MAX_GAME_DEPTH, string.concat(errorPrefix, "-100"), errors_);
        errors_ = internalRequire(
            Duration.unwrap(game.maxClockDuration) == EXPECTED_MAX_CLOCK_DURATION,
            string.concat(errorPrefix, "-110"),
            errors_
        );
        errors_ = internalRequire(Hash.unwrap(anchorRoot) != bytes32(0), string.concat(errorPrefix, "-120"), errors_);

        errors_ = assertValidDelayedWETH(
            errors_,
            _args.sysCfg,
            game.weth,
            _args.admin,
            _cfg.l1PAOMultisig,
            _impls.delayedWETHImpl,
            _cfg.withdrawalDelaySeconds,
            errorPrefix
        );
        errors_ = assertValidAnchorStateRegistry(
            errors_, _args.sysCfg, dgf, game.asr, _args.admin, _impls.anchorStateRegistryImpl, errorPrefix
        );

        errors_ = assertValidMipsVm(errors_, IMIPS64(address(game.vm)), _impls.mipsImpl, errorPrefix);

        // Only assert valid preimage oracle if the game VM is valid, since otherwise
        // the contract is likely to revert.
        if (address(game.vm) == _impls.mipsImpl) {
            errors_ = assertValidPreimageOracle(errors_, game.vm.oracle(), errorPrefix);
        }

        return errors_;
    }
}
