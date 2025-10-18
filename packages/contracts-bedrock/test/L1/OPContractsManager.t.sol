// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test, stdStorage, StdStorage } from "forge-std/Test.sol";
import { VmSafe } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { FeatureFlags } from "test/setup/FeatureFlags.sol";
import { DeployOPChain_TestBase } from "test/opcm/DeployOPChain.t.sol";
import { DelegateCaller } from "test/mocks/Callers.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { Deploy } from "scripts/deploy/Deploy.s.sol";
import { VerifyOPCM } from "scripts/deploy/VerifyOPCM.s.sol";
import { DeployOPChain } from "scripts/deploy/DeployOPChain.s.sol";
import { Config } from "scripts/libraries/Config.sol";
import { Types } from "scripts/libraries/Types.sol";

// Libraries
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { GameType, Duration, Hash, Claim } from "src/dispute/lib/LibUDT.sol";
import { Proposal, GameTypes } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProtocolVersions } from "interfaces/L1/IProtocolVersions.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import {
    IOPContractsManager,
    IOPContractsManagerGameTypeAdder,
    IOPContractsManagerInteropMigrator,
    IOPContractsManagerUpgrader,
    IOPContractsManagerStandardValidator
} from "interfaces/L1/IOPContractsManager.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";

// Contracts
import {
    OPContractsManager,
    OPContractsManagerGameTypeAdder,
    OPContractsManagerDeployer,
    OPContractsManagerUpgrader,
    OPContractsManagerContractsContainer,
    OPContractsManagerInteropMigrator,
    OPContractsManagerStandardValidator
} from "src/L1/OPContractsManager.sol";

/// @title OPContractsManager_Harness
/// @notice Exposes internal functions for testing.
contract OPContractsManager_Harness is OPContractsManager {
    constructor(
        OPContractsManagerGameTypeAdder _opcmGameTypeAdder,
        OPContractsManagerDeployer _opcmDeployer,
        OPContractsManagerUpgrader _opcmUpgrader,
        OPContractsManagerInteropMigrator _opcmInteropMigrator,
        OPContractsManagerStandardValidator _opcmStandardValidator,
        ISuperchainConfig _superchainConfig,
        IProtocolVersions _protocolVersions
    )
        OPContractsManager(
            _opcmGameTypeAdder,
            _opcmDeployer,
            _opcmUpgrader,
            _opcmInteropMigrator,
            _opcmStandardValidator,
            _superchainConfig,
            _protocolVersions
        )
    { }

    function chainIdToBatchInboxAddress_exposed(uint256 l2ChainId) public view returns (address) {
        return super.chainIdToBatchInboxAddress(l2ChainId);
    }
}

/// @title OPContractsManager_Upgrade_Harness
/// @notice Exposes internal functions for testing.
contract OPContractsManager_Upgrade_Harness is CommonTest {
    // The Upgraded event emitted by the Proxy contract.
    event Upgraded(address indexed implementation);

    // The Upgraded event emitted by the OPContractsManager contract.
    event Upgraded(uint256 indexed l2ChainId, ISystemConfig indexed systemConfig, address indexed upgrader);

    // The AddressSet event emitted by the AddressManager contract.
    event AddressSet(string indexed name, address newAddress, address oldAddress);

    // The AdminChanged event emitted by the Proxy contract at init time or when the admin is
    // changed.
    event AdminChanged(address previousAdmin, address newAdmin);

    // The ImplementationSet event emitted by the DisputeGameFactory contract.
    event ImplementationSet(address indexed impl, GameType indexed gameType);

    /// @notice Thrown when testing with an unsupported chain ID.
    error UnsupportedChainId();

    uint256 l2ChainId;
    address upgrader;
    IOPContractsManager.OpChainConfig[] opChainConfigs;
    Claim absolutePrestate;
    string public opChain = Config.forkOpChain();

    function setUp() public virtual override {
        super.disableUpgradedFork();
        super.setUp();
        if (!isForkTest()) {
            // This test is only supported in forked tests, as we are testing the upgrade.
            vm.skip(true);
        }

        skipIfOpsRepoTest(
            "OPContractsManager_Upgrade_Harness: cannot test upgrade on superchain ops repo upgrade tests"
        );

        absolutePrestate = Claim.wrap(bytes32(keccak256("absolutePrestate")));
        upgrader = proxyAdmin.owner();
        vm.label(upgrader, "ProxyAdmin Owner");

        // Set the upgrader to be a DelegateCaller so we can test the upgrade
        vm.etch(upgrader, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        opChainConfigs.push(
            IOPContractsManager.OpChainConfig({
                systemConfigProxy: systemConfig,
                proxyAdmin: proxyAdmin,
                absolutePrestate: absolutePrestate
            })
        );

        // Retrieve the l2ChainId, which was read from the superchain-registry, and saved in
        // Artifacts encoded as an address.
        l2ChainId = uint256(uint160(address(artifacts.mustGetAddress("L2ChainId"))));

        delayedWETHPermissionedGameProxy =
            IDelayedWETH(payable(artifacts.mustGetAddress("PermissionedDelayedWETHProxy")));
        delayedWeth = IDelayedWETH(payable(artifacts.mustGetAddress("PermissionlessDelayedWETHProxy")));
        permissionedDisputeGame = IPermissionedDisputeGame(address(artifacts.mustGetAddress("PermissionedDisputeGame")));
        faultDisputeGame = IFaultDisputeGame(address(artifacts.mustGetAddress("FaultDisputeGame")));

        // Since this superchainConfig is already at the expected reinitializer version...
        // We do this to pass the reinitializer check when trying to upgrade the superchainConfig contract.

        // Get the value of the 0th storage slot of the superchainConfig contract.
        bytes32 slot0 = vm.load(address(superchainConfig), bytes32(0));
        // Remove the value of initialized slot.
        slot0 = slot0 & bytes32(~uint256(0xff));
        // Store 1 there.
        slot0 = bytes32(uint256(slot0) + 1);
        // Store the new value.
        vm.store(address(superchainConfig), bytes32(0), slot0);
    }

    /// @notice Helper function that runs an OPCM upgrade, asserts that the upgrade was successful,
    ///         asserts that it fits within a certain amount of gas, and runs the StandardValidator
    ///         over the result.
    /// @param _opcm The OPCM contract to upgrade with.
    /// @param _delegateCaller The address of the delegate caller to use for the upgrade.
    /// @param _revertBytes The bytes of the revert to expect.
    function _runOpcmUpgradeAndChecks(
        IOPContractsManager _opcm,
        address _delegateCaller,
        bytes memory _revertBytes
    )
        internal
    {
        // Always start by upgrading the SuperchainConfig contract.
        // Temporarily replace the superchainPAO with a DelegateCaller.
        address superchainPAO = IProxyAdmin(EIP1967Helper.getAdmin(address(superchainConfig))).owner();
        bytes memory superchainPAOCode = address(superchainPAO).code;
        vm.etch(superchainPAO, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        // Execute the SuperchainConfig upgrade.
        // nosemgrep: sol-safety-trycatch-eip150
        try DelegateCaller(superchainPAO).dcForward(
            address(_opcm),
            abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (superchainConfig, superchainProxyAdmin))
        ) {
            // Great, the upgrade succeeded.
        } catch (bytes memory reason) {
            // Only acceptable revert reason is the SuperchainConfig already being up to date. This
            // try/catch is better than checking the version via the implementations struct because
            // the implementations struct interface can change between OPCM versions which would
            // cause the test to break and be a pain to resolve.
            assertTrue(
                bytes4(reason)
                    == IOPContractsManagerUpgrader.OPContractsManagerUpgrader_SuperchainConfigAlreadyUpToDate.selector,
                "Revert reason other than SuperchainConfigAlreadyUpToDate"
            );
        }

        // Reset the superchainPAO to the original code.
        vm.etch(superchainPAO, superchainPAOCode);

        // Temporarily replace the upgrader with a DelegateCaller.
        bytes memory delegateCallerCode = address(_delegateCaller).code;
        vm.etch(_delegateCaller, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        // Expect the revert if one is specified.
        if (_revertBytes.length > 0) {
            vm.expectRevert(_revertBytes);
        }

        // Execute the chain upgrade.
        DelegateCaller(_delegateCaller).dcForward(
            address(_opcm), abi.encodeCall(IOPContractsManager.upgrade, (opChainConfigs))
        );

        // Return early if a revert was expected. Otherwise we'll get errors below.
        if (_revertBytes.length > 0) {
            return;
        }

        // Less than 90% of the gas target of 2**24 (EIP-7825) to account for the gas used by
        // using Safe.
        uint256 fusakaLimit = 2 ** 24;
        VmSafe.Gas memory gas = vm.lastCallGas();
        assertLt(gas.gasTotalUsed, fusakaLimit * 9 / 10, "Upgrade exceeds gas target of 90% of 2**24 (EIP-7825)");

        // Reset the upgrader to the original code.
        vm.etch(_delegateCaller, delegateCallerCode);

        // We expect there to only be one chain config for these tests, you will have to rework
        // this test if you add more.
        assertEq(opChainConfigs.length, 1);

        // Coverage changes bytecode, so we get various errors. We can safely ignore the result of
        // the standard validator in the coverage case, if the validator is failing in coverage
        // then it will also fail in other CI tests (unless it's the expected issues, in which case
        // we can safely skip).
        if (vm.isContext(VmSafe.ForgeContext.Coverage)) {
            return;
        }

        // Create validationOverrides
        IOPContractsManagerStandardValidator.ValidationOverrides memory validationOverrides =
        IOPContractsManagerStandardValidator.ValidationOverrides({
            l1PAOMultisig: opChainConfigs[0].proxyAdmin.owner(),
            challenger: IPermissionedDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON)))
                .challenger()
        });

        // Grab the validator before we do the error assertion because otherwise the assertion will
        // try to apply to this function call instead.
        IOPContractsManagerStandardValidator validator = _opcm.opcmStandardValidator();

        // If the absolute prestate is zero, we will always get a PDDG-40,PLDG-40 error here in the
        // standard validator. This happens because an absolute prestate of zero means that the
        // user is requesting to use the existing prestate. We could avoid the error by grabbing
        // the prestate from the actual contracts, but that doesn't actually give us any valuable
        // checks. Easier to just expect the error in this case.
        // We add the prefix of OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER because we use validationOverrides.
        if (opChainConfigs[0].absolutePrestate.raw() == bytes32(0)) {
            vm.expectRevert(
                "OPContractsManagerStandardValidator: OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,PDDG-40,PLDG-40"
            );
        }

        // Run the StandardValidator checks.
        validator.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInput({
                proxyAdmin: opChainConfigs[0].proxyAdmin,
                sysCfg: opChainConfigs[0].systemConfigProxy,
                absolutePrestate: opChainConfigs[0].absolutePrestate.raw(),
                l2ChainID: l2ChainId
            }),
            false,
            validationOverrides
        );
    }

    /// @notice Executes all past upgrades that have not yet been executed on mainnet as of the
    ///         current simulation block defined in the justfile for this package. This function
    ///         might be empty if there are no previous upgrades to execute. You should remove
    ///         upgrades from this function once they've been executed on mainnet and the
    ///         simulation block has been bumped beyond the execution block.
    /// @param _delegateCaller The address of the delegate caller to use for the upgrade.
    function runPastUpgrades(address _delegateCaller) internal view {
        // Run past upgrades depending on network.
        if (block.chainid == 1) {
            // Mainnet
            // This is empty because the block number in the justfile is after the most recent upgrade so there are no
            // past upgrades to run.
            _delegateCaller;
        } else {
            revert UnsupportedChainId();
        }
    }

    /// @notice Executes the current upgrade and checks the results.
    /// @param _delegateCaller The address of the delegate caller to use for the upgrade.
    function runCurrentUpgrade(address _delegateCaller) public {
        _runOpcmUpgradeAndChecks(opcm, _delegateCaller, bytes(""));
    }

    /// @notice Executes the current upgrade and expects reverts.
    /// @param _delegateCaller The address of the delegate caller to use for the upgrade.
    /// @param _revertBytes The bytes of the revert to expect.
    function runCurrentUpgrade(address _delegateCaller, bytes memory _revertBytes) public {
        _runOpcmUpgradeAndChecks(opcm, _delegateCaller, _revertBytes);
    }
}

/// @title OPContractsManager_TestInit
/// @notice Reusable test initialization for `OPContractsManager` tests.
abstract contract OPContractsManager_TestInit is CommonTest {
    event GameTypeAdded(
        uint256 indexed l2ChainId, GameType indexed gameType, IDisputeGame newDisputeGame, IDisputeGame oldDisputeGame
    );

    IOPContractsManager.DeployOutput internal chainDeployOutput1;
    IOPContractsManager.DeployOutput internal chainDeployOutput2;

    function setUp() public virtual override {
        super.setUp();

        chainDeployOutput1 = createChainContracts(100);
        chainDeployOutput2 = createChainContracts(101);

        vm.deal(address(chainDeployOutput1.ethLockboxProxy), 100 ether);
        vm.deal(address(chainDeployOutput2.ethLockboxProxy), 100 ether);
    }

    /// @notice Sets up the environment variables for the VerifyOPCM test.
    function setupEnvVars() public {
        vm.setEnv("EXPECTED_SUPERCHAIN_CONFIG", vm.toString(address(opcm.superchainConfig())));
        vm.setEnv("EXPECTED_PROTOCOL_VERSIONS", vm.toString(address(opcm.protocolVersions())));
    }

    /// @notice Helper function to deploy a new set of L1 contracts via OPCM.
    /// @param _l2ChainId The L2 chain ID to deploy the contracts for.
    /// @return The deployed contracts.
    function createChainContracts(uint256 _l2ChainId) internal returns (IOPContractsManager.DeployOutput memory) {
        return opcm.deploy(
            IOPContractsManager.DeployInput({
                roles: IOPContractsManager.Roles({
                    opChainProxyAdminOwner: address(this),
                    systemConfigOwner: address(this),
                    batcher: address(this),
                    unsafeBlockSigner: address(this),
                    proposer: address(this),
                    challenger: address(this)
                }),
                basefeeScalar: 1,
                blobBasefeeScalar: 1,
                startingAnchorRoot: abi.encode(
                    Proposal({
                        root: Hash.wrap(0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef),
                        l2SequenceNumber: 0
                    })
                ),
                l2ChainId: _l2ChainId,
                saltMixer: "hello",
                gasLimit: 30_000_000,
                disputeGameType: GameType.wrap(1),
                disputeAbsolutePrestate: Claim.wrap(
                    bytes32(hex"038512e02c4c3f7bdaec27d00edf55b7155e0905301e1a88083e4e0a6764d54c")
                ),
                disputeMaxGameDepth: 73,
                disputeSplitDepth: 30,
                disputeClockExtension: Duration.wrap(10800),
                disputeMaxClockDuration: Duration.wrap(302400)
            })
        );
    }

    function addGameType(IOPContractsManager.AddGameInput memory input)
        internal
        returns (IOPContractsManager.AddGameOutput memory)
    {
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](1);
        inputs[0] = input;

        uint256 l2ChainId = IFaultDisputeGame(
            address(IDisputeGameFactory(input.systemConfig.disputeGameFactory()).gameImpls(GameType.wrap(1)))
        ).l2ChainId();

        // Expect the GameTypeAdded event to be emitted.
        vm.expectEmit(true, true, true, false, address(this));
        emit GameTypeAdded(
            l2ChainId, input.disputeGameType, IDisputeGame(payable(address(0))), IDisputeGame(payable(address(0)))
        );
        (bool success, bytes memory rawGameOut) =
            address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertTrue(success, "addGameType failed");

        IOPContractsManager.AddGameOutput[] memory addGameOutAll =
            abi.decode(rawGameOut, (IOPContractsManager.AddGameOutput[]));
        return addGameOutAll[0];
    }

    function newGameInputFactory(GameType _gameType) internal view returns (IOPContractsManager.AddGameInput memory) {
        return IOPContractsManager.AddGameInput({
            saltMixer: "hello",
            systemConfig: chainDeployOutput1.systemConfigProxy,
            proxyAdmin: chainDeployOutput1.opChainProxyAdmin,
            delayedWETH: IDelayedWETH(payable(address(0))),
            disputeGameType: _gameType,
            disputeAbsolutePrestate: Claim.wrap(bytes32(hex"deadbeef1234")),
            disputeMaxGameDepth: 73,
            disputeSplitDepth: 30,
            disputeClockExtension: Duration.wrap(10800),
            disputeMaxClockDuration: Duration.wrap(302400),
            initialBond: 1 ether,
            vm: IBigStepper(address(opcm.implementations().mipsImpl)),
            permissioned: _gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()
                || _gameType.raw() == GameTypes.SUPER_PERMISSIONED_CANNON.raw()
        });
    }
}

/// @title OPContractsManager_ChainIdToBatchInboxAddress_Test
/// @notice Tests the `chainIdToBatchInboxAddress` function of the `OPContractsManager` contract.
/// @dev These tests use the harness which exposes internal functions for testing.
contract OPContractsManager_ChainIdToBatchInboxAddress_Test is Test, FeatureFlags {
    OPContractsManager_Harness opcmHarness;
    address challenger = makeAddr("challenger");

    function setUp() public {
        ISuperchainConfig superchainConfigProxy = ISuperchainConfig(makeAddr("superchainConfig"));
        IProtocolVersions protocolVersionsProxy = IProtocolVersions(makeAddr("protocolVersions"));
        IProxyAdmin superchainProxyAdmin = IProxyAdmin(makeAddr("superchainProxyAdmin"));
        OPContractsManager.Blueprints memory emptyBlueprints;
        OPContractsManager.Implementations memory emptyImpls;
        vm.etch(address(superchainConfigProxy), hex"01");
        vm.etch(address(protocolVersionsProxy), hex"01");

        resolveFeaturesFromEnv();
        OPContractsManagerContractsContainer container =
            new OPContractsManagerContractsContainer(emptyBlueprints, emptyImpls, devFeatureBitmap);

        OPContractsManager.Implementations memory __opcmImplementations = container.implementations();
        OPContractsManagerStandardValidator.Implementations memory opcmImplementations;
        assembly {
            opcmImplementations := __opcmImplementations
        }

        opcmHarness = new OPContractsManager_Harness({
            _opcmGameTypeAdder: new OPContractsManagerGameTypeAdder(container),
            _opcmDeployer: new OPContractsManagerDeployer(container),
            _opcmUpgrader: new OPContractsManagerUpgrader(container),
            _opcmInteropMigrator: new OPContractsManagerInteropMigrator(container),
            _opcmStandardValidator: new OPContractsManagerStandardValidator(
                opcmImplementations, superchainConfigProxy, address(superchainProxyAdmin), challenger, 100, bytes32(0)
            ),
            _superchainConfig: superchainConfigProxy,
            _protocolVersions: protocolVersionsProxy
        });
    }

    function test_calculatesBatchInboxAddress_succeeds() public view {
        // These test vectors were calculated manually:
        //   1. Compute the bytes32 encoding of the chainId: bytes32(uint256(chainId));
        //   2. Hash it and manually take the first 19 bytes, and prefixed it with 0x00.
        uint256 chainId = 1234;
        address expected = 0x0017FA14b0d73Aa6A26D6b8720c1c84b50984f5C;
        address actual = opcmHarness.chainIdToBatchInboxAddress_exposed(chainId);
        vm.assertEq(expected, actual);

        chainId = type(uint256).max;
        expected = 0x00a9C584056064687E149968cBaB758a3376D22A;
        actual = opcmHarness.chainIdToBatchInboxAddress_exposed(chainId);
        vm.assertEq(expected, actual);
    }
}

/// @title OPContractsManager_AddGameType_Test
/// @notice Tests the `addGameType` function of the `OPContractsManager` contract.
contract OPContractsManager_AddGameType_Test is OPContractsManager_TestInit {
    function setUp() public virtual override {
        super.setUp();

        // Skip AddGameType tests when V2 dispute games are enabled
        // TODO(#17260): Remove skip when V2 dispute game support for addGameType implemented
        skipIfDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES);
    }

    /// @notice Tests that we can add a PermissionedDisputeGame implementation with addGameType.
    function test_addGameType_permissioned_succeeds() public {
        // Create the input for the Permissioned game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.PERMISSIONED_CANNON);

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);

        // Check the values on the new game type.
        IPermissionedDisputeGame newPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));
        IPermissionedDisputeGame oldPDG = chainDeployOutput1.permissionedDisputeGame;

        // Check the proposer and challenger values.
        assertEq(newPDG.proposer(), oldPDG.proposer(), "proposer mismatch");
        assertEq(newPDG.challenger(), oldPDG.challenger(), "challenger mismatch");

        // L2 chain ID call should not revert because this is not a Super game.
        assertNotEq(newPDG.l2ChainId(), 0, "l2ChainId should not be zero");
    }

    /// @notice Tests that we can add a FaultDisputeGame implementation with addGameType.
    function test_addGameType_cannon_succeeds() public {
        // Create the input for the Permissionless game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON);

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);

        // Check the values on the new game type.
        IPermissionedDisputeGame notPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));

        // Proposer call should revert because this is a permissionless game.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.proposer();

        // L2 chain ID call should not revert because this is not a Super game.
        assertNotEq(notPDG.l2ChainId(), 0, "l2ChainId should not be zero");
    }

    /// @notice Tests that we can add a SuperPermissionedDisputeGame implementation with addGameType.
    function test_addGameType_permissionedSuper_succeeds() public {
        // Create the input for the Super game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_PERMISSIONED_CANNON);

        // Since OPCM will start with the standard Permissioned (non-Super) game type we won't have
        // a Super dispute game to grab the proposer and challenger from. In production we'd either
        // already have a Super dispute game or we'd trigger the migration to make sure one exists.
        // Here for simplicity we'll just mock it out so the values exist.

        // Mock the DisputeGameFactory to return the non-Super implementation, good enough, it'll
        // have the right variables on it for the test to pass. We're basically just pretending
        // that the non-Super game is a Super game for the sake of this test.
        vm.mockCall(
            address(chainDeployOutput1.disputeGameFactoryProxy),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(chainDeployOutput1.permissionedDisputeGame)
        );
        vm.mockCall(
            address(chainDeployOutput1.permissionedDisputeGame),
            abi.encodeCall(IDisputeGame.gameType, ()),
            abi.encode(GameTypes.SUPER_PERMISSIONED_CANNON)
        );

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        vm.clearMockedCalls();
        assertValidGameType(input, output);

        // Check the values on the new game type.
        IPermissionedDisputeGame newPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));
        IPermissionedDisputeGame oldPDG = chainDeployOutput1.permissionedDisputeGame;
        assertEq(newPDG.proposer(), oldPDG.proposer(), "proposer mismatch");
        assertEq(newPDG.challenger(), oldPDG.challenger(), "challenger mismatch");

        // Super games don't have the l2ChainId function.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        newPDG.l2ChainId();
    }

    /// @notice Tests that we can add a SuperFaultDisputeGame implementation with addGameType.
    function test_addGameType_superCannon_succeeds() public {
        // Create the input for the Super game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_CANNON);

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);

        // Grab the new game type.
        IPermissionedDisputeGame notPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));

        // Proposer should fail, this is a permissionless game.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.proposer();

        // Super games don't have the l2ChainId function.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.l2ChainId();
    }

    /// @notice Tests that addGameType will revert if the game type is not supported.
    function test_addGameType_unsupportedGameType_reverts() public {
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameType.wrap(2000));

        // Run the addGameType call, should revert.
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](1);
        inputs[0] = input;
        (bool success,) = address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
    }

    /// @notice Tests that addGameType will revert if the game type is cannon-kona and the dev feature is not enabled
    function test_addGameType_cannonKonaGameTypeDisabled_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.CANNON_KONA);
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON_KONA);

        // Run the addGameType call, should revert.
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](1);
        inputs[0] = input;
        (bool success,) = address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
    }

    /// @notice Tests that addGameType will revert if the game type is cannon-kona and the dev feature is not enabled
    function test_addGameType_superCannonKonaGameTypeDisabled_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.CANNON_KONA);
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_CANNON_KONA);

        // Run the addGameType call, should revert.
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](1);
        inputs[0] = input;
        (bool success,) = address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
    }

    function test_addGameType_reusedDelayedWETH_succeeds() public {
        IDelayedWETH delayedWETH = IDelayedWETH(
            payable(
                address(
                    DeployUtils.create1({
                        _name: "DelayedWETH",
                        _args: DeployUtils.encodeConstructor(abi.encodeCall(IDelayedWETH.__constructor__, (1)))
                    })
                )
            )
        );
        vm.etch(address(delayedWETH), hex"01");
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON);
        input.delayedWETH = delayedWETH;
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);
        assertEq(address(output.delayedWETH), address(delayedWETH), "delayedWETH address mismatch");
    }

    function test_addGameType_outOfOrderInputs_reverts() public {
        IOPContractsManager.AddGameInput memory input1 = newGameInputFactory(GameType.wrap(2));
        IOPContractsManager.AddGameInput memory input2 = newGameInputFactory(GameType.wrap(1));
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](2);
        inputs[0] = input1;
        inputs[1] = input2;

        // For the sake of completeness, we run the call again to validate the success behavior.
        (bool success,) = address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
    }

    function test_addGameType_duplicateGameType_reverts() public {
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON);
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](2);
        inputs[0] = input;
        inputs[1] = input;

        // See test above for why we run the call twice.
        (bool success, bytes memory revertData) =
            address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
        assertEq(bytes4(revertData), IOPContractsManager.InvalidGameConfigs.selector, "revertData mismatch");
    }

    function test_addGameType_zeroLengthInput_reverts() public {
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](0);

        (bool success, bytes memory revertData) =
            address(opcm).delegatecall(abi.encodeCall(IOPContractsManager.addGameType, (inputs)));
        assertFalse(success, "addGameType should have failed");
        assertEq(bytes4(revertData), IOPContractsManager.InvalidGameConfigs.selector, "revertData mismatch");
    }

    function test_addGameType_notDelegateCall_reverts() public {
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.PERMISSIONED_CANNON);
        IOPContractsManager.AddGameInput[] memory inputs = new IOPContractsManager.AddGameInput[](1);
        inputs[0] = input;

        vm.expectRevert(IOPContractsManager.OnlyDelegatecall.selector);
        opcm.addGameType(inputs);
    }

    function assertValidGameType(
        IOPContractsManager.AddGameInput memory agi,
        IOPContractsManager.AddGameOutput memory ago
    )
        internal
        view
    {
        // Check the config for the game itself
        assertEq(ago.faultDisputeGame.gameType().raw(), agi.disputeGameType.raw(), "gameType mismatch");
        assertEq(
            ago.faultDisputeGame.absolutePrestate().raw(),
            agi.disputeAbsolutePrestate.raw(),
            "absolutePrestate mismatch"
        );
        assertEq(ago.faultDisputeGame.maxGameDepth(), agi.disputeMaxGameDepth, "maxGameDepth mismatch");
        assertEq(ago.faultDisputeGame.splitDepth(), agi.disputeSplitDepth, "splitDepth mismatch");
        assertEq(
            ago.faultDisputeGame.clockExtension().raw(), agi.disputeClockExtension.raw(), "clockExtension mismatch"
        );
        assertEq(
            ago.faultDisputeGame.maxClockDuration().raw(),
            agi.disputeMaxClockDuration.raw(),
            "maxClockDuration mismatch"
        );
        assertEq(address(ago.faultDisputeGame.vm()), address(agi.vm), "vm address mismatch");
        assertEq(address(ago.faultDisputeGame.weth()), address(ago.delayedWETH), "delayedWETH address mismatch");
        assertEq(
            address(ago.faultDisputeGame.anchorStateRegistry()),
            address(chainDeployOutput1.anchorStateRegistryProxy),
            "ASR address mismatch"
        );

        // Check the DGF
        assertEq(
            chainDeployOutput1.disputeGameFactoryProxy.gameImpls(agi.disputeGameType).gameType().raw(),
            agi.disputeGameType.raw(),
            "gameType mismatch"
        );
        assertEq(
            address(chainDeployOutput1.disputeGameFactoryProxy.gameImpls(agi.disputeGameType)),
            address(ago.faultDisputeGame),
            "gameImpl address mismatch"
        );
        assertEq(address(ago.faultDisputeGame.weth()), address(ago.delayedWETH), "weth address mismatch");
        assertEq(
            chainDeployOutput1.disputeGameFactoryProxy.initBonds(agi.disputeGameType), agi.initialBond, "bond mismatch"
        );
    }

    /// @notice Tests that addGameType will revert if the game type is cannon-kona and the dev feature is not enabled
    function test_addGameType_cannonKonaGameType_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        // Create the input for the cannon-kona game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON_KONA);

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);

        // Check the values on the new game type.
        IPermissionedDisputeGame notPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));

        // Proposer call should revert because this is a permissionless game.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.proposer();

        // L2 chain ID call should not revert because this is not a Super game.
        assertNotEq(notPDG.l2ChainId(), 0, "l2ChainId should not be zero");
    }

    /// @notice Tests that addGameType will revert if the game type is cannon-kona and the dev feature is not enabled
    function test_addGameType_superCannonKonaGameType_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        // Create the input for the cannon-kona game type.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_CANNON_KONA);

        // Run the addGameType call.
        IOPContractsManager.AddGameOutput memory output = addGameType(input);
        assertValidGameType(input, output);

        // Grab the new game type.
        IPermissionedDisputeGame notPDG = IPermissionedDisputeGame(address(output.faultDisputeGame));

        // Proposer should fail, this is a permissionless game.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.proposer();

        // Super games don't have the l2ChainId function.
        vm.expectRevert(); // nosemgrep: sol-safety-expectrevert-no-args
        notPDG.l2ChainId();
    }
}

/// @title OPContractsManager_UpdatePrestate_Test
/// @notice Tests the `updatePrestate` function of the `OPContractsManager` contract.
contract OPContractsManager_UpdatePrestate_Test is OPContractsManager_TestInit {
    IOPContractsManager internal prestateUpdater;
    OPContractsManager.AddGameInput[] internal gameInput;

    function setUp() public virtual override {
        super.setUp();

        // Skip UpdatePrestate tests when V2 dispute games enabled
        // UpdatePrestate feature not yet implemented for V2
        // TODO(#17261): Remove skip when V2 dispute game support for updatePrestate implemented
        skipIfDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES);

        prestateUpdater = opcm;
    }

    /// @notice Runs the OPCM updatePrestate function and checks the results.
    /// @param _input The input to the OPCM updatePrestate function.
    function _runUpdatePrestateAndChecks(IOPContractsManager.UpdatePrestateInput memory _input) internal {
        _runUpdatePrestateAndChecks(_input, bytes(""));
    }

    /// @notice Runs the OPCM updatePrestate function and checks the results.
    /// @param _input The input to the OPCM updatePrestate function.
    /// @param _revertBytes The bytes of the revert to expect, if any.
    function _runUpdatePrestateAndChecks(
        IOPContractsManager.UpdatePrestateInput memory _input,
        bytes memory _revertBytes
    )
        internal
    {
        bool expectCannonUpdated = address(
            IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(GameTypes.CANNON)
        ) != address(0);
        bool expectCannonKonaUpdated = address(
            IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                GameTypes.CANNON_KONA
            )
        ) != address(0);

        // Turn the ProxyAdmin owner into a DelegateCaller.
        address proxyAdminOwner = chainDeployOutput1.opChainProxyAdmin.owner();
        vm.etch(address(proxyAdminOwner), vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        IOPContractsManager.UpdatePrestateInput[] memory inputs = new IOPContractsManager.UpdatePrestateInput[](1);
        inputs[0] = _input;

        if (_revertBytes.length > 0) {
            vm.expectRevert(_revertBytes);
        }

        // Trigger the updatePrestate function.
        DelegateCaller(proxyAdminOwner).dcForward(
            address(prestateUpdater), abi.encodeCall(IOPContractsManager.updatePrestate, (inputs))
        );

        // Return early if a revert was expected. Otherwise we'll get errors below.
        if (_revertBytes.length > 0) {
            return;
        }

        // Grab the PermissionedDisputeGame.
        IPermissionedDisputeGame pdg = IPermissionedDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.PERMISSIONED_CANNON
                )
            )
        );
        assertEq(pdg.absolutePrestate().raw(), _input.cannonPrestate.raw(), "permissioned game prestate mismatch");
        // Ensure that the WETH contracts are not reverting
        pdg.weth().balanceOf(address(0));

        if (expectCannonUpdated) {
            IPermissionedDisputeGame game = IPermissionedDisputeGame(
                address(
                    IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                        GameTypes.CANNON
                    )
                )
            );
            assertEq(game.absolutePrestate().raw(), _input.cannonPrestate.raw(), "cannon game prestate mismatch");
            // Ensure that the WETH contracts are not reverting
            game.weth().balanceOf(address(0));
        } else {
            assertEq(
                address(
                    IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                        GameTypes.CANNON
                    )
                ),
                (address(0)),
                "cannon game should not exist"
            );
        }

        if (expectCannonKonaUpdated) {
            IPermissionedDisputeGame game = IPermissionedDisputeGame(
                address(
                    IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                        GameTypes.CANNON_KONA
                    )
                )
            );
            assertEq(game.absolutePrestate().raw(), _input.cannonKonaPrestate.raw(), "cannon game prestate mismatch");
            // Ensure that the WETH contracts are not reverting
            game.weth().balanceOf(address(0));
        } else {
            assertEq(
                address(
                    IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                        GameTypes.CANNON_KONA
                    )
                ),
                (address(0)),
                "cannon_kona game should not exist"
            );
        }
    }

    /// @notice Tests that we can update the prestate when only the PermissionedDisputeGame exists.
    function test_updatePrestate_pdgOnlyWithValidInput_succeeds() public {
        Claim prestate = Claim.wrap(bytes32(hex"ABBA"));
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput(
                chainDeployOutput1.systemConfigProxy, prestate, Claim.wrap(bytes32(0))
            )
        );
    }

    /// @notice Tests that we can update the prestate when both the PermissionedDisputeGame and
    ///         FaultDisputeGame exist.
    function test_updatePrestate_bothGamesWithValidInput_succeeds() public {
        // Add a FaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON);
        addGameType(input);

        Claim prestate = Claim.wrap(bytes32(hex"ABBA"));
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput(
                chainDeployOutput1.systemConfigProxy, prestate, Claim.wrap(bytes32(0))
            )
        );
    }

    /// @notice Tests that we can update the prestate when a SuperFaultDisputeGame exists. Note
    ///         that this test isn't ideal because the system starts with a PermissionedDisputeGame
    ///         and then adds a SuperPermissionedDisputeGame and SuperFaultDisputeGame. In the real
    ///         system we wouldn't have that PermissionedDisputeGame to start with, but it
    ///         shouldn't matter because the function is independent of other game types that
    ///         exist.
    function test_updatePrestate_withSuperGame_succeeds() public {
        // Mock out the existence of a previous SuperPermissionedDisputeGame so we can add a real
        // SuperPermissionedDisputeGame implementation.
        vm.mockCall(
            address(chainDeployOutput1.disputeGameFactoryProxy),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(chainDeployOutput1.permissionedDisputeGame)
        );
        vm.mockCall(
            address(chainDeployOutput1.permissionedDisputeGame),
            abi.encodeCall(IDisputeGame.gameType, ()),
            abi.encode(GameTypes.SUPER_PERMISSIONED_CANNON)
        );

        // Add a SuperPermissionedDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input1 = newGameInputFactory(GameTypes.SUPER_PERMISSIONED_CANNON);
        addGameType(input1);
        vm.clearMockedCalls();

        // Add a SuperFaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input2 = newGameInputFactory(GameTypes.SUPER_CANNON);
        addGameType(input2);

        // Clear out the PermissionedDisputeGame implementation.
        address owner = chainDeployOutput1.disputeGameFactoryProxy.owner();
        vm.prank(owner);
        chainDeployOutput1.disputeGameFactoryProxy.setImplementation(
            GameTypes.PERMISSIONED_CANNON, IDisputeGame(payable(address(0)))
        );

        // Create the input for the function call.
        Claim prestate = Claim.wrap(bytes32(hex"ABBA"));
        IOPContractsManager.UpdatePrestateInput[] memory inputs = new IOPContractsManager.UpdatePrestateInput[](1);
        inputs[0] = IOPContractsManager.UpdatePrestateInput(
            chainDeployOutput1.systemConfigProxy, prestate, Claim.wrap(bytes32(0))
        );

        // Turn the ProxyAdmin owner into a DelegateCaller.
        address proxyAdminOwner = chainDeployOutput1.opChainProxyAdmin.owner();
        vm.etch(address(proxyAdminOwner), vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        // Trigger the updatePrestate function.
        DelegateCaller(proxyAdminOwner).dcForward(
            address(prestateUpdater), abi.encodeCall(IOPContractsManager.updatePrestate, (inputs))
        );

        // Grab the SuperPermissionedDisputeGame.
        IPermissionedDisputeGame pdg = IPermissionedDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.SUPER_PERMISSIONED_CANNON
                )
            )
        );

        // Grab the SuperFaultDisputeGame.
        IPermissionedDisputeGame fdg = IPermissionedDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.SUPER_CANNON
                )
            )
        );

        // Check the prestate values.
        assertEq(pdg.absolutePrestate().raw(), prestate.raw(), "pdg prestate mismatch");
        assertEq(fdg.absolutePrestate().raw(), prestate.raw(), "fdg prestate mismatch");

        // Ensure that the WETH contracts are not reverting
        pdg.weth().balanceOf(address(0));
        fdg.weth().balanceOf(address(0));
    }

    /// @notice Tests that the updatePrestate function will revert if the provided prestate is for
    ///        mixed game types (i.e. CANNON and SUPER_CANNON).
    function test_updatePrestate_mixedGameTypes_reverts() public {
        // Add a SuperFaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_CANNON);
        addGameType(input);

        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(hex"ABBA")),
                cannonKonaPrestate: Claim.wrap(bytes32(0))
            }),
            abi.encodeWithSelector(
                IOPContractsManagerGameTypeAdder.OPContractsManagerGameTypeAdder_MixedGameTypes.selector
            )
        );
    }

    /// @notice Tests that the updatePrestate function will revert if the provided prestate is the
    ///         zero hash.
    function test_updatePrestate_whenPDGPrestateIsZero_reverts() public {
        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(0)),
                cannonKonaPrestate: Claim.wrap(bytes32(0))
            }),
            abi.encodeWithSelector(IOPContractsManager.PrestateRequired.selector)
        );
    }

    function test_updatePrestate_whenOnlyCannonPrestateIsZeroAndCannonGameTypeDisabled_reverts() public {
        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(0)),
                cannonKonaPrestate: Claim.wrap(bytes32(hex"ABBA"))
            }),
            abi.encodeWithSelector(IOPContractsManager.PrestateRequired.selector)
        );
    }

    /// @notice Tests that we can update the prestate for both CANNON and CANNON_KONA game types.
    function test_updatePrestate_bothGamesAndCannonKonaWithValidInput_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        // Add a FaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON);
        addGameType(input);
        input = newGameInputFactory(GameTypes.CANNON_KONA);
        addGameType(input);

        Claim cannonPrestate = Claim.wrap(bytes32(hex"ABBA"));
        Claim cannonKonaPrestate = Claim.wrap(bytes32(hex"ADDA"));
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: cannonPrestate,
                cannonKonaPrestate: cannonKonaPrestate
            })
        );
    }

    function test_updatePrestate_cannonKonaWithSuperGame_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        // Mock out the existence of a previous SuperPermissionedDisputeGame so we can add a real
        // SuperPermissionedDisputeGame implementation.
        vm.mockCall(
            address(chainDeployOutput1.disputeGameFactoryProxy),
            abi.encodeCall(IDisputeGameFactory.gameImpls, (GameTypes.SUPER_PERMISSIONED_CANNON)),
            abi.encode(chainDeployOutput1.permissionedDisputeGame)
        );
        vm.mockCall(
            address(chainDeployOutput1.permissionedDisputeGame),
            abi.encodeCall(IDisputeGame.gameType, ()),
            abi.encode(GameTypes.SUPER_PERMISSIONED_CANNON)
        );

        // Add a SuperPermissionedDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input1 = newGameInputFactory(GameTypes.SUPER_PERMISSIONED_CANNON);
        addGameType(input1);
        vm.clearMockedCalls();

        // Add a SuperFaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input2 = newGameInputFactory(GameTypes.SUPER_CANNON);
        addGameType(input2);
        IOPContractsManager.AddGameInput memory input3 = newGameInputFactory(GameTypes.SUPER_CANNON_KONA);
        addGameType(input3);

        // Clear out the PermissionedDisputeGame implementation.
        address owner = chainDeployOutput1.disputeGameFactoryProxy.owner();
        vm.prank(owner);
        chainDeployOutput1.disputeGameFactoryProxy.setImplementation(
            GameTypes.PERMISSIONED_CANNON, IDisputeGame(payable(address(0)))
        );

        // Create the input for the function call.
        Claim cannonPrestate = Claim.wrap(bytes32(hex"ABBA"));
        Claim cannonKonaPrestate = Claim.wrap(bytes32(hex"ABBA"));
        IOPContractsManager.UpdatePrestateInput[] memory inputs = new IOPContractsManager.UpdatePrestateInput[](1);
        inputs[0] = IOPContractsManager.UpdatePrestateInput({
            systemConfigProxy: chainDeployOutput1.systemConfigProxy,
            cannonPrestate: cannonPrestate,
            cannonKonaPrestate: cannonKonaPrestate
        });

        // Turn the ProxyAdmin owner into a DelegateCaller.
        address proxyAdminOwner = chainDeployOutput1.opChainProxyAdmin.owner();
        vm.etch(address(proxyAdminOwner), vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        // Trigger the updatePrestate function.
        DelegateCaller(proxyAdminOwner).dcForward(
            address(prestateUpdater), abi.encodeCall(IOPContractsManager.updatePrestate, (inputs))
        );

        IPermissionedDisputeGame pdg = IPermissionedDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.SUPER_PERMISSIONED_CANNON
                )
            )
        );

        IFaultDisputeGame fdg = IFaultDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.SUPER_CANNON
                )
            )
        );
        IFaultDisputeGame fdgKona = IFaultDisputeGame(
            address(
                IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory()).gameImpls(
                    GameTypes.SUPER_CANNON_KONA
                )
            )
        );

        // Check the prestate values.
        assertEq(pdg.absolutePrestate().raw(), cannonPrestate.raw(), "pdg prestate mismatch");
        assertEq(fdg.absolutePrestate().raw(), cannonPrestate.raw(), "fdg prestate mismatch");
        assertEq(fdgKona.absolutePrestate().raw(), cannonKonaPrestate.raw(), "fdgKona prestate mismatch");

        // Ensure that the WETH contracts are not reverting
        pdg.weth().balanceOf(address(0));
        fdg.weth().balanceOf(address(0));
        fdgKona.weth().balanceOf(address(0));
    }

    /// @notice Tests that we can update the prestate when both the PermissionedDisputeGame and
    ///        FaultDisputeGame exist, and the FaultDisputeGame is of type CANNON_KONA.
    function test_updatePrestate_pdgAndCannonKonaOnly_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON_KONA);
        addGameType(input);

        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(hex"ABBA")),
                cannonKonaPrestate: Claim.wrap(bytes32(hex"ADDA"))
            })
        );
    }

    /// @notice Tests that the updatePrestate function will revert if the provided prestate is for
    ///       mixed game types (i.e. CANNON and SUPER_CANNON_KONA).
    function test_updatePrestate_cannonKonaMixedGameTypes_reverts() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        // Add a SuperFaultDisputeGame implementation via addGameType.
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.SUPER_CANNON_KONA);
        addGameType(input);

        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(hex"ABBA")),
                cannonKonaPrestate: Claim.wrap(hex"ADDA")
            }),
            abi.encodeWithSelector(
                IOPContractsManagerGameTypeAdder.OPContractsManagerGameTypeAdder_MixedGameTypes.selector
            )
        );
    }

    /// @notice Tests that the updatePrestate function will revert if the provided prestate is the
    ///         zero hash.
    function test_updatePrestate_presetCannonKonaWhenOnlyCannonPrestateIsZeroAndCannonGameTypeDisabled_reverts()
        public
    {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON_KONA);
        addGameType(input);

        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(0)),
                cannonKonaPrestate: Claim.wrap(bytes32(hex"ABBA"))
            }),
            abi.encodeWithSelector(IOPContractsManager.PrestateRequired.selector)
        );
    }

    /// @notice Tests that the updatePrestate function will revert if the provided prestate is the
    ///         zero hash.
    function test_updatePrestate_whenCannonKonaPrestateIsZero_reverts() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        IOPContractsManager.AddGameInput memory input = newGameInputFactory(GameTypes.CANNON_KONA);
        addGameType(input);

        // nosemgrep: sol-style-use-abi-encodecall
        _runUpdatePrestateAndChecks(
            IOPContractsManager.UpdatePrestateInput({
                systemConfigProxy: chainDeployOutput1.systemConfigProxy,
                cannonPrestate: Claim.wrap(bytes32(hex"ABBA")),
                cannonKonaPrestate: Claim.wrap(bytes32(0))
            }),
            abi.encodeWithSelector(IOPContractsManager.PrestateRequired.selector)
        );
    }
}

/// @title OPContractsManager_Upgrade_Test
/// @notice Tests the `upgrade` function of the `OPContractsManager` contract.
contract OPContractsManager_Upgrade_Test is OPContractsManager_Upgrade_Harness {
    function setUp() public override {
        super.setUp();

        // Run all past upgrades.
        runPastUpgrades(upgrader);
    }

    function test_upgradeOPChainOnly_succeeds() public {
        // Run the upgrade test and checks
        runCurrentUpgrade(upgrader);
    }

    function test_verifyOpcmCorrectness_succeeds() public {
        skipIfCoverage(); // Coverage changes bytecode and breaks the verification script.

        // Set up environment variables with the actual OPCM addresses for tests that need themqq
        vm.setEnv("EXPECTED_SUPERCHAIN_CONFIG", vm.toString(address(opcm.superchainConfig())));
        vm.setEnv("EXPECTED_PROTOCOL_VERSIONS", vm.toString(address(opcm.protocolVersions())));

        // Run the upgrade test and checks
        runCurrentUpgrade(upgrader);

        // Run the verification script without etherscan verification. Hard to run with etherscan
        // verification in these tests, can do it but means we add even more dependencies to the
        // test environment.
        VerifyOPCM verify = new VerifyOPCM();
        verify.run(address(opcm), true);
    }

    function test_upgrade_duplicateL2ChainId_succeeds() public {
        // Deploy a new OPChain with the same L2 chain ID as the current OPChain
        Deploy deploy = Deploy(address(uint160(uint256(keccak256(abi.encode("optimism.deploy"))))));
        IOPContractsManager.DeployInput memory deployInput = deploy.getDeployInput();
        deployInput.l2ChainId = l2ChainId;
        deployInput.saltMixer = "v2.0.0";
        opcm.deploy(deployInput);

        // Try to upgrade the current OPChain
        runCurrentUpgrade(upgrader);
    }

    /// @notice Tests that the absolute prestate can be overridden using the upgrade config.
    function test_upgrade_absolutePrestateOverride_succeeds() public {
        // Get the pdg and fdg before the upgrade
        Claim pdgPrestateBefore = IPermissionedDisputeGame(
            address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON))
        ).absolutePrestate();
        Claim fdgPrestateBefore =
            IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON))).absolutePrestate();

        // Assert that the prestate is not zero.
        assertNotEq(pdgPrestateBefore.raw(), bytes32(0));
        assertNotEq(fdgPrestateBefore.raw(), bytes32(0));

        // Set the absolute prestate input to something non-zero.
        opChainConfigs[0].absolutePrestate = Claim.wrap(bytes32(uint256(1)));

        // Run the upgrade.
        runCurrentUpgrade(upgrader);

        // Get the absolute prestate after the upgrade
        Claim pdgPrestateAfter = IPermissionedDisputeGame(
            address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON))
        ).absolutePrestate();
        Claim fdgPrestateAfter =
            IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON))).absolutePrestate();

        // Assert that the absolute prestate is the non-zero value we set.
        assertEq(pdgPrestateAfter.raw(), bytes32(uint256(1)));
        assertEq(fdgPrestateAfter.raw(), bytes32(uint256(1)));
    }

    /// @notice Tests that the old absolute prestate is used if the upgrade config does not set an
    ///         absolute prestate.
    function test_upgrade_absolutePrestateNotSet_succeeds() public {
        // Get the pdg and fdg before the upgrade
        Claim pdgPrestateBefore = IPermissionedDisputeGame(
            address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON))
        ).absolutePrestate();
        Claim fdgPrestateBefore =
            IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON))).absolutePrestate();

        // Assert that the prestate is not zero.
        assertNotEq(pdgPrestateBefore.raw(), bytes32(0));
        assertNotEq(fdgPrestateBefore.raw(), bytes32(0));

        // Set the absolute prestate input to zero.
        opChainConfigs[0].absolutePrestate = Claim.wrap(bytes32(0));

        // Run the upgrade.
        runCurrentUpgrade(upgrader);

        // Get the absolute prestate after the upgrade
        Claim pdgPrestateAfter = IPermissionedDisputeGame(
            address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON))
        ).absolutePrestate();
        Claim fdgPrestateAfter =
            IFaultDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.CANNON))).absolutePrestate();

        // Assert that the absolute prestate is the same as before the upgrade.
        assertEq(pdgPrestateAfter.raw(), pdgPrestateBefore.raw());
        assertEq(fdgPrestateAfter.raw(), fdgPrestateBefore.raw());
    }

    function test_upgrade_notDelegateCalled_reverts() public {
        vm.prank(upgrader);
        vm.expectRevert(IOPContractsManager.OnlyDelegatecall.selector);
        opcm.upgrade(opChainConfigs);
    }

    function test_upgrade_notProxyAdminOwner_reverts() public {
        address delegateCaller = makeAddr("delegateCaller");
        vm.etch(delegateCaller, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        assertNotEq(superchainProxyAdmin.owner(), delegateCaller);
        assertNotEq(proxyAdmin.owner(), delegateCaller);

        runCurrentUpgrade(delegateCaller, bytes("Ownable: caller is not the owner"));
    }

    /// @notice Tests that upgrade reverts when absolutePrestate is zero and the existing game also
    ///         has an absolute prestate of zero.
    function test_upgrade_absolutePrestateNotSet_reverts() public {
        // Set the config to try to update the absolutePrestate to zero.
        opChainConfigs[0].absolutePrestate = Claim.wrap(bytes32(0));

        // Get the address of the PermissionedDisputeGame.
        IPermissionedDisputeGame pdg =
            IPermissionedDisputeGame(address(disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON)));

        // Mock the PDG to return a prestate of zero.
        vm.mockCall(
            address(pdg),
            abi.encodeCall(IPermissionedDisputeGame.absolutePrestate, ()),
            abi.encode(Claim.wrap(bytes32(0)))
        );

        // Expect the upgrade to revert with PrestateNotSet.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgrade(upgrader, abi.encodeWithSelector(IOPContractsManager.PrestateNotSet.selector));
    }

    /// @notice Tests that the upgrade function reverts when the superchainConfig is not at the expected target version.
    function test_upgrade_superchainConfigNeedsUpgrade_reverts() public {
        // Force the SuperchainConfig to return an obviously outdated version.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("0.0.0"));

        // Try upgrading an OPChain without upgrading its superchainConfig.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgrade(
            upgrader,
            abi.encodeWithSelector(
                IOPContractsManagerUpgrader.OPContractsManagerUpgrader_SuperchainConfigNeedsUpgrade.selector, (0)
            )
        );
    }
}

contract OPContractsManager_UpgradeSuperchainConfig_Test is OPContractsManager_Upgrade_Harness {
    function setUp() public override {
        super.setUp();

        // The superchainConfig is already at the expected version so we mock this call here to bypass that check and
        // get our expected error.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("2.2.0"));
    }

    /// @notice Tests that the upgradeSuperchainConfig function succeeds when the superchainConfig is at the expected
    ///         version and the delegate caller is the superchainProxyAdmin owner.
    function test_upgradeSuperchainConfig_succeeds() public {
        IOPContractsManager.Implementations memory impls = opcm.implementations();

        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));

        address superchainPAO = IProxyAdmin(EIP1967Helper.getAdmin(address(superchainConfig))).owner();
        vm.etch(superchainPAO, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        vm.expectEmit(address(superchainConfig));
        emit Upgraded(impls.superchainConfigImpl);
        DelegateCaller(superchainPAO).dcForward(
            address(opcm),
            abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (superchainConfig, superchainProxyAdmin))
        );
    }

    /// @notice Tests that the upgradeSuperchainConfig function reverts when it is not called via delegatecall.
    function test_upgradeSuperchainConfig_notDelegateCalled_reverts() public {
        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));

        vm.expectRevert(IOPContractsManager.OnlyDelegatecall.selector);
        opcm.upgradeSuperchainConfig(superchainConfig, superchainProxyAdmin);
    }

    /// @notice Tests that the upgradeSuperchainConfig function reverts when the delegate caller is not the
    ///         superchainProxyAdmin owner.
    function test_upgradeSuperchainConfig_notProxyAdminOwner_reverts() public {
        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));

        address delegateCaller = makeAddr("delegateCaller");
        vm.etch(delegateCaller, vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        assertNotEq(superchainProxyAdmin.owner(), delegateCaller);
        assertNotEq(proxyAdmin.owner(), delegateCaller);

        vm.expectRevert("Ownable: caller is not the owner");
        DelegateCaller(delegateCaller).dcForward(
            address(opcm),
            abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (superchainConfig, superchainProxyAdmin))
        );
    }

    /// @notice Tests that the upgradeSuperchainConfig function reverts when the superchainConfig version is the same or
    ///         newer than the target version.
    function test_upgradeSuperchainConfig_superchainConfigAlreadyUpToDate_reverts() public {
        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));

        // Set the version of the superchain config to a version that is the target version.
        vm.clearMockedCalls();

        // Mock the SuperchainConfig to return a very large version.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("99.99.99"));

        // Try to upgrade the SuperchainConfig contract again, should fail.
        vm.expectRevert(IOPContractsManagerUpgrader.OPContractsManagerUpgrader_SuperchainConfigAlreadyUpToDate.selector);
        DelegateCaller(upgrader).dcForward(
            address(opcm),
            abi.encodeCall(IOPContractsManager.upgradeSuperchainConfig, (superchainConfig, superchainProxyAdmin))
        );
    }
}

/// @title OPContractsManager_Migrate_Test
/// @notice Tests the `migrate` function of the `OPContractsManager` contract.
contract OPContractsManager_Migrate_Test is OPContractsManager_TestInit {
    Claim absolutePrestate1 = Claim.wrap(bytes32(hex"ABBA"));
    Claim absolutePrestate2 = Claim.wrap(bytes32(hex"DEAD"));

    /// @notice Function requires interop portal.
    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);
    }

    /// @notice Helper function to create the default migration input.
    function _getDefaultInput() internal view returns (IOPContractsManagerInteropMigrator.MigrateInput memory) {
        IOPContractsManagerInteropMigrator.GameParameters memory gameParameters = IOPContractsManagerInteropMigrator
            .GameParameters({
            proposer: address(1234),
            challenger: address(5678),
            maxGameDepth: 72,
            splitDepth: 32,
            initBond: 1 ether,
            clockExtension: Duration.wrap(10800),
            maxClockDuration: Duration.wrap(302400)
        });

        IOPContractsManager.OpChainConfig[] memory opChainConfigs = new IOPContractsManager.OpChainConfig[](2);
        opChainConfigs[0] = IOPContractsManager.OpChainConfig(
            chainDeployOutput1.systemConfigProxy, chainDeployOutput1.opChainProxyAdmin, absolutePrestate1
        );
        opChainConfigs[1] = IOPContractsManager.OpChainConfig(
            chainDeployOutput2.systemConfigProxy, chainDeployOutput2.opChainProxyAdmin, absolutePrestate1
        );

        return IOPContractsManagerInteropMigrator.MigrateInput({
            usePermissionlessGame: true,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(hex"ABBA")), l2SequenceNumber: 1234 }),
            gameParameters: gameParameters,
            opChainConfigs: opChainConfigs
        });
    }

    /// @notice Helper function to execute a migration.
    /// @param _input The input to the migration function.
    function _doMigration(IOPContractsManagerInteropMigrator.MigrateInput memory _input) internal {
        _doMigration(_input, bytes4(0));
    }

    /// @notice Helper function to execute a migration with a revert selector.
    /// @param _input The input to the migration function.
    /// @param _revertSelector The selector of the revert to expect.
    function _doMigration(
        IOPContractsManagerInteropMigrator.MigrateInput memory _input,
        bytes4 _revertSelector
    )
        internal
    {
        // Set the proxy admin owner to be a delegate caller.
        address proxyAdminOwner = chainDeployOutput1.opChainProxyAdmin.owner();
        vm.etch(address(proxyAdminOwner), vm.getDeployedCode("test/mocks/Callers.sol:DelegateCaller"));

        // Execute a delegatecall to the OPCM migration function.
        // Check gas usage of the migration function.
        uint256 gasBefore = gasleft();
        if (_revertSelector != bytes4(0)) {
            vm.expectRevert(_revertSelector);
        }
        DelegateCaller(proxyAdminOwner).dcForward(address(opcm), abi.encodeCall(IOPContractsManager.migrate, (_input)));
        uint256 gasAfter = gasleft();

        // Make sure the gas usage is less than 20 million so we can definitely fit in a block.
        assertLt(gasBefore - gasAfter, 20_000_000, "Gas usage too high");
    }

    /// @notice Helper function to assert that the old game implementations are now zeroed out.
    ///         We need a separate helper to avoid stack too deep errors.
    /// @param _disputeGameFactory The dispute game factory to check.
    function _assertOldGamesZeroed(IDisputeGameFactory _disputeGameFactory) internal view {
        // Assert that the old game implementations are now zeroed out.
        assertEq(address(_disputeGameFactory.gameImpls(GameTypes.CANNON)), address(0));
        assertEq(address(_disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON)), address(0));
        assertEq(address(_disputeGameFactory.gameImpls(GameTypes.PERMISSIONED_CANNON)), address(0));
        assertEq(address(_disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON)), address(0));
        if (isDevFeatureEnabled(DevFeatures.CANNON_KONA)) {
            // Only explicitly zeroed out if feature is enabled. Otherwise left unchanged (which may still be 0).
            assertEq(address(_disputeGameFactory.gameImpls(GameTypes.CANNON_KONA)), address(0));
            assertEq(address(_disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON_KONA)), address(0));
        }
    }

    /// @notice Tests that the migration function succeeds when requesting to use the
    ///         permissionless game.
    function test_migrate_withPermissionlessGame_succeeds() public {
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Separate context to avoid stack too deep errors.
        {
            // Grab the existing DisputeGameFactory for each chain.
            IDisputeGameFactory oldDisputeGameFactory1 =
                IDisputeGameFactory(payable(chainDeployOutput1.systemConfigProxy.disputeGameFactory()));
            IDisputeGameFactory oldDisputeGameFactory2 =
                IDisputeGameFactory(payable(chainDeployOutput2.systemConfigProxy.disputeGameFactory()));

            // Execute the migration.
            _doMigration(input);

            // Assert that the old game implementations are now zeroed out.
            _assertOldGamesZeroed(oldDisputeGameFactory1);
            _assertOldGamesZeroed(oldDisputeGameFactory2);
        }

        // Grab the two OptimismPortal addresses.
        IOptimismPortal2 optimismPortal1 =
            IOptimismPortal2(payable(chainDeployOutput1.systemConfigProxy.optimismPortal()));
        IOptimismPortal2 optimismPortal2 =
            IOptimismPortal2(payable(chainDeployOutput2.systemConfigProxy.optimismPortal()));

        // Grab the AnchorStateRegistry from the OptimismPortal for both chains, confirm same.
        assertEq(
            address(optimismPortal1.anchorStateRegistry()),
            address(optimismPortal2.anchorStateRegistry()),
            "AnchorStateRegistry mismatch"
        );

        // Extract the AnchorStateRegistry now that we know it's the same on both chains.
        IAnchorStateRegistry anchorStateRegistry = optimismPortal1.anchorStateRegistry();

        // Grab the DisputeGameFactory from the SystemConfig for both chains, confirm same.
        assertEq(
            chainDeployOutput1.systemConfigProxy.disputeGameFactory(),
            chainDeployOutput2.systemConfigProxy.disputeGameFactory(),
            "DisputeGameFactory mismatch"
        );

        // Extract the DisputeGameFactory now that we know it's the same on both chains.
        IDisputeGameFactory disputeGameFactory =
            IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory());

        // Grab the ETHLockbox from the OptimismPortal for both chains, confirm same.
        assertEq(address(optimismPortal1.ethLockbox()), address(optimismPortal2.ethLockbox()), "ETHLockbox mismatch");

        // Extract the ETHLockbox now that we know it's the same on both chains.
        IETHLockbox ethLockbox = optimismPortal1.ethLockbox();

        // Check that the ETHLockbox was migrated correctly.
        assertGt(address(ethLockbox).balance, 0, "ETHLockbox balance is zero");
        assertTrue(ethLockbox.authorizedPortals(optimismPortal1), "ETHLockbox does not have portal 1 authorized");
        assertTrue(ethLockbox.authorizedPortals(optimismPortal2), "ETHLockbox does not have portal 2 authorized");

        // Check that the respected game type is the Super Cannon game type.
        assertEq(
            anchorStateRegistry.respectedGameType().raw(),
            GameTypes.SUPER_CANNON.raw(),
            "Super Cannon game type mismatch"
        );

        // Check that the starting anchor root is the same as the input.
        (Hash root, uint256 l2SequenceNumber) = anchorStateRegistry.getAnchorRoot();
        assertEq(root.raw(), input.startingAnchorRoot.root.raw(), "Starting anchor root mismatch");
        assertEq(
            l2SequenceNumber,
            input.startingAnchorRoot.l2SequenceNumber,
            "Starting anchor root L2 sequence number mismatch"
        );

        // Check that the DisputeGameFactory has implementations for both games.
        assertEq(
            disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON).gameType().raw(),
            GameTypes.SUPER_CANNON.raw(),
            "Super Cannon game type not set properly"
        );
        assertEq(
            disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON).gameType().raw(),
            GameTypes.SUPER_PERMISSIONED_CANNON.raw(),
            "Super Permissioned Cannon game type not set properly"
        );
        assertEq(
            disputeGameFactory.initBonds(GameTypes.SUPER_CANNON),
            input.gameParameters.initBond,
            "Super Cannon init bond mismatch"
        );
        assertEq(
            disputeGameFactory.initBonds(GameTypes.SUPER_PERMISSIONED_CANNON),
            input.gameParameters.initBond,
            "Super Permissioned Cannon init bond mismatch"
        );

        // Check that the Super Cannon game has the correct parameters.
        IDisputeGame superFdgImpl = disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON);
        ISuperFaultDisputeGame superFdg = ISuperFaultDisputeGame(address(superFdgImpl));
        assertEq(superFdg.maxGameDepth(), input.gameParameters.maxGameDepth);
        assertEq(superFdg.splitDepth(), input.gameParameters.splitDepth);
        assertEq(superFdg.clockExtension().raw(), input.gameParameters.clockExtension.raw());
        assertEq(superFdg.maxClockDuration().raw(), input.gameParameters.maxClockDuration.raw());
        assertEq(superFdg.absolutePrestate().raw(), absolutePrestate1.raw());

        // Check that the Super Permissioned Cannon game has the correct parameters.
        IDisputeGame superPdgImpl = disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON);
        ISuperPermissionedDisputeGame superPdg = ISuperPermissionedDisputeGame(address(superPdgImpl));
        assertEq(superPdg.proposer(), input.gameParameters.proposer);
        assertEq(superPdg.challenger(), input.gameParameters.challenger);
        assertEq(superPdg.maxGameDepth(), input.gameParameters.maxGameDepth);
        assertEq(superPdg.splitDepth(), input.gameParameters.splitDepth);
        assertEq(superPdg.clockExtension().raw(), input.gameParameters.clockExtension.raw());
        assertEq(superPdg.maxClockDuration().raw(), input.gameParameters.maxClockDuration.raw());
        assertEq(superPdg.absolutePrestate().raw(), absolutePrestate1.raw());
    }

    /// @notice Tests that the migration function succeeds when requesting to not use the
    ///         permissioned game (no permissioned game is deployed).
    function test_migrate_withoutPermissionlessGame_succeeds() public {
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Change the input to not use the permissionless game.
        input.usePermissionlessGame = false;

        // Separate context to avoid stack too deep errors.
        {
            // Grab the existing DisputeGameFactory for each chain.
            IDisputeGameFactory oldDisputeGameFactory1 =
                IDisputeGameFactory(payable(chainDeployOutput1.systemConfigProxy.disputeGameFactory()));
            IDisputeGameFactory oldDisputeGameFactory2 =
                IDisputeGameFactory(payable(chainDeployOutput2.systemConfigProxy.disputeGameFactory()));

            // Execute the migration.
            _doMigration(input);

            // Assert that the old game implementations are now zeroed out.
            _assertOldGamesZeroed(oldDisputeGameFactory1);
            _assertOldGamesZeroed(oldDisputeGameFactory2);
        }

        // Grab the two OptimismPortal addresses.
        IOptimismPortal2 optimismPortal1 =
            IOptimismPortal2(payable(chainDeployOutput1.systemConfigProxy.optimismPortal()));
        IOptimismPortal2 optimismPortal2 =
            IOptimismPortal2(payable(chainDeployOutput2.systemConfigProxy.optimismPortal()));

        // Grab the AnchorStateRegistry from the SystemConfig for both chains, confirm same.
        assertEq(
            address(optimismPortal1.anchorStateRegistry()),
            address(optimismPortal2.anchorStateRegistry()),
            "AnchorStateRegistry mismatch"
        );

        // Extract the AnchorStateRegistry now that we know it's the same on both chains.
        IAnchorStateRegistry anchorStateRegistry = optimismPortal1.anchorStateRegistry();

        // Grab the DisputeGameFactory from the SystemConfig for both chains, confirm same.
        assertEq(
            chainDeployOutput1.systemConfigProxy.disputeGameFactory(),
            chainDeployOutput2.systemConfigProxy.disputeGameFactory(),
            "DisputeGameFactory mismatch"
        );

        // Extract the DisputeGameFactory now that we know it's the same on both chains.
        IDisputeGameFactory disputeGameFactory =
            IDisputeGameFactory(chainDeployOutput1.systemConfigProxy.disputeGameFactory());

        // Check that the respected game type is the Super Cannon game type.
        assertEq(
            anchorStateRegistry.respectedGameType().raw(),
            GameTypes.SUPER_PERMISSIONED_CANNON.raw(),
            "Super Permissioned Cannon game type mismatch"
        );

        // Grab the ETHLockbox from the SystemConfig for both chains, confirm same.
        assertEq(address(optimismPortal1.ethLockbox()), address(optimismPortal2.ethLockbox()), "ETHLockbox mismatch");

        // Extract the ETHLockbox now that we know it's the same on both chains.
        IETHLockbox ethLockbox = optimismPortal1.ethLockbox();

        // Check that the ETHLockbox was migrated correctly.
        assertGt(address(ethLockbox).balance, 0, "ETHLockbox balance is zero");
        assertTrue(ethLockbox.authorizedPortals(optimismPortal1), "ETHLockbox does not have portal 1 authorized");
        assertTrue(ethLockbox.authorizedPortals(optimismPortal2), "ETHLockbox does not have portal 2 authorized");

        // Check that the starting anchor root is the same as the input.
        (Hash root, uint256 l2SequenceNumber) = anchorStateRegistry.getAnchorRoot();
        assertEq(root.raw(), input.startingAnchorRoot.root.raw(), "Starting anchor root mismatch");
        assertEq(
            l2SequenceNumber,
            input.startingAnchorRoot.l2SequenceNumber,
            "Starting anchor root L2 sequence number mismatch"
        );

        // Check that the DisputeGameFactory has implementation for the Permissioned game.
        assertEq(
            disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON).gameType().raw(),
            GameTypes.SUPER_PERMISSIONED_CANNON.raw(),
            "Super Permissioned Cannon game type not set properly"
        );
        assertEq(
            disputeGameFactory.initBonds(GameTypes.SUPER_PERMISSIONED_CANNON),
            input.gameParameters.initBond,
            "Super Permissioned Cannon init bond mismatch"
        );

        // Check that the DisputeGameFactory does not have an implementation for the regular game.
        assertEq(
            address(disputeGameFactory.gameImpls(GameTypes.SUPER_CANNON)),
            address(0),
            "Super Cannon game type set when it should not be"
        );
        assertEq(disputeGameFactory.initBonds(GameTypes.SUPER_CANNON), 0, "Super Cannon init bond mismatch");

        // Check that the Super Permissioned Cannon game has the correct parameters.
        IDisputeGame superPdgImpl = disputeGameFactory.gameImpls(GameTypes.SUPER_PERMISSIONED_CANNON);
        ISuperPermissionedDisputeGame superPdg = ISuperPermissionedDisputeGame(address(superPdgImpl));
        assertEq(superPdg.proposer(), input.gameParameters.proposer);
        assertEq(superPdg.challenger(), input.gameParameters.challenger);
        assertEq(superPdg.maxGameDepth(), input.gameParameters.maxGameDepth);
        assertEq(superPdg.splitDepth(), input.gameParameters.splitDepth);
        assertEq(superPdg.clockExtension().raw(), input.gameParameters.clockExtension.raw());
        assertEq(superPdg.maxClockDuration().raw(), input.gameParameters.maxClockDuration.raw());
        assertEq(superPdg.absolutePrestate().raw(), absolutePrestate1.raw());
    }

    /// @notice Tests that the migration function reverts when the ProxyAdmin owners are
    ///         mismatched.
    function test_migrate_mismatchedProxyAdminOwners_reverts() public {
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Mock out the owners of the ProxyAdmins to be different.
        vm.mockCall(
            address(input.opChainConfigs[0].proxyAdmin),
            abi.encodeCall(IProxyAdmin.owner, ()),
            abi.encode(address(1234))
        );
        vm.mockCall(
            address(input.opChainConfigs[1].proxyAdmin),
            abi.encodeCall(IProxyAdmin.owner, ()),
            abi.encode(address(5678))
        );

        // Execute the migration.
        _doMigration(
            input, OPContractsManagerInteropMigrator.OPContractsManagerInteropMigrator_ProxyAdminOwnerMismatch.selector
        );
    }

    /// @notice Tests that the migration function reverts when the absolute prestates are
    ///         mismatched.
    function test_migrate_mismatchedAbsolutePrestates_reverts() public {
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Set the prestates to be different.
        input.opChainConfigs[0].absolutePrestate = absolutePrestate1;
        input.opChainConfigs[0].absolutePrestate = absolutePrestate2;

        // Execute the migration.
        _doMigration(
            input, OPContractsManagerInteropMigrator.OPContractsManagerInteropMigrator_AbsolutePrestateMismatch.selector
        );
    }

    /// @notice Tests that the migration function reverts when the SuperchainConfig addresses are
    ///         mismatched.
    function test_migrate_mismatchedSuperchainConfig_reverts() public {
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Mock out the SuperchainConfig addresses to be different.
        vm.mockCall(
            address(chainDeployOutput1.optimismPortalProxy),
            abi.encodeCall(IOptimismPortal2.superchainConfig, ()),
            abi.encode(address(1234))
        );
        vm.mockCall(
            address(chainDeployOutput2.optimismPortalProxy),
            abi.encodeCall(IOptimismPortal2.superchainConfig, ()),
            abi.encode(address(5678))
        );

        // Execute the migration.
        _doMigration(
            input, OPContractsManagerInteropMigrator.OPContractsManagerInteropMigrator_SuperchainConfigMismatch.selector
        );
    }

    function test_migrate_zerosOutCannonKonaGameTypes_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.CANNON_KONA);
        IOPContractsManagerInteropMigrator.MigrateInput memory input = _getDefaultInput();

        // Grab the existing DisputeGameFactory for each chain.
        IDisputeGameFactory oldDisputeGameFactory1 =
            IDisputeGameFactory(payable(chainDeployOutput1.systemConfigProxy.disputeGameFactory()));
        IDisputeGameFactory oldDisputeGameFactory2 =
            IDisputeGameFactory(payable(chainDeployOutput2.systemConfigProxy.disputeGameFactory()));
        // Ensure cannon kona games have implementations
        oldDisputeGameFactory1.setImplementation(GameTypes.CANNON_KONA, IDisputeGame(address(1)));
        oldDisputeGameFactory2.setImplementation(GameTypes.CANNON_KONA, IDisputeGame(address(1)));
        oldDisputeGameFactory1.setImplementation(GameTypes.SUPER_CANNON_KONA, IDisputeGame(address(2)));
        oldDisputeGameFactory2.setImplementation(GameTypes.SUPER_CANNON_KONA, IDisputeGame(address(2)));

        // Execute the migration.
        _doMigration(input);

        // Assert that the old game implementations are now zeroed out.
        _assertOldGamesZeroed(oldDisputeGameFactory1);
        _assertOldGamesZeroed(oldDisputeGameFactory2);
    }
}

/// @title OPContractsManager_Deploy_Test
/// @notice Tests the `deploy` function of the `OPContractsManager` contract.
/// @dev Unlike other test suites, we intentionally do not inherit from CommonTest or Setup. This
///      is because OPContractsManager acts as a deploy script, so we start from a clean slate here
///      and work OPContractsManager's deployment into the existing test setup, instead of using
///      the existing test setup to deploy OPContractsManager. We do however inherit from
///      DeployOPChain_TestBase so we can use its setup to deploy the implementations similarly
///      to how a real deployment would happen.
contract OPContractsManager_Deploy_Test is DeployOPChain_TestBase {
    using stdStorage for StdStorage;

    event Deployed(uint256 indexed l2ChainId, address indexed deployer, bytes deployOutput);

    // This helper function is used to convert the input struct type defined in DeployOPChain.s.sol
    // to the input struct type defined in OPContractsManager.sol.
    function toOPCMDeployInput(Types.DeployOPChainInput memory _doi)
        internal
        returns (IOPContractsManager.DeployInput memory)
    {
        bytes memory startingAnchorRoot = new DeployOPChain().startingAnchorRoot();
        return IOPContractsManager.DeployInput({
            roles: IOPContractsManager.Roles({
                opChainProxyAdminOwner: _doi.opChainProxyAdminOwner,
                systemConfigOwner: _doi.systemConfigOwner,
                batcher: _doi.batcher,
                unsafeBlockSigner: _doi.unsafeBlockSigner,
                proposer: _doi.proposer,
                challenger: _doi.challenger
            }),
            basefeeScalar: _doi.basefeeScalar,
            blobBasefeeScalar: _doi.blobBaseFeeScalar,
            l2ChainId: _doi.l2ChainId,
            startingAnchorRoot: startingAnchorRoot,
            saltMixer: _doi.saltMixer,
            gasLimit: _doi.gasLimit,
            disputeGameType: _doi.disputeGameType,
            disputeAbsolutePrestate: _doi.disputeAbsolutePrestate,
            disputeMaxGameDepth: _doi.disputeMaxGameDepth,
            disputeSplitDepth: _doi.disputeSplitDepth,
            disputeClockExtension: _doi.disputeClockExtension,
            disputeMaxClockDuration: _doi.disputeMaxClockDuration
        });
    }

    /// @notice Helper function to create a permissioned game through the factory
    function _createPermissionedGame(
        IDisputeGameFactory factory,
        address proposer,
        Claim claim,
        uint256 l2BlockNumber
    )
        internal
        returns (IPermissionedDisputeGame)
    {
        // Check if there's an init bond required for the game type
        uint256 initBond = factory.initBonds(GameTypes.PERMISSIONED_CANNON);

        // Fund the proposer if needed
        if (initBond > 0) {
            vm.deal(proposer, initBond);
        }

        // We use vm.startPrank to set both msg.sender and tx.origin to the proposer
        vm.startPrank(proposer, proposer);

        IDisputeGame gameProxy =
            factory.create{ value: initBond }(GameTypes.PERMISSIONED_CANNON, claim, abi.encode(bytes32(l2BlockNumber)));

        vm.stopPrank();

        return IPermissionedDisputeGame(address(gameProxy));
    }

    function test_deploy_l2ChainIdEqualsZero_reverts() public {
        IOPContractsManager.DeployInput memory input = toOPCMDeployInput(deployOPChainInput);
        input.l2ChainId = 0;

        vm.expectRevert(IOPContractsManager.InvalidChainId.selector);
        opcm.deploy(input);
    }

    function test_deploy_l2ChainIdEqualsCurrentChainId_reverts() public {
        IOPContractsManager.DeployInput memory input = toOPCMDeployInput(deployOPChainInput);
        input.l2ChainId = block.chainid;

        vm.expectRevert(IOPContractsManager.InvalidChainId.selector);
        opcm.deploy(input);
    }

    function test_deploy_succeeds() public {
        vm.expectEmit(true, true, true, false); // TODO precompute the expected `deployOutput`.
        emit Deployed(deployOPChainInput.l2ChainId, address(this), bytes(""));
        opcm.deploy(toOPCMDeployInput(deployOPChainInput));
    }

    /// @notice Test that deploy sets the permissioned dispute game implementation
    function test_deployPermissioned_succeeds() public {
        bool isV2 = isDevFeatureEnabled(DevFeatures.DEPLOY_V2_DISPUTE_GAMES);

        // Sanity-check setup is consistent with devFeatures flag
        IOPContractsManager.Implementations memory impls = opcm.implementations();
        address pdgImpl = address(impls.permissionedDisputeGameV2Impl);
        address fdgImpl = address(impls.faultDisputeGameV2Impl);
        if (isV2) {
            assertFalse(pdgImpl == address(0), "PDG implementation address should be non-zero");
            assertFalse(fdgImpl == address(0), "FDG implementation address should be non-zero");
        } else {
            assertTrue(pdgImpl == address(0), "PDG implementation address should be zero");
            assertTrue(fdgImpl == address(0), "FDG implementation address should be zero");
        }

        // Run OPCM.deploy
        IOPContractsManager.DeployInput memory opcmInput = toOPCMDeployInput(deployOPChainInput);
        IOPContractsManager.DeployOutput memory opcmOutput = opcm.deploy(opcmInput);

        // Verify that the DisputeGameFactory has registered an implementation for the PERMISSIONED_CANNON game type
        address expectedPDGAddress = isV2 ? pdgImpl : address(opcmOutput.permissionedDisputeGame);
        address actualPDGAddress = address(opcmOutput.disputeGameFactoryProxy.gameImpls(GameTypes.PERMISSIONED_CANNON));
        assertNotEq(actualPDGAddress, address(0), "DisputeGameFactory should have a registered PERMISSIONED_CANNON");
        assertEq(actualPDGAddress, address(expectedPDGAddress), "PDG address should match");

        // Create a game proxy to test immutable fields
        Claim claim = Claim.wrap(bytes32(uint256(9876)));
        uint256 l2BlockNumber = uint256(123);
        IPermissionedDisputeGame pdg =
            _createPermissionedGame(opcmOutput.disputeGameFactoryProxy, opcmInput.roles.proposer, claim, l2BlockNumber);

        // Verify immutable fields on the game proxy
        // Constructor args
        assertEq(pdg.gameType().raw(), GameTypes.PERMISSIONED_CANNON.raw(), "Game type should match");
        assertEq(pdg.clockExtension().raw(), opcmInput.disputeClockExtension.raw(), "Clock extension should match");
        assertEq(
            pdg.maxClockDuration().raw(), opcmInput.disputeMaxClockDuration.raw(), "Max clock duration should match"
        );
        assertEq(pdg.splitDepth(), opcmInput.disputeSplitDepth, "Split depth should match");
        assertEq(pdg.maxGameDepth(), opcmInput.disputeMaxGameDepth, "Max game depth should match");
        // Clone-with-immutable-args
        assertEq(pdg.gameCreator(), opcmInput.roles.proposer, "Game creator should match");
        assertEq(pdg.rootClaim().raw(), claim.raw(), "Claim should match");
        assertEq(pdg.l1Head().raw(), blockhash(block.number - 1), "L1 head should match");
        assertEq(pdg.l2BlockNumber(), l2BlockNumber, "L2 Block number should match");
        assertEq(
            pdg.absolutePrestate().raw(),
            opcmInput.disputeAbsolutePrestate.raw(),
            "Absolute prestate should match input"
        );
        assertEq(address(pdg.vm()), address(impls.mipsImpl), "VM should match MIPS implementation");
        assertEq(address(pdg.anchorStateRegistry()), address(opcmOutput.anchorStateRegistryProxy), "ASR should match");
        assertEq(address(pdg.weth()), address(opcmOutput.delayedWETHPermissionedGameProxy), "WETH should match");
        assertEq(pdg.l2ChainId(), opcmInput.l2ChainId, "L2 chain ID should match");
        // For permissioned game, check proposer and challenger
        assertEq(pdg.proposer(), opcmInput.roles.proposer, "Proposer should match");
        assertEq(pdg.challenger(), opcmInput.roles.challenger, "Challenger should match");
    }
}

/// @title OPContractsManager_Version_Test
/// @notice Tests the `version` function of the `OPContractsManager` contract.
contract OPContractsManager_Version_Test is OPContractsManager_TestInit {
    function test_semver_works() public view {
        assertNotEq(abi.encode(opcm.version()), abi.encode(0));
    }
}
