// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { LibString } from "@solady/utils/LibString.sol";

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

    uint256 public withdrawalDelaySeconds;
    uint256 public minProposalSizeBytes;
    uint256 public challengePeriodSeconds;
    uint256 public proofMaturityDelaySeconds;
    uint256 public disputeGameFinalityDelaySeconds;

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

        withdrawalDelaySeconds = _input.withdrawalDelaySeconds;
        minProposalSizeBytes = _input.minProposalSizeBytes;
        challengePeriodSeconds = _input.challengePeriodSeconds;
        proofMaturityDelaySeconds = _input.proofMaturityDelaySeconds;
        disputeGameFinalityDelaySeconds = _input.disputeGameFinalityDelaySeconds;
    }

    function input() public view returns (Input memory) {
        require(inputSet, "DeployImplementationsInput: input not set");
        return inputs;
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

    OptimismPortal2 public optimismPortal2Impl;
    DelayedWETH public delayedWETHImpl;
    PreimageOracle public preimageOracleSingleton;
    MIPS public mipsSingleton;
    SystemConfig public systemConfigImpl;
    L1CrossDomainMessenger public l1CrossDomainMessengerImpl;
    L1ERC721Bridge public l1ERC721BridgeImpl;
    L1StandardBridge public l1StandardBridgeImpl;
    OptimismMintableERC20Factory public optimismMintableERC20FactoryImpl;

    function set(bytes4 sel, address _addr) public {
        // forgefmt: disable-start
        if (sel == this.optimismPortal2Impl.selector) optimismPortal2Impl = OptimismPortal2(payable(_addr));
        else if (sel == this.delayedWETHImpl.selector) delayedWETHImpl = DelayedWETH(payable(_addr));
        else if (sel == this.preimageOracleSingleton.selector) preimageOracleSingleton = PreimageOracle(_addr);
        else if (sel == this.mipsSingleton.selector) mipsSingleton = MIPS(_addr);
        else if (sel == this.systemConfigImpl.selector) systemConfigImpl = SystemConfig(_addr);
        else if (sel == this.l1CrossDomainMessengerImpl.selector) l1CrossDomainMessengerImpl = L1CrossDomainMessenger(_addr);
        else if (sel == this.l1ERC721BridgeImpl.selector) l1ERC721BridgeImpl = L1ERC721Bridge(_addr);
        else if (sel == this.l1StandardBridgeImpl.selector) l1StandardBridgeImpl = L1StandardBridge(payable(_addr));
        else if (sel == this.optimismMintableERC20FactoryImpl.selector) optimismMintableERC20FactoryImpl = OptimismMintableERC20Factory(_addr);
        else revert("DeployImplementationsOutput: unknown selector");
        // forgefmt: disable-end
    }

    function writeOutputFile(string memory _outfile) public pure {
        _outfile;
        require(false, "DeployImplementationsOutput: not implemented");
    }

    function output() public view returns (Output memory) {
        return Output({
            optimismPortal2Impl: optimismPortal2Impl,
            delayedWETHImpl: delayedWETHImpl,
            preimageOracleSingleton: preimageOracleSingleton,
            mipsSingleton: mipsSingleton,
            systemConfigImpl: systemConfigImpl,
            l1CrossDomainMessengerImpl: l1CrossDomainMessengerImpl,
            l1ERC721BridgeImpl: l1ERC721BridgeImpl,
            l1StandardBridgeImpl: l1StandardBridgeImpl,
            optimismMintableERC20FactoryImpl: optimismMintableERC20FactoryImpl
        });
    }

    function checkOutput() public view {
        address[] memory addresses = new address[](9);
        addresses[0] = address(optimismPortal2Impl);
        addresses[1] = address(delayedWETHImpl);
        addresses[2] = address(preimageOracleSingleton);
        addresses[3] = address(mipsSingleton);
        addresses[4] = address(systemConfigImpl);
        addresses[5] = address(l1CrossDomainMessengerImpl);
        addresses[6] = address(l1ERC721BridgeImpl);
        addresses[7] = address(l1StandardBridgeImpl);
        addresses[8] = address(optimismMintableERC20FactoryImpl);

        for (uint256 i = 0; i < addresses.length; i++) {
            address who = addresses[i];
            require(who != address(0), string.concat("check failed: zero address at index ", LibString.toString(i)));
            require(
                who.code.length > 0, string.concat("check failed: no code at ", LibString.toHexStringChecksummed(who))
            );
        }

        for (uint256 i = 0; i < addresses.length; i++) {
            for (uint256 j = i + 1; j < addresses.length; j++) {
                string memory err =
                    string.concat("check failed: duplicates at ", LibString.toString(i), ",", LibString.toString(j));
                require(addresses[i] != addresses[j], err);
            }
        }
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

    function toIOAddress(address _sender, string memory _identifier) internal pure returns (address) {
        return address(uint160(uint256(keccak256(abi.encode(_sender, _identifier)))));
    }

    function etchIOContracts() internal returns (DeployImplementationsInput dsi_, DeployImplementationsOutput dso_) {
        (dsi_, dso_) = getIOContracts();
        vm.etch(address(dsi_), type(DeployImplementationsInput).runtimeCode);
        vm.etch(address(dso_), type(DeployImplementationsOutput).runtimeCode);
    }

    function getIOContracts() public view returns (DeployImplementationsInput dsi_, DeployImplementationsOutput dso_) {
        dsi_ = DeployImplementationsInput(toIOAddress(msg.sender, "optimism.DeployImplementationsInput"));
        dso_ = DeployImplementationsOutput(toIOAddress(msg.sender, "optimism.DeployImplementationsOutput"));
    }

    function assertValidContractAddress(address _address) internal view {
        require(_address != address(0), "DeployImplementations: zero address");
        require(_address.code.length > 0, "DeployImplementations: no code");
    }
}
