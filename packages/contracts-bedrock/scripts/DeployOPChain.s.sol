// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

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
    struct Roles {
        address opChainProxyAdminOwner;
        address systemConfigOwner;
        address batcher;
        address unsafeBlockSigner;
        address proposer;
        address challenger;
    }

    // TODO Add fault proofs inputs in a future PR.
    struct Input {
        Roles roles;
        uint32 basefeeScalar;
        uint32 blobBaseFeeScalar;
        uint256 l2ChainId;
        OPStackManager opsm;
    }

    bool public inputSet = false;
    Input internal inputs;

    function loadInputFile(string memory _infile) public {
        _infile;
        Input memory parsedInput;
        loadInput(parsedInput);
        require(false, "DeployOPChainInput: not implemented");
    }

    function loadInput(Input memory _input) public {
        require(!inputSet, "DeployOPChainInput: input already set");

        require(_input.roles.opChainProxyAdminOwner != address(0), "DeployOPChainInput: null opChainProxyAdminOwner");
        require(_input.roles.systemConfigOwner != address(0), "DeployOPChainInput: null systemConfigOwner");
        require(_input.roles.batcher != address(0), "DeployOPChainInput: null batcher");
        require(_input.roles.unsafeBlockSigner != address(0), "DeployOPChainInput: null unsafeBlockSigner");
        require(_input.roles.proposer != address(0), "DeployOPChainInput: null proposer");
        require(_input.roles.challenger != address(0), "DeployOPChainInput: null challenger");
        require(_input.l2ChainId != 0 && _input.l2ChainId != block.chainid, "DeployOPChainInput: invalid l2ChainId");
        require(address(_input.opsm) != address(0), "DeployOPChainInput: null opsm");

        inputSet = true;
        inputs = _input;
    }

    function assertInputSet() internal view {
        require(inputSet, "DeployOPChainInput: input not set");
    }

    function input() public view returns (Input memory) {
        assertInputSet();
        return inputs;
    }

    function opChainProxyAdminOwner() public view returns (address) {
        assertInputSet();
        return inputs.roles.opChainProxyAdminOwner;
    }

    function systemConfigOwner() public view returns (address) {
        assertInputSet();
        return inputs.roles.systemConfigOwner;
    }

    function batcher() public view returns (address) {
        assertInputSet();
        return inputs.roles.batcher;
    }

    function unsafeBlockSigner() public view returns (address) {
        assertInputSet();
        return inputs.roles.unsafeBlockSigner;
    }

    function proposer() public view returns (address) {
        assertInputSet();
        return inputs.roles.proposer;
    }

    function challenger() public view returns (address) {
        assertInputSet();
        return inputs.roles.challenger;
    }

    function basefeeScalar() public view returns (uint32) {
        assertInputSet();
        return inputs.basefeeScalar;
    }

    function blobBaseFeeScalar() public view returns (uint32) {
        assertInputSet();
        return inputs.blobBaseFeeScalar;
    }

    function l2ChainId() public view returns (uint256) {
        assertInputSet();
        return inputs.l2ChainId;
    }

    function opsm() public view returns (OPStackManager) {
        assertInputSet();
        return inputs.opsm;
    }
}

contract DeployOPChainOutput {
    struct Output {
        ProxyAdmin opChainProxyAdmin;
        AddressManager addressManager;
        L1ERC721Bridge l1ERC721BridgeProxy;
        SystemConfig systemConfigProxy;
        OptimismMintableERC20Factory optimismMintableERC20FactoryProxy;
        L1StandardBridge l1StandardBridgeProxy;
        L1CrossDomainMessenger l1CrossDomainMessengerProxy;
        // Fault proof contracts below.
        OptimismPortal2 optimismPortalProxy;
        DisputeGameFactory disputeGameFactoryProxy;
        DisputeGameFactory disputeGameFactoryImpl;
        AnchorStateRegistry anchorStateRegistryProxy;
        AnchorStateRegistry anchorStateRegistryImpl;
        FaultDisputeGame faultDisputeGame;
        PermissionedDisputeGame permissionedDisputeGame;
        DelayedWETH delayedWETHPermissionedGameProxy;
        DelayedWETH delayedWETHPermissionlessGameProxy;
    }

    Output internal outputs;

    function set(bytes4 sel, address _addr) public {
        // forgefmt: disable-start
        if (sel == this.opChainProxyAdmin.selector) outputs.opChainProxyAdmin = ProxyAdmin(_addr) ;
        else if (sel == this.addressManager.selector) outputs.addressManager = AddressManager(_addr) ;
        else if (sel == this.l1ERC721BridgeProxy.selector) outputs.l1ERC721BridgeProxy = L1ERC721Bridge(_addr) ;
        else if (sel == this.systemConfigProxy.selector) outputs.systemConfigProxy = SystemConfig(_addr) ;
        else if (sel == this.optimismMintableERC20FactoryProxy.selector) outputs.optimismMintableERC20FactoryProxy = OptimismMintableERC20Factory(_addr) ;
        else if (sel == this.l1StandardBridgeProxy.selector) outputs.l1StandardBridgeProxy = L1StandardBridge(payable(_addr)) ;
        else if (sel == this.l1CrossDomainMessengerProxy.selector) outputs.l1CrossDomainMessengerProxy = L1CrossDomainMessenger(_addr) ;
        else if (sel == this.optimismPortalProxy.selector) outputs.optimismPortalProxy = OptimismPortal2(payable(_addr)) ;
        else if (sel == this.disputeGameFactoryProxy.selector) outputs.disputeGameFactoryProxy = DisputeGameFactory(_addr) ;
        else if (sel == this.disputeGameFactoryImpl.selector) outputs.disputeGameFactoryImpl = DisputeGameFactory(_addr) ;
        else if (sel == this.anchorStateRegistryProxy.selector) outputs.anchorStateRegistryProxy = AnchorStateRegistry(_addr) ;
        else if (sel == this.anchorStateRegistryImpl.selector) outputs.anchorStateRegistryImpl = AnchorStateRegistry(_addr) ;
        else if (sel == this.faultDisputeGame.selector) outputs.faultDisputeGame = FaultDisputeGame(_addr) ;
        else if (sel == this.permissionedDisputeGame.selector) outputs.permissionedDisputeGame = PermissionedDisputeGame(_addr) ;
        else if (sel == this.delayedWETHPermissionedGameProxy.selector) outputs.delayedWETHPermissionedGameProxy = DelayedWETH(payable(_addr)) ;
        else if (sel == this.delayedWETHPermissionlessGameProxy.selector) outputs.delayedWETHPermissionlessGameProxy = DelayedWETH(payable(_addr)) ;
        else revert("DeployOPChainOutput: unknown selector");
        // forgefmt: disable-end
    }

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployOPChainOutput: not implemented");
    }

    function output() public view returns (Output memory) {
        return outputs;
    }

    function checkOutput() public view {
        // With 16 addresses, we'd get a stack too deep error if we tried to do this inline as a
        // single call to `Solarray.addresses`. So we split it into two calls.
        address[] memory addrs1 = Solarray.addresses(
            address(outputs.opChainProxyAdmin),
            address(outputs.addressManager),
            address(outputs.l1ERC721BridgeProxy),
            address(outputs.systemConfigProxy),
            address(outputs.optimismMintableERC20FactoryProxy),
            address(outputs.l1StandardBridgeProxy),
            address(outputs.l1CrossDomainMessengerProxy)
        );
        address[] memory addrs2 = Solarray.addresses(
            address(outputs.optimismPortalProxy),
            address(outputs.disputeGameFactoryProxy),
            address(outputs.disputeGameFactoryImpl),
            address(outputs.anchorStateRegistryProxy),
            address(outputs.anchorStateRegistryImpl),
            address(outputs.faultDisputeGame),
            address(outputs.permissionedDisputeGame),
            address(outputs.delayedWETHPermissionedGameProxy),
            address(outputs.delayedWETHPermissionlessGameProxy)
        );
        DeployUtils.assertValidContractAddresses(Solarray.extend(addrs1, addrs2));
    }

    function opChainProxyAdmin() public view returns (ProxyAdmin) {
        DeployUtils.assertValidContractAddress(address(outputs.opChainProxyAdmin));
        return outputs.opChainProxyAdmin;
    }

    function addressManager() public view returns (AddressManager) {
        DeployUtils.assertValidContractAddress(address(outputs.addressManager));
        return outputs.addressManager;
    }

    function l1ERC721BridgeProxy() public view returns (L1ERC721Bridge) {
        DeployUtils.assertValidContractAddress(address(outputs.l1ERC721BridgeProxy));
        return outputs.l1ERC721BridgeProxy;
    }

    function systemConfigProxy() public view returns (SystemConfig) {
        DeployUtils.assertValidContractAddress(address(outputs.systemConfigProxy));
        return outputs.systemConfigProxy;
    }

    function optimismMintableERC20FactoryProxy() public view returns (OptimismMintableERC20Factory) {
        DeployUtils.assertValidContractAddress(address(outputs.optimismMintableERC20FactoryProxy));
        return outputs.optimismMintableERC20FactoryProxy;
    }

    function l1StandardBridgeProxy() public view returns (L1StandardBridge) {
        DeployUtils.assertValidContractAddress(address(outputs.l1StandardBridgeProxy));
        return outputs.l1StandardBridgeProxy;
    }

    function l1CrossDomainMessengerProxy() public view returns (L1CrossDomainMessenger) {
        DeployUtils.assertValidContractAddress(address(outputs.l1CrossDomainMessengerProxy));
        return outputs.l1CrossDomainMessengerProxy;
    }

    function optimismPortalProxy() public view returns (OptimismPortal2) {
        DeployUtils.assertValidContractAddress(address(outputs.optimismPortalProxy));
        return outputs.optimismPortalProxy;
    }

    function disputeGameFactoryProxy() public view returns (DisputeGameFactory) {
        DeployUtils.assertValidContractAddress(address(outputs.disputeGameFactoryProxy));
        return outputs.disputeGameFactoryProxy;
    }

    function disputeGameFactoryImpl() public view returns (DisputeGameFactory) {
        DeployUtils.assertValidContractAddress(address(outputs.disputeGameFactoryImpl));
        return outputs.disputeGameFactoryImpl;
    }

    function anchorStateRegistryProxy() public view returns (AnchorStateRegistry) {
        DeployUtils.assertValidContractAddress(address(outputs.anchorStateRegistryProxy));
        return outputs.anchorStateRegistryProxy;
    }

    function anchorStateRegistryImpl() public view returns (AnchorStateRegistry) {
        DeployUtils.assertValidContractAddress(address(outputs.anchorStateRegistryImpl));
        return outputs.anchorStateRegistryImpl;
    }

    function faultDisputeGame() public view returns (FaultDisputeGame) {
        DeployUtils.assertValidContractAddress(address(outputs.faultDisputeGame));
        return outputs.faultDisputeGame;
    }

    function permissionedDisputeGame() public view returns (PermissionedDisputeGame) {
        DeployUtils.assertValidContractAddress(address(outputs.permissionedDisputeGame));
        return outputs.permissionedDisputeGame;
    }

    function delayedWETHPermissionedGameProxy() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(outputs.delayedWETHPermissionedGameProxy));
        return outputs.delayedWETHPermissionedGameProxy;
    }

    function delayedWETHPermissionlessGameProxy() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(outputs.delayedWETHPermissionlessGameProxy));
        return outputs.delayedWETHPermissionlessGameProxy;
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

    function run(DeployOPChainInput.Input memory _input) public returns (DeployOPChainOutput.Output memory) {
        (DeployOPChainInput doi, DeployOPChainOutput doo) = etchIOContracts();
        doi.loadInput(_input);
        run(doi, doo);
        return doo.output();
    }

    function run(DeployOPChainInput _doi, DeployOPChainOutput _doo) public {
        require(_doi.inputSet(), "DeployOPChain: input not set");

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

    function etchIOContracts() internal returns (DeployOPChainInput doi_, DeployOPChainOutput doo_) {
        (doi_, doo_) = getIOContracts();
        vm.etch(address(doi_), type(DeployOPChainInput).runtimeCode);
        vm.etch(address(doo_), type(DeployOPChainOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployOPChainInput doi_, DeployOPChainOutput doo_) {
        doi_ = DeployOPChainInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainInput"));
        doo_ = DeployOPChainOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainOutput"));
    }
}
