// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { InteropMigrationInput, InteropMigration, InteropMigrationOutput } from "scripts/deploy/InteropMigration.s.sol";

// Libraries
import { Claim, Duration, Hash, GameType, Proposal } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOPContractsManagerInteropMigrator, IOPContractsManager } from "interfaces/L1/IOPContractsManager.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

contract InteropMigrationInput_Test is Test {
    InteropMigrationInput input;

    function setUp() public {
        input = new InteropMigrationInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("InteropMigrationInput: prank not set");
        input.prank();

        vm.expectRevert("InteropMigrationInput: opcm not set");
        input.opcm();

        vm.expectRevert("InteropMigrationInput: migrateInput not set");
        input.migrateInput();
    }

    function test_setAddress_succeeds() public {
        address mockPrank = makeAddr("prank");
        address mockOPCM = makeAddr("opcm");

        input.set(input.prank.selector, mockPrank);
        input.set(input.opcm.selector, mockOPCM);

        assertEq(input.prank(), mockPrank);
        assertEq(input.opcm(), mockOPCM);
    }

    function test_setMigrateInputV1_succeeds() public {
        // Create sample V1 input
        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        address systemConfig1 = makeAddr("systemConfig1");
        vm.etch(systemConfig1, hex"01");

        configs[0] = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig1),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(11)))
        });

        IOPContractsManagerInteropMigrator.MigrateInput memory migrateInput = IOPContractsManagerInteropMigrator
            .MigrateInput({
            usePermissionlessGame: true,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(uint256(1))), l2SequenceNumber: 100 }),
            gameParameters: IOPContractsManagerInteropMigrator.GameParameters({
                proposer: makeAddr("proposer"),
                challenger: makeAddr("challenger"),
                maxGameDepth: 73,
                splitDepth: 30,
                initBond: 1 ether,
                clockExtension: Duration.wrap(10800),
                maxClockDuration: Duration.wrap(302400)
            }),
            opChainConfigs: configs
        });

        input.set(input.migrateInput.selector, migrateInput);

        bytes memory storedInput = input.migrateInput();
        assertEq(storedInput, abi.encode(migrateInput));
    }

    function test_setMigrateInputV2_succeeds() public {
        // Create sample V2 input
        ISystemConfig[] memory systemConfigs = new ISystemConfig[](1);
        address systemConfig1 = makeAddr("systemConfig1");
        vm.etch(systemConfig1, hex"01");
        systemConfigs[0] = ISystemConfig(systemConfig1);

        IOPContractsManagerUtils.DisputeGameConfig[] memory gameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        gameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 1 ether,
            gameType: GameType.wrap(0),
            gameArgs: abi.encodePacked(bytes32(uint256(0xabc)))
        });

        IOPContractsManagerMigrator.MigrateInput memory migrateInput = IOPContractsManagerMigrator.MigrateInput({
            chainSystemConfigs: systemConfigs,
            disputeGameConfigs: gameConfigs,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(uint256(1))), l2SequenceNumber: 100 }),
            startingRespectedGameType: GameType.wrap(0)
        });

        input.set(input.migrateInput.selector, migrateInput);

        bytes memory storedInput = input.migrateInput();
        assertEq(storedInput, abi.encode(migrateInput));
    }

    function test_setAddress_withZeroAddress_reverts() public {
        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.prank.selector, address(0));

        vm.expectRevert("InteropMigrationInput: cannot set zero address");
        input.set(input.opcm.selector, address(0));
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("InteropMigrationInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));
    }
}

contract MockOPCM {
    event MigrateV1Called(address indexed sysCfgProxy, bytes32 indexed cannonPrestate);
    event MigrateV2Called(address indexed sysCfg, uint32 indexed gameType);

    bool public opcmV2Enabled;

    constructor(bool _opcmV2Enabled) {
        opcmV2Enabled = _opcmV2Enabled;
    }

    function version() public view returns (string memory) {
        return opcmV2Enabled ? "7.0.0" : "6.0.0";
    }

    function migrate(IOPContractsManagerInteropMigrator.MigrateInput memory _input) public {
        emit MigrateV1Called(
            address(_input.opChainConfigs[0].systemConfigProxy), Claim.unwrap(_input.opChainConfigs[0].cannonPrestate)
        );
    }

    function migrate(IOPContractsManagerMigrator.MigrateInput memory _input) public {
        emit MigrateV2Called(address(_input.chainSystemConfigs[0]), GameType.unwrap(_input.startingRespectedGameType));
    }
}

contract MockOPCMRevert {
    bool public opcmV2Enabled;

    constructor(bool _opcmV2Enabled) {
        opcmV2Enabled = _opcmV2Enabled;
    }

    function version() public view returns (string memory) {
        return opcmV2Enabled ? "7.0.0" : "6.0.0";
    }

    function migrate(IOPContractsManagerInteropMigrator.MigrateInput memory /*_input*/ ) public pure {
        revert("MockOPCMRevert: revert migrate");
    }

    function migrate(IOPContractsManagerMigrator.MigrateInput memory /*_input*/ ) public pure {
        revert("MockOPCMRevert: revert migrate");
    }
}

contract InteropMigrationV1_Test is Test {
    MockOPCM mockOPCM;
    MockOPCMRevert mockOPCMRevert;
    InteropMigrationInput input;
    IOPContractsManager.OpChainConfig config;
    InteropMigration migration;
    address prank;

    event MigrateV1Called(address indexed sysCfgProxy, bytes32 indexed cannonPrestate);

    function setUp() public {
        // Create mock OPCM with V2 disabled
        mockOPCM = new MockOPCM(false);
        input = new InteropMigrationInput();
        input.set(input.opcm.selector, address(mockOPCM));

        // Setup V1 migration input
        address systemConfig = makeAddr("systemConfigProxy");
        vm.etch(systemConfig, hex"01");

        config = IOPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig),
            cannonPrestate: Claim.wrap(keccak256("cannonPrestate")),
            cannonKonaPrestate: Claim.wrap(keccak256("cannonKonaPrestate"))
        });

        IOPContractsManager.OpChainConfig[] memory configs = new IOPContractsManager.OpChainConfig[](1);
        configs[0] = config;

        IOPContractsManagerInteropMigrator.MigrateInput memory migrateInput = IOPContractsManagerInteropMigrator
            .MigrateInput({
            usePermissionlessGame: true,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(uint256(1))), l2SequenceNumber: 100 }),
            gameParameters: IOPContractsManagerInteropMigrator.GameParameters({
                proposer: makeAddr("proposer"),
                challenger: makeAddr("challenger"),
                maxGameDepth: 73,
                splitDepth: 30,
                initBond: 1 ether,
                clockExtension: Duration.wrap(10800),
                maxClockDuration: Duration.wrap(302400)
            }),
            opChainConfigs: configs
        });

        input.set(input.migrateInput.selector, migrateInput);

        prank = makeAddr("prank");
        input.set(input.prank.selector, prank);

        migration = new InteropMigration();
    }

    function test_migrateV1_succeeds() public {
        // MigrateV1Called should be emitted by the prank since it's a delegatecall.
        vm.expectEmit(address(prank));
        emit MigrateV1Called(address(config.systemConfigProxy), Claim.unwrap(config.cannonPrestate));

        // mocks for post-migration checks
        address portal = makeAddr("optimismPortal");
        address dgf = makeAddr("disputeGameFactory");
        vm.mockCall(
            address(config.systemConfigProxy), abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal)
        );
        vm.etch(dgf, hex"01");
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal.disputeGameFactory, ()), abi.encode(dgf));

        InteropMigrationOutput output = new InteropMigrationOutput();
        migration.run(input, output);

        assertEq(address(output.disputeGameFactory()), dgf);
    }

    function test_migrateV1_migrate_reverts() public {
        // Set mock OPCM to revert on migrate
        mockOPCMRevert = new MockOPCMRevert(false);
        input.set(input.opcm.selector, address(mockOPCMRevert));

        InteropMigrationOutput output = new InteropMigrationOutput();
        vm.expectRevert("MockOPCMRevert: revert migrate");
        migration.run(input, output);
    }

    function test_opcmV1_withNoCode_reverts() public {
        // Set an address with no code as OPCM
        address emptyOPCM = makeAddr("emptyOPCM");
        input.set(input.opcm.selector, emptyOPCM);

        InteropMigrationOutput output = new InteropMigrationOutput();

        vm.expectRevert("InteropMigration: OPCM address has no code");
        migration.run(input, output);
    }
}

contract InteropMigrationV2_Test is Test {
    MockOPCM mockOPCM;
    MockOPCMRevert mockOPCMRevert;
    InteropMigrationInput input;
    ISystemConfig systemConfig;
    InteropMigration migration;
    address prank;

    event MigrateV2Called(address indexed sysCfg, uint32 indexed gameType);

    function setUp() public {
        // Create mock OPCM with V2 enabled
        mockOPCM = new MockOPCM(true);
        input = new InteropMigrationInput();
        input.set(input.opcm.selector, address(mockOPCM));

        // Setup V2 migration input
        address systemConfigAddr = makeAddr("systemConfig");
        vm.etch(systemConfigAddr, hex"01");
        systemConfig = ISystemConfig(systemConfigAddr);

        ISystemConfig[] memory systemConfigs = new ISystemConfig[](1);
        systemConfigs[0] = systemConfig;

        IOPContractsManagerUtils.DisputeGameConfig[] memory gameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        gameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 1 ether,
            gameType: GameType.wrap(0),
            gameArgs: abi.encodePacked(bytes32(uint256(0xabc)))
        });

        IOPContractsManagerMigrator.MigrateInput memory migrateInput = IOPContractsManagerMigrator.MigrateInput({
            chainSystemConfigs: systemConfigs,
            disputeGameConfigs: gameConfigs,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(uint256(1))), l2SequenceNumber: 100 }),
            startingRespectedGameType: GameType.wrap(0)
        });

        input.set(input.migrateInput.selector, migrateInput);

        prank = makeAddr("prank");
        input.set(input.prank.selector, prank);

        migration = new InteropMigration();
    }

    function test_migrateV2_succeeds() public {
        // MigrateV2Called should be emitted by the prank since it's a delegatecall.
        vm.expectEmit(address(prank));
        emit MigrateV2Called(address(systemConfig), 0);

        // mocks for post-migration checks
        address portal = makeAddr("optimismPortal");
        address dgf = makeAddr("disputeGameFactory");
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.optimismPortal, ()), abi.encode(portal));
        vm.etch(dgf, hex"01");
        vm.mockCall(portal, abi.encodeCall(IOptimismPortal.disputeGameFactory, ()), abi.encode(dgf));

        InteropMigrationOutput output = new InteropMigrationOutput();
        migration.run(input, output);

        assertEq(address(output.disputeGameFactory()), dgf);
    }

    function test_migrateV2_migrate_reverts() public {
        // Set mock OPCM with V2 enabled to revert on migrate
        mockOPCMRevert = new MockOPCMRevert(true);
        input.set(input.opcm.selector, address(mockOPCMRevert));

        InteropMigrationOutput output = new InteropMigrationOutput();
        vm.expectRevert("MockOPCMRevert: revert migrate");
        migration.run(input, output);
    }

    function test_opcmv2_withNoCode_reverts() public {
        // Set an address with no code as OPCM
        address emptyOPCM = makeAddr("emptyOPCM");
        input.set(input.opcm.selector, emptyOPCM);

        InteropMigrationOutput output = new InteropMigrationOutput();

        vm.expectRevert("InteropMigration: OPCM address has no code");
        migration.run(input, output);
    }
}
