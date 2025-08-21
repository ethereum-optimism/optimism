// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { LibString } from "@solady/utils/LibString.sol";
import { GameType, Claim, GameTypes } from "src/dispute/lib/Types.sol";
import { Duration } from "src/dispute/lib/LibUDT.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Hash } from "src/dispute/lib/Types.sol";

// Interfaces
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { IL1ERC721Bridge } from "interfaces/L1/IL1ERC721Bridge.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IMIPS64 } from "interfaces/cannon/IMIPS64.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";

/// @title OPContractsManagerStandardValidator
/// @notice This contract is used to validate the configuration of the L1 contracts of an OP Stack chain.
/// It is a stateless contract that can be used to ensure that the L1 contracts are configured correctly.
/// It is intended to be used by the L1 PAO multisig to validate the configuration of the L1 contracts
/// before and after an upgrade.
contract OPContractsManagerStandardValidator is ISemver {
    /// @notice The semantic version of the OPContractsManagerStandardValidator contract.
    /// @custom:semver 1.8.0
    string public constant version = "1.8.0";

    /// @notice The SuperchainConfig contract.
    ISuperchainConfig public superchainConfig;

    /// @notice The L1 PAO multisig address.
    address public l1PAOMultisig;

    /// @notice The challenger address for permissioned dispute games.
    address public challenger;

    /// @notice The withdrawal delay in seconds for the DelayedWETH contract.
    uint256 public withdrawalDelaySeconds;

    // Implementation addresses as state variables

    /// @notice The L1ERC721Bridge implementation address.
    address public l1ERC721BridgeImpl;

    /// @notice The OptimismPortal implementation address.
    address public optimismPortalImpl;

    /// @notice The ETHLockbox implementation address.
    address public ethLockboxImpl;

    /// @notice The SystemConfig implementation address.
    address public systemConfigImpl;

    /// @notice The OptimismMintableERC20Factory implementation address.
    address public optimismMintableERC20FactoryImpl;

    /// @notice The L1CrossDomainMessenger implementation address.
    address public l1CrossDomainMessengerImpl;

    /// @notice The L1StandardBridge implementation address.
    address public l1StandardBridgeImpl;

    /// @notice The DisputeGameFactory implementation address.
    address public disputeGameFactoryImpl;

    /// @notice The AnchorStateRegistry implementation address.
    address public anchorStateRegistryImpl;

    /// @notice The DelayedWETH implementation address.
    address public delayedWETHImpl;

    /// @notice The MIPS implementation address.
    address public mipsImpl;

    /// @notice Struct containing the implementation addresses of the L1 contracts.
    struct Implementations {
        address l1ERC721BridgeImpl;
        address optimismPortalImpl;
        address ethLockboxImpl;
        address systemConfigImpl;
        address optimismMintableERC20FactoryImpl;
        address l1CrossDomainMessengerImpl;
        address l1StandardBridgeImpl;
        address disputeGameFactoryImpl;
        address anchorStateRegistryImpl;
        address delayedWETHImpl;
        address mipsImpl;
    }

    /// @notice Struct containing the input parameters for the validation process.
    struct ValidationInput {
        IProxyAdmin proxyAdmin;
        ISystemConfig sysCfg;
        bytes32 absolutePrestate;
        uint256 l2ChainID;
    }

    /// @notice Struct containing override parameters for the validation process.
    struct ValidationOverrides {
        address l1PAOMultisig;
        address challenger;
    }

    /// @notice Constructor for the OPContractsManagerStandardValidator contract.
    constructor(
        Implementations memory _implementations,
        ISuperchainConfig _superchainConfig,
        address _l1PAOMultisig,
        address _challenger,
        uint256 _withdrawalDelaySeconds
    ) {
        superchainConfig = _superchainConfig;
        l1PAOMultisig = _l1PAOMultisig;
        challenger = _challenger;
        withdrawalDelaySeconds = _withdrawalDelaySeconds;

        // Set implementation addresses from struct
        l1ERC721BridgeImpl = _implementations.l1ERC721BridgeImpl;
        optimismPortalImpl = _implementations.optimismPortalImpl;
        ethLockboxImpl = _implementations.ethLockboxImpl;
        systemConfigImpl = _implementations.systemConfigImpl;
        optimismMintableERC20FactoryImpl = _implementations.optimismMintableERC20FactoryImpl;
        l1CrossDomainMessengerImpl = _implementations.l1CrossDomainMessengerImpl;
        l1StandardBridgeImpl = _implementations.l1StandardBridgeImpl;
        disputeGameFactoryImpl = _implementations.disputeGameFactoryImpl;
        anchorStateRegistryImpl = _implementations.anchorStateRegistryImpl;
        delayedWETHImpl = _implementations.delayedWETHImpl;
        mipsImpl = _implementations.mipsImpl;
    }

    /// @notice Returns a string representing the overrides that are set.
    function getOverridesString(ValidationOverrides memory _overrides) private pure returns (string memory) {
        string memory overridesError;

        if (_overrides.l1PAOMultisig != address(0)) {
            overridesError = string.concat(overridesError, "OVERRIDES-L1PAOMULTISIG");
        }

        if (_overrides.challenger != address(0)) {
            if (bytes(overridesError).length > 0) overridesError = string.concat(overridesError, ",");
            overridesError = string.concat(overridesError, "OVERRIDES-CHALLENGER");
        }

        return overridesError;
    }

    /// @notice Returns the expected L1 PAO multisig address.
    function expectedL1PAOMultisig(ValidationOverrides memory _overrides) internal view returns (address) {
        if (_overrides.l1PAOMultisig != address(0)) {
            return _overrides.l1PAOMultisig;
        }
        return l1PAOMultisig;
    }

    /// @notice Returns the expected challenger address.
    function expectedChallenger(ValidationOverrides memory _overrides) internal view returns (address) {
        if (_overrides.challenger != address(0)) {
            return _overrides.challenger;
        }
        return challenger;
    }

    /// @notice Returns the expected SystemConfig version.
    function systemConfigVersion() public pure returns (string memory) {
        return "3.4.0";
    }

    /// @notice Returns the expected OptimismPortal version.
    function optimismPortalVersion() public pure returns (string memory) {
        return "4.6.0";
    }

    /// @notice Returns the expected L1CrossDomainMessenger version.
    function l1CrossDomainMessengerVersion() public pure returns (string memory) {
        return "2.9.0";
    }

    /// @notice Returns the expected L1ERC721Bridge version.
    function l1ERC721BridgeVersion() public pure returns (string memory) {
        return "2.7.0";
    }

    /// @notice Returns the expected L1StandardBridge version.
    function l1StandardBridgeVersion() public pure returns (string memory) {
        return "2.6.0";
    }

    /// @notice Returns the expected MIPS version.
    function mipsVersion() public pure returns (string memory) {
        return "1.9.0";
    }

    /// @notice Returns the expected OptimismMintableERC20Factory version.
    function optimismMintableERC20FactoryVersion() public pure returns (string memory) {
        return "1.10.1";
    }

    /// @notice Returns the expected DisputeGameFactory version.
    function disputeGameFactoryVersion() public pure returns (string memory) {
        return "1.2.0";
    }

    /// @notice Returns the expected AnchorStateRegistry version.
    function anchorStateRegistryVersion() public pure returns (string memory) {
        return "3.5.0";
    }

    /// @notice Returns the expected DelayedWETH version.
    function delayedWETHVersion() public pure returns (string memory) {
        return "1.5.0";
    }

    /// @notice Returns the expected PermissionedDisputeGame version.
    function permissionedDisputeGameVersion() public pure returns (string memory) {
        return "1.8.0";
    }

    /// @notice Returns the expected PreimageOracle version.
    function preimageOracleVersion() public pure returns (string memory) {
        return "1.1.4";
    }

    /// @notice Returns the expected ETHLockbox version.
    function ethLockboxVersion() public pure returns (string memory) {
        return "1.2.0";
    }

    /// @notice Internal function to get version from any contract implementing ISemver.
    function getVersion(address _contract) private view returns (string memory) {
        return ISemver(_contract).version();
    }

    /// @notice Internal function to get the ProxyAdmin contract from any contract implementing IProxyAdminOwnedBase.
    function getProxyAdmin(address _contract) private view returns (IProxyAdmin) {
        return IProxyAdminOwnedBase(_contract).proxyAdmin();
    }

    /// @notice Internal function to get the implementation address of any contract via the ProxyAdmin contract.
    function getProxyImplementation(IProxyAdmin _admin, address _contract) private view returns (address) {
        return _admin.getProxyImplementation(_contract);
    }

    /// @notice Asserts that the SuperchainConfig contract is valid.
    function assertValidSuperchainConfig(string memory _errors) internal view returns (string memory) {
        _errors = internalRequire(!superchainConfig.paused(address(0)), "SPRCFG-10", _errors);
        return _errors;
    }

    /// @notice Asserts that the ProxyAdmin contract is valid.
    function assertValidProxyAdmin(
        string memory _errors,
        IProxyAdmin _admin,
        ValidationOverrides memory _overrides
    )
        internal
        view
        returns (string memory)
    {
        address _l1PAOMultisig = expectedL1PAOMultisig(_overrides);
        _errors = internalRequire(_admin.owner() == _l1PAOMultisig, "PROXYA-10", _errors);
        return _errors;
    }

    /// @notice Asserts that the SystemConfig contract is valid.
    function assertValidSystemConfig(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        virtual
        returns (string memory)
    {
        _errors =
            internalRequire(LibString.eq(getVersion(address(_sysCfg)), systemConfigVersion()), "SYSCON-10", _errors);
        _errors = internalRequire(_sysCfg.gasLimit() <= uint64(500_000_000), "SYSCON-20", _errors);
        _errors = internalRequire(_sysCfg.scalar() != 0, "SYSCON-30", _errors);
        _errors =
            internalRequire(getProxyImplementation(_admin, address(_sysCfg)) == systemConfigImpl, "SYSCON-40", _errors);

        IResourceMetering.ResourceConfig memory outputConfig = _sysCfg.resourceConfig();
        _errors = internalRequire(outputConfig.maxResourceLimit == 20_000_000, "SYSCON-50", _errors);
        _errors = internalRequire(outputConfig.elasticityMultiplier == 10, "SYSCON-60", _errors);
        _errors = internalRequire(outputConfig.baseFeeMaxChangeDenominator == 8, "SYSCON-70", _errors);
        _errors = internalRequire(outputConfig.systemTxMaxGas == 1_000_000, "SYSCON-80", _errors);
        _errors = internalRequire(outputConfig.minimumBaseFee == 1 gwei, "SYSCON-90", _errors);
        _errors = internalRequire(outputConfig.maximumBaseFee == type(uint128).max, "SYSCON-100", _errors);
        _errors = internalRequire(_sysCfg.operatorFeeScalar() == 0, "SYSCON-110", _errors);
        _errors = internalRequire(_sysCfg.operatorFeeConstant() == 0, "SYSCON-120", _errors);
        _errors = internalRequire(getProxyAdmin(address(_sysCfg)) == _admin, "SYSCON-130", _errors);
        _errors = internalRequire(_sysCfg.superchainConfig() == superchainConfig, "SYSCON-140", _errors);
        return _errors;
    }

    /// @notice Asserts that the L1CrossDomainMessenger contract is valid.
    function assertValidL1CrossDomainMessenger(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IL1CrossDomainMessenger _messenger = IL1CrossDomainMessenger(_sysCfg.l1CrossDomainMessenger());
        _errors = internalRequire(
            LibString.eq(getVersion(address(_messenger)), l1CrossDomainMessengerVersion()), "L1xDM-10", _errors
        );
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_messenger)) == l1CrossDomainMessengerImpl, "L1xDM-20", _errors
        );

        IOptimismPortal2 _portal = IOptimismPortal2(payable(_sysCfg.optimismPortal()));

        _errors = internalRequire(
            address(_messenger.OTHER_MESSENGER()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-30", _errors
        );
        _errors = internalRequire(
            address(_messenger.otherMessenger()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-40", _errors
        );
        _errors = internalRequire(address(_messenger.PORTAL()) == address(_portal), "L1xDM-50", _errors);
        _errors = internalRequire(address(_messenger.portal()) == address(_portal), "L1xDM-60", _errors);
        _errors = internalRequire(address(_messenger.systemConfig()) == address(_sysCfg), "L1xDM-70", _errors);
        _errors = internalRequire(getProxyAdmin(address(_messenger)) == _admin, "L1xDM-80", _errors);
        return _errors;
    }

    /// @notice Asserts that the L1StandardBridge contract is valid.
    function assertValidL1StandardBridge(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IL1StandardBridge _bridge = IL1StandardBridge(payable(_sysCfg.l1StandardBridge()));
        _errors =
            internalRequire(LibString.eq(getVersion(address(_bridge)), l1StandardBridgeVersion()), "L1SB-10", _errors);
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_bridge)) == l1StandardBridgeImpl, "L1SB-20", _errors
        );

        IL1CrossDomainMessenger _messenger = IL1CrossDomainMessenger(_sysCfg.l1CrossDomainMessenger());

        _errors = internalRequire(address(_bridge.MESSENGER()) == address(_messenger), "L1SB-30", _errors);
        _errors = internalRequire(address(_bridge.messenger()) == address(_messenger), "L1SB-40", _errors);
        _errors = internalRequire(address(_bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-50", _errors);
        _errors = internalRequire(address(_bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-60", _errors);
        _errors = internalRequire(address(_bridge.systemConfig()) == address(_sysCfg), "L1SB-70", _errors);
        _errors = internalRequire(getProxyAdmin(address(_bridge)) == _admin, "L1SB-80", _errors);
        return _errors;
    }

    /// @notice Asserts that the OptimismMintableERC20Factory contract is valid.
    function assertValidOptimismMintableERC20Factory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IOptimismMintableERC20Factory _factory = IOptimismMintableERC20Factory(_sysCfg.optimismMintableERC20Factory());
        _errors = internalRequire(
            LibString.eq(getVersion(address(_factory)), optimismMintableERC20FactoryVersion()), "MERC20F-10", _errors
        );
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_factory)) == optimismMintableERC20FactoryImpl, "MERC20F-20", _errors
        );

        IL1StandardBridge _bridge = IL1StandardBridge(payable(_sysCfg.l1StandardBridge()));
        _errors = internalRequire(_factory.BRIDGE() == address(_bridge), "MERC20F-30", _errors);
        _errors = internalRequire(_factory.bridge() == address(_bridge), "MERC20F-40", _errors);
        return _errors;
    }

    /// @notice Asserts that the L1ERC721Bridge contract is valid.
    function assertValidL1ERC721Bridge(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IL1ERC721Bridge _bridge = IL1ERC721Bridge(_sysCfg.l1ERC721Bridge());
        _errors =
            internalRequire(LibString.eq(getVersion(address(_bridge)), l1ERC721BridgeVersion()), "L721B-10", _errors);
        _errors =
            internalRequire(getProxyImplementation(_admin, address(_bridge)) == l1ERC721BridgeImpl, "L721B-20", _errors);

        IL1CrossDomainMessenger _l1XDM = IL1CrossDomainMessenger(_sysCfg.l1CrossDomainMessenger());
        _errors = internalRequire(address(_bridge.OTHER_BRIDGE()) == Predeploys.L2_ERC721_BRIDGE, "L721B-30", _errors);
        _errors = internalRequire(address(_bridge.otherBridge()) == Predeploys.L2_ERC721_BRIDGE, "L721B-40", _errors);
        _errors = internalRequire(address(_bridge.MESSENGER()) == address(_l1XDM), "L721B-50", _errors);
        _errors = internalRequire(address(_bridge.messenger()) == address(_l1XDM), "L721B-60", _errors);
        _errors = internalRequire(address(_bridge.systemConfig()) == address(_sysCfg), "L721B-70", _errors);
        _errors = internalRequire(getProxyAdmin(address(_bridge)) == _admin, "L721B-80", _errors);
        return _errors;
    }

    /// @notice Asserts that the OptimismPortal contract is valid.
    function assertValidOptimismPortal(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IOptimismPortal2 _portal = IOptimismPortal2(payable(_sysCfg.optimismPortal()));
        _errors =
            internalRequire(LibString.eq(getVersion(address(_portal)), optimismPortalVersion()), "PORTAL-10", _errors);
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_portal)) == optimismPortalImpl, "PORTAL-20", _errors
        );

        IDisputeGameFactory _dgf = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors = internalRequire(address(_portal.disputeGameFactory()) == address(_dgf), "PORTAL-30", _errors);
        _errors = internalRequire(address(_portal.systemConfig()) == address(_sysCfg), "PORTAL-40", _errors);
        _errors = internalRequire(_portal.l2Sender() == Constants.DEFAULT_L2_SENDER, "PORTAL-80", _errors);
        _errors = internalRequire(getProxyAdmin(address(_portal)) == _admin, "PORTAL-90", _errors);
        return _errors;
    }

    /// @notice Asserts that the ETHLockbox contract is valid.
    function assertValidETHLockbox(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin
    )
        internal
        view
        returns (string memory)
    {
        IOptimismPortal2 _portal = IOptimismPortal2(payable(_sysCfg.optimismPortal()));
        IETHLockbox _lockbox = IETHLockbox(_portal.ethLockbox());

        _errors =
            internalRequire(LibString.eq(getVersion(address(_lockbox)), ethLockboxVersion()), "LOCKBOX-10", _errors);
        _errors =
            internalRequire(getProxyImplementation(_admin, address(_lockbox)) == ethLockboxImpl, "LOCKBOX-20", _errors);
        _errors = internalRequire(getProxyAdmin(address(_lockbox)) == _admin, "LOCKBOX-30", _errors);
        _errors = internalRequire(_lockbox.systemConfig() == _sysCfg, "LOCKBOX-40", _errors);
        _errors = internalRequire(_lockbox.authorizedPortals(_portal), "LOCKBOX-50", _errors);
        return _errors;
    }

    /// @notice Asserts that the DisputeGameFactory contract is valid.
    function assertValidDisputeGameFactory(
        string memory _errors,
        ISystemConfig _sysCfg,
        IProxyAdmin _admin,
        ValidationOverrides memory _overrides
    )
        internal
        view
        returns (string memory)
    {
        address _l1PAOMultisig = expectedL1PAOMultisig(_overrides);
        IDisputeGameFactory _factory = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        _errors =
            internalRequire(LibString.eq(getVersion(address(_factory)), disputeGameFactoryVersion()), "DF-10", _errors);
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_factory)) == disputeGameFactoryImpl, "DF-20", _errors
        );
        _errors = internalRequire(_factory.owner() == _l1PAOMultisig, "DF-30", _errors);
        _errors = internalRequire(getProxyAdmin(address(_factory)) == _admin, "DF-40", _errors);
        return _errors;
    }

    /// @notice Asserts that the PermissionedDisputeGame contract is valid.
    function assertValidPermissionedDisputeGame(
        string memory _errors,
        ISystemConfig _sysCfg,
        bytes32 _absolutePrestate,
        uint256 _l2ChainID,
        IProxyAdmin _admin,
        ValidationOverrides memory _overrides
    )
        internal
        view
        returns (string memory)
    {
        IDisputeGameFactory _factory = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        IPermissionedDisputeGame _game =
            IPermissionedDisputeGame(address(_factory.gameImpls(GameTypes.PERMISSIONED_CANNON)));

        if (address(_game) == address(0)) {
            _errors = internalRequire(false, "PDDG-10", _errors);
            // Return early to avoid reverting, since this means that there is no valid game impl
            // for this game type.
            return _errors;
        }

        _errors = assertValidDisputeGame(
            _errors,
            _sysCfg,
            _game,
            _factory,
            _absolutePrestate,
            _l2ChainID,
            _admin,
            GameTypes.PERMISSIONED_CANNON,
            _overrides,
            "PDDG"
        );

        // Challenger is specific to the PermissionedDisputeGame contract.
        address _challenger = expectedChallenger(_overrides);
        _errors = internalRequire(_game.challenger() == _challenger, "PDDG-130", _errors);

        return _errors;
    }

    /// @notice Asserts that the PermissionlessDisputeGame contract is valid.
    function assertValidPermissionlessDisputeGame(
        string memory _errors,
        ISystemConfig _sysCfg,
        bytes32 _absolutePrestate,
        uint256 _l2ChainID,
        IProxyAdmin _admin,
        ValidationOverrides memory _overrides
    )
        internal
        view
        returns (string memory)
    {
        IDisputeGameFactory _factory = IDisputeGameFactory(_sysCfg.disputeGameFactory());
        IPermissionedDisputeGame _game = IPermissionedDisputeGame(address(_factory.gameImpls(GameTypes.CANNON)));

        if (address(_game) == address(0)) {
            _errors = internalRequire(false, "PLDG-10", _errors);
            // Return early to avoid reverting, since this means that there is no valid game impl
            // for this game type.
            return _errors;
        }

        _errors = assertValidDisputeGame(
            _errors,
            _sysCfg,
            _game,
            _factory,
            _absolutePrestate,
            _l2ChainID,
            _admin,
            GameTypes.CANNON,
            _overrides,
            "PLDG"
        );

        return _errors;
    }

    /// @notice Asserts that a DisputeGame contract is valid.
    function assertValidDisputeGame(
        string memory _errors,
        ISystemConfig _sysCfg,
        IPermissionedDisputeGame _game,
        IDisputeGameFactory _factory,
        bytes32 _absolutePrestate,
        uint256 _l2ChainID,
        IProxyAdmin _admin,
        GameType _gameType,
        ValidationOverrides memory _overrides,
        string memory _errorPrefix
    )
        internal
        view
        returns (string memory)
    {
        IAnchorStateRegistry _asr = _game.anchorStateRegistry();
        (Hash anchorRoot,) = _asr.getAnchorRoot();

        _errors = internalRequire(
            LibString.eq(getVersion(address(_game)), permissionedDisputeGameVersion()),
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        _errors = internalRequire(
            GameType.unwrap(_game.gameType()) == GameType.unwrap(_gameType), string.concat(_errorPrefix, "-30"), _errors
        );
        _errors = internalRequire(
            Claim.unwrap(_game.absolutePrestate()) == _absolutePrestate, string.concat(_errorPrefix, "-40"), _errors
        );
        _errors = internalRequire(_game.l2ChainId() == _l2ChainID, string.concat(_errorPrefix, "-60"), _errors);
        _errors = internalRequire(_game.l2SequenceNumber() == 0, string.concat(_errorPrefix, "-70"), _errors);
        _errors = internalRequire(
            Duration.unwrap(_game.clockExtension()) == 10800, string.concat(_errorPrefix, "-80"), _errors
        );
        _errors = internalRequire(_game.splitDepth() == 30, string.concat(_errorPrefix, "-90"), _errors);
        _errors = internalRequire(_game.maxGameDepth() == 73, string.concat(_errorPrefix, "-100"), _errors);
        _errors = internalRequire(
            Duration.unwrap(_game.maxClockDuration()) == 302400, string.concat(_errorPrefix, "-110"), _errors
        );
        _errors = internalRequire(Hash.unwrap(anchorRoot) != bytes32(0), string.concat(_errorPrefix, "-120"), _errors);

        _errors = assertValidDelayedWETH(_errors, _sysCfg, _game.weth(), _admin, _overrides, _errorPrefix);
        _errors = assertValidAnchorStateRegistry(_errors, _sysCfg, _factory, _asr, _admin, _errorPrefix);

        _errors = assertValidMipsVm(_errors, IMIPS64(address(_game.vm())), _errorPrefix);

        // Only assert valid preimage oracle if the game VM is valid, since otherwise
        // the contract is likely to revert.
        if (address(_game.vm()) == mipsImpl) {
            _errors = assertValidPreimageOracle(_errors, _game.vm().oracle(), _errorPrefix);
        }

        return _errors;
    }

    /// @notice Asserts that the DelayedWETH contract is valid.
    function assertValidDelayedWETH(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDelayedWETH _weth,
        IProxyAdmin _admin,
        ValidationOverrides memory _overrides,
        string memory _errorPrefix
    )
        internal
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-DWETH");
        _errors = internalRequire(
            LibString.eq(getVersion(address(_weth)), delayedWETHVersion()), string.concat(_errorPrefix, "-10"), _errors
        );
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_weth)) == delayedWETHImpl,
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        address _l1PAOMultisig = expectedL1PAOMultisig(_overrides);
        _errors =
            internalRequire(_weth.proxyAdminOwner() == _l1PAOMultisig, string.concat(_errorPrefix, "-30"), _errors);
        _errors = internalRequire(_weth.delay() == withdrawalDelaySeconds, string.concat(_errorPrefix, "-40"), _errors);
        _errors = internalRequire(_weth.systemConfig() == _sysCfg, string.concat(_errorPrefix, "-50"), _errors);
        _errors = internalRequire(getProxyAdmin(address(_weth)) == _admin, string.concat(_errorPrefix, "-60"), _errors);
        return _errors;
    }

    /// @notice Asserts that the AnchorStateRegistry contract is valid.
    function assertValidAnchorStateRegistry(
        string memory _errors,
        ISystemConfig _sysCfg,
        IDisputeGameFactory _dgf,
        IAnchorStateRegistry _asr,
        IProxyAdmin _admin,
        string memory _errorPrefix
    )
        internal
        view
        virtual
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-ANCHORP");
        _errors = internalRequire(
            LibString.eq(getVersion(address(_asr)), anchorStateRegistryVersion()),
            string.concat(_errorPrefix, "-10"),
            _errors
        );
        _errors = internalRequire(
            getProxyImplementation(_admin, address(_asr)) == anchorStateRegistryImpl,
            string.concat(_errorPrefix, "-20"),
            _errors
        );
        _errors = internalRequire(
            address(_asr.disputeGameFactory()) == address(_dgf), string.concat(_errorPrefix, "-30"), _errors
        );
        _errors = internalRequire(_asr.systemConfig() == _sysCfg, string.concat(_errorPrefix, "-40"), _errors);
        _errors = internalRequire(getProxyAdmin(address(_asr)) == _admin, string.concat(_errorPrefix, "-50"), _errors);
        _errors = internalRequire(_asr.retirementTimestamp() > 0, string.concat(_errorPrefix, "-60"), _errors);
        return _errors;
    }

    /// @notice Asserts that the MipsVm contract is valid.
    function assertValidMipsVm(
        string memory _errors,
        IMIPS64 _mips,
        string memory _errorPrefix
    )
        internal
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-VM");
        _errors = internalRequire(address(_mips) == mipsImpl, string.concat(_errorPrefix, "-10"), _errors);
        _errors = internalRequire(
            LibString.eq(getVersion(address(_mips)), mipsVersion()), string.concat(_errorPrefix, "-20"), _errors
        );
        _errors = internalRequire(_mips.stateVersion() == 8, string.concat(_errorPrefix, "-30"), _errors);
        return _errors;
    }

    /// @notice Asserts that the PreimageOracle contract is valid.
    function assertValidPreimageOracle(
        string memory _errors,
        IPreimageOracle _oracle,
        string memory _errorPrefix
    )
        internal
        view
        returns (string memory)
    {
        _errorPrefix = string.concat(_errorPrefix, "-PIMGO");
        // The preimage oracle's address is correct if the MIPS address is correct.
        _errors = internalRequire(
            LibString.eq(getVersion(address(_oracle)), preimageOracleVersion()),
            string.concat(_errorPrefix, "-10"),
            _errors
        );
        _errors = internalRequire(_oracle.challengePeriod() == 86400, string.concat(_errorPrefix, "-20"), _errors);
        _errors = internalRequire(_oracle.minProposalSize() == 126000, string.concat(_errorPrefix, "-30"), _errors);
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

    /// @notice Validates the configuration of the L1 contracts.
    function validate(ValidationInput memory _input, bool _allowFailure) external view returns (string memory) {
        return validateWithOverrides(
            _input, _allowFailure, ValidationOverrides({ l1PAOMultisig: address(0), challenger: address(0) })
        );
    }

    /// @notice Validates the configuration of the L1 contracts. Supports overrides of certain storage values denoted in
    /// the ValidationOverrides struct.
    function validateWithOverrides(
        ValidationInput memory _input,
        bool _allowFailure,
        ValidationOverrides memory _overrides
    )
        public
        view
        returns (string memory)
    {
        string memory _errors = "";

        _errors = assertValidSuperchainConfig(_errors);
        _errors = assertValidProxyAdmin(_errors, _input.proxyAdmin, _overrides);
        _errors = assertValidSystemConfig(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidL1CrossDomainMessenger(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidL1StandardBridge(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidOptimismMintableERC20Factory(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidL1ERC721Bridge(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidOptimismPortal(_errors, _input.sysCfg, _input.proxyAdmin);
        _errors = assertValidDisputeGameFactory(_errors, _input.sysCfg, _input.proxyAdmin, _overrides);
        _errors = assertValidPermissionedDisputeGame(
            _errors, _input.sysCfg, _input.absolutePrestate, _input.l2ChainID, _input.proxyAdmin, _overrides
        );
        _errors = assertValidPermissionlessDisputeGame(
            _errors, _input.sysCfg, _input.absolutePrestate, _input.l2ChainID, _input.proxyAdmin, _overrides
        );
        _errors = assertValidETHLockbox(_errors, _input.sysCfg, _input.proxyAdmin);

        string memory overridesString = getOverridesString(_overrides);
        string memory finalErrors = _errors;

        // Handle overrides if present
        if (bytes(overridesString).length > 0) {
            // If we have both overrides and errors, combine them
            if (bytes(_errors).length > 0) {
                finalErrors = string.concat(overridesString, ",", _errors);
            } else {
                // If we only have overrides, use them as the final message
                finalErrors = overridesString;
            }
        }

        // Handle validation failure
        if (bytes(_errors).length > 0 && !_allowFailure) {
            revert(string.concat("OPContractsManagerStandardValidator: ", finalErrors));
        }

        return finalErrors;
    }
}
