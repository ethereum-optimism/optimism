// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";
import { BaseDeployIO } from "scripts/utils/BaseDeployIO.sol";

import { IResourceMetering } from "src/L1/interfaces/IResourceMetering.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";
import { IBigStepper } from "src/dispute/interfaces/IBigStepper.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { DelayedWETH } from "src/dispute/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";
import { Claim, GameType, GameTypes, Hash, OutputRoot } from "src/dispute/lib/Types.sol";

import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

contract DeployOPChainInput is BaseDeployIO {
    address internal _opChainProxyAdminOwner;
    address internal _systemConfigOwner;
    address internal _batcher;
    address internal _unsafeBlockSigner;
    address internal _proposer;
    address internal _challenger;

    // TODO Add fault proofs inputs in a future PR.
    uint32 internal _basefeeScalar;
    uint32 internal _blobBaseFeeScalar;
    uint256 internal _l2ChainId;
    OPContractsManager internal _opcmProxy;

    function set(bytes4 _sel, address _addr) public {
        require(_addr != address(0), "DeployOPChainInput: cannot set zero address");
        if (_sel == this.opChainProxyAdminOwner.selector) _opChainProxyAdminOwner = _addr;
        else if (_sel == this.systemConfigOwner.selector) _systemConfigOwner = _addr;
        else if (_sel == this.batcher.selector) _batcher = _addr;
        else if (_sel == this.unsafeBlockSigner.selector) _unsafeBlockSigner = _addr;
        else if (_sel == this.proposer.selector) _proposer = _addr;
        else if (_sel == this.challenger.selector) _challenger = _addr;
        else if (_sel == this.opcmProxy.selector) _opcmProxy = OPContractsManager(_addr);
        else revert("DeployOPChainInput: unknown selector");
    }

    function set(bytes4 _sel, uint256 _value) public {
        if (_sel == this.basefeeScalar.selector) {
            _basefeeScalar = SafeCast.toUint32(_value);
        } else if (_sel == this.blobBaseFeeScalar.selector) {
            _blobBaseFeeScalar = SafeCast.toUint32(_value);
        } else if (_sel == this.l2ChainId.selector) {
            require(_value != 0 && _value != block.chainid, "DeployOPChainInput: invalid l2ChainId");
            _l2ChainId = _value;
        } else {
            revert("DeployOPChainInput: unknown selector");
        }
    }

    function opChainProxyAdminOwner() public view returns (address) {
        require(_opChainProxyAdminOwner != address(0), "DeployOPChainInput: not set");
        return _opChainProxyAdminOwner;
    }

    function systemConfigOwner() public view returns (address) {
        require(_systemConfigOwner != address(0), "DeployOPChainInput: not set");
        return _systemConfigOwner;
    }

    function batcher() public view returns (address) {
        require(_batcher != address(0), "DeployOPChainInput: not set");
        return _batcher;
    }

    function unsafeBlockSigner() public view returns (address) {
        require(_unsafeBlockSigner != address(0), "DeployOPChainInput: not set");
        return _unsafeBlockSigner;
    }

    function proposer() public view returns (address) {
        require(_proposer != address(0), "DeployOPChainInput: not set");
        return _proposer;
    }

    function challenger() public view returns (address) {
        require(_challenger != address(0), "DeployOPChainInput: not set");
        return _challenger;
    }

    function basefeeScalar() public view returns (uint32) {
        require(_basefeeScalar != 0, "DeployOPChainInput: not set");
        return _basefeeScalar;
    }

    function blobBaseFeeScalar() public view returns (uint32) {
        require(_blobBaseFeeScalar != 0, "DeployOPChainInput: not set");
        return _blobBaseFeeScalar;
    }

    function l2ChainId() public view returns (uint256) {
        require(_l2ChainId != 0, "DeployOPChainInput: not set");
        require(_l2ChainId != block.chainid, "DeployOPChainInput: invalid l2ChainId");
        return _l2ChainId;
    }

    function startingAnchorRoots() public pure returns (bytes memory) {
        // WARNING: For now always hardcode the starting permissioned game anchor root to 0xdead,
        // and we do not set anything for the permissioned game. This is because we currently only
        // support deploying straight to permissioned games, and the starting root does not
        // matter for that, as long as it is non-zero, since no games will be played. We do not
        // deploy the permissionless game (and therefore do not set a starting root for it here)
        // because to to update to the permissionless game, we will need to update its starting
        // anchor root and deploy a new permissioned dispute game contract anyway.
        //
        // You can `console.logBytes(abi.encode(defaultStartingAnchorRoots))` to get the bytes that
        // are hardcoded into `op-chain-ops/deployer/opcm/opchain.go`
        AnchorStateRegistry.StartingAnchorRoot[] memory defaultStartingAnchorRoots =
            new AnchorStateRegistry.StartingAnchorRoot[](1);
        defaultStartingAnchorRoots[0] = AnchorStateRegistry.StartingAnchorRoot({
            gameType: GameTypes.PERMISSIONED_CANNON,
            outputRoot: OutputRoot({ root: Hash.wrap(bytes32(hex"dead")), l2BlockNumber: 0 })
        });
        return abi.encode(defaultStartingAnchorRoots);
    }

    function opcmProxy() public returns (OPContractsManager) {
        require(address(_opcmProxy) != address(0), "DeployOPChainInput: not set");
        DeployUtils.assertValidContractAddress(address(_opcmProxy));
        DeployUtils.assertImplementationSet(address(_opcmProxy));
        return _opcmProxy;
    }
}

contract DeployOPChainOutput is BaseDeployIO {
    ProxyAdmin internal _opChainProxyAdmin;
    AddressManager internal _addressManager;
    L1ERC721Bridge internal _l1ERC721BridgeProxy;
    SystemConfig internal _systemConfigProxy;
    OptimismMintableERC20Factory internal _optimismMintableERC20FactoryProxy;
    L1StandardBridge internal _l1StandardBridgeProxy;
    L1CrossDomainMessenger internal _l1CrossDomainMessengerProxy;
    OptimismPortal2 internal _optimismPortalProxy;
    DisputeGameFactory internal _disputeGameFactoryProxy;
    AnchorStateRegistry internal _anchorStateRegistryProxy;
    AnchorStateRegistry internal _anchorStateRegistryImpl;
    FaultDisputeGame internal _faultDisputeGame;
    PermissionedDisputeGame internal _permissionedDisputeGame;
    DelayedWETH internal _delayedWETHPermissionedGameProxy;
    DelayedWETH internal _delayedWETHPermissionlessGameProxy;

    function set(bytes4 sel, address _addr) public {
        require(_addr != address(0), "DeployOPChainOutput: cannot set zero address");
        // forgefmt: disable-start
        if (sel == this.opChainProxyAdmin.selector) _opChainProxyAdmin = ProxyAdmin(_addr) ;
        else if (sel == this.addressManager.selector) _addressManager = AddressManager(_addr) ;
        else if (sel == this.l1ERC721BridgeProxy.selector) _l1ERC721BridgeProxy = L1ERC721Bridge(_addr) ;
        else if (sel == this.systemConfigProxy.selector) _systemConfigProxy = SystemConfig(_addr) ;
        else if (sel == this.optimismMintableERC20FactoryProxy.selector) _optimismMintableERC20FactoryProxy = OptimismMintableERC20Factory(_addr) ;
        else if (sel == this.l1StandardBridgeProxy.selector) _l1StandardBridgeProxy = L1StandardBridge(payable(_addr)) ;
        else if (sel == this.l1CrossDomainMessengerProxy.selector) _l1CrossDomainMessengerProxy = L1CrossDomainMessenger(_addr) ;
        else if (sel == this.optimismPortalProxy.selector) _optimismPortalProxy = OptimismPortal2(payable(_addr)) ;
        else if (sel == this.disputeGameFactoryProxy.selector) _disputeGameFactoryProxy = DisputeGameFactory(_addr) ;
        else if (sel == this.anchorStateRegistryProxy.selector) _anchorStateRegistryProxy = AnchorStateRegistry(_addr) ;
        else if (sel == this.anchorStateRegistryImpl.selector) _anchorStateRegistryImpl = AnchorStateRegistry(_addr) ;
        else if (sel == this.faultDisputeGame.selector) _faultDisputeGame = FaultDisputeGame(_addr) ;
        else if (sel == this.permissionedDisputeGame.selector) _permissionedDisputeGame = PermissionedDisputeGame(_addr) ;
        else if (sel == this.delayedWETHPermissionedGameProxy.selector) _delayedWETHPermissionedGameProxy = DelayedWETH(payable(_addr)) ;
        else if (sel == this.delayedWETHPermissionlessGameProxy.selector) _delayedWETHPermissionlessGameProxy = DelayedWETH(payable(_addr)) ;
        else revert("DeployOPChainOutput: unknown selector");
        // forgefmt: disable-end
    }

    function checkOutput(DeployOPChainInput _doi) public {
        // With 16 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
        address[] memory addrs1 = Solarray.addresses(
            address(_opChainProxyAdmin),
            address(_addressManager),
            address(_l1ERC721BridgeProxy),
            address(_systemConfigProxy),
            address(_optimismMintableERC20FactoryProxy),
            address(_l1StandardBridgeProxy),
            address(_l1CrossDomainMessengerProxy)
        );
        address[] memory addrs2 = Solarray.addresses(
            address(_optimismPortalProxy),
            address(_disputeGameFactoryProxy),
            address(_anchorStateRegistryProxy),
            address(_anchorStateRegistryImpl),
            // address(_faultDisputeGame),
            address(_permissionedDisputeGame),
            address(_delayedWETHPermissionedGameProxy),
            address(_delayedWETHPermissionlessGameProxy)
        );
        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));

        assertValidDeploy(_doi);
    }

    function opChainProxyAdmin() public view returns (ProxyAdmin) {
        DeployUtils.assertValidContractAddress(address(_opChainProxyAdmin));
        return _opChainProxyAdmin;
    }

    function addressManager() public view returns (AddressManager) {
        DeployUtils.assertValidContractAddress(address(_addressManager));
        return _addressManager;
    }

    function l1ERC721BridgeProxy() public view returns (L1ERC721Bridge) {
        DeployUtils.assertValidContractAddress(address(_l1ERC721BridgeProxy));
        return _l1ERC721BridgeProxy;
    }

    function systemConfigProxy() public view returns (SystemConfig) {
        DeployUtils.assertValidContractAddress(address(_systemConfigProxy));
        return _systemConfigProxy;
    }

    function optimismMintableERC20FactoryProxy() public view returns (OptimismMintableERC20Factory) {
        DeployUtils.assertValidContractAddress(address(_optimismMintableERC20FactoryProxy));
        return _optimismMintableERC20FactoryProxy;
    }

    function l1StandardBridgeProxy() public view returns (L1StandardBridge) {
        DeployUtils.assertValidContractAddress(address(_l1StandardBridgeProxy));
        return _l1StandardBridgeProxy;
    }

    function l1CrossDomainMessengerProxy() public view returns (L1CrossDomainMessenger) {
        DeployUtils.assertValidContractAddress(address(_l1CrossDomainMessengerProxy));
        return _l1CrossDomainMessengerProxy;
    }

    function optimismPortalProxy() public view returns (OptimismPortal2) {
        DeployUtils.assertValidContractAddress(address(_optimismPortalProxy));
        return _optimismPortalProxy;
    }

    function disputeGameFactoryProxy() public view returns (DisputeGameFactory) {
        DeployUtils.assertValidContractAddress(address(_disputeGameFactoryProxy));
        return _disputeGameFactoryProxy;
    }

    function anchorStateRegistryProxy() public view returns (AnchorStateRegistry) {
        DeployUtils.assertValidContractAddress(address(_anchorStateRegistryProxy));
        return _anchorStateRegistryProxy;
    }

    function anchorStateRegistryImpl() public view returns (AnchorStateRegistry) {
        DeployUtils.assertValidContractAddress(address(_anchorStateRegistryImpl));
        return _anchorStateRegistryImpl;
    }

    function faultDisputeGame() public view returns (FaultDisputeGame) {
        DeployUtils.assertValidContractAddress(address(_faultDisputeGame));
        return _faultDisputeGame;
    }

    function permissionedDisputeGame() public view returns (PermissionedDisputeGame) {
        DeployUtils.assertValidContractAddress(address(_permissionedDisputeGame));
        return _permissionedDisputeGame;
    }

    function delayedWETHPermissionedGameProxy() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(_delayedWETHPermissionedGameProxy));
        return _delayedWETHPermissionedGameProxy;
    }

    function delayedWETHPermissionlessGameProxy() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(_delayedWETHPermissionlessGameProxy));
        return _delayedWETHPermissionlessGameProxy;
    }

    // -------- Deployment Assertions --------

    function assertValidDeploy(DeployOPChainInput _doi) internal {
        assertValidAnchorStateRegistryImpl(_doi);
        assertValidAnchorStateRegistryProxy(_doi);
        assertValidDelayedWETHs(_doi);
        assertValidDisputeGameFactory(_doi);
        assertValidL1CrossDomainMessenger(_doi);
        assertValidL1ERC721Bridge(_doi);
        assertValidL1StandardBridge(_doi);
        assertValidOptimismMintableERC20Factory(_doi);
        assertValidOptimismPortal(_doi);
        assertValidPermissionedDisputeGame(_doi);
        assertValidSystemConfig(_doi);
    }

    function assertValidPermissionedDisputeGame(DeployOPChainInput _doi) internal {
        PermissionedDisputeGame game = permissionedDisputeGame();

        require(GameType.unwrap(game.gameType()) == GameType.unwrap(GameTypes.PERMISSIONED_CANNON), "DPG-10");
        require(Claim.unwrap(game.absolutePrestate()) == bytes32(hex"dead"), "DPG-20");

        OPContractsManager opcm = _doi.opcmProxy();
        (address mips,) = opcm.implementations(opcm.latestRelease(), "MIPS");
        require(game.vm() == IBigStepper(mips), "DPG-30");

        require(address(game.weth()) == address(delayedWETHPermissionedGameProxy()), "DPG-40");
        require(address(game.anchorStateRegistry()) == address(anchorStateRegistryProxy()), "DPG-50");
        require(game.l2ChainId() == _doi.l2ChainId(), "DPG-60");
    }

    function assertValidAnchorStateRegistryProxy(DeployOPChainInput) internal {
        // First we check the proxy as itself.
        Proxy proxy = Proxy(payable(address(anchorStateRegistryProxy())));
        vm.prank(address(0));
        address admin = proxy.admin();
        require(admin == address(opChainProxyAdmin()), "ANCHORP-10");

        // Then we check the proxy as ASR.
        DeployUtils.assertInitialized({ _contractAddress: address(anchorStateRegistryProxy()), _slot: 0, _offset: 0 });

        vm.prank(address(0));
        address impl = proxy.implementation();
        require(impl == address(anchorStateRegistryImpl()), "ANCHORP-20");
        require(
            address(anchorStateRegistryProxy().disputeGameFactory()) == address(disputeGameFactoryProxy()), "ANCHORP-30"
        );
    }

    function assertValidAnchorStateRegistryImpl(DeployOPChainInput) internal view {
        AnchorStateRegistry registry = anchorStateRegistryImpl();

        DeployUtils.assertInitialized({ _contractAddress: address(registry), _slot: 0, _offset: 0 });

        require(address(registry.disputeGameFactory()) == address(disputeGameFactoryProxy()), "ANCHORI-10");
    }

    function assertValidSystemConfig(DeployOPChainInput _doi) internal {
        SystemConfig systemConfig = systemConfigProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(systemConfig), _slot: 0, _offset: 0 });

        require(systemConfig.owner() == _doi.systemConfigOwner(), "SYSCON-10");
        require(systemConfig.basefeeScalar() == _doi.basefeeScalar(), "SYSCON-20");
        require(systemConfig.blobbasefeeScalar() == _doi.blobBaseFeeScalar(), "SYSCON-30");
        require(systemConfig.batcherHash() == bytes32(uint256(uint160(_doi.batcher()))), "SYSCON-40");
        require(systemConfig.gasLimit() == uint64(30000000), "SYSCON-50"); // TODO allow other gas limits?
        require(systemConfig.unsafeBlockSigner() == _doi.unsafeBlockSigner(), "SYSCON-60");
        require(systemConfig.scalar() >> 248 == 1, "SYSCON-70");

        IResourceMetering.ResourceConfig memory rConfig = Constants.DEFAULT_RESOURCE_CONFIG();
        IResourceMetering.ResourceConfig memory outputConfig = systemConfig.resourceConfig();
        require(outputConfig.maxResourceLimit == rConfig.maxResourceLimit, "SYSCON-80");
        require(outputConfig.elasticityMultiplier == rConfig.elasticityMultiplier, "SYSCON-90");
        require(outputConfig.baseFeeMaxChangeDenominator == rConfig.baseFeeMaxChangeDenominator, "SYSCON-100");
        require(outputConfig.systemTxMaxGas == rConfig.systemTxMaxGas, "SYSCON-110");
        require(outputConfig.minimumBaseFee == rConfig.minimumBaseFee, "SYSCON-120");
        require(outputConfig.maximumBaseFee == rConfig.maximumBaseFee, "SYSCON-130");

        require(systemConfig.startBlock() == block.number, "SYSCON-140");
        require(
            systemConfig.batchInbox() == _doi.opcmProxy().chainIdToBatchInboxAddress(_doi.l2ChainId()), "SYSCON-150"
        );

        require(systemConfig.l1CrossDomainMessenger() == address(l1CrossDomainMessengerProxy()), "SYSCON-160");
        require(systemConfig.l1ERC721Bridge() == address(l1ERC721BridgeProxy()), "SYSCON-170");
        require(systemConfig.l1StandardBridge() == address(l1StandardBridgeProxy()), "SYSCON-180");
        require(systemConfig.disputeGameFactory() == address(disputeGameFactoryProxy()), "SYSCON-190");
        require(systemConfig.optimismPortal() == address(optimismPortalProxy()), "SYSCON-200");
        require(
            systemConfig.optimismMintableERC20Factory() == address(optimismMintableERC20FactoryProxy()), "SYSCON-210"
        );
        (address gasPayingToken,) = systemConfig.gasPayingToken();
        require(gasPayingToken == Constants.ETHER, "SYSCON-220");
    }

    function assertValidL1CrossDomainMessenger(DeployOPChainInput _doi) internal {
        L1CrossDomainMessenger messenger = l1CrossDomainMessengerProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(messenger), _slot: 0, _offset: 20 });

        require(address(messenger.OTHER_MESSENGER()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-10");
        require(address(messenger.otherMessenger()) == Predeploys.L2_CROSS_DOMAIN_MESSENGER, "L1xDM-20");

        require(address(messenger.PORTAL()) == address(optimismPortalProxy()), "L1xDM-30");
        require(address(messenger.portal()) == address(optimismPortalProxy()), "L1xDM-40");
        require(address(messenger.superchainConfig()) == address(_doi.opcmProxy().superchainConfig()), "L1xDM-50");

        bytes32 xdmSenderSlot = vm.load(address(messenger), bytes32(uint256(204)));
        require(address(uint160(uint256(xdmSenderSlot))) == Constants.DEFAULT_L2_SENDER, "L1xDM-60");
    }

    function assertValidL1StandardBridge(DeployOPChainInput _doi) internal {
        L1StandardBridge bridge = l1StandardBridgeProxy();
        L1CrossDomainMessenger messenger = l1CrossDomainMessengerProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.MESSENGER()) == address(messenger), "L1SB-10");
        require(address(bridge.messenger()) == address(messenger), "L1SB-20");
        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-30");
        require(address(bridge.otherBridge()) == Predeploys.L2_STANDARD_BRIDGE, "L1SB-40");
        require(address(bridge.superchainConfig()) == address(_doi.opcmProxy().superchainConfig()), "L1SB-50");
    }

    function assertValidOptimismMintableERC20Factory(DeployOPChainInput) internal view {
        OptimismMintableERC20Factory factory = optimismMintableERC20FactoryProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(factory.BRIDGE() == address(l1StandardBridgeProxy()), "MERC20F-10");
        require(factory.bridge() == address(l1StandardBridgeProxy()), "MERC20F-20");
    }

    function assertValidL1ERC721Bridge(DeployOPChainInput _doi) internal {
        L1ERC721Bridge bridge = l1ERC721BridgeProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(bridge), _slot: 0, _offset: 0 });

        require(address(bridge.OTHER_BRIDGE()) == Predeploys.L2_ERC721_BRIDGE, "L721B-10");
        require(address(bridge.otherBridge()) == Predeploys.L2_ERC721_BRIDGE, "L721B-20");

        require(address(bridge.MESSENGER()) == address(l1CrossDomainMessengerProxy()), "L721B-30");
        require(address(bridge.messenger()) == address(l1CrossDomainMessengerProxy()), "L721B-40");
        require(address(bridge.superchainConfig()) == address(_doi.opcmProxy().superchainConfig()), "L721B-50");
    }

    function assertValidOptimismPortal(DeployOPChainInput _doi) internal {
        OptimismPortal2 portal = optimismPortalProxy();
        ISuperchainConfig superchainConfig = ISuperchainConfig(address(_doi.opcmProxy().superchainConfig()));

        require(address(portal.disputeGameFactory()) == address(disputeGameFactoryProxy()), "PORTAL-10");
        require(address(portal.systemConfig()) == address(systemConfigProxy()), "PORTAL-20");
        require(address(portal.superchainConfig()) == address(superchainConfig), "PORTAL-30");
        require(portal.guardian() == superchainConfig.guardian(), "PORTAL-40");
        require(portal.paused() == superchainConfig.paused(), "PORTAL-50");
        require(portal.l2Sender() == Constants.DEFAULT_L2_SENDER, "PORTAL-60");

        // This slot is the custom gas token _balance and this check ensures
        // that it stays unset for forwards compatibility with custom gas token.
        require(vm.load(address(portal), bytes32(uint256(61))) == bytes32(0));
    }

    function assertValidDisputeGameFactory(DeployOPChainInput) internal view {
        DisputeGameFactory factory = disputeGameFactoryProxy();

        DeployUtils.assertInitialized({ _contractAddress: address(factory), _slot: 0, _offset: 0 });

        require(
            address(factory.gameImpls(GameTypes.PERMISSIONED_CANNON)) == address(permissionedDisputeGame()), "DF-10"
        );
        require(factory.owner() == address(opChainProxyAdmin()), "DF-20");
    }

    function assertValidDelayedWETHs(DeployOPChainInput) internal view {
        // TODO add in once FP support is added.
    }
}

contract DeployOPChain is Script {
    // -------- Core Deployment Methods --------

    function run(DeployOPChainInput _doi, DeployOPChainOutput _doo) public {
        OPContractsManager opcmProxy = _doi.opcmProxy();

        OPContractsManager.Roles memory roles = OPContractsManager.Roles({
            opChainProxyAdminOwner: _doi.opChainProxyAdminOwner(),
            systemConfigOwner: _doi.systemConfigOwner(),
            batcher: _doi.batcher(),
            unsafeBlockSigner: _doi.unsafeBlockSigner(),
            proposer: _doi.proposer(),
            challenger: _doi.challenger()
        });
        OPContractsManager.DeployInput memory deployInput = OPContractsManager.DeployInput({
            roles: roles,
            basefeeScalar: _doi.basefeeScalar(),
            blobBasefeeScalar: _doi.blobBaseFeeScalar(),
            l2ChainId: _doi.l2ChainId(),
            startingAnchorRoots: _doi.startingAnchorRoots()
        });

        vm.broadcast(msg.sender);
        OPContractsManager.DeployOutput memory deployOutput = opcmProxy.deploy(deployInput);

        vm.label(address(deployOutput.opChainProxyAdmin), "opChainProxyAdmin");
        vm.label(address(deployOutput.addressManager), "addressManager");
        vm.label(address(deployOutput.l1ERC721BridgeProxy), "l1ERC721BridgeProxy");
        vm.label(address(deployOutput.systemConfigProxy), "systemConfigProxy");
        vm.label(address(deployOutput.optimismMintableERC20FactoryProxy), "optimismMintableERC20FactoryProxy");
        vm.label(address(deployOutput.l1StandardBridgeProxy), "l1StandardBridgeProxy");
        vm.label(address(deployOutput.l1CrossDomainMessengerProxy), "l1CrossDomainMessengerProxy");
        vm.label(address(deployOutput.optimismPortalProxy), "optimismPortalProxy");
        vm.label(address(deployOutput.disputeGameFactoryProxy), "disputeGameFactoryProxy");
        vm.label(address(deployOutput.anchorStateRegistryProxy), "anchorStateRegistryProxy");
        vm.label(address(deployOutput.anchorStateRegistryImpl), "anchorStateRegistryImpl");
        // vm.label(address(deployOutput.faultDisputeGame), "faultDisputeGame");
        vm.label(address(deployOutput.permissionedDisputeGame), "permissionedDisputeGame");
        vm.label(address(deployOutput.delayedWETHPermissionedGameProxy), "delayedWETHPermissionedGameProxy");
        vm.label(address(deployOutput.delayedWETHPermissionlessGameProxy), "delayedWETHPermissionlessGameProxy");

        _doo.set(_doo.opChainProxyAdmin.selector, address(deployOutput.opChainProxyAdmin));
        _doo.set(_doo.addressManager.selector, address(deployOutput.addressManager));
        _doo.set(_doo.l1ERC721BridgeProxy.selector, address(deployOutput.l1ERC721BridgeProxy));
        _doo.set(_doo.systemConfigProxy.selector, address(deployOutput.systemConfigProxy));
        _doo.set(
            _doo.optimismMintableERC20FactoryProxy.selector, address(deployOutput.optimismMintableERC20FactoryProxy)
        );
        _doo.set(_doo.l1StandardBridgeProxy.selector, address(deployOutput.l1StandardBridgeProxy));
        _doo.set(_doo.l1CrossDomainMessengerProxy.selector, address(deployOutput.l1CrossDomainMessengerProxy));
        _doo.set(_doo.optimismPortalProxy.selector, address(deployOutput.optimismPortalProxy));
        _doo.set(_doo.disputeGameFactoryProxy.selector, address(deployOutput.disputeGameFactoryProxy));
        _doo.set(_doo.anchorStateRegistryProxy.selector, address(deployOutput.anchorStateRegistryProxy));
        _doo.set(_doo.anchorStateRegistryImpl.selector, address(deployOutput.anchorStateRegistryImpl));
        // _doo.set(_doo.faultDisputeGame.selector, address(deployOutput.faultDisputeGame));
        _doo.set(_doo.permissionedDisputeGame.selector, address(deployOutput.permissionedDisputeGame));
        _doo.set(_doo.delayedWETHPermissionedGameProxy.selector, address(deployOutput.delayedWETHPermissionedGameProxy));
        _doo.set(
            _doo.delayedWETHPermissionlessGameProxy.selector, address(deployOutput.delayedWETHPermissionlessGameProxy)
        );

        _doo.checkOutput(_doi);
    }

    // -------- Utilities --------

    function etchIOContracts() public returns (DeployOPChainInput doi_, DeployOPChainOutput doo_) {
        (doi_, doo_) = getIOContracts();
        vm.etch(address(doi_), type(DeployOPChainInput).runtimeCode);
        vm.etch(address(doo_), type(DeployOPChainOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployOPChainInput doi_, DeployOPChainOutput doo_) {
        doi_ = DeployOPChainInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainInput"));
        doo_ = DeployOPChainOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainOutput"));
    }
}
