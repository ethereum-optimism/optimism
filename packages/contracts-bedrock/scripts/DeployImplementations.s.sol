// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

import { LibString } from "@solady/utils/LibString.sol";

import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

import { IProxyAdmin } from "src/universal/interfaces/IProxyAdmin.sol";
import { IProxy } from "src/universal/interfaces/IProxy.sol";
import { IOptimismMintableERC20Factory } from "src/universal/interfaces/IOptimismMintableERC20Factory.sol";

import { IL1ChugSplashProxy } from "src/legacy/interfaces/IL1ChugSplashProxy.sol";
import { IResolvedDelegateProxy } from "src/legacy/interfaces/IResolvedDelegateProxy.sol";
import { IAddressManager } from "src/legacy/interfaces/IAddressManager.sol";

import { IDelayedWETH } from "src/dispute/interfaces/IDelayedWETH.sol";
import { IDisputeGameFactory } from "src/dispute/interfaces/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "src/dispute/interfaces/IAnchorStateRegistry.sol";

import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { IMIPS } from "src/cannon/interfaces/IMIPS.sol";

import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { IProtocolVersions } from "src/L1/interfaces/IProtocolVersions.sol";
import { IOptimismPortal2 } from "src/L1/interfaces/IOptimismPortal2.sol";
import { ISystemConfig } from "src/L1/interfaces/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "src/L1/interfaces/IL1CrossDomainMessenger.sol";
import { IL1ERC721Bridge } from "src/L1/interfaces/IL1ERC721Bridge.sol";
import { IL1StandardBridge } from "src/L1/interfaces/IL1StandardBridge.sol";
import { IOptimismPortalInterop } from "src/L1/interfaces/IOptimismPortalInterop.sol";
import { ISystemConfigInterop } from "src/L1/interfaces/ISystemConfigInterop.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OPStackManagerInterop } from "src/L1/OPStackManagerInterop.sol";

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

    // The release version to set OPSM implementations for, of the format `op-contracts/vX.Y.Z`.
    string internal _release;

    // Outputs from DeploySuperchain.s.sol.
    ISuperchainConfig internal _superchainConfigProxy;
    IProtocolVersions internal _protocolVersionsProxy;

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
        else revert("DeployImplementationsInput: unknown selector");
    }

    function set(bytes4 sel, address _addr) public {
        require(_addr != address(0), "DeployImplementationsInput: cannot set zero address");
        if (sel == this.superchainConfigProxy.selector) _superchainConfigProxy = ISuperchainConfig(_addr);
        else if (sel == this.protocolVersionsProxy.selector) _protocolVersionsProxy = IProtocolVersions(_addr);
        else revert("DeployImplementationsInput: unknown selector");
    }

    function set(bytes4 sel, bytes32 _value) public {
        if (sel == this.salt.selector) _salt = _value;
        else revert("DeployImplementationsInput: unknown selector");
    }

    function loadInputFile(string memory _infile) public pure {
        _infile;
        require(false, "DeployImplementationsInput: not implemented");
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

    function superchainConfigProxy() public view returns (ISuperchainConfig) {
        require(address(_superchainConfigProxy) != address(0), "DeployImplementationsInput: not set");
        return _superchainConfigProxy;
    }

    function protocolVersionsProxy() public view returns (IProtocolVersions) {
        require(address(_protocolVersionsProxy) != address(0), "DeployImplementationsInput: not set");
        return _protocolVersionsProxy;
    }

    function superchainProxyAdmin() public returns (IProxyAdmin) {
        ISuperchainConfig proxy = this.superchainConfigProxy();
        // Can infer the superchainProxyAdmin from the superchainConfigProxy.
        vm.prank(address(0));
        IProxyAdmin proxyAdmin = IProxyAdmin(IProxy(payable(address(proxy))).admin());
        require(address(proxyAdmin) != address(0), "DeployImplementationsInput: not set");
        return proxyAdmin;
    }
}

contract DeployImplementationsOutput is BaseDeployIO {
    OPStackManager internal _opsmProxy;
    IDelayedWETH internal _delayedWETHImpl;
    IOptimismPortal2 internal _optimismPortalImpl;
    IPreimageOracle internal _preimageOracleSingleton;
    IMIPS internal _mipsSingleton;
    ISystemConfig internal _systemConfigImpl;
    IL1CrossDomainMessenger internal _l1CrossDomainMessengerImpl;
    IL1ERC721Bridge internal _l1ERC721BridgeImpl;
    IL1StandardBridge internal _l1StandardBridgeImpl;
    IOptimismMintableERC20Factory internal _optimismMintableERC20FactoryImpl;
    IDisputeGameFactory internal _disputeGameFactoryImpl;

    function set(bytes4 sel, address _addr) public {
        require(_addr != address(0), "DeployImplementationsOutput: cannot set zero address");

        // forgefmt: disable-start
        if (sel == this.opsmProxy.selector) _opsmProxy = OPStackManager(payable(_addr));
        else if (sel == this.optimismPortalImpl.selector) _optimismPortalImpl = IOptimismPortal2(payable(_addr));
        else if (sel == this.delayedWETHImpl.selector) _delayedWETHImpl = IDelayedWETH(payable(_addr));
        else if (sel == this.preimageOracleSingleton.selector) _preimageOracleSingleton = IPreimageOracle(_addr);
        else if (sel == this.mipsSingleton.selector) _mipsSingleton = IMIPS(_addr);
        else if (sel == this.systemConfigImpl.selector) _systemConfigImpl = ISystemConfig(_addr);
        else if (sel == this.l1CrossDomainMessengerImpl.selector) _l1CrossDomainMessengerImpl = IL1CrossDomainMessenger(_addr);
        else if (sel == this.l1ERC721BridgeImpl.selector) _l1ERC721BridgeImpl = IL1ERC721Bridge(_addr);
        else if (sel == this.l1StandardBridgeImpl.selector) _l1StandardBridgeImpl = IL1StandardBridge(payable(_addr));
        else if (sel == this.optimismMintableERC20FactoryImpl.selector) _optimismMintableERC20FactoryImpl = IOptimismMintableERC20Factory(_addr);
        else if (sel == this.disputeGameFactoryImpl.selector) _disputeGameFactoryImpl = IDisputeGameFactory(_addr);
        else revert("DeployImplementationsOutput: unknown selector");
        // forgefmt: disable-end
    }

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployImplementationsOutput: not implemented");
    }

    function checkOutput(DeployImplementationsInput _dii) public {
        address[] memory addrs = Solarray.addresses(
            address(this.opsmProxy()),
            address(this.optimismPortalImpl()),
            address(this.delayedWETHImpl()),
            address(this.preimageOracleSingleton()),
            address(this.mipsSingleton()),
            address(this.systemConfigImpl()),
            address(this.l1CrossDomainMessengerImpl()),
            address(this.l1ERC721BridgeImpl()),
            address(this.l1StandardBridgeImpl()),
            address(this.optimismMintableERC20FactoryImpl()),
            address(this.disputeGameFactoryImpl())
        );
        DeployUtils.assertValidContractAddresses(addrs);

        assertValidDeploy(_dii);
    }

    function opsmProxy() public returns (OPStackManager) {
        DeployUtils.assertValidContractAddress(address(_opsmProxy));
        DeployUtils.assertImplementationSet(address(_opsmProxy));
        return _opsmProxy;
    }

    function optimismPortalImpl() public view returns (IOptimismPortal2) {
        DeployUtils.assertValidContractAddress(address(_optimismPortalImpl));
        return _optimismPortalImpl;
    }

    function delayedWETHImpl() public view returns (IDelayedWETH) {
        DeployUtils.assertValidContractAddress(address(_delayedWETHImpl));
        return _delayedWETHImpl;
    }

    function preimageOracleSingleton() public view returns (IPreimageOracle) {
        DeployUtils.assertValidContractAddress(address(_preimageOracleSingleton));
        return _preimageOracleSingleton;
    }

    function mipsSingleton() public view returns (IMIPS) {
        DeployUtils.assertValidContractAddress(address(_mipsSingleton));
        return _mipsSingleton;
    }

    function systemConfigImpl() public view returns (ISystemConfig) {
        DeployUtils.assertValidContractAddress(address(_systemConfigImpl));
        return _systemConfigImpl;
    }

    function l1CrossDomainMessengerImpl() public view returns (IL1CrossDomainMessenger) {
        DeployUtils.assertValidContractAddress(address(_l1CrossDomainMessengerImpl));
        return _l1CrossDomainMessengerImpl;
    }

    function l1ERC721BridgeImpl() public view returns (IL1ERC721Bridge) {
        DeployUtils.assertValidContractAddress(address(_l1ERC721BridgeImpl));
        return _l1ERC721BridgeImpl;
    }

    function l1StandardBridgeImpl() public view returns (IL1StandardBridge) {
        DeployUtils.assertValidContractAddress(address(_l1StandardBridgeImpl));
        return _l1StandardBridgeImpl;
    }

    function optimismMintableERC20FactoryImpl() public view returns (IOptimismMintableERC20Factory) {
        DeployUtils.assertValidContractAddress(address(_optimismMintableERC20FactoryImpl));
        return _optimismMintableERC20FactoryImpl;
    }

    function disputeGameFactoryImpl() public view returns (IDisputeGameFactory) {
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
        assertValidOpsmProxy(_dii);
        assertValidOpsmImpl(_dii);
        assertValidOptimismMintableERC20FactoryImpl(_dii);
        assertValidOptimismPortalImpl(_dii);
        assertValidPreimageOracleSingleton(_dii);
        assertValidSystemConfigImpl(_dii);
    }

    function assertValidOpsmProxy(DeployImplementationsInput _dii) internal {
        // First we check the proxy as itself.
        IProxy proxy = IProxy(payable(address(opsmProxy())));
        vm.prank(address(0));
        address admin = proxy.admin();
        require(admin == address(_dii.superchainProxyAdmin()), "OPSMP-10");

        // Then we check the proxy as OPSM.
        DeployUtils.assertInitialized({ _contractAddress: address(opsmProxy()), _slot: 0, _offset: 0 });
        require(address(opsmProxy().superchainConfig()) == address(_dii.superchainConfigProxy()), "OPSMP-20");
        require(address(opsmProxy().protocolVersions()) == address(_dii.protocolVersionsProxy()), "OPSMP-30");
        require(LibString.eq(opsmProxy().latestRelease(), _dii.release()), "OPSMP-50"); // Initial release is latest.
    }

    function assertValidOpsmImpl(DeployImplementationsInput _dii) internal {
        IProxy proxy = IProxy(payable(address(opsmProxy())));
        vm.prank(address(0));
        OPStackManager impl = OPStackManager(proxy.implementation());
        DeployUtils.assertInitialized({ _contractAddress: address(impl), _slot: 0, _offset: 0 });
        require(address(impl.superchainConfig()) == address(_dii.superchainConfigProxy()), "OPSMI-10");
        require(address(impl.protocolVersions()) == address(_dii.protocolVersionsProxy()), "OPSMI-20");
    }

    function assertValidOptimismPortalImpl(DeployImplementationsInput) internal view {
        IOptimismPortal2 portal = optimismPortalImpl();

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
        IDelayedWETH delayedWETH = delayedWETHImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(delayedWETH), _slot: 0, _offset: 0 });

        require(delayedWETH.owner() == address(0), "DW-10");
        require(delayedWETH.delay() == _dii.withdrawalDelaySeconds(), "DW-20");
        require(delayedWETH.config() == ISuperchainConfig(address(0)), "DW-30");
    }

    function assertValidPreimageOracleSingleton(DeployImplementationsInput _dii) internal view {
        IPreimageOracle oracle = preimageOracleSingleton();

        require(oracle.minProposalSize() == _dii.minProposalSizeBytes(), "PO-10");
        require(oracle.challengePeriod() == _dii.challengePeriodSeconds(), "PO-20");
    }

    function assertValidMipsSingleton(DeployImplementationsInput) internal view {
        IMIPS mips = mipsSingleton();

        require(address(mips.oracle()) == address(preimageOracleSingleton()), "MIPS-10");
    }

    function assertValidSystemConfigImpl(DeployImplementationsInput) internal view {
        ISystemConfig systemConfig = systemConfigImpl();

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
        IL1CrossDomainMessenger messenger = l1CrossDomainMessengerImpl();

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
        IL1ERC721Bridge bridge = l1ERC721BridgeImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_ERC721_BRIDGE, "L721B-10");
        require(address(bridge.otherBridge()) == Predeploys.L2_ERC721_BRIDGE, "L721B-20");
        require(address(bridge.MESSENGER()) == address(0), "L721B-30");
        require(address(bridge.messenger()) == address(0), "L721B-40");
        require(address(bridge.superchainConfig()) == address(0), "L721B-50");
    }

    function assertValidL1StandardBridgeImpl(DeployImplementationsInput) internal view {
        IL1StandardBridge bridge = l1StandardBridgeImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.MESSENGER()) == address(0), "L1SB-10");
        require(address(bridge.messenger()) == address(0), "L1SB-20");
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-30");
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-40");
        require(address(bridge.superchainConfig()) == address(0), "L1SB-50");
    }

    function assertValidOptimismMintableERC20FactoryImpl(DeployImplementationsInput) internal view {
        IOptimismMintableERC20Factory factory = optimismMintableERC20FactoryImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(address(factory.BRIDGE()) == address(0), "MERC20F-10");
        require(address(factory.bridge()) == address(0), "MERC20F-20");
    }

    function assertValidDisputeGameFactoryImpl(DeployImplementationsInput) internal view {
        IDisputeGameFactory factory = disputeGameFactoryImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(address(factory.owner()) == address(0), "DG-10");
    }
}

contract DeployImplementations is Script {
    // -------- Core Deployment Methods --------

    function run(string memory _infile) public {
        (DeployImplementationsInput dii, DeployImplementationsOutput dio) = etchIOContracts();
        dii.loadInputFile(_infile);
        run(dii, dio);
        string memory outfile = ""; // This will be derived from input file name, e.g. `foo.in.toml` -> `foo.out.toml`
        dio.writeOutputFile(outfile);
        require(false, "DeployImplementations: run is not implemented");
    }

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

        // Deploy the OP Stack Manager with the new implementations set.
        deployOPStackManager(_dii, _dio);

        _dio.checkOutput(_dii);
    }

    // -------- Deployment Steps --------

    // --- OP Stack Manager ---

    function opsmSystemConfigSetter(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        internal
        view
        virtual
        returns (OPStackManager.ImplementationSetter memory)
    {
        return OPStackManager.ImplementationSetter({
            name: "SystemConfig",
            info: OPStackManager.Implementation(address(_dio.systemConfigImpl()), ISystemConfig.initialize.selector)
        });
    }

    // Deploy and initialize a proxied OPStackManager.
    function createOPSMContract(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput,
        OPStackManager.Blueprints memory blueprints,
        string memory release,
        OPStackManager.ImplementationSetter[] memory setters
    )
        internal
        virtual
        returns (OPStackManager opsmProxy_)
    {
        ISuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        IProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();
        IProxyAdmin proxyAdmin = _dii.superchainProxyAdmin();

        vm.startBroadcast(msg.sender);
        IProxy proxy = IProxy(payable(DeployUtils.deploy("Proxy", abi.encode(address(msg.sender)))));
        OPStackManager opsm = new OPStackManager(superchainConfigProxy, protocolVersionsProxy);

        OPStackManager.InitializerInputs memory initializerInputs =
            OPStackManager.InitializerInputs(blueprints, setters, release, true);
        proxy.upgradeToAndCall(address(opsm), abi.encodeWithSelector(opsm.initialize.selector, initializerInputs));

        proxy.changeAdmin(address(proxyAdmin)); // transfer ownership of Proxy contract to the ProxyAdmin contract
        vm.stopBroadcast();

        opsmProxy_ = OPStackManager(address(proxy));
    }

    function deployOPStackManager(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        string memory release = _dii.release();

        // First we deploy the blueprints for the singletons deployed by OPSM.
        // forgefmt: disable-start
        bytes32 salt = _dii.salt();
        OPStackManager.Blueprints memory blueprints;

        vm.startBroadcast(msg.sender);
        blueprints.addressManager = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("AddressManager")), salt);
        blueprints.proxy = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("Proxy")), salt);
        blueprints.proxyAdmin = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("ProxyAdmin")), salt);
        blueprints.l1ChugSplashProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("L1ChugSplashProxy")), salt);
        blueprints.resolvedDelegateProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("ResolvedDelegateProxy")), salt);
        blueprints.anchorStateRegistry = deployBytecode(Blueprint.blueprintDeployerBytecode(vm.getCode("AnchorStateRegistry")), salt);
        vm.stopBroadcast();
        // forgefmt: disable-end

        OPStackManager.ImplementationSetter[] memory setters = new OPStackManager.ImplementationSetter[](7);
        setters[0] = OPStackManager.ImplementationSetter({
            name: "L1ERC721Bridge",
            info: OPStackManager.Implementation(address(_dio.l1ERC721BridgeImpl()), IL1ERC721Bridge.initialize.selector)
        });
        setters[1] = OPStackManager.ImplementationSetter({
            name: "OptimismPortal",
            info: OPStackManager.Implementation(address(_dio.optimismPortalImpl()), IOptimismPortal2.initialize.selector)
        });
        setters[2] = opsmSystemConfigSetter(_dii, _dio);
        setters[3] = OPStackManager.ImplementationSetter({
            name: "OptimismMintableERC20Factory",
            info: OPStackManager.Implementation(
                address(_dio.optimismMintableERC20FactoryImpl()), IOptimismMintableERC20Factory.initialize.selector
            )
        });
        setters[4] = OPStackManager.ImplementationSetter({
            name: "L1CrossDomainMessenger",
            info: OPStackManager.Implementation(
                address(_dio.l1CrossDomainMessengerImpl()), IL1CrossDomainMessenger.initialize.selector
            )
        });
        setters[5] = OPStackManager.ImplementationSetter({
            name: "L1StandardBridge",
            info: OPStackManager.Implementation(address(_dio.l1StandardBridgeImpl()), IL1StandardBridge.initialize.selector)
        });

        setters[6] = OPStackManager.ImplementationSetter({
            name: "DisputeGameFactory",
            info: OPStackManager.Implementation(
                address(_dio.disputeGameFactoryImpl()), IDisputeGameFactory.initialize.selector
            )
        });

        // This call contains a broadcast to deploy OPSM which is proxied.
        OPStackManager opsmProxy = createOPSMContract(_dii, _dio, blueprints, release, setters);

        vm.label(address(opsmProxy), "OPStackManager");
        _dio.set(_dio.opsmProxy.selector, address(opsmProxy));
    }

    // --- Core Contracts ---

    function deploySystemConfigImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        ISystemConfig systemConfigImpl = ISystemConfig(DeployUtils.deploy("SystemConfig", hex""));

        vm.label(address(systemConfigImpl), "SystemConfigImpl");
        _dio.set(_dio.systemConfigImpl.selector, address(systemConfigImpl));
    }

    function deployL1CrossDomainMessengerImpl(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        vm.broadcast(msg.sender);
        IL1CrossDomainMessenger l1CrossDomainMessengerImpl =
            IL1CrossDomainMessenger(DeployUtils.deploy("L1CrossDomainMessenger", hex""));

        vm.label(address(l1CrossDomainMessengerImpl), "L1CrossDomainMessengerImpl");
        _dio.set(_dio.l1CrossDomainMessengerImpl.selector, address(l1CrossDomainMessengerImpl));
    }

    function deployL1ERC721BridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        IL1ERC721Bridge l1ERC721BridgeImpl = IL1ERC721Bridge(DeployUtils.deploy("L1ERC721Bridge", hex""));

        vm.label(address(l1ERC721BridgeImpl), "L1ERC721BridgeImpl");
        _dio.set(_dio.l1ERC721BridgeImpl.selector, address(l1ERC721BridgeImpl));
    }

    function deployL1StandardBridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        IL1StandardBridge l1StandardBridgeImpl =
            IL1StandardBridge(payable(DeployUtils.deploy("L1StandardBridge", hex"")));

        vm.label(address(l1StandardBridgeImpl), "L1StandardBridgeImpl");
        _dio.set(_dio.l1StandardBridgeImpl.selector, address(l1StandardBridgeImpl));
    }

    function deployOptimismMintableERC20FactoryImpl(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        vm.broadcast(msg.sender);
        IOptimismMintableERC20Factory optimismMintableERC20FactoryImpl =
            IOptimismMintableERC20Factory(DeployUtils.deploy("OptimismMintableERC20Factory", hex""));

        vm.label(address(optimismMintableERC20FactoryImpl), "OptimismMintableERC20FactoryImpl");
        _dio.set(_dio.optimismMintableERC20FactoryImpl.selector, address(optimismMintableERC20FactoryImpl));
    }

    // --- Fault Proofs Contracts ---

    // The fault proofs contracts are configured as follows:
    // | Contract                | Proxied | Deployment                        | MCP Ready  |
    // |-------------------------|---------|-----------------------------------|------------|
    // | DisputeGameFactory      | Yes     | Bespoke                           | Yes        |
    // | AnchorStateRegistry     | Yes     | Bespoke                           | No         |
    // | FaultDisputeGame        | No      | Bespoke                           | No         |
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
        uint256 proofMaturityDelaySeconds = _dii.proofMaturityDelaySeconds();
        uint256 disputeGameFinalityDelaySeconds = _dii.disputeGameFinalityDelaySeconds();

        vm.broadcast(msg.sender);
        IOptimismPortal2 optimismPortalImpl = IOptimismPortal2(
            payable(
                DeployUtils.deploy(
                    "OptimismPortal2", abi.encode(proofMaturityDelaySeconds, disputeGameFinalityDelaySeconds)
                )
            )
        );

        vm.label(address(optimismPortalImpl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(optimismPortalImpl));
    }

    function deployDelayedWETHImpl(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        uint256 withdrawalDelaySeconds = _dii.withdrawalDelaySeconds();

        vm.broadcast(msg.sender);
        IDelayedWETH delayedWETHImpl =
            IDelayedWETH(payable(DeployUtils.deploy("DelayedWETH", abi.encode(withdrawalDelaySeconds))));

        vm.label(address(delayedWETHImpl), "DelayedWETHImpl");
        _dio.set(_dio.delayedWETHImpl.selector, address(delayedWETHImpl));
    }

    function deployPreimageOracleSingleton(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        uint256 minProposalSizeBytes = _dii.minProposalSizeBytes();
        uint256 challengePeriodSeconds = _dii.challengePeriodSeconds();

        vm.broadcast(msg.sender);
        IPreimageOracle preimageOracleSingleton = IPreimageOracle(
            DeployUtils.deploy("PreimageOracle", abi.encode(minProposalSizeBytes, challengePeriodSeconds))
        );

        vm.label(address(preimageOracleSingleton), "PreimageOracleSingleton");
        _dio.set(_dio.preimageOracleSingleton.selector, address(preimageOracleSingleton));
    }

    function deployMipsSingleton(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        IPreimageOracle preimageOracle = IPreimageOracle(_dio.preimageOracleSingleton());

        vm.broadcast(msg.sender);
        IMIPS mipsSingleton = IMIPS(DeployUtils.deploy("MIPS", abi.encode(preimageOracle)));

        vm.label(address(mipsSingleton), "MIPSSingleton");
        _dio.set(_dio.mipsSingleton.selector, address(mipsSingleton));
    }

    function deployDisputeGameFactoryImpl(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        public
        virtual
    {
        vm.broadcast(msg.sender);
        IDisputeGameFactory disputeGameFactoryImpl =
            IDisputeGameFactory(DeployUtils.deploy("DisputeGameFactory", hex""));

        vm.label(address(disputeGameFactoryImpl), "DisputeGameFactoryImpl");
        _dio.set(_dio.disputeGameFactoryImpl.selector, address(disputeGameFactoryImpl));
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
}

// Similar to how DeploySuperchain.s.sol contains a lot of comments to thoroughly document the script
// architecture, this comment block documents how to update the deploy scripts to support new features.
//
// Using the base scripts and contracts (DeploySuperchain, DeployImplementations, DeployOPChain, and
// the corresponding OPStackManager) deploys a standard chain. For nonstandard and in-development
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
//   - An `OPStackManagerInterop is OPStackManager` that knows how to encode the calldata for the
//     new system config initializer.
//   - A `DeployImplementationsInterop is DeployImplementations` that:
//     - Deploys OptimismPortalInterop instead of OptimismPortal.
//     - Deploys SystemConfigInterop instead of SystemConfig.
//     - Deploys OPStackManagerInterop instead of OPStackManager, which contains the updated logic
//       for encoding the SystemConfig initializer.
//     - Updates the OPSM release setter logic to use the updated initializer.
//  - A `DeployOPChainInterop is DeployOPChain` that allows the updated input parameter to be passed.
//
// Most of the complexity in the above flow comes from the the new input for the updated SystemConfig
// initializer. If all function signatures were the same, all we'd have to change is the contract
// implementations that are deployed then set in the OPSM. For now, to simplify things until we
// resolve https://github.com/ethereum-optimism/optimism/issues/11783, we just assume this new role
// is the same as the proxy admin owner.
contract DeployImplementationsInterop is DeployImplementations {
    function createOPSMContract(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput,
        OPStackManager.Blueprints memory blueprints,
        string memory release,
        OPStackManager.ImplementationSetter[] memory setters
    )
        internal
        override
        returns (OPStackManager opsmProxy_)
    {
        ISuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        IProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();
        IProxyAdmin proxyAdmin = _dii.superchainProxyAdmin();

        vm.startBroadcast(msg.sender);
        IProxy proxy = IProxy(payable(DeployUtils.deploy("Proxy", abi.encode(address(msg.sender)))));
        OPStackManager opsm = new OPStackManagerInterop(superchainConfigProxy, protocolVersionsProxy);

        OPStackManager.InitializerInputs memory initializerInputs =
            OPStackManager.InitializerInputs(blueprints, setters, release, true);
        proxy.upgradeToAndCall(address(opsm), abi.encodeWithSelector(opsm.initialize.selector, initializerInputs));

        proxy.changeAdmin(address(proxyAdmin)); // transfer ownership of Proxy contract to the ProxyAdmin contract
        vm.stopBroadcast();

        opsmProxy_ = OPStackManagerInterop(address(proxy));
    }

    function deployOptimismPortalImpl(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput _dio
    )
        public
        override
    {
        uint256 proofMaturityDelaySeconds = _dii.proofMaturityDelaySeconds();
        uint256 disputeGameFinalityDelaySeconds = _dii.disputeGameFinalityDelaySeconds();

        vm.broadcast(msg.sender);
        IOptimismPortal2 optimismPortalImpl = IOptimismPortal2(
            payable(
                DeployUtils.deploy(
                    "OptimismPortal2", abi.encode(proofMaturityDelaySeconds, disputeGameFinalityDelaySeconds)
                )
            )
        );

        vm.label(address(optimismPortalImpl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(optimismPortalImpl));
    }

    function deploySystemConfigImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public override {
        vm.broadcast(msg.sender);
        ISystemConfigInterop systemConfigImpl = ISystemConfigInterop(DeployUtils.deploy("SystemConfigInterop", hex""));

        vm.label(address(systemConfigImpl), "SystemConfigImpl");
        _dio.set(_dio.systemConfigImpl.selector, address(systemConfigImpl));
    }

    function opsmSystemConfigSetter(
        DeployImplementationsInput,
        DeployImplementationsOutput _dio
    )
        internal
        view
        override
        returns (OPStackManager.ImplementationSetter memory)
    {
        return OPStackManager.ImplementationSetter({
            name: "SystemConfig",
            info: OPStackManager.Implementation(address(_dio.systemConfigImpl()), ISystemConfigInterop.initialize.selector)
        });
    }
}
