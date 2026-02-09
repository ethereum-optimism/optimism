// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";
import { PastUpgrades } from "test/setup/PastUpgrades.sol";
import { BatchUpgrader } from "test/L1/opcm/helpers/BatchUpgrader.sol";

// Libraries
import { Config } from "scripts/libraries/Config.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Claim, Hash } from "src/dispute/lib/LibUDT.sol";
import { GameType, GameTypes, Proposal } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Features } from "src/libraries/Features.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";
import { IOPContractsManagerMigrator } from "interfaces/L1/opcm/IOPContractsManagerMigrator.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IOptimismPortalInterop } from "interfaces/L1/IOptimismPortalInterop.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";

/// @title OPContractsManagerV2_TestInit
/// @notice Base test initialization contract for OPContractsManagerV2.
contract OPContractsManagerV2_TestInit is CommonTest {
    /// @notice Fake prestate for Cannon games.
    Claim cannonPrestate = Claim.wrap(bytes32(keccak256("cannonPrestate")));

    /// @notice Fake prestate for Cannon Kona games.
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice Special string constant used to indicate that we expect a revert without any data.
    bytes public constant EXPECT_REVERT_WITHOUT_DATA = bytes("EXPECT_REVERT_WITHOUT_DATA");

    /// @notice Buffer percentage (relative to EIP-7825 gas limit) allowed for deployments.
    uint256 public constant DEPLOY_GAS_BUFFER_PERCENTAGE = 80; // 80%

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);
    }

    /// @notice Helper function that runs an OPCM V2 deploy, asserts that the deploy was successful,
    ///         and runs post-deploy standard validator checks.
    /// @param _opcm The OPCM contract to use for deployment.
    /// @param _deployConfig The full config for the deployment.
    /// @param _revertBytes The bytes of the revert to expect (empty if no revert expected).
    /// @param _expectedValidatorErrors The StandardValidator errors to expect.
    /// @return cts_ The deployed chain contracts.
    function _runOpcmV2DeployAndChecks(
        IOPContractsManagerV2 _opcm,
        IOPContractsManagerV2.FullConfig memory _deployConfig,
        bytes memory _revertBytes,
        string memory _expectedValidatorErrors
    )
        internal
        returns (IOPContractsManagerV2.ChainContracts memory cts_)
    {
        // Grab the proposer and challenger from deploy config for validator.
        address deployProposer;
        address deployChallenger;
        for (uint256 i = 0; i < _deployConfig.disputeGameConfigs.length; i++) {
            if (_deployConfig.disputeGameConfigs[i].gameType.raw() == GameTypes.PERMISSIONED_CANNON.raw()) {
                IOPContractsManagerUtils.PermissionedDisputeGameConfig memory parsedArgs = abi.decode(
                    _deployConfig.disputeGameConfigs[i].gameArgs,
                    (IOPContractsManagerUtils.PermissionedDisputeGameConfig)
                );
                deployProposer = parsedArgs.proposer;
                deployChallenger = parsedArgs.challenger;
                break;
            }
        }

        // Expect the revert if one is specified.
        if (_revertBytes.length > 0) {
            if (keccak256(_revertBytes) == keccak256(EXPECT_REVERT_WITHOUT_DATA)) {
                // nosemgrep: sol-safety-expectrevert-no-args
                vm.expectRevert();
            } else {
                vm.expectRevert(_revertBytes);
            }
        }

        // Execute the V2 deploy.
        cts_ = _opcm.deploy(_deployConfig);

        // Return early if a revert was expected. Otherwise we'll get errors below.
        if (_revertBytes.length > 0) {
            return cts_;
        }

        // Less than the buffer percentage of the EIP-7825 gas limit to account for the gas used
        // by using Safe.
        uint256 fusakaLimit = 2 ** 24;
        VmSafe.Gas memory gas = vm.lastCallGas();
        assertLt(
            gas.gasTotalUsed,
            fusakaLimit * DEPLOY_GAS_BUFFER_PERCENTAGE / 100,
            string.concat(
                "Deploy exceeds gas target of ", vm.toString(DEPLOY_GAS_BUFFER_PERCENTAGE), "% of 2**24 (EIP-7825)"
            )
        );

        // Coverage changes bytecode, so we get various errors. We can safely ignore the result of
        // the standard validator in the coverage case.
        if (vm.isContext(VmSafe.ForgeContext.Coverage)) {
            return cts_;
        }

        // Create validationOverrides for the newly deployed chain.
        IOPContractsManagerStandardValidator.ValidationOverrides memory validationOverrides =
        IOPContractsManagerStandardValidator.ValidationOverrides({
            l1PAOMultisig: _deployConfig.proxyAdminOwner,
            challenger: deployChallenger
        });

        // Grab the validator before we do the error assertion.
        IOPContractsManagerStandardValidator validator = _opcm.opcmStandardValidator();

        // Expect validator errors if the user provides them.
        if (bytes(_expectedValidatorErrors).length > 0) {
            vm.expectRevert(
                bytes(
                    string.concat(
                        "OPContractsManagerStandardValidator: OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,",
                        _expectedValidatorErrors
                    )
                )
            );
        }

        // Run the StandardValidator checks on the newly deployed chain.
        validator.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: cts_.systemConfig,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                l2ChainID: _deployConfig.l2ChainId,
                proposer: deployProposer
            }),
            false,
            validationOverrides
        );

        return cts_;
    }

    /// @notice Executes a V2 deploy and checks the results.
    /// @param _deployConfig The full config for the deployment.
    /// @return The deployed chain contracts.
    function runDeployV2(IOPContractsManagerV2.FullConfig memory _deployConfig)
        public
        returns (IOPContractsManagerV2.ChainContracts memory)
    {
        return _runOpcmV2DeployAndChecks(opcmV2, _deployConfig, bytes(""), "");
    }

    /// @notice Executes a V2 deploy and expects reverts.
    /// @param _deployConfig The full config for the deployment.
    /// @param _revertBytes The bytes of the revert to expect.
    /// @return The deployed chain contracts.
    function runDeployV2(
        IOPContractsManagerV2.FullConfig memory _deployConfig,
        bytes memory _revertBytes
    )
        public
        returns (IOPContractsManagerV2.ChainContracts memory)
    {
        return _runOpcmV2DeployAndChecks(opcmV2, _deployConfig, _revertBytes, "");
    }

    /// @notice Executes a V2 deploy and expects reverts with validator errors.
    /// @param _deployConfig The full config for the deployment.
    /// @param _revertBytes The bytes of the revert to expect.
    /// @param _expectedValidatorErrors The StandardValidator errors to expect.
    /// @return The deployed chain contracts.
    function runDeployV2(
        IOPContractsManagerV2.FullConfig memory _deployConfig,
        bytes memory _revertBytes,
        string memory _expectedValidatorErrors
    )
        public
        returns (IOPContractsManagerV2.ChainContracts memory)
    {
        return _runOpcmV2DeployAndChecks(opcmV2, _deployConfig, _revertBytes, _expectedValidatorErrors);
    }
}

/// @title OPContractsManagerV2_Upgrade_TestInit
/// @notice Test initialization contract for OPContractsManagerV2 upgrade functions.
contract OPContractsManagerV2_Upgrade_TestInit is OPContractsManagerV2_TestInit {
    // The Upgraded event emitted by the Proxy contract.
    event Upgraded(address indexed implementation);

    /// @notice Chain ID for the L2 chain being upgraded in this test.
    uint256 l2ChainId;

    /// @notice Address of the ProxyAdmin owner for the chain being upgraded.
    address chainPAO;

    /// @notice Address of the Superchain ProxyAdmin owner.
    address superchainPAO;

    /// @notice Name of the chain being forked.
    string public opChain = Config.forkOpChain();

    /// @notice Default v2 upgrade input.
    IOPContractsManagerV2.UpgradeInput v2UpgradeInput;

    /// @notice Buffer percentage (relative to EIP-7825 gas limit) allowed for upgrades.
    uint256 public constant UPGRADE_GAS_BUFFER_PERCENTAGE = 50; // 50%

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.disableUpgradedFork();
        super.setUp();

        skipIfNotForkTest("OPContractsManagerV2_Upgrade_TestInit: only runs in forked tests");
        skipIfOpsRepoTest("OPContractsManagerV2_Upgrade_TestInit: skipped in superchain-ops");

        // Set the chain PAO.
        chainPAO = proxyAdmin.owner();
        vm.label(chainPAO, "ProxyAdmin Owner");

        // Set the SuperchainConfig PAO.
        superchainPAO = IProxyAdmin(EIP1967Helper.getAdmin(address(superchainConfig))).owner();
        vm.label(superchainPAO, "SuperchainConfig ProxyAdmin Owner");

        // Grab and set the L2 chain ID.
        l2ChainId = uint256(uint160(address(artifacts.mustGetAddress("L2ChainId"))));

        // Set up the default v2 upgrade input dispute game configs.
        address initialChallengerForV2 = DisputeGames.permissionedGameChallenger(disputeGameFactory);
        address initialProposerForV2 = DisputeGames.permissionedGameProposer(disputeGameFactory);
        v2UpgradeInput.systemConfig = systemConfig;
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: disputeGameFactory.initBonds(GameTypes.CANNON),
                gameType: GameTypes.CANNON,
                gameArgs: abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonPrestate }))
            })
        );
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: disputeGameFactory.initBonds(GameTypes.PERMISSIONED_CANNON),
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: cannonPrestate,
                        proposer: initialProposerForV2,
                        challenger: initialChallengerForV2
                    })
                )
            })
        );
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: disputeGameFactory.initBonds(GameTypes.CANNON_KONA),
                gameType: GameTypes.CANNON_KONA,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate })
                )
            })
        );

        // Allow the DelayedWETH proxy to be (re)deployed during upgrades if it is missing.
        v2UpgradeInput.extraInstructions.push(
            IOPContractsManagerUtils.ExtraInstruction({ key: "PermittedProxyDeployment", data: bytes("DelayedWETH") })
        );
    }

    /// @notice Helper function that runs an OPCM V2 upgrade, asserts that the upgrade was successful,
    ///         and runs post-upgrade smoke tests.
    /// @param _opcm The OPCM contract to reference for shared components.
    /// @param _delegateCaller The address of the delegate caller to use for superchain upgrade.
    /// @param _revertBytes The bytes of the revert to expect.
    /// @param _expectedValidatorErrors The StandardValidator errors to expect.
    function _runOpcmV2UpgradeAndChecks(
        IOPContractsManagerV2 _opcm,
        address _delegateCaller,
        bytes memory _revertBytes,
        string memory _expectedValidatorErrors
    )
        internal
    {
        // Grab some values before we upgrade, to be checked later
        address initialChallenger = DisputeGames.permissionedGameChallenger(disputeGameFactory);
        address initialProposer = DisputeGames.permissionedGameProposer(disputeGameFactory);

        // Execute the SuperchainConfig upgrade.
        prankDelegateCall(superchainPAO);
        (bool success, bytes memory reason) = address(opcmV2).delegatecall(
            abi.encodeCall(
                IOPContractsManagerV2.upgradeSuperchain,
                (
                    IOPContractsManagerV2.SuperchainUpgradeInput({
                        superchainConfig: superchainConfig,
                        extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
                    })
                )
            )
        );
        if (success == false) {
            // Only acceptable revert reason is the SuperchainConfig already being up to date. This
            // try/catch is better than checking the version via the implementations struct because
            // the implementations struct interface can change between OPCM versions which would
            // cause the test to break and be a pain to resolve.
            assertTrue(
                bytes4(reason) == IOPContractsManagerUtils.OPContractsManagerUtils_DowngradeNotAllowed.selector,
                "Revert reason other than DowngradeNotAllowed"
            );
        }

        // Expect the revert if one is specified.
        if (_revertBytes.length > 0) {
            if (keccak256(_revertBytes) == keccak256(EXPECT_REVERT_WITHOUT_DATA)) {
                // nosemgrep: sol-safety-expectrevert-no-args
                vm.expectRevert();
            } else {
                vm.expectRevert(_revertBytes);
            }
        }

        // Execute the V2 chain upgrade via delegate caller.
        prankDelegateCall(_delegateCaller);
        (bool upgradeSuccess,) =
            address(opcmV2).delegatecall(abi.encodeCall(IOPContractsManagerV2.upgrade, (v2UpgradeInput)));
        assertTrue(upgradeSuccess, "upgrade failed");

        // Return early if a revert was expected. Otherwise we'll get errors below.
        if (_revertBytes.length > 0) {
            return;
        }

        // Less than the buffer percentage of the EIP-7825 gas limit to account for the gas used
        // by using Safe.
        uint256 fusakaLimit = 2 ** 24;
        VmSafe.Gas memory gas = vm.lastCallGas();
        assertLt(
            gas.gasTotalUsed,
            fusakaLimit * UPGRADE_GAS_BUFFER_PERCENTAGE / 100,
            string.concat(
                "Upgrade exceeds gas target of ", vm.toString(UPGRADE_GAS_BUFFER_PERCENTAGE), "% of 2**24 (EIP-7825)"
            )
        );

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
            l1PAOMultisig: v2UpgradeInput.systemConfig.proxyAdminOwner(),
            challenger: initialChallenger
        });

        // Grab the validator before we do the error assertion because otherwise the assertion will
        // try to apply to this function call instead.
        IOPContractsManagerStandardValidator validator = _opcm.opcmStandardValidator();

        // Expect validator errors if the user provides them. We always expect the L1PAOMultisig
        // and Challenger overrides so we don't need to repeat them here.
        if (bytes(_expectedValidatorErrors).length > 0) {
            vm.expectRevert(
                bytes(
                    string.concat(
                        "OPContractsManagerStandardValidator: OVERRIDES-L1PAOMULTISIG,OVERRIDES-CHALLENGER,",
                        _expectedValidatorErrors
                    )
                )
            );
        }

        // Run the StandardValidator checks.
        validator.validateWithOverrides(
            IOPContractsManagerStandardValidator.ValidationInputDev({
                sysCfg: v2UpgradeInput.systemConfig,
                cannonPrestate: cannonPrestate.raw(),
                cannonKonaPrestate: cannonKonaPrestate.raw(),
                l2ChainID: l2ChainId,
                proposer: initialProposer
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
    function runPastUpgrades(address _delegateCaller) internal {
        PastUpgrades.runPastUpgrades(_delegateCaller, v2UpgradeInput.systemConfig, superchainConfig, disputeGameFactory);
    }

    /// @notice Executes the current V2 upgrade and checks the results.
    /// @param _delegateCaller The address of the delegate caller to use for the superchain upgrade.
    function runCurrentUpgradeV2(address _delegateCaller) public {
        _runOpcmV2UpgradeAndChecks(opcmV2, _delegateCaller, bytes(""), "");
    }

    /// @notice Executes the current V2 upgrade and expects reverts.
    /// @param _delegateCaller The address of the delegate caller to use for the superchain upgrade.
    /// @param _revertBytes The bytes of the revert to expect.
    function runCurrentUpgradeV2(address _delegateCaller, bytes memory _revertBytes) public {
        _runOpcmV2UpgradeAndChecks(opcmV2, _delegateCaller, _revertBytes, "");
    }

    /// @notice Executes the current V2 upgrade and expects reverts.
    /// @param _delegateCaller The address of the delegate caller to use for the superchain upgrade.
    /// @param _revertBytes The bytes of the revert to expect.
    /// @param _expectedValidatorErrors The StandardValidator errors to expect.
    function runCurrentUpgradeV2(
        address _delegateCaller,
        bytes memory _revertBytes,
        string memory _expectedValidatorErrors
    )
        public
    {
        _runOpcmV2UpgradeAndChecks(opcmV2, _delegateCaller, _revertBytes, _expectedValidatorErrors);
    }

    /// @notice Extracts the absolute prestate embedded in a dispute game config.
    /// @param _gameType Game type to inspect.
    /// @return prestate_ The absolute prestate stored in the factory's game args.
    function _gameArgsAbsolutePrestate(GameType _gameType) internal view returns (bytes32 prestate_) {
        bytes memory args = disputeGameFactory.gameArgs(_gameType);
        if (args.length == 0) {
            return bytes32(0);
        }
        assembly {
            prestate_ := mload(add(args, 0x20))
        }
    }
}

/// @title OPContractsManagerV2_Upgrade_Test
/// @notice Tests OPContractsManagerV2.upgrade
contract OPContractsManagerV2_Upgrade_Test is OPContractsManagerV2_Upgrade_TestInit {
    /// @notice Sets up the test.
    function setUp() public override {
        super.setUp();

        // Run all past upgrades.
        runPastUpgrades(chainPAO);
    }

    /// @notice Tests that the upgrade function succeeds when executed normally.
    function test_upgrade_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // Run the upgrade test and checks
        runCurrentUpgradeV2(chainPAO);
    }

    /// @notice Tests that the upgrade function reverts if not called by the correct ProxyAdmin
    ///         owner address.
    function test_upgrade_notProxyAdminOwner_reverts() public {
        address delegateCaller = makeAddr("delegateCaller");

        assertNotEq(superchainProxyAdmin.owner(), delegateCaller);
        assertNotEq(proxyAdmin.owner(), delegateCaller);

        runCurrentUpgradeV2(delegateCaller, "Ownable: caller is not the owner");
    }

    /// @notice Tests that the upgrade function reverts when the superchainConfig is not at the
    ///         expected target version.
    function test_upgrade_superchainConfigNeedsUpgrade_reverts() public {
        // Force the SuperchainConfig to return an obviously outdated version.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("0.0.0"));

        // Try upgrading an OPChain without upgrading its superchainConfig.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO,
            abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_SuperchainConfigNeedsUpgrade.selector)
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the SystemConfig address is zero.
    function test_upgrade_zeroSystemConfig_reverts() public {
        v2UpgradeInput.systemConfig = ISystemConfig(address(0));

        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidUpgradeInput.selector)
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the user does not provide a game
    ///         config for each valid game type.
    function test_upgrade_missingGameConfigs_reverts() public {
        // Delete the Permissionless game configuration.
        delete v2UpgradeInput.disputeGameConfigs[1];

        // Expect upgrade to revert.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the user provides the game configs
    ///         in the wrong order.
    function test_upgrade_wrongGameConfigOrder_reverts() public {
        // Swap the game config order.
        IOPContractsManagerUtils.DisputeGameConfig memory temp = v2UpgradeInput.disputeGameConfigs[0];
        v2UpgradeInput.disputeGameConfigs[0] = v2UpgradeInput.disputeGameConfigs[1];
        v2UpgradeInput.disputeGameConfigs[1] = temp;

        // Expect upgrade to revert due to invalid game config order.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the user wants to disable the
    ///         PermissionedDisputeGame.
    function test_upgrade_disabledPermissionedGame_reverts() public {
        // Disable the PermissionedDisputeGame.
        v2UpgradeInput.disputeGameConfigs[1].enabled = false;

        // Expect upgrade to revert due to missing game config.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that the V2 upgrade function rejects the ALL sentinel in permitted proxy
    ///         deployments.
    function test_upgrade_allPermittedProxyDeployments_reverts() public {
        delete v2UpgradeInput.extraInstructions;
        v2UpgradeInput.extraInstructions.push(
            IOPContractsManagerUtils.ExtraInstruction({ key: "PermitProxyDeployment", data: abi.encode("ALL") })
        );

        // Expect upgrade to revert due to invalid upgrade input.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO,
            abi.encodeWithSelector(
                IOPContractsManagerV2.OPContractsManagerV2_InvalidUpgradeInstruction.selector, "PermitProxyDeployment"
            )
        );
    }

    /// @notice Tests that the V2 upgrade function reverts if a permitted proxy deployment is
    ///         required but missing.
    function test_upgrade_missingPermittedProxyDeployment_reverts() public {
        delete v2UpgradeInput.extraInstructions;

        // Simulate a missing DelayedWETH proxy so the upgrade path would need to deploy it.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCallRevert(address(systemConfig), abi.encodeWithSelector(ISystemConfig.delayedWETH.selector), "");

        // Expect the upgrade to revert because the DelayedWETH proxy must load but the user did not permit
        // redeployment.
        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO,
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ProxyMustLoad.selector, "DelayedWETH"
            )
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the function that attempts to load
    ///         an existing proxy returns data that isn't an abi-encoded address.
    /// @param _len Length of the data to generate.
    function testFuzz_upgrade_proxyLoadBadReturn_reverts(uint8 _len) public {
        // Ensure we do not produce a 32-byte payload, which would be interpreted as a valid
        // abi-encoded address and could change the revert reason.
        vm.assume(_len != 32);

        // Build an arbitrary bytes payload of length `_len`.
        bytes memory bad = new bytes(_len);
        for (uint256 i = 0; i < bad.length; i++) {
            bad[i] = bytes1(uint8(0xAA));
        }

        // Mock the first proxy load source call to succeed but return a payload with a length
        // not equal to 32 bytes, triggering OPContractsManagerUtils_ProxyLoadMustLoad.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), bad);

        // Expect a revert without any data (due to abi decoding failure).
        runCurrentUpgradeV2(chainPAO, EXPECT_REVERT_WITHOUT_DATA);
    }

    /// @notice Tests that the V2 upgrade function reverts when the function that attempts to load
    ///         an existing proxy returns the zero address but we asked it to load.
    function test_upgrade_proxyMustLoadButZeroAddress_reverts() public {
        // Mock the first proxy load to succeed and return address(0) with 32 bytes,
        // which triggers OPContractsManagerUtils_ProxyMustLoad since _mustLoad is true in upgrade.
        vm.mockCall(
            address(systemConfig), abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), abi.encode(address(0))
        );

        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO,
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ProxyMustLoad.selector, "L1CrossDomainMessenger"
            )
        );
    }

    /// @notice Tests that the V2 upgrade function reverts when the function that attempts to load
    ///         an existing proxy returns an error but we asked it to load.
    function test_upgrade_proxyMustLoadButReverts_reverts() public {
        // Mock the first proxy load source to revert, which with _mustLoad=true triggers
        // OPContractsManagerUtils_ProxyMustLoad.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCallRevert(address(systemConfig), abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()), bytes(""));

        // nosemgrep: sol-style-use-abi-encodecall
        runCurrentUpgradeV2(
            chainPAO,
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_ProxyMustLoad.selector, "L1CrossDomainMessenger"
            )
        );
    }

    /// @notice Tests that repeatedly upgrading can enable a previously disabled game type.
    function test_upgrade_enableGameType_succeeds() public {
        uint256 originalBond = disputeGameFactory.initBonds(GameTypes.CANNON);

        // First, disable Cannon and clear its bond so the factory entry is removed.
        v2UpgradeInput.disputeGameConfigs[0].enabled = false;
        v2UpgradeInput.disputeGameConfigs[0].initBond = 0;
        runCurrentUpgradeV2(chainPAO, hex"", "PLDG-10");
        assertEq(address(disputeGameFactory.gameImpls(GameTypes.CANNON)), address(0), "game impl not cleared");

        // Re-enable Cannon and restore its bond so that it is re-installed.
        v2UpgradeInput.disputeGameConfigs[0].enabled = true;
        v2UpgradeInput.disputeGameConfigs[0].initBond = originalBond;
        runCurrentUpgradeV2(chainPAO);
        assertEq(
            address(disputeGameFactory.gameImpls(GameTypes.CANNON)),
            opcmV2.implementations().faultDisputeGameImpl,
            "game impl not restored"
        );
        assertEq(disputeGameFactory.initBonds(GameTypes.CANNON), originalBond, "init bond not restored");
    }

    /// @notice Tests that disabling a game type removes it from the factory.
    function test_upgrade_disableGameType_succeeds() public {
        // Establish the baseline where Cannon is enabled.
        runCurrentUpgradeV2(chainPAO);
        assertEq(
            address(disputeGameFactory.gameImpls(GameTypes.CANNON)),
            opcmV2.implementations().faultDisputeGameImpl,
            "initial game impl mismatch"
        );

        // Disable Cannon and zero its bond, then ensure it is removed.
        v2UpgradeInput.disputeGameConfigs[0].enabled = false;
        v2UpgradeInput.disputeGameConfigs[0].initBond = 0;
        runCurrentUpgradeV2(chainPAO, hex"", "PLDG-10");
        assertEq(address(disputeGameFactory.gameImpls(GameTypes.CANNON)), address(0), "game impl not cleared");
        assertEq(disputeGameFactory.initBonds(GameTypes.CANNON), 0, "init bond not cleared");
        assertEq(disputeGameFactory.gameArgs(GameTypes.CANNON), bytes(""), "game args not cleared");
    }

    /// @notice Tests that the upgrade flow can update the Cannon and Permissioned prestate.
    function test_upgrade_updatePrestate_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // Run baseline upgrade and capture the current prestates.
        runCurrentUpgradeV2(chainPAO);
        assertEq(
            _gameArgsAbsolutePrestate(GameTypes.CANNON),
            Claim.unwrap(cannonPrestate),
            "baseline cannon prestate mismatch"
        );
        assertEq(
            _gameArgsAbsolutePrestate(GameTypes.PERMISSIONED_CANNON),
            Claim.unwrap(cannonPrestate),
            "baseline permissioned prestate mismatch"
        );

        // Prepare new prestates.
        Claim newPrestate = Claim.wrap(bytes32(keccak256("new cannon prestate")));
        cannonPrestate = newPrestate;

        // Update the dispute game configs to point at the new prestates.
        v2UpgradeInput.disputeGameConfigs[0].gameArgs =
            abi.encode(IOPContractsManagerUtils.FaultDisputeGameConfig({ absolutePrestate: newPrestate }));
        v2UpgradeInput.disputeGameConfigs[1].gameArgs = abi.encode(
            IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                absolutePrestate: newPrestate,
                proposer: DisputeGames.permissionedGameProposer(disputeGameFactory),
                challenger: DisputeGames.permissionedGameChallenger(disputeGameFactory)
            })
        );

        // Run the upgrade again and ensure prestates updated.
        runCurrentUpgradeV2(chainPAO);
        assertEq(_gameArgsAbsolutePrestate(GameTypes.CANNON), Claim.unwrap(newPrestate), "cannon prestate not updated");
        assertEq(
            _gameArgsAbsolutePrestate(GameTypes.PERMISSIONED_CANNON),
            Claim.unwrap(newPrestate),
            "permissioned prestate not updated"
        );
    }

    /// @notice INVARIANT: Upgrades must always work when the system is paused.
    ///         This test validates that the OPCMv2 upgrade function can execute successfully
    ///         even when the SuperchainConfig has the system globally paused. This is critical
    ///         because upgrades may be needed during incident response when the system is paused.
    function test_upgrade_whenPaused_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);

        // First, pause the system globally using the guardian.
        address guardian = superchainConfig.guardian();
        vm.prank(guardian);
        superchainConfig.pause(address(0));

        // Verify the system is actually paused.
        assertTrue(superchainConfig.paused(address(0)), "System should be globally paused");

        // Run the upgrade - this should succeed even while paused.
        // The StandardValidator will report SPRCFG-10 because the system is paused,
        // but the upgrade itself must complete successfully.
        runCurrentUpgradeV2(chainPAO, hex"", "SPRCFG-10");

        // Verify the system is still paused after the upgrade.
        assertTrue(superchainConfig.paused(address(0)), "System should still be paused after upgrade");
    }
}

/// @title OPContractsManagerV2_IsPermittedUpgradeSequence_Test
/// @notice Tests OPContractsManagerV2.isPermittedUpgradeSequence
contract OPContractsManagerV2_IsPermittedUpgradeSequence_Test is OPContractsManagerV2_TestInit {
    /// @notice Tests that the upgrade sequence is permitted when using the same OPCM (re-running upgrade).
    function test_isPermittedUpgradeSequence_sameOPCM_succeeds() public {
        // Mock the OPCM version to be >= 8.0.0 so the check activates.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("8.0.0"));

        // Mock lastUsedOPCM to return the same OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(address(opcmV2)));

        // Should return true because it's the same OPCM.
        assertTrue(opcmV2.isPermittedUpgradeSequence(systemConfig), "same OPCM should be permitted");
    }

    /// @notice Tests that the upgrade sequence is permitted when upgrading to same major but higher minor.
    function test_isPermittedUpgradeSequence_sameMajorHigherMinor_succeeds() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 8.1.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("8.1.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 8.0.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("8.0.0"));

        // Should return true because 8.1.0 > 8.0.0 (same major, higher minor).
        assertTrue(opcmV2.isPermittedUpgradeSequence(systemConfig), "same major higher minor should be permitted");
    }

    /// @notice Tests that the upgrade sequence is permitted when upgrading to the next major version.
    function test_isPermittedUpgradeSequence_nextMajorVersion_succeeds() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 9.0.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("9.0.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 8.2.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("8.2.0"));

        // Should return true because 9.0.0 is the next major after 8.x.x.
        assertTrue(opcmV2.isPermittedUpgradeSequence(systemConfig), "next major version should be permitted");
    }

    /// @notice Tests that the upgrade sequence is not permitted when downgrading (same major, lower minor).
    function test_isPermittedUpgradeSequence_sameMajorLowerMinor_fails() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 8.0.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("8.0.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 8.1.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("8.1.0"));

        // Should return false because 8.0.0 < 8.1.0 (downgrade).
        assertFalse(opcmV2.isPermittedUpgradeSequence(systemConfig), "same major lower minor should not be permitted");
    }

    /// @notice Tests that the upgrade sequence is not permitted when using same minor with different OPCM.
    function test_isPermittedUpgradeSequence_sameMajorSameMinor_fails() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 8.1.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("8.1.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 8.1.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("8.1.0"));

        // Should return false because 8.1.0 == 8.1.0 (not higher).
        assertFalse(opcmV2.isPermittedUpgradeSequence(systemConfig), "same major same minor should not be permitted");
    }

    /// @notice Tests that the upgrade sequence is not permitted when skipping major versions.
    function test_isPermittedUpgradeSequence_skipMajorVersion_fails() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 10.0.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("10.0.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 8.0.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("8.0.0"));

        // Should return false because 10.0.0 skips major version 9.
        assertFalse(opcmV2.isPermittedUpgradeSequence(systemConfig), "skipping major version should not be permitted");
    }

    /// @notice Tests that the upgrade sequence is not permitted when downgrading major versions.
    function test_isPermittedUpgradeSequence_downgradeMajorVersion_fails() public {
        // Create a mock address for the "old" OPCM.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 8.0.0.
        vm.mockCall(address(opcmV2), abi.encodeCall(IOPContractsManagerV2.version, ()), abi.encode("8.0.0"));

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 9.0.0.
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("9.0.0"));

        // Should return false because 8.0.0 < 9.0.0 (major downgrade).
        assertFalse(opcmV2.isPermittedUpgradeSequence(systemConfig), "major downgrade should not be permitted");
    }

    /// @notice Tests that the upgrade sequence check returns true for OPCM versions < 8.0.0.
    function test_isPermittedUpgradeSequence_versionBelowThreshold_succeeds() public {
        // Create a mock address for the "old" OPCM that would fail the check if it ran.
        address oldOPCM = makeAddr("oldOPCM");

        // Mock the current OPCM version to be 7.0.0 (below threshold).
        vm.mockCall(
            address(opcmV2),
            abi.encodeCall(IOPContractsManagerV2.version, ()),
            abi.encode(Constants.OPCM_V2_MIN_VERSION)
        );

        // Mock lastUsedOPCM to return the old OPCM address.
        vm.mockCall(address(systemConfig), abi.encodeCall(ISystemConfig.lastUsedOPCM, ()), abi.encode(oldOPCM));

        // Mock the old OPCM version to be 10.0.0 (would fail if check ran).
        vm.mockCall(oldOPCM, abi.encodeCall(ISemver.version, ()), abi.encode("10.0.0"));

        // Should return true because the check is skipped for versions < 8.0.0.
        assertTrue(opcmV2.isPermittedUpgradeSequence(systemConfig), "version below threshold should be permitted");
    }

    /// @notice Tests that the upgrade sequence check returns true for initial deployments.
    function test_isPermittedUpgradeSequence_initialDeployment_succeeds() public view {
        // Should return true for initial deployments (address(0) SystemConfig).
        assertTrue(
            opcmV2.isPermittedUpgradeSequence(ISystemConfig(address(0))), "initial deployment should be permitted"
        );
    }
}

/// @title OPContractsManagerV2_UpgradeSuperchain_Test
/// @notice Tests OPContractsManagerV2.upgradeSuperchain
contract OPContractsManagerV2_UpgradeSuperchain_Test is OPContractsManagerV2_Upgrade_TestInit {
    /// @notice Input for the upgradeSuperchain function.
    IOPContractsManagerV2.SuperchainUpgradeInput internal superchainUpgradeInput;

    /// @notice Sets up the test.
    function setUp() public override {
        super.setUp();

        // Set the superchain config.
        // No extra instructions, so don't set them.
        superchainUpgradeInput.superchainConfig = superchainConfig;
    }

    /// @notice Tests that the upgradeSuperchain function succeeds when the superchainConfig is at
    ///         the expected version and the delegate caller is the SuperchainConfig PAO.
    function test_upgradeSuperchain_succeeds() public {
        // Expect the SuperchainConfig to be upgraded.
        address superchainConfigImpl = opcmV2.implementations().superchainConfigImpl;
        vm.expectEmit(address(superchainConfig));
        emit Upgraded(superchainConfigImpl);

        // Do the upgrade.
        prankDelegateCall(superchainPAO);
        (bool success,) = address(opcmV2).delegatecall(
            abi.encodeCall(IOPContractsManagerV2.upgradeSuperchain, (superchainUpgradeInput))
        );
        assertTrue(success, "upgradeSuperchain failed");
    }

    /// @notice Tests that the upgradeSuperchain function reverts when not delegatecalled.
    function test_upgradeSuperchain_notDelegateCalled_reverts() public {
        vm.expectRevert("Ownable: caller is not the owner");
        opcmV2.upgradeSuperchain(superchainUpgradeInput);
    }

    /// @notice Tests that the upgradeSuperchain function reverts when the delegate caller is not
    ///         the superchainProxyAdmin owner.
    function test_upgradeSuperchain_notProxyAdminOwner_reverts() public {
        // Make a new address for testing.
        address delegateCaller = makeAddr("delegateCaller");

        // Sanity check that the address we generated isn't the superchainPAO or chainPAO.
        assertNotEq(superchainPAO, delegateCaller);
        assertNotEq(chainPAO, delegateCaller);

        // Should revert.
        vm.expectRevert("Ownable: caller is not the owner");
        prankDelegateCall(delegateCaller);
        (bool success,) = address(opcmV2).delegatecall(
            abi.encodeCall(IOPContractsManagerV2.upgradeSuperchain, (superchainUpgradeInput))
        );
        assertTrue(success, "upgradeSuperchain failed");
    }

    /// @notice Tests that the upgradeSuperchain function reverts when the superchainConfig version
    ///         is the same or newer than the target version.
    function test_upgradeSuperchain_superchainConfigAlreadyUpToDate_reverts() public {
        ISuperchainConfig superchainConfig = ISuperchainConfig(artifacts.mustGetAddress("SuperchainConfigProxy"));

        // Set the version of the superchain config to a version that is the target version.
        vm.clearMockedCalls();

        // Mock the SuperchainConfig to return a very large version.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("99.99.99"));

        // Should revert.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.expectRevert(
            abi.encodeWithSelector(
                IOPContractsManagerUtils.OPContractsManagerUtils_DowngradeNotAllowed.selector, address(superchainConfig)
            )
        );
        prankDelegateCall(superchainPAO);
        (bool success,) = address(opcmV2).delegatecall(
            abi.encodeCall(IOPContractsManagerV2.upgradeSuperchain, (superchainUpgradeInput))
        );
        assertTrue(success, "upgradeSuperchain failed");
    }
}

/// @title OPContractsManagerV2_Deploy_Test
/// @notice Tests OPContractsManagerV2.deploy
contract OPContractsManagerV2_Deploy_Test is OPContractsManagerV2_TestInit {
    /// @notice Default deploy config.
    IOPContractsManagerV2.FullConfig deployConfig;

    /// @notice Sets up the test.
    function setUp() public override {
        super.setUp();

        // Set up default deploy config.
        // We can't set storage structs directly, so we need to set each field individually.
        deployConfig.saltMixer = "test-salt-mixer";
        deployConfig.superchainConfig = superchainConfig;
        deployConfig.proxyAdminOwner = makeAddr("proxyAdminOwner");
        deployConfig.systemConfigOwner = makeAddr("systemConfigOwner");
        deployConfig.unsafeBlockSigner = makeAddr("unsafeBlockSigner");
        deployConfig.batcher = makeAddr("batcher");
        deployConfig.startingAnchorRoot = Proposal({ root: Hash.wrap(bytes32(hex"1234")), l2SequenceNumber: 123 });
        deployConfig.startingRespectedGameType = GameTypes.PERMISSIONED_CANNON;
        deployConfig.basefeeScalar = 1368;
        deployConfig.blobBasefeeScalar = 801949;
        deployConfig.gasLimit = 60_000_000;
        deployConfig.l2ChainId = 999_999_999;
        deployConfig.resourceConfig = IResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1 gwei,
            systemTxMaxGas: 1_000_000,
            maximumBaseFee: type(uint128).max
        });

        // Set up dispute game configs using the same pattern as upgrade tests.
        address initialChallenger = DisputeGames.permissionedGameChallenger(disputeGameFactory);
        address initialProposer = DisputeGames.permissionedGameProposer(disputeGameFactory);
        deployConfig.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.CANNON,
                gameArgs: bytes("")
            })
        );
        deployConfig.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: true,
                initBond: DEFAULT_DISPUTE_GAME_INIT_BOND, // Standard init bond
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: abi.encode(
                    IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                        absolutePrestate: cannonPrestate,
                        proposer: initialProposer,
                        challenger: initialChallenger
                    })
                )
            })
        );
        deployConfig.disputeGameConfigs.push(
            IOPContractsManagerUtils.DisputeGameConfig({
                enabled: false,
                initBond: 0,
                gameType: GameTypes.CANNON_KONA,
                gameArgs: bytes("")
            })
        );
    }

    /// @notice Tests that the deploy function succeeds and passes standard validation.
    function test_deploy_succeeds() public {
        // Run the deploy and standard validator checks.
        // We expect PLDG-10 and CKDG-10 validator errors because CANNON and CANNON_KONA are
        // disabled during initial deployment (no implementations registered).
        IOPContractsManagerV2.ChainContracts memory cts = runDeployV2(deployConfig, bytes(""), "PLDG-10,CKDG-10");

        // Verify key contracts are deployed.
        assertTrue(address(cts.systemConfig) != address(0), "systemConfig not deployed");
        assertTrue(address(cts.proxyAdmin) != address(0), "proxyAdmin not deployed");
        assertTrue(address(cts.optimismPortal) != address(0), "optimismPortal not deployed");
        assertTrue(address(cts.disputeGameFactory) != address(0), "disputeGameFactory not deployed");
        assertTrue(address(cts.anchorStateRegistry) != address(0), "anchorStateRegistry not deployed");
        assertTrue(address(cts.delayedWETH) != address(0), "delayedWETH not deployed");

        // Verify ownership is transferred to proxyAdminOwner.
        assertEq(cts.proxyAdmin.owner(), deployConfig.proxyAdminOwner, "proxyAdmin owner mismatch");
        assertEq(cts.disputeGameFactory.owner(), deployConfig.proxyAdminOwner, "disputeGameFactory owner mismatch");
    }

    /// @notice Tests that deploy reverts when the superchainConfig needs upgrade.
    function test_deploy_superchainConfigNeedsUpgrade_reverts() public {
        // Force the SuperchainConfig to return an obviously outdated version.
        vm.mockCall(address(superchainConfig), abi.encodeCall(ISuperchainConfig.version, ()), abi.encode("0.0.0"));

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig,
            abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_SuperchainConfigNeedsUpgrade.selector)
        );
    }

    /// @notice Tests that deploy reverts when missing game configs.
    function test_deploy_missingGameConfigs_reverts() public {
        // Delete the Cannon Kona game configuration.
        delete deployConfig.disputeGameConfigs[2];

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that deploy reverts when game configs are in wrong order.
    function test_deploy_wrongGameConfigOrder_reverts() public {
        // Swap the game config order.
        IOPContractsManagerUtils.DisputeGameConfig memory temp = deployConfig.disputeGameConfigs[0];
        deployConfig.disputeGameConfigs[0] = deployConfig.disputeGameConfigs[1];
        deployConfig.disputeGameConfigs[1] = temp;

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that deploy reverts when the PermissionedDisputeGame is disabled.
    function test_deploy_disabledPermissionedGame_reverts() public {
        // Disable the PermissionedDisputeGame.
        deployConfig.disputeGameConfigs[1].enabled = false;

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    /// @notice Tests that deploy reverts when a disabled game has non-zero init bond.
    function test_deploy_disabledGameNonZeroBond_reverts() public {
        // Disable Cannon but keep a non-zero init bond.
        deployConfig.disputeGameConfigs[0].enabled = false;
        deployConfig.disputeGameConfigs[0].initBond = 1 ether;

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    function test_deploy_cannonGameEnabled_reverts() public {
        deployConfig.disputeGameConfigs[0].enabled = true;
        deployConfig.disputeGameConfigs[0].initBond = 1 ether;

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }

    function test_deploy_cannonKonaGameEnabled_reverts() public {
        deployConfig.disputeGameConfigs[2].enabled = true;
        deployConfig.disputeGameConfigs[2].initBond = 1 ether;

        // nosemgrep: sol-style-use-abi-encodecall
        runDeployV2(
            deployConfig, abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidGameConfigs.selector)
        );
    }
}

/// @title OPContractsManagerV2_DevFeatureBitmap_Test
/// @notice Tests OPContractsManagerV2.devFeatureBitmap
contract OPContractsManagerV2_DevFeatureBitmap_Test is OPContractsManagerV2_TestInit {
    /// @notice Tests that the devFeatureBitmap returned by opcmV2 matches the contractsContainer address's own.
    function test_devFeatureBitmap_succeeds() public view {
        assertEq(
            opcmV2.devFeatureBitmap(),
            opcmV2.contractsContainer().devFeatureBitmap(),
            "devFeatureBitmap on opcmV2 does not match contractsContainer bitmap"
        );
    }
}

/// @title OPContractsManagerV2_Migrate_Test
/// @notice Tests the `migrate` function of the `OPContractsManagerV2` contract.
contract OPContractsManagerV2_Migrate_Test is OPContractsManagerV2_TestInit {
    /// @notice Deployed chain contracts for chain 1.
    IOPContractsManagerV2.ChainContracts chainContracts1;

    /// @notice Deployed chain contracts for chain 2.
    IOPContractsManagerV2.ChainContracts chainContracts2;

    /// @notice Super root prestate for super cannon games.
    Claim superPrestate = Claim.wrap(bytes32(keccak256("superPrestate")));

    /// @notice Function requires interop portal.
    function setUp() public override {
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Deploy two chains via OPCMv2 for migration testing.
        chainContracts1 = _deployChainForMigration(1000001);
        chainContracts2 = _deployChainForMigration(1000002);
    }

    /// @notice Helper function to deploy a chain for migration testing.
    /// @param _l2ChainId The L2 chain ID for the deployed chain.
    /// @return cts_ The deployed chain contracts.
    function _deployChainForMigration(uint256 _l2ChainId)
        internal
        returns (IOPContractsManagerV2.ChainContracts memory cts_)
    {
        // Set up dispute game configs first since they're needed for the struct literal.
        address initialChallenger = DisputeGames.permissionedGameChallenger(disputeGameFactory);
        address initialProposer = DisputeGames.permissionedGameProposer(disputeGameFactory);
        IOPContractsManagerUtils.DisputeGameConfig[] memory dgConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](3);
        dgConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: bytes("")
        });
        dgConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: cannonPrestate,
                    proposer: initialProposer,
                    challenger: initialChallenger
                })
            )
        });
        dgConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON_KONA,
            gameArgs: bytes("")
        });

        // Set up the deploy config using struct literal for compile-time field checking.
        IOPContractsManagerV2.FullConfig memory deployConfig = IOPContractsManagerV2.FullConfig({
            saltMixer: string(abi.encodePacked("migrate-test-", _l2ChainId)),
            superchainConfig: superchainConfig,
            proxyAdminOwner: makeAddr("migrateProxyAdminOwner"),
            systemConfigOwner: makeAddr("migrateSystemConfigOwner"),
            unsafeBlockSigner: makeAddr("migrateUnsafeBlockSigner"),
            batcher: makeAddr("migrateBatcher"),
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(hex"1234")), l2SequenceNumber: 123 }),
            startingRespectedGameType: GameTypes.PERMISSIONED_CANNON,
            basefeeScalar: 1368,
            blobBasefeeScalar: 801949,
            gasLimit: 60_000_000,
            l2ChainId: _l2ChainId,
            resourceConfig: IResourceMetering.ResourceConfig({
                maxResourceLimit: 20_000_000,
                elasticityMultiplier: 10,
                baseFeeMaxChangeDenominator: 8,
                minimumBaseFee: 1 gwei,
                systemTxMaxGas: 1_000_000,
                maximumBaseFee: type(uint128).max
            }),
            disputeGameConfigs: dgConfigs,
            useCustomGasToken: false
        });

        // Deploy the chain.
        cts_ = opcmV2.deploy(deployConfig);
    }

    /// @notice Helper function to create the default migration input.
    /// @return input_ The default migration input.
    function _getDefaultMigrateInput() internal returns (IOPContractsManagerMigrator.MigrateInput memory input_) {
        // Set up the chain system configs.
        ISystemConfig[] memory chainSystemConfigs = new ISystemConfig[](2);
        chainSystemConfigs[0] = chainContracts1.systemConfig;
        chainSystemConfigs[1] = chainContracts2.systemConfig;

        // Set up the dispute game configs for super root games.
        address proposer = makeAddr("superProposer");
        address challenger = makeAddr("superChallenger");

        IOPContractsManagerUtils.DisputeGameConfig[] memory disputeGameConfigs =
            new IOPContractsManagerUtils.DisputeGameConfig[](1);
        disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: 0.08 ether,
            gameType: GameTypes.SUPER_PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: superPrestate,
                    proposer: proposer,
                    challenger: challenger
                })
            )
        });

        input_ = IOPContractsManagerMigrator.MigrateInput({
            chainSystemConfigs: chainSystemConfigs,
            disputeGameConfigs: disputeGameConfigs,
            startingAnchorRoot: Proposal({ root: Hash.wrap(bytes32(hex"ABBA")), l2SequenceNumber: 1234 }),
            startingRespectedGameType: GameTypes.SUPER_PERMISSIONED_CANNON
        });
    }

    /// @notice Helper function to execute a migration.
    /// @param _input The input to the migration function.
    function _doMigration(IOPContractsManagerMigrator.MigrateInput memory _input) internal {
        _doMigration(_input, bytes4(0));
    }

    /// @notice Helper function to execute a migration with a revert selector.
    /// @param _input The input to the migration function.
    /// @param _revertSelector The selector of the revert to expect.
    function _doMigration(IOPContractsManagerMigrator.MigrateInput memory _input, bytes4 _revertSelector) internal {
        // Set the proxy admin owner to be a delegate caller.
        address proxyAdminOwner = chainContracts1.proxyAdmin.owner();

        // Execute a delegatecall to the OPCM migration function.
        // Check gas usage of the migration function.
        uint256 gasBefore = gasleft();
        if (_revertSelector != bytes4(0)) {
            vm.expectRevert(_revertSelector);
        }
        prankDelegateCall(proxyAdminOwner);
        (bool success,) = address(opcmV2).delegatecall(abi.encodeCall(IOPContractsManagerV2.migrate, (_input)));
        assertTrue(success, "migrate failed");
        uint256 gasAfter = gasleft();

        // Make sure the gas usage is less than 20 million so we can definitely fit in a block.
        assertLt(gasBefore - gasAfter, 20_000_000, "Gas usage too high");
    }

    /// @notice Helper function to assert that the old game implementations are now zeroed out.
    /// @param _disputeGameFactory The dispute game factory to check.
    function _assertOldGamesZeroed(IDisputeGameFactory _disputeGameFactory) internal view {
        // Assert that the old game implementations are now zeroed out.
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.CANNON, "CANNON");
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.SUPER_CANNON, "SUPER_CANNON");
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.PERMISSIONED_CANNON, "PERMISSIONED_CANNON");
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.SUPER_PERMISSIONED_CANNON, "SUPER_PERMISSIONED_CANNON");
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.CANNON_KONA, "CANNON_KONA");
        _assertGameIsEmpty(_disputeGameFactory, GameTypes.SUPER_CANNON_KONA, "SUPER_CANNON_KONA");
    }

    /// @notice Helper function to assert a game is empty.
    /// @param _dgf The dispute game factory.
    /// @param _gameType The game type.
    /// @param _label The label for the game type.
    function _assertGameIsEmpty(IDisputeGameFactory _dgf, GameType _gameType, string memory _label) internal view {
        assertEq(
            address(_dgf.gameImpls(_gameType)),
            address(0),
            string.concat("Game type set when it should not be: ", _label)
        );
        assertEq(_dgf.gameArgs(_gameType), hex"", string.concat("Game args should be empty: ", _label));
    }

    /// @notice Tests that the migration function succeeds and liquidity is migrated.
    function test_migrate_succeeds() public {
        IOPContractsManagerMigrator.MigrateInput memory input = _getDefaultMigrateInput();

        // Pre-migration setup: Get old lockboxes and fund them.
        IETHLockbox oldLockbox1;
        IETHLockbox oldLockbox2;
        uint256 lockbox1Balance = 10 ether;
        uint256 lockbox2Balance = 5 ether;
        {
            IOptimismPortal2 oldPortal1 = IOptimismPortal2(payable(chainContracts1.systemConfig.optimismPortal()));
            IOptimismPortal2 oldPortal2 = IOptimismPortal2(payable(chainContracts2.systemConfig.optimismPortal()));
            oldLockbox1 = oldPortal1.ethLockbox();
            oldLockbox2 = oldPortal2.ethLockbox();
            vm.deal(address(oldLockbox1), lockbox1Balance);
            vm.deal(address(oldLockbox2), lockbox2Balance);
        }

        // Pre-migration: Get old DisputeGameFactories.
        IDisputeGameFactory oldDGF1 = IDisputeGameFactory(payable(chainContracts1.systemConfig.disputeGameFactory()));
        IDisputeGameFactory oldDGF2 = IDisputeGameFactory(payable(chainContracts2.systemConfig.disputeGameFactory()));

        // Execute the migration.
        _doMigration(input);

        // Assert that the old game implementations are now zeroed out.
        _assertOldGamesZeroed(oldDGF1);
        _assertOldGamesZeroed(oldDGF2);

        // Grab the two OptimismPortal addresses.
        IOptimismPortal2 portal1 = IOptimismPortal2(payable(chainContracts1.systemConfig.optimismPortal()));
        IOptimismPortal2 portal2 = IOptimismPortal2(payable(chainContracts2.systemConfig.optimismPortal()));

        // Grab the AnchorStateRegistry from the OptimismPortal for both chains, confirm same.
        assertEq(
            address(portal1.anchorStateRegistry()),
            address(portal2.anchorStateRegistry()),
            "AnchorStateRegistry mismatch"
        );

        // Extract the AnchorStateRegistry now that we know it's the same on both chains.
        IAnchorStateRegistry asr = portal1.anchorStateRegistry();

        // Check that the starting anchor root is the same as the input.
        {
            (Hash root, uint256 l2SeqNum) = asr.getAnchorRoot();
            assertEq(root.raw(), input.startingAnchorRoot.root.raw(), "Starting anchor root mismatch");
            assertEq(l2SeqNum, input.startingAnchorRoot.l2SequenceNumber, "Starting anchor root L2 seq num mismatch");
        }

        // Grab the ETHLockbox from the OptimismPortal for both chains, confirm same.
        assertEq(address(portal1.ethLockbox()), address(portal2.ethLockbox()), "ETHLockbox mismatch");

        // Extract the new ETHLockbox now that we know it's the same on both chains.
        IETHLockbox newLockbox = portal1.ethLockbox();

        // Check that the ETHLockbox has authorized portals.
        assertTrue(newLockbox.authorizedPortals(portal1), "ETHLockbox does not have portal 1 authorized");
        assertTrue(newLockbox.authorizedPortals(portal2), "ETHLockbox does not have portal 2 authorized");

        // Check that superRootsActive is true on both portals.
        assertTrue(
            IOptimismPortalInterop(payable(address(portal1))).superRootsActive(),
            "Portal 1 superRootsActive should be true"
        );
        assertTrue(
            IOptimismPortalInterop(payable(address(portal2))).superRootsActive(),
            "Portal 2 superRootsActive should be true"
        );

        // Check that the ETH_LOCKBOX feature is enabled on both SystemConfigs.
        assertTrue(
            chainContracts1.systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX),
            "Chain 1 ETH_LOCKBOX feature should be enabled"
        );
        assertTrue(
            chainContracts2.systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX),
            "Chain 2 ETH_LOCKBOX feature should be enabled"
        );

        // Check that the init bonds are set correctly on the new DisputeGameFactory.
        assertEq(
            IDisputeGameFactory(asr.disputeGameFactory()).initBonds(GameTypes.SUPER_PERMISSIONED_CANNON),
            0.08 ether,
            "SUPER_PERMISSIONED_CANNON init bond mismatch"
        );

        // Check that liquidity was migrated from old lockboxes to the new shared lockbox.
        assertEq(address(oldLockbox1).balance, 0, "Old lockbox 1 should have 0 balance after migration");
        assertEq(address(oldLockbox2).balance, 0, "Old lockbox 2 should have 0 balance after migration");
        assertEq(
            address(newLockbox).balance,
            lockbox1Balance + lockbox2Balance,
            "New lockbox should have combined balance from both old lockboxes"
        );

        // Check that the old lockboxes are authorized on the new lockbox.
        assertTrue(newLockbox.authorizedLockboxes(oldLockbox1), "Old lockbox 1 should be authorized on new lockbox");
        assertTrue(newLockbox.authorizedLockboxes(oldLockbox2), "Old lockbox 2 should be authorized on new lockbox");
    }

    /// @notice Tests that the migration function reverts when the ProxyAdmin owners are mismatched.
    /// @param _owner1 The owner address for the first chain's ProxyAdmin.
    /// @param _owner2 The owner address for the second chain's ProxyAdmin.
    function testFuzz_migrate_mismatchedProxyAdminOwners_reverts(address _owner1, address _owner2) public {
        vm.assume(_owner1 != _owner2);
        assumeNotPrecompile(_owner1);
        assumeNotPrecompile(_owner2);
        assumeNotForgeAddress(_owner1);
        assumeNotForgeAddress(_owner2);
        IOPContractsManagerMigrator.MigrateInput memory input = _getDefaultMigrateInput();

        // Mock out the owners of the ProxyAdmins to be different.
        vm.mockCall(
            address(input.chainSystemConfigs[0].proxyAdmin()),
            abi.encodeCall(IProxyAdmin.owner, ()),
            abi.encode(_owner1)
        );
        vm.mockCall(
            address(input.chainSystemConfigs[1].proxyAdmin()),
            abi.encodeCall(IProxyAdmin.owner, ()),
            abi.encode(_owner2)
        );

        // Execute the migration, expect revert.
        _doMigration(input, IOPContractsManagerMigrator.OPContractsManagerMigrator_ProxyAdminOwnerMismatch.selector);
    }

    /// @notice Tests that the migration function reverts when the SuperchainConfig addresses are mismatched.
    /// @param _config1 The SuperchainConfig address for the first chain.
    /// @param _config2 The SuperchainConfig address for the second chain.
    function testFuzz_migrate_mismatchedSuperchainConfig_reverts(address _config1, address _config2) public {
        vm.assume(_config1 != _config2);
        assumeNotPrecompile(_config1);
        assumeNotPrecompile(_config2);
        assumeNotForgeAddress(_config1);
        assumeNotForgeAddress(_config2);
        IOPContractsManagerMigrator.MigrateInput memory input = _getDefaultMigrateInput();

        // Mock out the SuperchainConfig addresses to be different.
        vm.mockCall(
            address(input.chainSystemConfigs[0]),
            abi.encodeCall(ISystemConfig.superchainConfig, ()),
            abi.encode(_config1)
        );
        vm.mockCall(
            address(input.chainSystemConfigs[1]),
            abi.encodeCall(ISystemConfig.superchainConfig, ()),
            abi.encode(_config2)
        );

        // Execute the migration, expect revert.
        _doMigration(input, IOPContractsManagerMigrator.OPContractsManagerMigrator_SuperchainConfigMismatch.selector);
    }

    /// @notice Tests that the migration function reverts when the starting respected game type is invalid.
    /// @param _gameTypeRaw The raw game type value to test.
    function testFuzz_migrate_invalidStartingRespectedGameType_reverts(uint32 _gameTypeRaw) public {
        // Only SUPER_CANNON (4) and SUPER_PERMISSIONED_CANNON (5) are valid for migration.
        vm.assume(_gameTypeRaw != GameTypes.SUPER_CANNON.raw());
        vm.assume(_gameTypeRaw != GameTypes.SUPER_PERMISSIONED_CANNON.raw());

        IOPContractsManagerMigrator.MigrateInput memory input = _getDefaultMigrateInput();

        // Set an invalid starting respected game type.
        input.startingRespectedGameType = GameType.wrap(_gameTypeRaw);

        // Execute the migration, expect revert.
        _doMigration(
            input, IOPContractsManagerMigrator.OPContractsManagerMigrator_InvalidStartingRespectedGameType.selector
        );
    }
}

/// @title OPContractsManagerV2_FeatBatchUpgrade_Test
/// @notice Tests batch upgrade functionality with freshly deployed chains (non-forked).
contract OPContractsManagerV2_FeatBatchUpgrade_Test is OPContractsManagerV2_TestInit {
    /// @notice Tests that multiple upgrade operations (15 chains) can be executed within a single transaction.
    ///         This enforces the OPCMV2 invariant that approximately 15 upgrade operations should be
    ///         executable in one transaction.
    function test_batchUpgrade_multipleChains_succeeds() public {
        skipIfCoverage();

        uint256 numberOfChains = 15;

        // 1. Deploy BatchUpgrader helper contract.
        BatchUpgrader batchUpgrader = new BatchUpgrader(opcmV2);

        // 2. Set up base configuration for deploying chains.
        IOPContractsManagerV2.FullConfig memory baseConfig;
        baseConfig.superchainConfig = superchainConfig;
        baseConfig.proxyAdminOwner = address(batchUpgrader);
        baseConfig.systemConfigOwner = makeAddr("systemConfigOwner");
        baseConfig.batcher = makeAddr("batcher");
        baseConfig.unsafeBlockSigner = makeAddr("unsafeBlockSigner");
        baseConfig.startingAnchorRoot = Proposal({ root: Hash.wrap(bytes32(hex"1234")), l2SequenceNumber: 123 });
        baseConfig.startingRespectedGameType = GameTypes.PERMISSIONED_CANNON;
        baseConfig.basefeeScalar = 1368;
        baseConfig.blobBasefeeScalar = 801949;
        baseConfig.gasLimit = 60_000_000;
        baseConfig.resourceConfig = IResourceMetering.ResourceConfig({
            maxResourceLimit: 20_000_000,
            elasticityMultiplier: 10,
            baseFeeMaxChangeDenominator: 8,
            minimumBaseFee: 1 gwei,
            systemTxMaxGas: 1_000_000,
            maximumBaseFee: type(uint128).max
        });

        // Set up dispute game configs.
        address initialChallenger = makeAddr("challenger");
        address initialProposer = makeAddr("proposer");
        baseConfig.disputeGameConfigs = new IOPContractsManagerUtils.DisputeGameConfig[](3);
        baseConfig.disputeGameConfigs[0] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON,
            gameArgs: bytes("")
        });
        baseConfig.disputeGameConfigs[1] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: true,
            initBond: DEFAULT_DISPUTE_GAME_INIT_BOND,
            gameType: GameTypes.PERMISSIONED_CANNON,
            gameArgs: abi.encode(
                IOPContractsManagerUtils.PermissionedDisputeGameConfig({
                    absolutePrestate: cannonPrestate,
                    proposer: initialProposer,
                    challenger: initialChallenger
                })
            )
        });
        baseConfig.disputeGameConfigs[2] = IOPContractsManagerUtils.DisputeGameConfig({
            enabled: false,
            initBond: 0,
            gameType: GameTypes.CANNON_KONA,
            gameArgs: bytes("")
        });

        // 3. Deploy 15 separate chains using opcmV2.deploy().
        IOPContractsManagerV2.ChainContracts[] memory chains =
            new IOPContractsManagerV2.ChainContracts[](numberOfChains);
        for (uint256 i = 0; i < numberOfChains; i++) {
            IOPContractsManagerV2.FullConfig memory config = baseConfig;
            config.saltMixer = string.concat("chain-", vm.toString(i));
            config.l2ChainId = 1000 + i;
            chains[i] = opcmV2.deploy(config);
        }

        // 4. Prepare upgrade inputs for each chain.
        IOPContractsManagerV2.UpgradeInput[] memory upgradeInputs =
            new IOPContractsManagerV2.UpgradeInput[](numberOfChains);
        for (uint256 i = 0; i < numberOfChains; i++) {
            upgradeInputs[i] = IOPContractsManagerV2.UpgradeInput({
                systemConfig: chains[i].systemConfig,
                disputeGameConfigs: baseConfig.disputeGameConfigs,
                extraInstructions: new IOPContractsManagerUtils.ExtraInstruction[](0)
            });
        }

        // 5. Execute batch upgrade - all 15 upgrades in a single transaction.
        batchUpgrader.batchUpgrade(upgradeInputs);
        VmSafe.Gas memory gas = vm.lastCallGas();

        // 6. Verify that the upgrade gas usage is less than the EIP-7825 gas limit.
        // See https://eip.tools/eip/eip-7825.md for more details.
        // The upgradeGasBuffer amount below is an approximation of the overhead that is required
        // to execute a call to Safe.executeTransaction() prior to the call to IOPContractsManagerV2.upgrade().
        // The approximate value of 65,000 gas, was taken from a previous upgrade transaction on OP Mainnet:
        // https://dashboard.tenderly.co/oplabs/op-mainnet/tx/0x9b9aa2d8e857e1a28e55b124e931eac706b3ae04c1b33ba949f0366359860993/gas-usage?trace=0.1.7
        uint256 fusakaLimit = 2 ** 24;
        uint256 upgradeGasBuffer = 65_000;
        assertLt(gas.gasTotalUsed, fusakaLimit - upgradeGasBuffer, "Upgrade exceeds gas target");

        // 7. Verify all chains upgraded successfully.
        for (uint256 i = 0; i < numberOfChains; i++) {
            ISystemConfig systemConfig = chains[i].systemConfig;

            // Verify OPCM release version was updated.
            string memory version = systemConfig.lastUsedOPCMVersion();
            assertEq(version, opcmV2.version(), string.concat("Chain ", vm.toString(i), " version mismatch"));

            // Verify SystemConfig implementation was upgraded.
            address impl = EIP1967Helper.getImplementation(address(systemConfig));
            assertEq(
                impl,
                opcmV2.implementations().systemConfigImpl,
                string.concat("Chain ", vm.toString(i), " SystemConfig impl not upgraded")
            );
        }
    }
}
