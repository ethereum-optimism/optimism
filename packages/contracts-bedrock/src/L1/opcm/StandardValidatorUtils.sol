// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { GameTypes } from "src/dispute/lib/Types.sol";
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

uint256 constant EXPECTED_MAX_GAME_DEPTH = 73;
uint256 constant EXPECTED_SPLIT_DEPTH = 30;
uint256 constant EXPECTED_CLOCK_EXTENSION = 10800;
uint256 constant EXPECTED_MAX_CLOCK_DURATION = 302400;
string constant EXPECTED_PREIMAGE_ORACLE_VERSION = "1.1.4";
uint256 constant EXPECTED_CHALLENGE_PERIOD = 86400;
uint256 constant EXPECTED_MIN_PROPOSAL_SIZE = 126000;

/// @title StandardValidatorUtils
/// @notice StandardValidatorUtils is a contract that provides some validation logic
/// for the OPContractsManagerStandardValidator to split the bytecode across multiple
/// contracts to meet the EIP-170 bytecode size limit
contract StandardValidatorUtils {
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

    /// @notice Struct containing override parameters for the validation process.
    struct ValidationOverrides {
        address l1PAOMultisig;
        address challenger;
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
        external
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
        external
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
}
