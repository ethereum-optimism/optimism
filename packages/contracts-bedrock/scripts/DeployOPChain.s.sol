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
        (DeployOPChainInput dsi, DeployOPChainOutput dso) = etchIOContracts();
        dsi.loadInputFile(_infile);
        run(dsi, dso);
        string memory outfile = ""; // This will be derived from input file name, e.g. `foo.in.toml` -> `foo.out.toml`
        dso.writeOutputFile(outfile);
        require(false, "DeployOPChain: run is not implemented");
    }

    function run(DeployOPChainInput.Input memory _input) public returns (DeployOPChainOutput.Output memory) {
        (DeployOPChainInput dsi, DeployOPChainOutput dso) = etchIOContracts();
        dsi.loadInput(_input);
        run(dsi, dso);
        return dso.output();
    }

    function run(DeployOPChainInput _dsi, DeployOPChainOutput _dso) public view {
        require(_dsi.inputSet(), "DeployOPChain: input not set");

        // TODO call OP Stack Manager deploy method

        _dso.checkOutput();
    }

    // -------- Utilities --------

    function etchIOContracts() internal returns (DeployOPChainInput dsi_, DeployOPChainOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeployOPChainInput).runtimeCode);
        vm.etch(address(dso_), type(DeployOPChainOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployOPChainInput dsi_, DeployOPChainOutput dso_) {
        dsi_ = DeployOPChainInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainInput"));
        dso_ = DeployOPChainOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployOPChainOutput"));
    }
}
