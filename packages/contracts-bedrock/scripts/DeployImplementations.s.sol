// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { LibString } from "@solady/utils/LibString.sol";

import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";

import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Bytes } from "src/libraries/Bytes.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";

import { DelayedWETH } from "src/dispute/DelayedWETH.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

import { OPContractsManagerInterop } from "src/L1/OPContractsManagerInterop.sol";
import { OptimismPortalInterop } from "src/L1/OptimismPortalInterop.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";

import { Blueprint } from "src/libraries/Blueprint.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
import { BaseDeployIO } from "scripts/utils/BaseDeployIO.sol";

// See DeploySuperchain.s.sol for detailed comments on the script architecture used here.
contract DeployImplementationsInput is BaseDeployIO {
    bytes32 internal _salt;
    uint256 internal _withdrawalDelaySeconds;
    uint256 internal _minProposalSizeBytes;
    uint256 internal _challengePeriodSeconds;
    uint256 internal _proofMaturityDelaySeconds;
    uint256 internal _disputeGameFinalityDelaySeconds;

    // The release version to set OPCM implementations for, of the format `op-contracts/vX.Y.Z`.
    string internal _release;

    // Outputs from DeploySuperchain.s.sol.
    SuperchainConfig internal _superchainConfigProxy;
    ProtocolVersions internal _protocolVersionsProxy;

    string internal _standardVersionsToml;

    function set(bytes4 sel, uint256 _value) public {
        require(_value != 0, "DeployImplementationsInput: cannot set zero value");

        if (sel == this.withdrawalDelaySeconds.selector) {
            _withdrawalDelaySeconds = _value;
        } else if (sel == this.minProposalSizeBytes.selector) {
            _minProposalSizeBytes = _value;
        } else if (sel == this.challengePeriodSeconds.selector) {
            require(_value <= type(uint64).max, "DeployImplementationsInput: challengePeriodSeconds too large");
            _challengePeriodSeconds = _value;
        } else if (sel == this.proofMaturityDelaySeconds.selector) {
            _proofMaturityDelaySeconds = _value;
        } else if (sel == this.disputeGameFinalityDelaySeconds.selector) {
            _disputeGameFinalityDelaySeconds = _value;
        } else {
            revert("DeployImplementationsInput: unknown selector");
        }
    }

    function set(bytes4 sel, string memory _value) public {
        require(!LibString.eq(_value, ""), "DeployImplementationsInput: cannot set empty string");
        if (sel == this.release.selector) _release = _value;
        else if (sel == this.standardVersionsToml.selector) _standardVersionsToml = _value;
        else revert("DeployImplementationsInput: unknown selector");
    }

    function set(bytes4 sel, address _addr) public {
        require(_addr != address(0), "DeployImplementationsInput: cannot set zero address");
        if (sel == this.superchainConfigProxy.selector) _superchainConfigProxy = SuperchainConfig(_addr);
        else if (sel == this.protocolVersionsProxy.selector) _protocolVersionsProxy = ProtocolVersions(_addr);
        else revert("DeployImplementationsInput: unknown selector");
    }

    function set(bytes4 sel, bytes32 _value) public {
        if (sel == this.salt.selector) _salt = _value;
        else revert("DeployImplementationsInput: unknown selector");
    }

    function salt() public view returns (bytes32) {
        // TODO check if implementations are deployed based on code+salt and skip deploy if so.
        return _salt;
    }

    function withdrawalDelaySeconds() public view returns (uint256) {
        require(_withdrawalDelaySeconds != 0, "DeployImplementationsInput: not set");
        return _withdrawalDelaySeconds;
    }

    function minProposalSizeBytes() public view returns (uint256) {
        require(_minProposalSizeBytes != 0, "DeployImplementationsInput: not set");
        return _minProposalSizeBytes;
    }

    function challengePeriodSeconds() public view returns (uint256) {
        require(_challengePeriodSeconds != 0, "DeployImplementationsInput: not set");
        require(
            _challengePeriodSeconds <= type(uint64).max, "DeployImplementationsInput: challengePeriodSeconds too large"
        );
        return _challengePeriodSeconds;
    }

    function proofMaturityDelaySeconds() public view returns (uint256) {
        require(_proofMaturityDelaySeconds != 0, "DeployImplementationsInput: not set");
        return _proofMaturityDelaySeconds;
    }

    function disputeGameFinalityDelaySeconds() public view returns (uint256) {
        require(_disputeGameFinalityDelaySeconds != 0, "DeployImplementationsInput: not set");
        return _disputeGameFinalityDelaySeconds;
    }

    function release() public view returns (string memory) {
        require(!LibString.eq(_release, ""), "DeployImplementationsInput: not set");
        return _release;
    }

    function standardVersionsToml() public view returns (string memory) {
        require(!LibString.eq(_standardVersionsToml, ""), "DeployImplementationsInput: not set");
        return _standardVersionsToml;
    }

    function superchainConfigProxy() public view returns (SuperchainConfig) {
        require(address(_superchainConfigProxy) != address(0), "DeployImplementationsInput: not set");
        return _superchainConfigProxy;
    }

    function protocolVersionsProxy() public view returns (ProtocolVersions) {
        require(address(_protocolVersionsProxy) != address(0), "DeployImplementationsInput: not set");
        return _protocolVersionsProxy;
    }

    function superchainProxyAdmin() public returns (ProxyAdmin) {
        SuperchainConfig proxy = this.superchainConfigProxy();
        // Can infer the superchainProxyAdmin from the superchainConfigProxy.
        vm.prank(address(0));
        ProxyAdmin proxyAdmin = ProxyAdmin(Proxy(payable(address(proxy))).admin());
        require(address(proxyAdmin) != address(0), "DeployImplementationsInput: not set");
        return proxyAdmin;
    }
}

contract DeployImplementationsOutput is BaseDeployIO {
    OPContractsManager internal _opcmProxy;
    OPContractsManager internal _opcmImpl;
    DelayedWETH internal _delayedWETHImpl;
    OptimismPortal2 internal _optimismPortalImpl;
    PreimageOracle internal _preimageOracleSingleton;
    MIPS internal _mipsSingleton;
    SystemConfig internal _systemConfigImpl;
    L1CrossDomainMessenger internal _l1CrossDomainMessengerImpl;
    L1ERC721Bridge internal _l1ERC721BridgeImpl;
    L1StandardBridge internal _l1StandardBridgeImpl;
    OptimismMintableERC20Factory internal _optimismMintableERC20FactoryImpl;
    DisputeGameFactory internal _disputeGameFactoryImpl;

    function set(bytes4 sel, address _addr) public {
        require(_addr != address(0), "DeployImplementationsOutput: cannot set zero address");

        // forgefmt: disable-start
        if (sel == this.opcmProxy.selector) _opcmProxy = OPContractsManager(payable(_addr));
        else if (sel == this.opcmImpl.selector) _opcmImpl = OPContractsManager(payable(_addr));
        else if (sel == this.optimismPortalImpl.selector) _optimismPortalImpl = OptimismPortal2(payable(_addr));
        else if (sel == this.delayedWETHImpl.selector) _delayedWETHImpl = DelayedWETH(payable(_addr));
        else if (sel == this.preimageOracleSingleton.selector) _preimageOracleSingleton = PreimageOracle(_addr);
        else if (sel == this.mipsSingleton.selector) _mipsSingleton = MIPS(_addr);
        else if (sel == this.systemConfigImpl.selector) _systemConfigImpl = SystemConfig(_addr);
        else if (sel == this.l1CrossDomainMessengerImpl.selector) _l1CrossDomainMessengerImpl = L1CrossDomainMessenger(_addr);
        else if (sel == this.l1ERC721BridgeImpl.selector) _l1ERC721BridgeImpl = L1ERC721Bridge(_addr);
        else if (sel == this.l1StandardBridgeImpl.selector) _l1StandardBridgeImpl = L1StandardBridge(payable(_addr));
        else if (sel == this.optimismMintableERC20FactoryImpl.selector) _optimismMintableERC20FactoryImpl = OptimismMintableERC20Factory(_addr);
        else if (sel == this.disputeGameFactoryImpl.selector) _disputeGameFactoryImpl = DisputeGameFactory(_addr);
        else revert("DeployImplementationsOutput: unknown selector");
        // forgefmt: disable-end
    }

    function checkOutput(DeployImplementationsInput _dii) public {
        // With 12 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
        address[] memory addrs1 = Solarray.addresses(
            address(this.opcmProxy()),
            address(this.opcmImpl()),
            address(this.optimismPortalImpl()),
            address(this.delayedWETHImpl()),
            address(this.preimageOracleSingleton()),
            address(this.mipsSingleton())
        );

        address[] memory addrs2 = Solarray.addresses(
            address(this.systemConfigImpl()),
            address(this.l1CrossDomainMessengerImpl()),
            address(this.l1ERC721BridgeImpl()),
            address(this.l1StandardBridgeImpl()),
            address(this.optimismMintableERC20FactoryImpl()),
            address(this.disputeGameFactoryImpl())
        );

        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));

        assertValidDeploy(_dii);
    }

    function opcmProxy() public returns (OPContractsManager) {
        DeployUtils.assertValidContractAddress(address(_opcmProxy));
        DeployUtils.assertImplementationSet(address(_opcmProxy));
        return _opcmProxy;
    }

    function opcmImpl() public view returns (OPContractsManager) {
        DeployUtils.assertValidContractAddress(address(_opcmImpl));
        return _opcmImpl;
    }

    function optimismPortalImpl() public view returns (OptimismPortal2) {
        DeployUtils.assertValidContractAddress(address(_optimismPortalImpl));
        return _optimismPortalImpl;
    }

    function delayedWETHImpl() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(_delayedWETHImpl));
        return _delayedWETHImpl;
    }

    function preimageOracleSingleton() public view returns (PreimageOracle) {
        DeployUtils.assertValidContractAddress(address(_preimageOracleSingleton));
        return _preimageOracleSingleton;
    }

    function mipsSingleton() public view returns (MIPS) {
        DeployUtils.assertValidContractAddress(address(_mipsSingleton));
        return _mipsSingleton;
    }

    function systemConfigImpl() public view returns (SystemConfig) {
        DeployUtils.assertValidContractAddress(address(_systemConfigImpl));
        return _systemConfigImpl;
    }

    function l1CrossDomainMessengerImpl() public view returns (L1CrossDomainMessenger) {
        DeployUtils.assertValidContractAddress(address(_l1CrossDomainMessengerImpl));
        return _l1CrossDomainMessengerImpl;
    }

    function l1ERC721BridgeImpl() public view returns (L1ERC721Bridge) {
        DeployUtils.assertValidContractAddress(address(_l1ERC721BridgeImpl));
        return _l1ERC721BridgeImpl;
    }

    function l1StandardBridgeImpl() public view returns (L1StandardBridge) {
        DeployUtils.assertValidContractAddress(address(_l1StandardBridgeImpl));
        return _l1StandardBridgeImpl;
    }

    function optimismMintableERC20FactoryImpl() public view returns (OptimismMintableERC20Factory) {
        DeployUtils.assertValidContractAddress(address(_optimismMintableERC20FactoryImpl));
        return _optimismMintableERC20FactoryImpl;
    }

    function disputeGameFactoryImpl() public view returns (DisputeGameFactory) {
        DeployUtils.assertValidContractAddress(address(_disputeGameFactoryImpl));
        return _disputeGameFactoryImpl;
    }

    // -------- Deployment Assertions --------
    function assertValidDeploy(DeployImplementationsInput _dii) public {
        assertValidDelayedWETHImpl(_dii);
        assertValidDisputeGameFactoryImpl(_dii);
        assertValidL1CrossDomainMessengerImpl(_dii);
        assertValidL1ERC721BridgeImpl(_dii);
        assertValidL1StandardBridgeImpl(_dii);
        assertValidMipsSingleton(_dii);
        assertValidOpcmProxy(_dii);
        assertValidOpcmImpl(_dii);
        assertValidOptimismMintableERC20FactoryImpl(_dii);
        assertValidOptimismPortalImpl(_dii);
        assertValidPreimageOracleSingleton(_dii);
        assertValidSystemConfigImpl(_dii);
    }

    function assertValidOpcmProxy(DeployImplementationsInput _dii) internal {
        // First we check the proxy as itself.
        Proxy proxy = Proxy(payable(address(opcmProxy())));
        vm.prank(address(0));
        address admin = proxy.admin();
        require(admin == address(_dii.superchainProxyAdmin()), "OPCMP-10");

        // Then we check the proxy as OPCM.
        DeployUtils.assertInitialized({ _contractAddress: address(opcmProxy()), _slot: 0, _offset: 0 });
        require(address(opcmProxy().superchainConfig()) == address(_dii.superchainConfigProxy()), "OPCMP-20");
        require(address(opcmProxy().protocolVersions()) == address(_dii.protocolVersionsProxy()), "OPCMP-30");
        require(LibString.eq(opcmProxy().latestRelease(), _dii.release()), "OPCMP-50"); // Initial release is latest.
    }

    function assertValidOpcmImpl(DeployImplementationsInput _dii) internal {
        Proxy proxy = Proxy(payable(address(opcmProxy())));
        vm.prank(address(0));
        OPContractsManager impl = OPContractsManager(proxy.implementation());
        DeployUtils.assertInitialized({ _contractAddress: address(impl), _slot: 0, _offset: 0 });
        require(address(impl.superchainConfig()) == address(_dii.superchainConfigProxy()), "OPCMI-10");
        require(address(impl.protocolVersions()) == address(_dii.protocolVersionsProxy()), "OPCMI-20");
    }

    function assertValidOptimismPortalImpl(DeployImplementationsInput) internal view {
        OptimismPortal2 portal = optimismPortalImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(portal), _slot: 0, _offset: 0 });

        require(address(portal.disputeGameFactory()) == address(0), "PORTAL-10");
        require(address(portal.systemConfig()) == address(0), "PORTAL-20");
        require(address(portal.superchainConfig()) == address(0), "PORTAL-30");
        require(portal.l2Sender() == Constants.DEFAULT_L2_SENDER, "PORTAL-40");

        // This slot is the custom gas token _balance and this check ensures
        // that it stays unset for forwards compatibility with custom gas token.
        require(vm.load(address(portal), bytes32(uint256(61))) == bytes32(0), "PORTAL-50");
    }

    function assertValidDelayedWETHImpl(DeployImplementationsInput _dii) internal view {
        DelayedWETH delayedWETH = delayedWETHImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(delayedWETH), _slot: 0, _offset: 0 });

        require(delayedWETH.owner() == address(0), "DW-10");
        require(delayedWETH.delay() == _dii.withdrawalDelaySeconds(), "DW-20");
        require(delayedWETH.config() == ISuperchainConfig(address(0)), "DW-30");
    }

    function assertValidPreimageOracleSingleton(DeployImplementationsInput _dii) internal view {
        PreimageOracle oracle = preimageOracleSingleton();

        require(oracle.minProposalSize() == _dii.minProposalSizeBytes(), "PO-10");
        require(oracle.challengePeriod() == _dii.challengePeriodSeconds(), "PO-20");
    }

    function assertValidMipsSingleton(DeployImplementationsInput) internal view {
        MIPS mips = mipsSingleton();

        require(address(mips.oracle()) == address(preimageOracleSingleton()), "MIPS-10");
    }

    function assertValidSystemConfigImpl(DeployImplementationsInput) internal view {
        SystemConfig systemConfig = systemConfigImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(systemConfig), _slot: 0, _offset: 0 });

        require(systemConfig.owner() == address(0xdead), "SYSCON-10");
        require(systemConfig.overhead() == 0, "SYSCON-20");
        require(systemConfig.scalar() == uint256(0x01) << 248, "SYSCON-30");
        require(systemConfig.basefeeScalar() == 0, "SYSCON-40");
        require(systemConfig.blobbasefeeScalar() == 0, "SYSCON-50");
        require(systemConfig.batcherHash() == bytes32(0), "SYSCON-60");
        require(systemConfig.gasLimit() == 1, "SYSCON-70");
        require(systemConfig.unsafeBlockSigner() == address(0), "SYSCON-80");

        IResourceMetering.ResourceConfig memory resourceConfig = systemConfig.resourceConfig();
        require(resourceConfig.maxResourceLimit == 1, "SYSCON-90");
        require(resourceConfig.elasticityMultiplier == 1, "SYSCON-100");
        require(resourceConfig.baseFeeMaxChangeDenominator == 2, "SYSCON-110");
        require(resourceConfig.systemTxMaxGas == 0, "SYSCON-120");
        require(resourceConfig.minimumBaseFee == 0, "SYSCON-130");
        require(resourceConfig.maximumBaseFee == 0, "SYSCON-140");

        require(systemConfig.startBlock() == type(uint256).max, "SYSCON-150");
        require(systemConfig.batchInbox() == address(0), "SYSCON-160");
        require(systemConfig.l1CrossDomainMessenger() == address(0), "SYSCON-170");
        require(systemConfig.l1ERC721Bridge() == address(0), "SYSCON-180");
        require(systemConfig.l1StandardBridge() == address(0), "SYSCON-190");
        require(systemConfig.disputeGameFactory() == address(0), "SYSCON-200");
        require(systemConfig.optimismPortal() == address(0), "SYSCON-210");
        require(systemConfig.optimismMintableERC20Factory() == address(0), "SYSCON-220");
    }

    function assertValidL1CrossDomainMessengerImpl(DeployImplementationsInput) internal view {
        L1CrossDomainMessenger messenger = l1CrossDomainMessengerImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(messenger), _slot: 0, _offset: 20 });

        require(address(messenger.OTHER_MESSENGER()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-10");
        require(address(messenger.otherMessenger()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-20");
        require(address(messenger.PORTAL()) == address(0), "L1xDM-30");
        require(address(messenger.portal()) == address(0), "L1xDM-40");
        require(address(messenger.superchainConfig()) == address(0), "L1xDM-50");

        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER, "L1xDM-60");
    }

    function assertValidL1ERC721BridgeImpl(DeployImplementationsInput) internal view {
        L1ERC721Bridge bridge = l1ERC721BridgeImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_ERC721_BRIDGE, "L721B-10");
        require(address(bridge.otherBridge()) == Predeploys.L2_ERC721_BRIDGE, "L721B-20");
        require(address(bridge.MESSENGER()) == address(0), "L721B-30");
        require(address(bridge.messenger()) == address(0), "L721B-40");
        require(address(bridge.superchainConfig()) == address(0), "L721B-50");
    }

    function assertValidL1StandardBridgeImpl(DeployImplementationsInput) internal view {
        L1StandardBridge bridge = l1StandardBridgeImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.MESSENGER()) == address(0), "L1SB-10");
        require(address(bridge.messenger()) == address(0), "L1SB-20");
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-30");
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-40");
        require(address(bridge.superchainConfig()) == address(0), "L1SB-50");
    }

    function assertValidOptimismMintableERC20FactoryImpl(DeployImplementationsInput) internal view {
        OptimismMintableERC20Factory factory = optimismMintableERC20FactoryImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(address(factory.BRIDGE()) == address(0), "MERC20F-10");
        require(address(factory.bridge()) == address(0), "MERC20F-20");
    }

    function assertValidDisputeGameFactoryImpl(DeployImplementationsInput) internal view {
        DisputeGameFactory factory = disputeGameFactoryImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(address(factory.owner()) == address(0), "DG-10");
    }
}

contract DeployImplementations is Script {
    // -------- Core Deployment Methods --------

    function run(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public {
        // Deploy the implementations.
        deploySystemConfigImpl(_dii, _dio);
        deployL1CrossDomainMessengerImpl(_dii, _dio);
        deployL1ERC721BridgeImpl(_dii, _dio);
        deployL1StandardBridgeImpl(_dii, _dio);
        deployOptimismMintableERC20FactoryImpl(_dii, _dio);
        deployOptimismPortalImpl(_dii, _dio);
        deployDelayedWETHImpl(_dii, _dio);
        deployPreimageOracleSingleton(_dii, _dio);
        deployMipsSingleton(_dii, _dio);
        deployDisputeGameFactoryImpl(_dii, _dio);

        // Deploy the OP Contracts Manager with the new implementations set.
        deployOPContractsManager(_dii, _dio);

        _dio.checkOutput(_dii);
    }

    // -------- Deployment Steps --------

    // --- OP Contracts Manager ---

    function opcmSystemConfigSetter(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        internal
        view
        virtual
        returns (OPContractsManager.ImplementationSetter memory)
    {
        return OPContractsManager.ImplementationSetter({
            name: "SystemConfig",
            info: OPContractsManager.Implementation(address(_dio.systemConfigImpl()), SystemConfig.initialize.selector)
        });
    }

    // Deploy and initialize a proxied OPContractsManager.
    function createOPCMContract(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio,
        OPContractsManager.Blueprints memory _blueprints,
        string memory _release,
        OPContractsManager.ImplementationSetter[] memory _setters
    )
        internal
        virtual
        returns (OPContractsManager opcmProxy_)
    {
        ProxyAdmin proxyAdmin = _dii.superchainProxyAdmin();

        vm.broadcast(msg.sender);
        Proxy proxy = new Proxy(address(msg.sender));

        deployOPContractsManagerImpl(_dii, _dio);
        OPContractsManager opcmImpl = _dio.opcmImpl();

        OPContractsManager.InitializerInputs memory initializerInputs =
            OPContractsManager.InitializerInputs(_blueprints, _setters, _release, true);

        vm.startBroadcast(msg.sender);
        proxy.upgradeToAndCall(
            address(opcmImpl), abi.encodeWithSelector(opcmImpl.initialize.selector, initializerInputs)
        );

        proxy.changeAdmin(address(proxyAdmin)); // transfer ownership of Proxy contract to the ProxyAdmin contract
        vm.stopBroadcast();

        opcmProxy_ = OPContractsManager(address(proxy));
    }

    function deployOPContractsManager(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();

        // First we deploy the blueprints for the singletons deployed by OPCM.
        // forgefmt: disable-start
        bytes32 salt = _dii.salt();
        OPContractsManager.Blueprints memory blueprints;

        vm.startBroadcast(msg.sender);
        blueprints.addressManager = deployBytecode(Blueprint.blueprintDeployerBytecode(type(AddressManager).creationCode), salt);
        blueprints.proxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(Proxy).creationCode), salt);
        blueprints.proxyAdmin = deployBytecode(Blueprint.blueprintDeployerBytecode(type(ProxyAdmin).creationCode), salt);
        blueprints.l1ChugSplashProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(L1ChugSplashProxy).creationCode), salt);
        blueprints.resolvedDelegateProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(ResolvedDelegateProxy).creationCode), salt);
        blueprints.anchorStateRegistry = deployBytecode(Blueprint.blueprintDeployerBytecode(type(AnchorStateRegistry).creationCode), salt);
        (blueprints.permissionedDisputeGame1, blueprints.permissionedDisputeGame2)  = deployBigBytecode(type(PermissionedDisputeGame).creationCode, salt);
        vm.stopBroadcast();
        // forgefmt: disable-end

        OPContractsManager.ImplementationSetter[] memory setters = new OPContractsManager.ImplementationSetter[](9);
        setters[0] = OPContractsManager.ImplementationSetter({
            name: "L1ERC721Bridge",
            info: OPContractsManager.Implementation(address(_dio.l1ERC721BridgeImpl()), L1ERC721Bridge.initialize.selector)
        });
        setters[1] = OPContractsManager.ImplementationSetter({
            name: "OptimismPortal",
            info: OPContractsManager.Implementation(address(_dio.optimismPortalImpl()), OptimismPortal2.initialize.selector)
        });
        setters[2] = opcmSystemConfigSetter(_dii, _dio);
        setters[3] = OPContractsManager.ImplementationSetter({
            name: "OptimismMintableERC20Factory",
            info: OPContractsManager.Implementation(
                address(_dio.optimismMintableERC20FactoryImpl()), OptimismMintableERC20Factory.initialize.selector
            )
        });
        setters[4] = OPContractsManager.ImplementationSetter({
            name: "L1CrossDomainMessenger",
            info: OPContractsManager.Implementation(
                address(_dio.l1CrossDomainMessengerImpl()), L1CrossDomainMessenger.initialize.selector
            )
        });
        setters[5] = OPContractsManager.ImplementationSetter({
            name: "L1StandardBridge",
            info: OPContractsManager.Implementation(
                address(_dio.l1StandardBridgeImpl()), L1StandardBridge.initialize.selector
            )
        });
        setters[6] = OPContractsManager.ImplementationSetter({
            name: "DisputeGameFactory",
            info: OPContractsManager.Implementation(
                address(_dio.disputeGameFactoryImpl()), DisputeGameFactory.initialize.selector
            )
        });
        setters[7] = OPContractsManager.ImplementationSetter({
            name: "DelayedWETH",
            info: OPContractsManager.Implementation(address(_dio.delayedWETHImpl()), DelayedWETH.initialize.selector)
        });
        setters[8] = OPContractsManager.ImplementationSetter({
            name: "MIPS",
            // MIPS is a singleton for all chains, so it doesn't need to be initialized, so the
            // selector is just `bytes4(0)`.
            info: OPContractsManager.Implementation(address(_dio.mipsSingleton()), bytes4(0))
        });

        // This call contains a broadcast to deploy OPCM which is proxied.
        OPContractsManager opcmProxy = createOPCMContract(_dii, _dio, blueprints, release, setters);

        vm.label(address(opcmProxy), "OPContractsManager");
        _dio.set(_dio.opcmProxy.selector, address(opcmProxy));
    }

    // --- Core Contracts ---

    function deploySystemConfigImpl(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        // Using snake case for contract name to match the TOML file in superchain-registry.
        string memory contractName = "system_config";
        SystemConfig impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = SystemConfig(existingImplementation);
        } else if (isDevelopRelease(release)) {
            // Deploy a new implementation for development builds.
            vm.broadcast(msg.sender);
            impl = new SystemConfig();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "SystemConfigImpl");
        _dio.set(_dio.systemConfigImpl.selector, address(impl));
    }

    function deployL1CrossDomainMessengerImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "l1_cross_domain_messenger";
        L1CrossDomainMessenger impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = L1CrossDomainMessenger(existingImplementation);
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new L1CrossDomainMessenger();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "L1CrossDomainMessengerImpl");
        _dio.set(_dio.l1CrossDomainMessengerImpl.selector, address(impl));
    }

    function deployL1ERC721BridgeImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "l1_erc721_bridge";
        L1ERC721Bridge impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = L1ERC721Bridge(existingImplementation);
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new L1ERC721Bridge();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "L1ERC721BridgeImpl");
        _dio.set(_dio.l1ERC721BridgeImpl.selector, address(impl));
    }

    function deployL1StandardBridgeImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "l1_standard_bridge";
        L1StandardBridge impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = L1StandardBridge(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new L1StandardBridge();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "L1StandardBridgeImpl");
        _dio.set(_dio.l1StandardBridgeImpl.selector, address(impl));
    }

    function deployOptimismMintableERC20FactoryImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "optimism_mintable_erc20_factory";
        OptimismMintableERC20Factory impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = OptimismMintableERC20Factory(existingImplementation);
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new OptimismMintableERC20Factory();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "OptimismMintableERC20FactoryImpl");
        _dio.set(_dio.optimismMintableERC20FactoryImpl.selector, address(impl));
    }

    function deployOPContractsManagerImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        SuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        ProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();

        vm.broadcast(msg.sender);
        // TODO: Eventually we will want to select the correct implementation based on the release.
        OPContractsManager impl = new OPContractsManager(superchainConfigProxy, protocolVersionsProxy);

        vm.label(address(impl), "OPContractsManagerImpl");
        _dio.set(_dio.opcmImpl.selector, address(impl));
    }

    // --- Fault Proofs Contracts ---

    // The fault proofs contracts are configured as follows:
    // | Contract                | Proxied | Deployment                        | MCP Ready  |
    // |-------------------------|---------|-----------------------------------|------------|
    // | DisputeGameFactory      | Yes     | Bespoke                           | Yes        |
    // | AnchorStateRegistry     | Yes     | Bespoke                           | No         |
    // | FaultDisputeGame        | No      | Bespoke                           | No         | Not yet supported by OPCM
    // | PermissionedDisputeGame | No      | Bespoke                           | No         |
    // | DelayedWETH             | Yes     | Two bespoke (one per DisputeGame) | No         |
    // | PreimageOracle          | No      | Shared                            | N/A        |
    // | MIPS                    | No      | Shared                            | N/A        |
    // | OptimismPortal2         | Yes     | Shared                            | No         |
    //
    // This script only deploys the shared contracts. The bespoke contracts are deployed by
    // `DeployOPChain.s.sol`. When the shared contracts are proxied, the contracts deployed here are
    // "implementations", and when shared contracts are not proxied, they are "singletons". So
    // here we deploy:
    //
    //   - DisputeGameFactory (implementation)
    //   - OptimismPortal2 (implementation)
    //   - DelayedWETH (implementation)
    //   - PreimageOracle (singleton)
    //   - MIPS (singleton)
    //
    // For contracts which are not MCP ready neither the Proxy nor the implementation can be shared, therefore they
    // are deployed by `DeployOpChain.s.sol`.

    function deployOptimismPortalImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "optimism_portal";
        OptimismPortal2 impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = OptimismPortal2(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            uint256 proofMaturityDelaySeconds = _dii.proofMaturityDelaySeconds();
            uint256 disputeGameFinalityDelaySeconds = _dii.disputeGameFinalityDelaySeconds();
            vm.broadcast(msg.sender);
            impl = new OptimismPortal2(proofMaturityDelaySeconds, disputeGameFinalityDelaySeconds);
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(impl));
    }

    function deployDelayedWETHImpl(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "delayed_weth";
        DelayedWETH impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = DelayedWETH(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            uint256 withdrawalDelaySeconds = _dii.withdrawalDelaySeconds();
            vm.broadcast(msg.sender);
            impl = new DelayedWETH(withdrawalDelaySeconds);
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "DelayedWETHImpl");
        _dio.set(_dio.delayedWETHImpl.selector, address(impl));
    }

    function deployPreimageOracleSingleton(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "preimage_oracle";
        PreimageOracle singleton;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            singleton = PreimageOracle(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            uint256 minProposalSizeBytes = _dii.minProposalSizeBytes();
            uint256 challengePeriodSeconds = _dii.challengePeriodSeconds();
            vm.broadcast(msg.sender);
            singleton = new PreimageOracle(minProposalSizeBytes, challengePeriodSeconds);
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(singleton), "PreimageOracleSingleton");
        _dio.set(_dio.preimageOracleSingleton.selector, address(singleton));
    }

    function deployMipsSingleton(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "mips";
        MIPS singleton;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            singleton = MIPS(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            IPreimageOracle preimageOracle = IPreimageOracle(_dio.preimageOracleSingleton());
            vm.broadcast(msg.sender);
            singleton = new MIPS(preimageOracle);
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(singleton), "MIPSSingleton");
        _dio.set(_dio.mipsSingleton.selector, address(singleton));
    }

    function deployDisputeGameFactoryImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "dispute_game_factory";
        DisputeGameFactory impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = DisputeGameFactory(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new DisputeGameFactory();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "DisputeGameFactoryImpl");
        _dio.set(_dio.disputeGameFactoryImpl.selector, address(impl));
    }

    // -------- Utilities --------

    function etchIOContracts() public returns (DeployImplementationsInput dii_, DeployImplementationsOutput dio_) {
        (dii_, dio_) = getIOContracts();
        vm.etch(address(dii_), type(DeployImplementationsInput).runtimeCode);
        vm.etch(address(dio_), type(DeployImplementationsOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployImplementationsInput dii_, DeployImplementationsOutput dio_) {
        dii_ = DeployImplementationsInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployImplementationsInput"));
        dio_ = DeployImplementationsOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployImplementationsOutput"));
    }

    function deployBytecode(bytes memory _bytecode, bytes32 _salt) public returns (address newContract_) {
        assembly ("memory-safe") {
            newContract_ := create2(0, add(_bytecode, 0x20), mload(_bytecode), _salt)
        }
        require(newContract_ != address(0), "DeployImplementations: create2 failed");
    }

    function deployBigBytecode(
        bytes memory _bytecode,
        bytes32 _salt
    )
        public
        returns (address newContract1_, address newContract2_)
    {
        // Preamble needs 3 bytes.
        uint256 maxInitCodeSize = 24576 - 3;
        require(_bytecode.length > maxInitCodeSize, "DeployImplementations: Use deployBytecode instead");

        bytes memory part1Slice = Bytes.slice(_bytecode, 0, maxInitCodeSize);
        bytes memory part1 = Blueprint.blueprintDeployerBytecode(part1Slice);
        bytes memory part2Slice = Bytes.slice(_bytecode, maxInitCodeSize, _bytecode.length - maxInitCodeSize);
        bytes memory part2 = Blueprint.blueprintDeployerBytecode(part2Slice);

        newContract1_ = deployBytecode(part1, _salt);
        newContract2_ = deployBytecode(part2, _salt);
    }

    // Zero address is returned if the address is not found in '_standardVersionsToml'.
    function getReleaseAddress(
        string memory _version,
        string memory _contractName,
        string memory _standardVersionsToml
    )
        internal
        pure
        returns (address addr_)
    {
        string memory baseKey = string.concat('.releases["', _version, '"].', _contractName);
        string memory implAddressKey = string.concat(baseKey, ".implementation_address");
        string memory addressKey = string.concat(baseKey, ".address");
        try vm.parseTomlAddress(_standardVersionsToml, implAddressKey) returns (address parsedAddr_) {
            addr_ = parsedAddr_;
        } catch {
            try vm.parseTomlAddress(_standardVersionsToml, addressKey) returns (address parsedAddr_) {
                addr_ = parsedAddr_;
            } catch {
                addr_ = address(0);
            }
        }
    }

    // A release is considered a 'develop' release if it does not start with 'op-contracts'.
    function isDevelopRelease(string memory _release) internal pure returns (bool) {
        return !LibString.startsWith(_release, "op-contracts");
    }
}

// Similar to how DeploySuperchain.s.sol contains a lot of comments to thoroughly document the script
// architecture, this comment block documents how to update the deploy scripts to support new features.
//
// Using the base scripts and contracts (DeploySuperchain, DeployImplementations, DeployOPChain, and
// the corresponding OPContractsManager) deploys a standard chain. For nonstandard and in-development
// features we need to modify some or all of those contracts, and we do that via inheritance. Using
// interop as an example, they've made the following changes to L1 contracts:
//   - `OptimismPortalInterop is OptimismPortal`: A different portal implementation is used, and
//     it's ABI is the same.
//   - `SystemConfigInterop is SystemConfig`: A different system config implementation is used, and
//     it's initializer has a different signature. This signature is different because there is a
//     new input parameter, the `dependencyManager`.
//   - Because of the different system config initializer, there is a new input parameter (dependencyManager).
//
// Similar to how inheritance was used to develop the new portal and system config contracts, we use
// inheritance to modify up to all of the deployer contracts. For this interop example, what this
// means is we need:
//   - An `OPContractsManagerInterop is OPContractsManager` that knows how to encode the calldata for the
//     new system config initializer.
//   - A `DeployImplementationsInterop is DeployImplementations` that:
//     - Deploys OptimismPortalInterop instead of OptimismPortal.
//     - Deploys SystemConfigInterop instead of SystemConfig.
//     - Deploys OPContractsManagerInterop instead of OPContractsManager, which contains the updated logic
//       for encoding the SystemConfig initializer.
//     - Updates the OPCM release setter logic to use the updated initializer.
//  - A `DeployOPChainInterop is DeployOPChain` that allows the updated input parameter to be passed.
//
// Most of the complexity in the above flow comes from the the new input for the updated SystemConfig
// initializer. If all function signatures were the same, all we'd have to change is the contract
// implementations that are deployed then set in the OPCM. For now, to simplify things until we
// resolve https://github.com/ethereum-optimism/optimism/issues/11783, we just assume this new role
// is the same as the proxy admin owner.
contract DeployImplementationsInterop is DeployImplementations {
    function createOPCMContract(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio,
        OPContractsManager.Blueprints memory _blueprints,
        string memory _release,
        OPContractsManager.ImplementationSetter[] memory _setters
    )
        internal
        override
        returns (OPContractsManager opcmProxy_)
    {
        ProxyAdmin proxyAdmin = _dii.superchainProxyAdmin();

        vm.broadcast(msg.sender);
        Proxy proxy = new Proxy(address(msg.sender));

        deployOPContractsManagerImpl(_dii, _dio); // overriding function
        OPContractsManager opcmImpl = _dio.opcmImpl();

        OPContractsManager.InitializerInputs memory initializerInputs =
            OPContractsManager.InitializerInputs(_blueprints, _setters, _release, true);

        vm.startBroadcast(msg.sender);
        proxy.upgradeToAndCall(
            address(opcmImpl), abi.encodeWithSelector(opcmImpl.initialize.selector, initializerInputs)
        );

        proxy.changeAdmin(address(proxyAdmin)); // transfer ownership of Proxy contract to the ProxyAdmin contract
        vm.stopBroadcast();

        opcmProxy_ = OPContractsManagerInterop(address(proxy));
    }

    function deployOptimismPortalImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        override
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();
        string memory contractName = "optimism_portal";
        OptimismPortal2 impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = OptimismPortalInterop(payable(existingImplementation));
        } else if (isDevelopRelease(release)) {
            uint256 proofMaturityDelaySeconds = _dii.proofMaturityDelaySeconds();
            uint256 disputeGameFinalityDelaySeconds = _dii.disputeGameFinalityDelaySeconds();
            vm.broadcast(msg.sender);
            impl = new OptimismPortalInterop(proofMaturityDelaySeconds, disputeGameFinalityDelaySeconds);
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(impl));
    }

    function deploySystemConfigImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        override
    {
        string memory release = _dii.release();
        string memory stdVerToml = _dii.standardVersionsToml();

        string memory contractName = "system_config";
        SystemConfig impl;

        address existingImplementation = getReleaseAddress(release, contractName, stdVerToml);
        if (existingImplementation != address(0)) {
            impl = SystemConfigInterop(existingImplementation);
        } else if (isDevelopRelease(release)) {
            vm.broadcast(msg.sender);
            impl = new SystemConfigInterop();
        } else {
            revert(string.concat("DeployImplementations: failed to deploy release ", release));
        }

        vm.label(address(impl), "SystemConfigImpl");
        _dio.set(_dio.systemConfigImpl.selector, address(impl));
    }

    function deployOPContractsManagerImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        override
    {
        SuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        ProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();

        vm.broadcast(msg.sender);
        // TODO: Eventually we will want to select the correct implementation based on the release.
        OPContractsManager impl = new OPContractsManagerInterop(superchainConfigProxy, protocolVersionsProxy);

        vm.label(address(impl), "OPContractsManagerImpl");
        _dio.set(_dio.opcmImpl.selector, address(impl));
    }

    function opcmSystemConfigSetter(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        internal
        view
        override
        returns (OPContractsManager.ImplementationSetter memory)
    {
        return OPContractsManager.ImplementationSetter({
            name: "SystemConfig",
            info: OPContractsManager.Implementation(
                address(_dio.systemConfigImpl()), SystemConfigInterop.initialize.selector
            )
        });
    }
}
