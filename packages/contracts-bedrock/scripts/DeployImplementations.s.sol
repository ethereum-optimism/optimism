// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";

import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { MIPS } from "src/cannon/MIPS.sol";

import { OptimismPortal2 } from "src/L1/OptimismPortal2.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Solarray } from "scripts/libraries/Solarray.sol";

// See DeploySuperchain.s.sol for detailed comments on the script architecture used here.
contract DeployImplementationsInput {
    struct Input {
        uint256 withdrawalDelaySeconds;
        uint256 minProposalSizeBytes;
        uint256 challengePeriodSeconds;
        uint256 proofMaturityDelaySeconds;
        uint256 disputeGameFinalityDelaySeconds;
    }

    bool public inputSet = false;
    Input internal inputs;

    function loadInputFile(string memory _infile) public {
        _infile;
        Input memory parsedInput;
        loadInput(parsedInput);
        require(false, "DeployImplementationsInput: not implemented");
    }

    function loadInput(Input memory _input) public {
        require(!inputSet, "DeployImplementationsInput: input already set");
        require(
            _input.challengePeriodSeconds <= type(uint64).max, "DeployImplementationsInput: challenge period too large"
        );

        inputSet = true;
        inputs = _input;
    }

    function assertInputSet() internal view {
        require(inputSet, "DeployImplementationsInput: input not set");
    }

    function input() public view returns (Input memory) {
        assertInputSet();
        return inputs;
    }

    function withdrawalDelaySeconds() public view returns (uint256) {
        assertInputSet();
        return inputs.withdrawalDelaySeconds;
    }

    function minProposalSizeBytes() public view returns (uint256) {
        assertInputSet();
        return inputs.minProposalSizeBytes;
    }

    function challengePeriodSeconds() public view returns (uint256) {
        assertInputSet();
        return inputs.challengePeriodSeconds;
    }

    function proofMaturityDelaySeconds() public view returns (uint256) {
        assertInputSet();
        return inputs.proofMaturityDelaySeconds;
    }

    function disputeGameFinalityDelaySeconds() public view returns (uint256) {
        assertInputSet();
        return inputs.disputeGameFinalityDelaySeconds;
    }
}

contract DeployImplementationsOutput {
    struct Output {
        DelayedWETH delayedWETHImpl;
        OptimismPortal2 optimismPortal2Impl;
        PreimageOracle preimageOracleSingleton;
        MIPS mipsSingleton;
        SystemConfig systemConfigImpl;
        L1CrossDomainMessenger l1CrossDomainMessengerImpl;
        L1ERC721Bridge l1ERC721BridgeImpl;
        L1StandardBridge l1StandardBridgeImpl;
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
    }

    Output internal outputs;

    function set(bytes4 sel, address _addr) public {
        // forgefmt: disable-start
        if (sel == this.optimismPortal2Impl.selector) outputs.optimismPortal2Impl = OptimismPortal2(payable(_addr));
        else if (sel == this.delayedWETHImpl.selector) outputs.delayedWETHImpl = DelayedWETH(payable(_addr));
        else if (sel == this.preimageOracleSingleton.selector) outputs.preimageOracleSingleton = PreimageOracle(_addr);
        else if (sel == this.mipsSingleton.selector) outputs.mipsSingleton = MIPS(_addr);
        else if (sel == this.systemConfigImpl.selector) outputs.systemConfigImpl = SystemConfig(_addr);
        else if (sel == this.l1CrossDomainMessengerImpl.selector) outputs.l1CrossDomainMessengerImpl = L1CrossDomainMessenger(_addr);
        else if (sel == this.l1ERC721BridgeImpl.selector) outputs.l1ERC721BridgeImpl = L1ERC721Bridge(_addr);
        else if (sel == this.l1StandardBridgeImpl.selector) outputs.l1StandardBridgeImpl = L1StandardBridge(payable(_addr));
        else if (sel == this.optimismMintableERC20FactoryImpl.selector) outputs.optimismMintableERC20FactoryImpl = OptimismMintableERC20Factory(_addr);
        else revert("DeployImplementationsOutput: unknown selector");
        // forgefmt: disable-end
    }

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployImplementationsOutput: not implemented");
    }

    function output() public view returns (Output memory) {
        return outputs;
    }

    function checkOutput() public view {
        address[] memory addrs = Solarray.addresses(
            address(outputs.optimismPortal2Impl),
            address(outputs.delayedWETHImpl),
            address(outputs.preimageOracleSingleton),
            address(outputs.mipsSingleton),
            address(outputs.systemConfigImpl),
            address(outputs.l1CrossDomainMessengerImpl),
            address(outputs.l1ERC721BridgeImpl),
            address(outputs.l1StandardBridgeImpl),
            address(outputs.optimismMintableERC20FactoryImpl)
        );
        DeployUtils.assertValidContractAddresses(addrs);
    }

    function optimismPortal2Impl() public view returns (OptimismPortal2) {
        DeployUtils.assertValidContractAddress(address(outputs.optimismPortal2Impl));
        return outputs.optimismPortal2Impl;
    }

    function delayedWETHImpl() public view returns (DelayedWETH) {
        DeployUtils.assertValidContractAddress(address(outputs.delayedWETHImpl));
        return outputs.delayedWETHImpl;
    }

    function preimageOracleSingleton() public view returns (PreimageOracle) {
        DeployUtils.assertValidContractAddress(address(outputs.preimageOracleSingleton));
        return outputs.preimageOracleSingleton;
    }

    function mipsSingleton() public view returns (MIPS) {
        DeployUtils.assertValidContractAddress(address(outputs.mipsSingleton));
        return outputs.mipsSingleton;
    }

    function systemConfigImpl() public view returns (SystemConfig) {
        DeployUtils.assertValidContractAddress(address(outputs.systemConfigImpl));
        return outputs.systemConfigImpl;
    }

    function l1CrossDomainMessengerImpl() public view returns (L1CrossDomainMessenger) {
        DeployUtils.assertValidContractAddress(address(outputs.l1CrossDomainMessengerImpl));
        return outputs.l1CrossDomainMessengerImpl;
    }

    function l1ERC721BridgeImpl() public view returns (L1ERC721Bridge) {
        DeployUtils.assertValidContractAddress(address(outputs.l1ERC721BridgeImpl));
        return outputs.l1ERC721BridgeImpl;
    }

    function l1StandardBridgeImpl() public view returns (L1StandardBridge) {
        DeployUtils.assertValidContractAddress(address(outputs.l1StandardBridgeImpl));
        return outputs.l1StandardBridgeImpl;
    }

    function optimismMintableERC20FactoryImpl() public view returns (OptimismMintableERC20Factory) {
        DeployUtils.assertValidContractAddress(address(outputs.optimismMintableERC20FactoryImpl));
        return outputs.optimismMintableERC20FactoryImpl;
    }
}

contract DeployImplementations is Script {
    // -------- Core Deployment Methods --------

    function run(string memory _infile) public {
        (DeployImplementationsInput dsi, DeployImplementationsOutput dso) = etchIOContracts();
        dsi.loadInputFile(_infile);
        run(dsi, dso);
        string memory outfile = ""; // This will be derived from input file name, e.g. `foo.in.toml` -> `foo.out.toml`
        dso.writeOutputFile(outfile);
        require(false, "DeployImplementations: run is not implemented");
    }

    function run(DeployImplementationsInput.Input memory _input)
        public
        returns (DeployImplementationsOutput.Output memory)
    {
        (DeployImplementationsInput dsi, DeployImplementationsOutput dso) = etchIOContracts();
        dsi.loadInput(_input);
        run(dsi, dso);
        return dso.output();
    }

    function run(DeployImplementationsInput _dsi, DeployImplementationsOutput _dso) public {
        require(_dsi.inputSet(), "DeployImplementations: input not set");

        deploySystemConfigImpl(_dsi, _dso);
        deployL1CrossDomainMessengerImpl(_dsi, _dso);
        deployL1ERC721BridgeImpl(_dsi, _dso);
        deployL1StandardBridgeImpl(_dsi, _dso);
        deployOptimismMintableERC20FactoryImpl(_dsi, _dso);
        deployOptimismPortalImpl(_dsi, _dso);
        deployDelayedWETHImpl(_dsi, _dso);
        deployPreimageOracleSingleton(_dsi, _dso);
        deployMipsSingleton(_dsi, _dso);

        _dso.checkOutput();
    }

    // -------- Deployment Steps --------

    function deploySystemConfigImpl(DeployImplementationsInput, DeployImplementationsOutput _dso) public {
        vm.broadcast(msg.sender);
        SystemConfig systemConfigImpl = new SystemConfig();

        vm.label(address(systemConfigImpl), "systemConfigImpl");
        _dso.set(_dso.systemConfigImpl.selector, address(systemConfigImpl));
    }

    function deployL1CrossDomainMessengerImpl(DeployImplementationsInput, DeployImplementationsOutput _dso) public {
        vm.broadcast(msg.sender);
        L1CrossDomainMessenger l1CrossDomainMessengerImpl = new L1CrossDomainMessenger();

        vm.label(address(l1CrossDomainMessengerImpl), "L1CrossDomainMessengerImpl");
        _dso.set(_dso.l1CrossDomainMessengerImpl.selector, address(l1CrossDomainMessengerImpl));
    }

    function deployL1ERC721BridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dso) public {
        vm.broadcast(msg.sender);
        L1ERC721Bridge l1ERC721BridgeImpl = new L1ERC721Bridge();

        vm.label(address(l1ERC721BridgeImpl), "L1ERC721BridgeImpl");
        _dso.set(_dso.l1ERC721BridgeImpl.selector, address(l1ERC721BridgeImpl));
    }

    function deployL1StandardBridgeImpl(DeployImplementationsInput, DeployImplementationsOutput _dso) public {
        vm.broadcast(msg.sender);
        L1StandardBridge l1StandardBridgeImpl = new L1StandardBridge();

        vm.label(address(l1StandardBridgeImpl), "L1StandardBridgeImpl");
        _dso.set(_dso.l1StandardBridgeImpl.selector, address(l1StandardBridgeImpl));
    }

    function deployOptimismMintableERC20FactoryImpl(
        DeployImplementationsInput,
        DeployImplementationsOutput _dso
    )
        public
    {
        vm.broadcast(msg.sender);
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl = new OptimismMintableERC20Factory();

        vm.label(address(optimismMintableERC20FactoryImpl), "OptimismMintableERC20FactoryImpl");
        _dso.set(_dso.optimismMintableERC20FactoryImpl.selector, address(optimismMintableERC20FactoryImpl));
    }

    // The fault proofs contracts are configured as follows:
    //   - DisputeGameFactory: Proxied, bespoke per chain.
    //   - AnchorStateRegistry: Proxied, bespoke per chain.
    //   - FaultDisputeGame: Not proxied, bespoke per chain.
    //   - PermissionedDisputeGame: Not proxied, bespoke per chain.
    //   - DelayedWETH: Proxied, shared by all standard chains.
    //   - PreimageOracle: Not proxied, shared by all standard chains.
    //   - MIPS: Not proxied, shared by all standard chains.
    //   - OptimismPortal2: Proxied, shared by all standard chains.
    //
    // This script only deploys the shared contracts. The bespoke contracts are deployed by
    // `DeployOPChain.s.sol`. When the shared contracts are proxied, we call the "implementations",
    // and when they are not proxied, we call them "singletons". So here we deploy:
    //
    //   - OptimismPortal2 (implementation)
    //   - DelayedWETH (implementation)
    //   - PreimageOracle (singleton)
    //   - MIPS (singleton)

    function deployOptimismPortalImpl(DeployImplementationsInput _dsi, DeployImplementationsOutput _dso) public {
        uint256 proofMaturityDelaySeconds = _dsi.proofMaturityDelaySeconds();
        uint256 disputeGameFinalityDelaySeconds = _dsi.disputeGameFinalityDelaySeconds();

        vm.broadcast(msg.sender);
        OptimismPortal2 optimismPortal2Impl = new OptimismPortal2({
            _proofMaturityDelaySeconds: proofMaturityDelaySeconds,
            _disputeGameFinalityDelaySeconds: disputeGameFinalityDelaySeconds
        });

        vm.label(address(optimismPortal2Impl), "OptimismPortal2Impl");
        _dso.set(_dso.optimismPortal2Impl.selector, address(optimismPortal2Impl));
    }

    function deployDelayedWETHImpl(DeployImplementationsInput _dsi, DeployImplementationsOutput _dso) public {
        uint256 withdrawalDelaySeconds = _dsi.withdrawalDelaySeconds();

        vm.broadcast(msg.sender);
        DelayedWETH delayedWETHImpl = new DelayedWETH({ _delay: withdrawalDelaySeconds });

        vm.label(address(delayedWETHImpl), "DelayedWETHImpl");
        _dso.set(_dso.delayedWETHImpl.selector, address(delayedWETHImpl));
    }

    function deployPreimageOracleSingleton(DeployImplementationsInput _dsi, DeployImplementationsOutput _dso) public {
        uint256 minProposalSizeBytes = _dsi.minProposalSizeBytes();
        uint256 challengePeriodSeconds = _dsi.challengePeriodSeconds();

        vm.broadcast(msg.sender);
        PreimageOracle preimageOracleSingleton =
            new PreimageOracle({ _minProposalSize: minProposalSizeBytes, _challengePeriod: challengePeriodSeconds });

        vm.label(address(preimageOracleSingleton), "PreimageOracleSingleton");
        _dso.set(_dso.preimageOracleSingleton.selector, address(preimageOracleSingleton));
    }

    function deployMipsSingleton(DeployImplementationsInput, DeployImplementationsOutput _dso) public {
        IPreimageOracle preimageOracle = IPreimageOracle(_dso.preimageOracleSingleton());

        vm.broadcast(msg.sender);
        MIPS mipsSingleton = new MIPS(preimageOracle);

        vm.label(address(mipsSingleton), "MIPSSingleton");
        _dso.set(_dso.mipsSingleton.selector, address(mipsSingleton));
    }

    // -------- Utilities --------

    function etchIOContracts() internal returns (DeployImplementationsInput dsi_, DeployImplementationsOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeployImplementationsInput).runtimeCode);
        vm.etch(address(dso_), type(DeployImplementationsOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployImplementationsInput dsi_, DeployImplementationsOutput dso_) {
        dsi_ = DeployImplementationsInput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployImplementationsInput"));
        dso_ = DeployImplementationsOutput(DeployUtils.toIOAddress(msg.sender, "optimism.DeployImplementationsOutput"));
    }
}
