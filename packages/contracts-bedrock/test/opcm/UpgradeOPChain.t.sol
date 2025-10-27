// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Claim } from "src/dispute/lib/Types.sol";

import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

import { IOPContractsManager, IOPContractsManagerPre4_1_0 } from "interfaces/L1/IOPContractsManager.sol";
import { UpgradeOPChain, UpgradeOPChainInput } from "scripts/deploy/UpgradeOPChain.s.sol";

contract UpgradeOPChainInput_Test is Test {
    UpgradeOPChainInput input;

    function setUp() public {
        input = new UpgradeOPChainInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("UpgradeOPChainInput: prank not set");
        input.prank();

        vm.expectRevert("UpgradeOPChainInput: opcm not set");
        input.opcm();

        vm.expectRevert("UpgradeOPChainInput: opChainConfigs not set");
        input.opChainConfigs();
    }

    function test_setAddress_succeeds() public {
        address mockPrank = makeAddr("prank");
        address mockOPCM = makeAddr("opcm");

        // Create mock contract at OPCM address
        vm.etch(mockOPCM, hex"01");

        input.set(input.prank.selector, mockPrank);
        input.set(input.opcm.selector, mockOPCM);

        assertEq(input.prank(), mockPrank);
        assertEq(address(input.opcm()), mockOPCM);
    }

    function test_setOpChainConfigs_succeeds() public {
        // Create sample OpChainConfig array
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](2);

        // Setup mock addresses and contracts for first config
        address systemConfig1 = makeAddr("systemConfig1");
        address proxyAdmin1 = makeAddr("proxyAdmin1");
        vm.etch(systemConfig1, hex"01");
        vm.etch(proxyAdmin1, hex"01");

        configs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig1),
            absolutePrestate: Claim.wrap(bytes32(uint256(1)))
        });

        // Setup mock addresses and contracts for second config
        address systemConfig2 = makeAddr("systemConfig2");
        address proxyAdmin2 = makeAddr("proxyAdmin2");
        vm.etch(systemConfig2, hex"01");
        vm.etch(proxyAdmin2, hex"01");

        configs[1] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig2),
            absolutePrestate: Claim.wrap(bytes32(uint256(2)))
        });

        input.set(input.opChainConfigs.selector, abi.encode(configs));

        bytes memory storedConfigs = input.opChainConfigs();
        assertEq(storedConfigs, abi.encode(configs));

        // Additional verification of stored claims if needed
        IOPContractsManager.OpChainConfig[] memory decodedConfigs =
            abi.decode(storedConfigs, (IOPContractsManager.OpChainConfig[]));
        assertEq(Claim.unwrap(decodedConfigs[0].absolutePrestate), bytes32(uint256(1)));
        assertEq(Claim.unwrap(decodedConfigs[1].absolutePrestate), bytes32(uint256(2)));
    }

    function test_setAddress_withZeroAddress_reverts() public {
        vm.expectRevert("UpgradeOPChainInput: cannot set zero address");
        input.set(input.prank.selector, address(0));

        vm.expectRevert("UpgradeOPChainInput: cannot set zero address");
        input.set(input.opcm.selector, address(0));
    }

    function test_setOpChainConfigs_withEmptyArray_reverts() public {
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](0);
        vm.expectRevert("UpgradeOPChainInput: cannot set empty array");
        input.set(input.opChainConfigs.selector, abi.encode(configs));

        vm.expectRevert("UpgradeOPChainInput: cannot set empty array");
        input.set(input.opChainConfigs.selector, new bytes(0));
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("UpgradeOPChainInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));

        // Create a single config for testing invalid selector
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        address mockSystemConfig = makeAddr("systemConfig");
        address mockProxyAdmin = makeAddr("proxyAdmin");
        vm.etch(mockSystemConfig, hex"01");
        vm.etch(mockProxyAdmin, hex"01");

        configs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(mockSystemConfig),
            absolutePrestate: Claim.wrap(bytes32(uint256(1)))
        });

        vm.expectRevert("UpgradeOPChainInput: unknown selector");
        input.set(bytes4(0xdeadbeef), abi.encode(configs));
    }
}

contract MockOPCM {
    event UpgradeCalled(address indexed sysCfgProxy, bytes32 indexed absolutePrestate);

    function upgrade(IOPContractsManager.OpChainConfig[] memory _opChainConfigs) public {
        emit UpgradeCalled(
            address(_opChainConfigs[0].systemConfigProxy), Claim.unwrap(_opChainConfigs[0].absolutePrestate)
        );
    }

    function version() public pure returns (string memory) {
        return "4.1.0";
    }
}

contract MockOPCMPre410 {
    event UpgradeCalled(address indexed sysCfgProxy, address indexed proxyAdmin, bytes32 indexed absolutePrestate);

    function upgrade(IOPContractsManagerPre4_1_0.OpChainConfig[] memory _opChainConfigs) public {
        emit UpgradeCalled(
            address(_opChainConfigs[0].systemConfigProxy),
            address(_opChainConfigs[0].proxyAdmin),
            Claim.unwrap(_opChainConfigs[0].absolutePrestate)
        );
    }

    function version() public pure returns (string memory) {
        return "4.0.0";
    }
}

contract UpgradeOPChain_Test is Test {
    address mockOPCM;
    UpgradeOPChainInput uoci;
    IOPContractsManager.OpChainConfig config;
    UpgradeOPChain upgradeOPChain;
    address prank;

    event UpgradeCalled(address indexed sysCfgProxy, bytes32 indexed absolutePrestate);
    event UpgradeCalled(address indexed sysCfgProxy, address indexed proxyAdmin, bytes32 indexed absolutePrestate);

    function setUp() public virtual {
        mockOPCM = address(new MockOPCM());
        uoci = new UpgradeOPChainInput();
        uoci.set(uoci.opcm.selector, mockOPCM);
        config = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(makeAddr("systemConfigProxy")),
            absolutePrestate: Claim.wrap(keccak256("absolutePrestate"))
        });
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        configs[0] = config;
        uoci.set(uoci.opChainConfigs.selector, abi.encode(configs));
        prank = makeAddr("prank");
        uoci.set(uoci.prank.selector, prank);
        upgradeOPChain = new UpgradeOPChain();
    }

    function test_upgrade_succeeds() public {
        // For opcm >= 4.1.0
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(address(config.systemConfigProxy), Claim.unwrap(config.absolutePrestate));
        upgradeOPChain.run(uoci);

        // For opcm < 4.1.0
        IOPContractsManagerPre4_1_0.OpChainConfig memory configPre410 = IOPContractsManagerPre4_1_0.OpChainConfig({
            systemConfigProxy: ISystemConfig(makeAddr("systemConfigProxy")),
            proxyAdmin: IProxyAdmin(makeAddr("proxyAdmin")),
            absolutePrestate: Claim.wrap(keccak256("absolutePrestate"))
        });
        IOPContractsManagerPre4_1_0.OpChainConfig[] memory configsPre410 =
            new IOPContractsManagerPre4_1_0.OpChainConfig[](1);
        configsPre410[0] = configPre410;
        uoci.set(uoci.opChainConfigs.selector, abi.encode(configsPre410));
        mockOPCM = address(new MockOPCMPre410());
        uoci.set(uoci.opcm.selector, mockOPCM);
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(
            address(configPre410.systemConfigProxy),
            address(configPre410.proxyAdmin),
            Claim.unwrap(configPre410.absolutePrestate)
        );
        upgradeOPChain.run(uoci);
    }

    function testFuzz_upgrade_unexpectedEncoding_reverts(
        IOPContractsManager.OpChainConfig[] memory _opChainConfigs,
        IOPContractsManagerPre4_1_0.OpChainConfig[] memory _opChainConfigsPre410
    )
        public
    {
        vm.assume(_opChainConfigsPre410.length > 0);
        vm.assume(_opChainConfigs.length > 0);

        // Currently the mockOPCM set in the input contract is that of a version >= 4.1.0
        // So lets try to set opChainConfigs to be one of < 4.1.0 and expect a revert.
        uoci.set(uoci.opChainConfigs.selector, abi.encode(_opChainConfigsPre410));
        // It can revert because it could not decode the opChainConfigs (this happens if it expects an address but sees
        // a value with upper 12 bits dirty). If it does not see this it will still revert with a custom string error
        // because it decoded into the wrong type.
        (bool success, bytes memory result) = address(upgradeOPChain).call(abi.encodeCall(upgradeOPChain.run, (uoci)));
        assertFalse(success, "UpgradeOPChain_Test: call should revert");
        assertTrue(
            keccak256(result)
                == keccak256(
                    bytes.concat(
                        bytes4(keccak256("Error(string)")), abi.encode("UpgradeOPChain: opChainConfigs Unexpected encoding")
                    )
                ) || keccak256(result) == keccak256(""),
            "UpgradeOPChain_Test: result should be the expected error"
        );

        // Now lets set the mockOPCM to be a version < 4.1.0 and expect a revert when we set the opChainConfigs to be
        // that of version >= 4.1.0 and calling run.
        mockOPCM = address(new MockOPCMPre410());
        uoci.set(uoci.opcm.selector, mockOPCM);
        uoci.set(uoci.opChainConfigs.selector, abi.encode(_opChainConfigs));
        // It can revert because it could not decode the opChainConfigs (this happens if it expects an address but sees
        // a value with upper 12 bits dirty). If it does not see this it will still revert with a custom string error
        // because it decoded into the wrong type.
        (success, result) = address(upgradeOPChain).call(abi.encodeCall(upgradeOPChain.run, (uoci)));
        assertFalse(success, "UpgradeOPChain_Test: opChainConfigsPre410 call should revert");
        assertTrue(
            keccak256(result)
                == keccak256(
                    bytes.concat(
                        bytes4(keccak256("Error(string)")),
                        abi.encode("UpgradeOPChain: opChainConfigsPre410 Unexpected encoding")
                    )
                ) || keccak256(result) == keccak256(""),
            "UpgradeOPChain_Test: opChainConfigsPre410 result should be the expected error"
        );
    }
}
