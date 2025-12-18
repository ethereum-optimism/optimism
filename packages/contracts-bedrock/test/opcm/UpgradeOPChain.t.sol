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
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

contract UpgradeOPChainInput_Test is Test {
    UpgradeOPChainInput input;

    function setUp() public {
        input = new UpgradeOPChainInput();
    }

    function test_getters_whenNotSet_reverts() public {
        vm.expectRevert("UpgradeOPCMInput: prank not set");
        input.prank();

        vm.expectRevert("UpgradeOPCMInput: not set");
        input.opcm();

        vm.expectRevert("UpgradeOPCMInput: not set");
        input.upgradeInput();
    }

    function test_setAddress_succeeds() public {
        address mockPrank = makeAddr("prank");
        address mockOPCM = makeAddr("opcm");

        // Create mock contract at OPCM address
        vm.etch(mockOPCM, hex"01");

        input.set(input.prank.selector, mockPrank);
        input.set(input.opcm.selector, mockOPCM);

        assertEq(input.prank(), mockPrank);
        assertEq(input.opcm(), mockOPCM);
    }

    function test_setOpChainConfigs_succeeds() public {
        // Create sample OpChainConfig array
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](2);

        // Setup mock addresses and contracts for first config
        address systemConfig1 = makeAddr("systemConfig1");
        address proxyAdmin1 = makeAddr("proxyAdmin1");
        vm.etch(systemConfig1, hex"01");
        vm.etch(proxyAdmin1, hex"01");

        configs[0] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig1),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(2)))
        });

        // Setup mock addresses and contracts for second config
        address systemConfig2 = makeAddr("systemConfig2");
        address proxyAdmin2 = makeAddr("proxyAdmin2");
        vm.etch(systemConfig2, hex"01");
        vm.etch(proxyAdmin2, hex"01");

        configs[1] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(systemConfig2),
            cannonPrestate: Claim.wrap(bytes32(uint256(2))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(3)))
        });

        input.set(input.upgradeInput.selector, configs);

        bytes memory storedConfigs = input.upgradeInput();
        assertEq(storedConfigs, abi.encode(configs));

        // Additional verification of stored claims if needed
        OPContractsManager.OpChainConfig[] memory decodedConfigs =
            abi.decode(storedConfigs, (OPContractsManager.OpChainConfig[]));
        assertEq(Claim.unwrap(decodedConfigs[0].cannonPrestate), bytes32(uint256(1)));
        assertEq(Claim.unwrap(decodedConfigs[1].cannonPrestate), bytes32(uint256(2)));
    }

    /// @notice Tests that the upgrade input can be set using the OPContractsManagerV2.UpgradeInput type.
    function test_setUpgradeInputV2_succeeds() public {
        // Create sample UpgradeInputV2
        OPContractsManagerV2.DisputeGameConfig[] memory disputeGameConfigs =
            new OPContractsManagerV2.DisputeGameConfig[](1);
        disputeGameConfigs[0] = OPContractsManagerV2.DisputeGameConfig({
            enabled: true,
            initBond: 1000,
            gameType: GameType.wrap(1),
            gameArgs: abi.encode("test")
        });

        IOPContractsManagerUtils.ExtraInstruction[] memory extraInstructions =
            new IOPContractsManagerUtils.ExtraInstruction[](1);
        extraInstructions[0] = IOPContractsManagerUtils.ExtraInstruction({ key: "test", data: abi.encode("test") });

        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(makeAddr("systemConfig")),
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
        assertEq(decodedUpgradeInput.disputeGameConfigs[0].enabled, disputeGameConfigs[0].enabled);
        assertEq(decodedUpgradeInput.disputeGameConfigs[0].initBond, disputeGameConfigs[0].initBond);
        assertEq(
            GameType.unwrap(decodedUpgradeInput.disputeGameConfigs[0].gameType),
            GameType.unwrap(disputeGameConfigs[0].gameType)
        );
        assertEq(
            keccak256(decodedUpgradeInput.disputeGameConfigs[0].gameArgs), keccak256(disputeGameConfigs[0].gameArgs)
        );
        // Check extra instructions match
        assertEq(decodedUpgradeInput.extraInstructions.length, extraInstructions.length);
        assertEq(decodedUpgradeInput.extraInstructions[0].key, extraInstructions[0].key);
        assertEq(keccak256(decodedUpgradeInput.extraInstructions[0].data), keccak256(extraInstructions[0].data));
    }

    function test_setAddress_withZeroAddress_reverts() public {
        vm.expectRevert("UpgradeOPCMInput: cannot set zero address");
        input.set(input.prank.selector, address(0));

        vm.expectRevert("UpgradeOPCMInput: cannot set zero address");
        input.set(input.opcm.selector, address(0));
    }

    function test_setOpChainConfigs_withEmptyArray_reverts() public {
        OPContractsManager.OpChainConfig[] memory emptyConfigs = new OPContractsManager.OpChainConfig[](0);

        vm.expectRevert("UpgradeOPCMInput: cannot set empty array");
        input.set(input.upgradeInput.selector, emptyConfigs);
    }

    function test_set_withInvalidSelector_reverts() public {
        vm.expectRevert("UpgradeOPCMInput: unknown selector");
        input.set(bytes4(0xdeadbeef), makeAddr("test"));

        // Create a single config for testing invalid selector
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        address mockSystemConfig = makeAddr("systemConfig");
        address mockProxyAdmin = makeAddr("proxyAdmin");
        vm.etch(mockSystemConfig, hex"01");
        vm.etch(mockProxyAdmin, hex"01");

        configs[0] = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(mockSystemConfig),
            cannonPrestate: Claim.wrap(bytes32(uint256(1))),
            cannonKonaPrestate: Claim.wrap(bytes32(uint256(2)))
        });

        vm.expectRevert("UpgradeOPCMInput: unknown selector");
        input.set(bytes4(0xdeadbeef), configs);
    }
}

contract MockOPCMV1 {
    event UpgradeCalled(
        address indexed sysCfgProxy, bytes32 indexed absolutePrestate, bytes32 indexed cannonKonaPrestate
    );

    function isDevFeatureEnabled(bytes32 /* _feature */ ) public pure returns (bool) {
        return false;
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
        OPContractsManagerV2.DisputeGameConfig[] indexed disputeGameConfigs,
        IOPContractsManagerUtils.ExtraInstruction[] indexed extraInstructions
    );

    function isDevFeatureEnabled(bytes32 _feature) public pure returns (bool) {
        return _feature == DevFeatures.OPCM_V2;
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

    function setUp() public virtual {
        mockOPCM = new MockOPCMV1();
        uoci = new UpgradeOPChainInput();
        uoci.set(uoci.opcm.selector, address(mockOPCM));
        config = OPContractsManager.OpChainConfig({
            systemConfigProxy: ISystemConfig(makeAddr("systemConfigProxy")),
            cannonPrestate: Claim.wrap(keccak256("cannonPrestate")),
            cannonKonaPrestate: Claim.wrap(keccak256("cannonKonaPrestate"))
        });
        OPContractsManager.OpChainConfig[] memory configs = new OPContractsManager.OpChainConfig[](1);
        configs[0] = config;
        uoci.set(uoci.upgradeInput.selector, configs);
        prank = makeAddr("prank");
        uoci.set(uoci.prank.selector, prank);
        upgradeOPChain = new UpgradeOPChain();
    }

    function test_upgrade_succeeds() public {
        // UpgradeCalled should be emitted by the prank since it's a delegate call.
        vm.expectEmit(address(prank));
        emit UpgradeCalled(
            address(config.systemConfigProxy),
            Claim.unwrap(config.cannonPrestate),
            Claim.unwrap(config.cannonKonaPrestate)
        );
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
        OPContractsManagerV2.DisputeGameConfig[] indexed disputeGameConfigs,
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

    function test_upgrade_succeeds() public {
        // NOTE: Setting the upgrade input here to avoid `Copying of type struct OPContractsManagerV2.DisputeGameConfig
        // memory[] memory to storage not yet supported.` error.
        OPContractsManagerV2.UpgradeInput memory upgradeInput = OPContractsManagerV2.UpgradeInput({
            systemConfig: ISystemConfig(makeAddr("systemConfig")),
            disputeGameConfigs: new OPContractsManagerV2.DisputeGameConfig[](1),
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
}
