// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

import { InteropMigrationInput, InteropMigration, InteropMigrationOutput } from "scripts/deploy/InteropMigration.s.sol";
import { IOPContractsManagerInteropMigrator, IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { Claim } from "src/dispute/lib/Types.sol";

contract InteropMigrationInput_Test is Test {
    InteropMigrationInput input;

    function setUp() public {
        input = new InteropMigrationInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("InteropMigrationInput: prank not set");
        input.prank();

        vm.expectRevert("InteropMigrationInput: not set");
        input.opcm();

        vm.expectRevert("InteropMigrationInput: proposer not set");
        input.proposer();

        vm.expectRevert("InteropMigrationInput: challenger not set");
        input.challenger();

        vm.expectRevert("InteropMigrationInput: maxGameDepth not set");
        input.maxGameDepth();

        vm.expectRevert("InteropMigrationInput: splitDepth not set");
        input.splitDepth();

        vm.expectRevert("InteropMigrationInput: initBond not set");
        input.initBond();

        vm.expectRevert("InteropMigrationInput: clockExtension not set");
        input.clockExtension();

        vm.expectRevert("InteropMigrationInput: maxClockDuration not set");
        input.maxClockDuration();

        vm.expectRevert("InteropMigrationInput: startingAnchorL2SequenceNumber not set");
        input.startingAnchorL2SequenceNumber();

        vm.expectRevert("InteropMigrationInput: startingAnchorRoot not set");
        input.startingAnchorRoot();

        vm.expectRevert("InteropMigrationInput: not set");
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
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(11)))
        });

        // Setup mock addresses and contracts for second config
        address systemConfig2 = makeAddr("systemConfig2");
        address proxyAdmin2 = makeAddr("proxyAdmin2");
        vm.etch(systemConfig2, hex"01");
        vm.etch(proxyAdmin2, hex"01");

        configs[1] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig2),
            cannonPrestate: Claim.wrap(bytes32(uint256(2))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(22)))
        });

        input.set(input.opChainConfigs.selector, configs);

        bytes memory storedConfigs = input.opChainConfigs();
        assertEq(storedConfigs, abi.encode(configs));

        // Additional verification of stored claims if needed
        IOPContractsManager.OpChainConfig[] memory decodedConfigs =
            abi.decode(storedConfigs, (IOPContractsManager.OpChainConfig[]));
        assertEq(Claim.unwrap(decodedConfigs[0].cannonPrestate), bytes32(uint256(1)));
        assertEq(Claim.unwrap(decodedConfigs[1].cannonPrestate), bytes32(uint256(2)));
    }

    function test_setAddress_withZeroAddress_reverts() public {
        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.prank.selector, address(0));

        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.opcm.selector, address(0));

        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.proposer.selector, address(0));

        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.challenger.selector, address(0));
    }

    function test_setOpChainConfigs_withEmptyArray_reverts() public {
        IOPContractsManager.OpChainConfig[] memory emptyConfigs = new IOPContractsManager.OpChainConfig[](0);

        vm.expectRevert("InteropMigrationInput: cannot set empty array");
        input.set(input.opChainConfigs.selector, emptyConfigs);
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("InteropMigrationInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));

        // Create a single config for testing invalid selector
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        address mockSystemConfig = makeAddr("systemConfig");
        address mockProxyAdmin = makeAddr("proxyAdmin");
        vm.etch(mockSystemConfig, hex"01");
        vm.etch(mockProxyAdmin, hex"01");

        configs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(mockSystemConfig),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(11)))
        });

        vm.expectRevert("InteropMigrationInput: unknown selector");
        input.set(bytes4(0xdeadbeef), configs);
    }
}

contract MockOPCM {
    event MigrateCalled(address indexed sysCfgProxy, bytes32 indexed cannonPrestate);

    function migrate(IOPContractsManagerInteropMigrator.MigrateInput memory _input) public {
        emit MigrateCalled(
            address(_input.opChainConfigs[0].systemConfigProxy), Claim.unwrap(_input.opChainConfigs[0].cannonPrestate)
        );
    }
}

contract InteropMigration_Test is Test {
    MockOPCM mockOPCM;
    InteropMigrationInput input;
    IOPContractsManager.OpChainConfig config;
    InteropMigration migration;
    address prank;

    event MigrateCalled(address indexed sysCfgProxy, bytes32 indexed cannonPrestate);

    function setUp() public {
        mockOPCM = new MockOPCM();
        input = new InteropMigrationInput();
        input.set(input.opcm.selector, address(mockOPCM));
        config = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(makeAddr("systemConfigProxy")),
            cannonPrestate: Claim.wrap(keccak256("cannonPrestate")),
            cannonKonaPrestate: Claim.wrap(keccak256("cannonKonaPrestate"))
        });
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        configs[0] = config;
        input.set(input.opChainConfigs.selector, configs);
        prank = makeAddr("prank");
        input.set(input.prank.selector, prank);

        input.set(input.usePermissionlessGame.selector, true);
        input.set(input.startingAnchorL2SequenceNumber.selector, 1);
        input.set(input.startingAnchorRoot.selector, bytes32(uint256(1)));
        input.set(input.proposer.selector, makeAddr("proposer"));
        input.set(input.challenger.selector, makeAddr("challenger"));
        input.set(input.maxGameDepth.selector, 100);
        input.set(input.splitDepth.selector, 10);
        input.set(input.initBond.selector, 1000);
        input.set(input.clockExtension.selector, 100);
        input.set(input.maxClockDuration.selector, 1000);

        migration = new InteropMigration();
    }

    function test_migrate_succeeds() public {
        // MigrateCalled should be emitted by the prank since it's a delegatecall.
        vm.expectEmit(address(prank));
        emit MigrateCalled(address(config.systemConfigProxy), Claim.unwrap(config.cannonPrestate));

        // mocks for post-migration checks
        address portal = makeAddr("optimismPortal");
        address dgf = makeAddr("disputeGameFactory");
        IOPContractsManager.OpChainConfig[] memory opChainConfigs =
            abi.decode(input.opChainConfigs(), (IOPContractsManager.OpChainConfig[]));
        vm.mockCall(
            address(opChainConfigs[0].systemConfigProxy),
            abi.encodeCall(ISystemConfig.optimismPortal, ()),
            abi.encode(portal)
        );
        vm.etch(dgf, hex"01");
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal.disputeGameFactory, ()), abi.encode(dgf));

        InteropMigrationOutput output = new InteropMigrationOutput();
        migration.run(input, output);
    }
}
