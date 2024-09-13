// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { LibString } from "@solady/utils/LibString.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { L1ChugSplashProxy } from "src/legacy/L1ChugSplashProxy.sol";
import { ResolvedDelegateProxy } from "src/legacy/ResolvedDelegateProxy.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";

import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";

import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

import { OPStackManagerInterop } from "src/L1/OPStackManagerInterop.sol";
import { OptimismPortalInterop } from "src/L1/OptimismPortalInterop.sol";
import { SystemConfigInterop } from "src/L1/SystemConfigInterop.sol";

import { Blueprint } from "src/libraries/Blueprint.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// See DeploySuperchain.s.sol for detailed comments on the script architecture used here.
contract DeployImplementationsInput {
    uint256 internal _withdrawalDelaySeconds;
    uint256 internal _minProposalSizeBytes;
    uint256 internal _challengePeriodSeconds;
    uint256 internal _proofMaturityDelaySeconds;
    uint256 internal _disputeGameFinalityDelaySeconds;

    // The release version to set OPSM implementations for, of the format `op-contracts/vX.Y.Z`.
    string internal _release;

    // Outputs from DeploySuperchain.s.sol.
    SuperchainConfig internal _superchainConfigProxy;
    ProtocolVersions internal _protocolVersionsProxy;

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
        if (sel == this.superchainConfigProxy.selector) _superchainConfigProxy = SuperchainConfig(_addr);
        else if (sel == this.protocolVersionsProxy.selector) _protocolVersionsProxy = ProtocolVersions(_addr);
        else revert("DeployImplementationsInput: unknown selector");
    }

    function loadInputFile(string memory _infile) public pure {
        _infile;
        require(false, "DeployImplementationsInput: not implemented");
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

    function superchainConfigProxy() public view returns (SuperchainConfig) {
        require(address(_superchainConfigProxy) != address(0), "DeployImplementationsInput: not set");
        return _superchainConfigProxy;
    }

    function protocolVersionsProxy() public view returns (ProtocolVersions) {
        require(address(_protocolVersionsProxy) != address(0), "DeployImplementationsInput: not set");
        return _protocolVersionsProxy;
    }
}

contract DeployImplementationsOutput {
    OPStackManager internal _opsm;
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
        if (sel == this.opsm.selector) _opsm = OPStackManager(payable(_addr));
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

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployImplementationsOutput: not implemented");
    }

    function checkOutput() public view {
        address[] memory addrs = Solarray.addresses(
            address(this.opsm()),
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
    }

    function opsm() public view returns (OPStackManager) {
        DeployUtils.assertValidContractAddress(address(_opsm));
        return _opsm;
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

        _dio.checkOutput();
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
            info: OPStackManager.Implementation(address(_dio.systemConfigImpl()), SystemConfig.initialize.selector)
        });
    }

    function createOPSMContract(
        DeployImplementationsInput _dii,
        DeployImplementationsOutput,
        OPStackManager.Blueprints memory blueprints
    )
        internal
        virtual
        returns (OPStackManager opsm_)
    {
        SuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        ProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();

        vm.broadcast(msg.sender);
        opsm_ = new OPStackManager({
            _superchainConfig: superchainConfigProxy,
            _protocolVersions: protocolVersionsProxy,
            _blueprints: blueprints
        });
    }

    function deployOPStackManager(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        string memory release = _dii.release();

        // First we deploy the blueprints for the singletons deployed by OPSM.
        // forgefmt: disable-start
        bytes32 salt = bytes32(0);
        OPStackManager.Blueprints memory blueprints;

        vm.startBroadcast(msg.sender);
        blueprints.addressManager = deployBytecode(Blueprint.blueprintDeployerBytecode(type(AddressManager).creationCode), salt);
        blueprints.proxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(Proxy).creationCode), salt);
        blueprints.proxyAdmin = deployBytecode(Blueprint.blueprintDeployerBytecode(type(ProxyAdmin).creationCode), salt);
        blueprints.l1ChugSplashProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(L1ChugSplashProxy).creationCode), salt);
        blueprints.resolvedDelegateProxy = deployBytecode(Blueprint.blueprintDeployerBytecode(type(ResolvedDelegateProxy).creationCode), salt);
        vm.stopBroadcast();
        // forgefmt: disable-end

        // This call contains a broadcast to deploy OPSM.
        OPStackManager opsm = createOPSMContract(_dii, _dio, blueprints);

        OPStackManager.ImplementationSetter[] memory setters = new OPStackManager.ImplementationSetter[](6);
        setters[0] = OPStackManager.ImplementationSetter({
            name: "L1ERC721Bridge",
            info: OPStackManager.Implementation(address(_dio.l1ERC721BridgeImpl()), L1ERC721Bridge.initialize.selector)
        });
        setters[1] = OPStackManager.ImplementationSetter({
            name: "OptimismPortal",
            info: OPStackManager.Implementation(address(_dio.optimismPortalImpl()), OptimismPortal2.initialize.selector)
        });
        setters[2] = opsmSystemConfigSetter(_dii, _dio);
        setters[3] = OPStackManager.ImplementationSetter({
            name: "OptimismMintableERC20Factory",
            info: OPStackManager.Implementation(
                address(_dio.optimismMintableERC20FactoryImpl()), OptimismMintableERC20Factory.initialize.selector
            )
        });
        setters[4] = OPStackManager.ImplementationSetter({
            name: "L1CrossDomainMessenger",
            info: OPStackManager.Implementation(
                address(_dio.l1CrossDomainMessengerImpl()), L1CrossDomainMessenger.initialize.selector
            )
        });
        setters[5] = OPStackManager.ImplementationSetter({
            name: "L1StandardBridge",
            info: OPStackManager.Implementation(address(_dio.l1StandardBridgeImpl()), L1StandardBridge.initialize.selector)
        });

        vm.broadcast(msg.sender);
        opsm.setRelease({ _release: release, _isLatest: true, _setters: setters });

        vm.label(address(opsm), "OPStackManager");
        _dio.set(_dio.opsm.selector, address(opsm));
    }

    // --- Core Contracts ---

    function deploySystemConfigImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        SystemConfig systemConfigImpl = new SystemConfig();

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
        L1CrossDomainMessenger l1CrossDomainMessengerImpl = new L1CrossDomainMessenger();

        vm.label(address(l1CrossDomainMessengerImpl), "L1CrossDomainMessengerImpl");
        _dio.set(_dio.l1CrossDomainMessengerImpl.selector, address(l1CrossDomainMessengerImpl));
    }

    function deployL1ERC721BridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        L1ERC721Bridge l1ERC721BridgeImpl = new L1ERC721Bridge();

        vm.label(address(l1ERC721BridgeImpl), "L1ERC721BridgeImpl");
        _dio.set(_dio.l1ERC721BridgeImpl.selector, address(l1ERC721BridgeImpl));
    }

    function deployL1StandardBridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        vm.broadcast(msg.sender);
        L1StandardBridge l1StandardBridgeImpl = new L1StandardBridge();

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
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl = new OptimismMintableERC20Factory();

        vm.label(address(optimismMintableERC20FactoryImpl), "OptimismMintableERC20FactoryImpl");
        _dio.set(_dio.optimismMintableERC20FactoryImpl.selector, address(optimismMintableERC20FactoryImpl));
    }

    // --- Fault Proofs Contracts ---

    // The fault proofs contracts are configured as follows:
    //   - DisputeGameFactory: Proxied, bespoke per chain.
    //   - AnchorStateRegistry: Proxied, bespoke per chain.
    //   - FaultDisputeGame: Not proxied, bespoke per chain.
    //   - PermissionedDisputeGame: Not proxied, bespoke per chain.
    //   - DelayedWETH: Proxied, and two bespoke ones per chain (one for each DisputeGame).
    //   - PreimageOracle: Not proxied, shared by all standard chains.
    //   - MIPS: Not proxied, shared by all standard chains.
    //   - OptimismPortal2: Proxied, shared by all standard chains.
    //
    // This script only deploys the shared contracts. The bespoke contracts are deployed by
    // `DeployOPChain.s.sol`. When the shared contracts are proxied, the contracts deployed here are
    // "implementations", and when shared contracts are not proxied, they are "singletons". So
    // here we deploy:
    //
    //   - OptimismPortal2 (implementation)
    //   - DelayedWETH (implementation)
    //   - PreimageOracle (singleton)
    //   - MIPS (singleton)

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
        OptimismPortal2 optimismPortalImpl = new OptimismPortal2({
            _proofMaturityDelaySeconds: proofMaturityDelaySeconds,
            _disputeGameFinalityDelaySeconds: disputeGameFinalityDelaySeconds
        });

        vm.label(address(optimismPortalImpl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(optimismPortalImpl));
    }

    function deployDelayedWETHImpl(DeployImplementationsInput _dii, DeployImplementationsOutput _dio) public virtual {
        uint256 withdrawalDelaySeconds = _dii.withdrawalDelaySeconds();

        vm.broadcast(msg.sender);
        DelayedWETH delayedWETHImpl = new DelayedWETH({ _delay: withdrawalDelaySeconds });

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
        PreimageOracle preimageOracleSingleton =
            new PreimageOracle({ _minProposalSize: minProposalSizeBytes, _challengePeriod: challengePeriodSeconds });

        vm.label(address(preimageOracleSingleton), "PreimageOracleSingleton");
        _dio.set(_dio.preimageOracleSingleton.selector, address(preimageOracleSingleton));
    }

    function deployMipsSingleton(DeployImplementationsInput, DeployImplementationsOutput _dio) public virtual {
        IPreimageOracle preimageOracle = IPreimageOracle(_dio.preimageOracleSingleton());

        vm.broadcast(msg.sender);
        MIPS mipsSingleton = new MIPS(preimageOracle);

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
        DisputeGameFactory disputeGameFactoryImpl = new DisputeGameFactory();

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
        OPStackManager.Blueprints memory blueprints
    )
        internal
        override
        returns (OPStackManager opsm_)
    {
        SuperchainConfig superchainConfigProxy = _dii.superchainConfigProxy();
        ProtocolVersions protocolVersionsProxy = _dii.protocolVersionsProxy();

        vm.broadcast(msg.sender);
        opsm_ = new OPStackManagerInterop({
            _superchainConfig: superchainConfigProxy,
            _protocolVersions: protocolVersionsProxy,
            _blueprints: blueprints
        });
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
        OptimismPortalInterop optimismPortalImpl = new OptimismPortalInterop({
            _proofMaturityDelaySeconds: proofMaturityDelaySeconds,
            _disputeGameFinalityDelaySeconds: disputeGameFinalityDelaySeconds
        });

        vm.label(address(optimismPortalImpl), "OptimismPortalImpl");
        _dio.set(_dio.optimismPortalImpl.selector, address(optimismPortalImpl));
    }

    function deploySystemConfigImpl(DeployImplementationsInput, DeployImplementationsOutput _dio) public override {
        vm.broadcast(msg.sender);
        SystemConfigInterop systemConfigImpl = new SystemConfigInterop();

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
            info: OPStackManager.Implementation(address(_dio.systemConfigImpl()), SystemConfigInterop.initialize.selector)
        });
    }
}
