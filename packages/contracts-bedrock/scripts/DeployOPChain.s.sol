// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { SafeCast } from "@openzeppelin/contracts/utils/math/SafeCast.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";

import { AddressManager } from "src/legacy/AddressManager.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { FaultDisputeGame } from "src/dispute/FaultDisputeGame.sol";
import { PermissionedDisputeGame } from "src/dispute/PermissionedDisputeGame.sol";

import { OPStackManager } from "src/L1/OPStackManager.sol";
import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

contract DeployOPChainInput {
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
    OPStackManager internal _opsm;

    function set(bytes4 _sel, address _addr) public {
        require(_addr != address(0), "DeployOPChainInput: cannot set zero address");
        if (_sel == this.opChainProxyAdminOwner.selector) _opChainProxyAdminOwner = _addr;
        else if (_sel == this.systemConfigOwner.selector) _systemConfigOwner = _addr;
        else if (_sel == this.batcher.selector) _batcher = _addr;
        else if (_sel == this.unsafeBlockSigner.selector) _unsafeBlockSigner = _addr;
        else if (_sel == this.proposer.selector) _proposer = _addr;
        else if (_sel == this.challenger.selector) _challenger = _addr;
        else if (_sel == this.opsm.selector) _opsm = OPStackManager(_addr);
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

    function loadInputFile(string memory _infile) public pure {
        _infile;
        require(false, "DeployOPChainInput: not implemented");
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

    function opsm() public view returns (OPStackManager) {
        require(address(_opsm) != address(0), "DeployOPChainInput: not set");
        return _opsm;
    }
}

contract DeployOPChainOutput {
    ProxyAdmin internal _opChainProxyAdmin;
    AddressManager internal _addressManager;
    L1ERC721Bridge internal _l1ERC721BridgeProxy;
    SystemConfig internal _systemConfigProxy;
    OptimismMintableERC20Factory internal _optimismMintableERC20FactoryProxy;
    L1StandardBridge internal _l1StandardBridgeProxy;
    L1CrossDomainMessenger internal _l1CrossDomainMessengerProxy;
    OptimismPortal2 internal _optimismPortalProxy;
    DisputeGameFactory internal _disputeGameFactoryProxy;
    DisputeGameFactory internal _disputeGameFactoryImpl;
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
        else if (sel == this.disputeGameFactoryImpl.selector) _disputeGameFactoryImpl = DisputeGameFactory(_addr) ;
        else if (sel == this.anchorStateRegistryProxy.selector) _anchorStateRegistryProxy = AnchorStateRegistry(_addr) ;
        else if (sel == this.anchorStateRegistryImpl.selector) _anchorStateRegistryImpl = AnchorStateRegistry(_addr) ;
        else if (sel == this.faultDisputeGame.selector) _faultDisputeGame = FaultDisputeGame(_addr) ;
        else if (sel == this.permissionedDisputeGame.selector) _permissionedDisputeGame = PermissionedDisputeGame(_addr) ;
        else if (sel == this.delayedWETHPermissionedGameProxy.selector) _delayedWETHPermissionedGameProxy = DelayedWETH(payable(_addr)) ;
        else if (sel == this.delayedWETHPermissionlessGameProxy.selector) _delayedWETHPermissionlessGameProxy = DelayedWETH(payable(_addr)) ;
        else revert("DeployOPChainOutput: unknown selector");
        // forgefmt: disable-end
    }

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployOPChainOutput: not implemented");
    }

    function checkOutput() public view {
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
            address(_disputeGameFactoryImpl),
            address(_anchorStateRegistryProxy),
            address(_anchorStateRegistryImpl),
            address(_faultDisputeGame),
            address(_permissionedDisputeGame),
            address(_delayedWETHPermissionedGameProxy),
            address(_delayedWETHPermissionlessGameProxy)
        );
        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));
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

    function disputeGameFactoryImpl() public view returns (DisputeGameFactory) {
        DeployUtils.assertValidContractAddress(address(_disputeGameFactoryImpl));
        return _disputeGameFactoryImpl;
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
}

contract DeployOPChain is Script {
    // -------- Core Deployment Methods --------
    function run(string memory _infile) public {
        (DeployOPChainInput doi, DeployOPChainOutput doo) = etchIOContracts();
        doi.loadInputFile(_infile);
        run(doi, doo);
        string memory outfile = ""; // This will be derived from input file name, e.g. `foo.in.toml` -> `foo.out.toml`
        doo.writeOutputFile(outfile);
        require(false, "DeployOPChain: run is not implemented");
    }

    function run(DeployOPChainInput _doi, DeployOPChainOutput _doo) public {
        OPStackManager opsm = _doi.opsm();

        OPStackManager.Roles memory roles = OPStackManager.Roles({
            opChainProxyAdminOwner: _doi.opChainProxyAdminOwner(),
            systemConfigOwner: _doi.systemConfigOwner(),
            batcher: _doi.batcher(),
            unsafeBlockSigner: _doi.unsafeBlockSigner(),
            proposer: _doi.proposer(),
            challenger: _doi.challenger()
        });
        OPStackManager.DeployInput memory deployInput = OPStackManager.DeployInput({
            roles: roles,
            basefeeScalar: _doi.basefeeScalar(),
            blobBasefeeScalar: _doi.blobBaseFeeScalar(),
            l2ChainId: _doi.l2ChainId()
        });

        vm.broadcast(msg.sender);
        OPStackManager.DeployOutput memory deployOutput = opsm.deploy(deployInput);

        vm.label(address(deployOutput.opChainProxyAdmin), "opChainProxyAdmin");
        vm.label(address(deployOutput.addressManager), "addressManager");
        vm.label(address(deployOutput.l1ERC721BridgeProxy), "l1ERC721BridgeProxy");
        vm.label(address(deployOutput.systemConfigProxy), "systemConfigProxy");
        vm.label(address(deployOutput.optimismMintableERC20FactoryProxy), "optimismMintableERC20FactoryProxy");
        vm.label(address(deployOutput.l1StandardBridgeProxy), "l1StandardBridgeProxy");
        vm.label(address(deployOutput.l1CrossDomainMessengerProxy), "l1CrossDomainMessengerProxy");
        vm.label(address(deployOutput.optimismPortalProxy), "optimismPortalProxy");
        vm.label(address(deployOutput.disputeGameFactoryProxy), "disputeGameFactoryProxy");
        vm.label(address(deployOutput.disputeGameFactoryImpl), "disputeGameFactoryImpl");
        vm.label(address(deployOutput.anchorStateRegistryProxy), "anchorStateRegistryProxy");
        vm.label(address(deployOutput.anchorStateRegistryImpl), "anchorStateRegistryImpl");
        vm.label(address(deployOutput.faultDisputeGame), "faultDisputeGame");
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
        _doo.set(_doo.disputeGameFactoryImpl.selector, address(deployOutput.disputeGameFactoryImpl));
        _doo.set(_doo.anchorStateRegistryProxy.selector, address(deployOutput.anchorStateRegistryProxy));
        _doo.set(_doo.anchorStateRegistryImpl.selector, address(deployOutput.anchorStateRegistryImpl));
        _doo.set(_doo.faultDisputeGame.selector, address(deployOutput.faultDisputeGame));
        _doo.set(_doo.permissionedDisputeGame.selector, address(deployOutput.permissionedDisputeGame));
        _doo.set(_doo.delayedWETHPermissionedGameProxy.selector, address(deployOutput.delayedWETHPermissionedGameProxy));
        _doo.set(
            _doo.delayedWETHPermissionlessGameProxy.selector, address(deployOutput.delayedWETHPermissionlessGameProxy)
        );

        _doo.checkOutput();
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
