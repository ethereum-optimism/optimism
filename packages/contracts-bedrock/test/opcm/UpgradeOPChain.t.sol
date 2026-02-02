// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Scripts
import { UpgradeOPChain, UpgradeOPChainInput } from "scripts/deploy/UpgradeOPChain.s.sol";

// Contracts
import { OPContractsManager } from "src/L1/OPContractsManager.sol";
import { OPContractsManagerV2 } from "src/L1/opcm/OPContractsManagerV2.sol";
import { UpgradeOPChain, UpgradeOPChainInput } from "scripts/deploy/UpgradeOPChain.s.sol";

// Libraries
import { Claim } from "src/dispute/lib/Types.sol";
import { GameType } from "src/dispute/lib/LibUDT.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

contract UpgradeOPChainInput_Test is Test {
    UpgradeOPChainInput input;
    MockOPCMV1 _mockOPCM;

    function setUp() public {
        input = new UpgradeOPChainInput();
        _mockOPCM = new MockOPCMV1();
        input.set(input.opcm.selector, address(_mockOPCM));
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when the upgrade input is not
    /// completely set.
    function test_getters_whenNotSet_reverts() public {
        UpgradeOPChainInput freshInput = new UpgradeOPChainInput();

        vm.expectRevert("UpgradeOPCMInput: prank not set");
        freshInput.prank();

        vm.expectRevert("UpgradeOPCMInput: not set");
        freshInput.opcm();

        vm.expectRevert("UpgradeOPCMInput: not set");
        freshInput.upgradeInput();
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly sets the upgrade input with
    /// the address type.
    function testFuzz_setAddress_succeeds(address mockPrank, address mockOPCM) public {
        vm.assume(mockPrank != address(0));
        vm.assume(mockOPCM != address(0));

        UpgradeOPChainInput freshInput = new UpgradeOPChainInput();
        freshInput.set(freshInput.prank.selector, mockPrank);
        freshInput.set(freshInput.opcm.selector, mockOPCM);

        assertEq(freshInput.prank(), mockPrank);
        assertEq(freshInput.opcm(), mockOPCM);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly sets the upgrade input with
    /// the OPContractsManager.OpChainConfig[] type.
    function testFuzz_setOpChainConfigs_succeeds(
        address systemConfig1,
        address systemConfig2,
        bytes32 prestate1,
        bytes32 konaPrestate1,
        bytes32 prestate2,
        bytes32 konaPrestate2
    )
        public
    {
        // Assume non-zero addresses for system configs
        vm.assume(systemConfig1 != address(0));
        vm.assume(systemConfig2 != address(0));
        // Assume not precompiles for system configs
        assumeNotPrecompile(systemConfig1);
        assumeNotPrecompile(systemConfig2);
        // Ensure system configs don't collide with test contracts
        vm.assume(systemConfig1 != address(input));
        vm.assume(systemConfig1 != address(_mockOPCM));
        vm.assume(systemConfig2 != address(input));
        vm.assume(systemConfig2 != address(_mockOPCM));

        // Create sample OpChainConfig array
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](2);

        // Setup mock addresses and contracts for first config
        vm.etch(systemConfig1, hex"01");

        configs[0] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig1),
            cannonPrestate: Claim.wrap(prestate1),
            cannonKonaPrestate: Claim.wrap(konaPrestate1)
        });

        // Setup mock addresses and contracts for second config
        vm.etch(systemConfig2, hex"01");

        configs[1] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig2),
            cannonPrestate: Claim.wrap(prestate2),
            cannonKonaPrestate: Claim.wrap(konaPrestate2)
        });

        input.set(input.upgradeInput.selector, configs);

        bytes memory storedConfigs = input.upgradeInput();
        assertEq(storedConfigs, abi.encode(configs));

        // Additional verification of stored claims if needed
        OPContractsManager.OpChainConfig[] memory decodedConfigs =
            abi.decode(storedConfigs, (OPContractsManager.OpChainConfig[]));
        assertEq(Claim.unwrap(decodedConfigs[0].cannonPrestate), prestate1);
        assertEq(Claim.unwrap(decodedConfigs[1].cannonPrestate), prestate2);
        assertEq(Claim.unwrap(decodedConfigs[0].cannonKonaPrestate), konaPrestate1);
        assertEq(Claim.unwrap(decodedConfigs[1].cannonKonaPrestate), konaPrestate2);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// a zero address.
    function test_setAddress_withZeroAddress_reverts() public {
        UpgradeOPChainInput freshInput = new UpgradeOPChainInput();

        vm.expectRevert("UpgradeOPCMInput: cannot set zero address");
        freshInput.set(freshInput.prank.selector, address(0));

        vm.expectRevert("UpgradeOPCMInput: cannot set zero address");
        freshInput.set(freshInput.opcm.selector, address(0));
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// an empty array.
    function test_setOpChainConfigs_withEmptyArray_reverts() public {
        OPContractsManager.OpChainConfig[] memory emptyConfigs = new OPContractsManager.OpChainConfig[](0);

        vm.expectRevert("UpgradeOPCMInput: cannot set empty array");
        input.set(input.upgradeInput.selector, emptyConfigs);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// an invalid selector.
    function testFuzz_set_withInvalidSelector_reverts(bytes4 invalidSelector, address testAddr) public {
        // Assume the selector is not one of the valid selectors
        vm.assume(invalidSelector != input.prank.selector);
        vm.assume(invalidSelector != input.opcm.selector);
        vm.assume(invalidSelector != input.upgradeInput.selector);
        vm.assume(testAddr != address(0));

        vm.expectRevert("UpgradeOPCMInput: unknown selector");
        input.set(invalidSelector, testAddr);

        // Create a single config for testing invalid selector
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        address mockSystemConfig = makeAddr("systemConfig");
        vm.etch(mockSystemConfig, hex"01");

        configs[0] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(mockSystemConfig),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(2)))
        });

        vm.expectRevert("UpgradeOPCMInput: unknown selector");
        input.set(invalidSelector, configs);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// OPCM v2 input when OPCM v1 is enabled.
    function testFuzz_setUpgradeInputV2_onV1OPCM_reverts(
        address systemConfig,
        bool enabled,
        uint256 initBond,
        uint32 gameType
    )
        public
    {
        vm.assume(systemConfig != address(0));
        vm.assume(initBond > 0);

        // Try to set V2 input when V1 is enabled
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: enabled,
            initBond: initBond,
            gameType: GameType.wrap(gameType),
            gameArgs: abi.encode("test")
        });

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(systemConfig),
            disputeGameConfigs: disputeGameConfigs,
            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
        });

        vm.expectRevert("UpgradeOPCMInput: cannot set OPCM v2 upgrade input when OPCM v1 is enabled");
        input.set(input.upgradeInput.selector, upgradeInput);
    }
}

contract UpgradeOPChainInput_TestV2 is Test {
    UpgradeOPChainInput input;
    MockOPCMV2 mockOPCM;

    function setUp() public {
        input = new UpgradeOPChainInput();
        mockOPCM = new MockOPCMV2();
        input.set(input.opcm.selector, address(mockOPCM));
    }

    /// @notice Tests that the upgrade input can be set using the OPContractsManagerV2.UpgradeInput type.
    function testFuzz_setUpgradeInputV2_succeeds(
        address systemConfig,
        bool enabled,
        uint256 initBond,
        uint32 gameType,
        bytes memory gameArgs,
        string memory extraKey,
        bytes memory extraData
    )
        public
    {
        // Assume non-zero address for system config
        vm.assume(systemConfig != address(0));
        vm.assume(initBond > 0);

        // Create sample UpgradeInputV2
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: enabled,
            initBond: initBond,
            gameType: GameType.wrap(gameType),
            gameArgs: gameArgs
        });

        IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({ key: extraKey, data: extraData });

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(systemConfig),
            disputeGameConfigs: disputeGameConfigs,
            extraInstructions: extraInstructions
        });

        input.set(input.upgradeInput.selector, upgradeInput);

        bytes memory storedUpgradeInput = input.upgradeInput();
        assertEq(storedUpgradeInput, abi.encode(upgradeInput));

        // Additional verification of stored values if needed
        OPContractsManagerV2.UpgradeInput memory decodedUpgradeInput =
            abi.decode(storedUpgradeInput, (OPContractsManagerV2.UpgradeInput));
        // Check system config matches
        assertEq(address(decodedUpgradeInput.systemConfig), address(upgradeInput.systemConfig));
        // Check dispute game configs match
        assertEq(decodedUpgradeInput.disputeGameConfigs.length, disputeGameConfigs.length);
        assertEq(decodedUpgradeInput.disputeGameConfigs[0].enabled, enabled);
        assertEq(decodedUpgradeInput.disputeGameConfigs[0].initBond, initBond);
        assertEq(GameType.unwrap(decodedUpgradeInput.disputeGameConfigs[0].gameType), gameType);
        assertEq(keccak256(decodedUpgradeInput.disputeGameConfigs[0].gameArgs), keccak256(gameArgs));
        // Check extra instructions match
        assertEq(decodedUpgradeInput.extraInstructions.length, extraInstructions.length);
        assertEq(decodedUpgradeInput.extraInstructions[0].key, extraKey);
        assertEq(keccak256(decodedUpgradeInput.extraInstructions[0].data), keccak256(extraData));
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// a zero system config.
    function testFuzz_setUpgradeInputV2_withZeroSystemConfig_reverts() public {
        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(address(0)),
            disputeGameConfigs: new IOPContractsManagerUtils.DisputeGameConfig[](1),
            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
        });

        vm.expectRevert("UpgradeOPCMInput: cannot set zero address");
        input.set(input.upgradeInput.selector, upgradeInput);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// an empty dispute game configs array.
    function testFuzz_setUpgradeInputV2_withEmptyDisputeGameConfigs_reverts(address systemConfig) public {
        vm.assume(systemConfig != address(0));

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(systemConfig),
            disputeGameConfigs: new IOPContractsManagerUtils.DisputeGameConfig[](0),
            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
        });

        vm.expectRevert("UpgradeOPCMInput: cannot set empty dispute game configs array");
        input.set(input.upgradeInput.selector, upgradeInput);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when setting the upgrade input with
    /// OPCM v1 input when OPCM v2 is enabled.
    function testFuzz_setUpgradeInputV1_onV2OPCM_reverts(
        address systemConfigProxy,
        bytes32 cannonPrestate,
        bytes32 cannonKonaPrestate
    )
        public
    {
        vm.assume(systemConfigProxy != address(0));

        // Try to set V1 input when V2 is enabled
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        configs[0] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfigProxy),
            cannonPrestate: Claim.wrap(cannonPrestate),
            cannonKonaPrestate: Claim.wrap(cannonKonaPrestate)
        });

        vm.expectRevert("UpgradeOPCMInput: cannot set OPCM v1 upgrade input when OPCM v2 is enabled");
        input.set(input.upgradeInput.selector, configs);
    }
}

contract MockOPCMV1 {
    event UpgradeCalled(
        address indexed sysCfgProxy, bytes32 indexed absolutePrestate, bytes32 indexed cannonKonaPrestate
    );

    function version() public pure returns (string memory) {
        return "6.0.0";
    }

    function upgrade(OPContractsManager.OpChainConfig[] memory _opChainConfigs) public {
        emit UpgradeCalled(
            address(_opChainConfigs[0].systemConfigProxy),
            Claim.unwrap(_opChainConfigs[0].cannonPrestate),
            Claim.unwrap(_opChainConfigs[0].cannonKonaPrestate)
        );
    }
}

contract MockOPCMV2 {
    event UpgradeCalled(
        address indexed systemConfig,
        IOPContractsManagerUtils.DisputeGameConfig[] indexed disputeGameConfigs,
        IOPContractsManagerUtils.ExtraInstruction[] indexed extraInstructions
    );

    function version() public pure returns (string memory) {
        return Constants.OPCM_V2_MIN_VERSION;
    }

    function upgrade(OPContractsManagerV2.UpgradeInput memory _upgradeInput) public {
        emit UpgradeCalled(
            address(_upgradeInput.systemConfig), _upgradeInput.disputeGameConfigs, _upgradeInput.extraInstructions
        );
    }
}

contract UpgradeOPChain_Test is Test {
    MockOPCMV1 mockOPCM;
    UpgradeOPChainInput uoci;
    OPContractsManager.OpChainConfig config;
    UpgradeOPChain upgradeOPChain;
    address prank;

    event UpgradeCalled(
        address indexed sysCfgProxy, bytes32 indexed absolutePrestate, bytes32 indexed cannonKonaPrestate
    );

    function setUp() public {
        mockOPCM = new MockOPCMV1();
        uoci = new UpgradeOPChainInput();
        uoci.set(uoci.opcm.selector, address(mockOPCM));
        prank = makeAddr("prank");
        uoci.set(uoci.prank.selector, prank);
        upgradeOPChain = new UpgradeOPChain();
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly encodes and passes down the upgrade input
    /// arguments to the OPCM contract's upgrade function.
    /// @dev It does not test the actual upgrade functionality.
    function testFuzz_upgrade_succeeds(
        address systemConfigProxy,
        bytes32 cannonPrestate,
        bytes32 cannonKonaPrestate
    )
        public
    {
        vm.assume(systemConfigProxy != address(0));

        config = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfigProxy),
            cannonPrestate: Claim.wrap(cannonPrestate),
            cannonKonaPrestate: Claim.wrap(cannonKonaPrestate)
        });
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        configs[0] = config;
        uoci.set(uoci.upgradeInput.selector, configs);

        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(
            address(config.systemConfigProxy),
            Claim.unwrap(config.cannonPrestate),
            Claim.unwrap(config.cannonKonaPrestate)
        );
        upgradeOPChain.run(uoci);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when the OPCM upgrade
    /// call fails
    function test_upgrade_whenOPCMReverts_reverts() public {
        address systemConfigProxy = makeAddr("systemConfig");
        config = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfigProxy),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(2)))
        });
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        configs[0] = config;
        uoci.set(uoci.upgradeInput.selector, configs);

        vm.mockCallRevert(prank, OPContractsManager.upgrade.selector, abi.encode("UpgradeOPChain: upgrade failed"));

        vm.expectRevert("UpgradeOPChain: upgrade failed");
        upgradeOPChain.run(uoci);
    }
}

contract UpgradeOPChain_TestV2 is Test {
    MockOPCMV2 mockOPCM;
    UpgradeOPChainInput uoci;
    UpgradeOPChain upgradeOPChain;
    address prank;

    event UpgradeCalled(
        address indexed systemConfig,
        IOPContractsManagerUtils.DisputeGameConfig[] indexed disputeGameConfigs,
        IOPContractsManagerUtils.ExtraInstruction[] indexed extraInstructions
    );

    function setUp() public {
        mockOPCM = new MockOPCMV2();
        uoci = new UpgradeOPChainInput();
        uoci.set(uoci.opcm.selector, address(mockOPCM));

        prank = makeAddr("prank");
        uoci.set(uoci.prank.selector, prank);
        upgradeOPChain = new UpgradeOPChain();
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly encodes and passes down the upgrade input
    /// arguments to the OPCM contract's upgrade function.
    /// @dev It does not test the actual upgrade functionality.
    function testFuzz_upgrade_succeeds(
        address systemConfig,
        bool enabled,
        uint256 initBond,
        uint32 gameType,
        bytes memory gameArgs
    )
        public
    {
        vm.assume(systemConfig != address(0));

        // NOTE: Setting the upgrade input here to avoid `Copying of type struct
        // IOPContractsManagerUtils.DisputeGameConfig memory[] memory to storage
        // not yet supported.` error.
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: enabled,
            initBond: initBond,
            gameType: GameType.wrap(gameType),
            gameArgs: gameArgs
        });

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(systemConfig),
            disputeGameConfigs: disputeGameConfigs,
            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
        });
        uoci.set(uoci.upgradeInput.selector, upgradeInput);

        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(
            address(upgradeInput.systemConfig), upgradeInput.disputeGameConfigs, upgradeInput.extraInstructions
        );
        upgradeOPChain.run(uoci);
    }

    /// @notice This test verifies that the UpgradeOPChain script correctly reverts when the OPCM v2 upgrade
    /// call fails.
    function test_upgrade_whenOPCMV2Reverts_reverts() public {
        address systemConfig = makeAddr("systemConfig");
        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 1 ether,
            gameType: GameType.wrap(0),
            gameArgs: abi.encode("test")
        });

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(systemConfig),
            disputeGameConfigs: disputeGameConfigs,
            extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
        });
        uoci.set(uoci.upgradeInput.selector, upgradeInput);

        vm.mockCallRevert(prank, OPContractsManagerV2.upgrade.selector, abi.encode("UpgradeOPChain: upgrade failed"));

        vm.expectRevert("UpgradeOPChain: upgrade failed");
        upgradeOPChain.run(uoci);
    }
}
