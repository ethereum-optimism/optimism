// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { OpStackManager } from "src/L1/OpStackManager.sol";
import { AddressManager } from "src/legacy/AddressManager.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { ProtocolVersions } from "src/L1/ProtocolVersions.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { L1ERC721Bridge } from "src/L1/L1ERC721Bridge.sol";
import { L1StandardBridge } from "src/L1/L1StandardBridge.sol";
import { L2OutputOracle } from "src/L1/L2OutputOracle.sol";
import { OptimismPortal } from "src/L1/OptimismPortal.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";

contract BlueprintDeployer {
    function blueprintDeployerBytecode(bytes memory initcode) public pure returns (bytes memory) {
        bytes memory blueprintPreamble = hex"FE7100"; // ERC-5202 preamble.
        bytes memory blueprintBytecode = abi.encodePacked(blueprintPreamble, initcode);

        // The length of the deployed code in bytes.
        bytes2 lenBytes = bytes2(uint16(blueprintBytecode.length));

        // copy <blueprint_bytecode> to memory and `RETURN` it per EVM creation semantics
        // PUSH2 <len> RETURNDATASIZE DUP2 PUSH1 10 RETURNDATASIZE CODECOPY RETURN
        bytes memory deployBytecode = abi.encodePacked(hex"61", lenBytes, hex"3d8160093d39f3");

        return abi.encodePacked(deployBytecode, blueprintBytecode);
    }
}

contract OPStackManagerTest is Test {
    // Test data.
    OpStackManager opStackManagerImpl;
    OpStackManager opStackManager;
    address opStackManagerOwner = makeAddr("opStackManagerOwner");

    // Sepolia data.
    SystemConfig systemConfig = SystemConfig(0x034edD2A225f7f429A63E0f1D2084B9E0A93b538);
    ProtocolVersions protocolVersions = ProtocolVersions(0x79ADD5713B383DAa0a138d3C4780C7A1804a8090);
    AddressManager addressManager = AddressManager(0x9bFE9c5609311DF1c011c47642253B78a4f33F4B);
    address proxyAdminOwner = 0xDEe57160aAfCF04c34C887B5962D0a69676d3C8B;

    OpStackManager.ImplementationSetter[] impls;

    constructor() {
        impls.push(
            OpStackManager.ImplementationSetter(
                "L1CrossDomainMessenger",
                OpStackManager.Implementation(
                    0xD3494713A5cfaD3F5359379DfA074E2Ac8C6Fd65, L1CrossDomainMessenger.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "L1ERC721Bridge",
                OpStackManager.Implementation(
                    0xAE2AF01232a6c4a4d3012C5eC5b1b35059caF10d, L1ERC721Bridge.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "L1StandardBridge",
                OpStackManager.Implementation(
                    0x64B5a5Ed26DCb17370Ff4d33a8D503f0fbD06CfF, L1StandardBridge.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "L2OutputOracle",
                OpStackManager.Implementation(
                    0xF243BEd163251380e78068d317ae10f26042B292, L2OutputOracle.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "OptimismPortal",
                OpStackManager.Implementation(
                    0x2D778797049FE9259d947D1ED8e5442226dFB589, OptimismPortal.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "SystemConfig",
                OpStackManager.Implementation(
                    0xba2492e52F45651B60B8B38d4Ea5E2390C64Ffb1, SystemConfig.initialize.selector
                )
            )
        );
        impls.push(
            OpStackManager.ImplementationSetter(
                "OptimismMintableERC20Factory",
                OpStackManager.Implementation(
                    0xE01efbeb1089D1d1dB9c6c8b135C934C0734c846, OptimismMintableERC20Factory.initialize.selector
                )
            )
        );
    }

    function setUp() public {
        // Forking sepolia from one block before the FPAC upgrade.
        uint64 opSepoliaChainId = 11155420;
        vm.createSelectFork(vm.envString("SEPOLIA_RPC_URL"), 5519723);

        // Deploy and configure OpStackManager behind a proxy.
        ProxyAdmin opStackManagerProxyAdmin = new ProxyAdmin({ _owner: address(this) });

        opStackManagerImpl =
            new OpStackManager(opSepoliaChainId, systemConfig, protocolVersions, addressManager, proxyAdminOwner);
        opStackManager = OpStackManager(address(new Proxy(address(opStackManagerProxyAdmin))));

        bytes memory data = abi.encodeCall(OpStackManager.initialize, (opStackManagerOwner));
        opStackManagerProxyAdmin.upgradeAndCall(payable(address(opStackManager)), address(opStackManagerImpl), data);

        // Release op-contracts/v1.3.0
        vm.prank(opStackManagerOwner);
        opStackManager.release({ version: "op-contracts/v1.3.0", isLatest: true, impls: impls });

        // Registers the existing OP Mainnet chain.
        vm.prank(opStackManagerOwner);
        opStackManager.register(
            opSepoliaChainId,
            "op-contracts/v1.3.0",
            SystemConfig(0x034edD2A225f7f429A63E0f1D2084B9E0A93b538),
            ProtocolVersions(0x79ADD5713B383DAa0a138d3C4780C7A1804a8090)
        );
    }

    function test_DeployGasCost() public {
        uint64 l2ChainId = 0x999;
        address dummyProxyAdminOwner = makeAddr("dummyProxyAdminOwner");
        OpStackManager.SystemConfigInputs memory systemConfigInputs = OpStackManager.SystemConfigInputs({
            systemConfigOwner: makeAddr("dummySystemConfigOwner"),
            overhead: 1,
            scalar: 2,
            batcherHash: bytes32(uint256(0x1234)),
            unsafeBlockSigner: makeAddr("dummyUnsafeBlockSigner")
        });
        OpStackManager.L2OutputOracleInputs memory l2OutputOracleInputs = OpStackManager.L2OutputOracleInputs({
            submissionInterval: 1800,
            proposer: makeAddr("proposer"),
            challenger: makeAddr("challenger")
        });

        opStackManager.deploy(l2ChainId, dummyProxyAdminOwner, systemConfigInputs, l2OutputOracleInputs);
    }
}
