// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { VmSafe } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Libraries
import { Config } from "scripts/libraries/Config.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Claim } from "src/dispute/lib/LibUDT.sol";
import { GameType, GameTypes } from "src/dispute/lib/Types.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IOPContractsManagerStandardValidator } from "interfaces/L1/IOPContractsManagerStandardValidator.sol";
import { IOPContractsManagerV2 } from "interfaces/L1/opcm/IOPContractsManagerV2.sol";
import { IOPContractsManagerUtils } from "interfaces/L1/opcm/IOPContractsManagerUtils.sol";

/// @title OPContractsManagerV2_Upgrade_TestInit
/// @notice Test initialization contract for OPContractsManagerV2 upgrade functions.
contract OPContractsManagerV2_Upgrade_TestInit is CommonTest, DisputeGames {
    // The Upgraded event emitted by the Proxy contract.
    event Upgraded(address indexed implementation);

    /// @notice Chain ID for the L2 chain being upgraded in this test.
    uint256 l2ChainId;

    /// @notice Address of the ProxyAdmin owner for the chain being upgraded.
    address chainPAO;

    /// @notice Address of the Superchain ProxyAdmin owner.
    address superchainPAO;

    /// @notice Fake prestate for Cannon games.
    Claim cannonPrestate = Claim.wrap(bytes32(keccak256("cannonPrestate")));

    /// @notice Fake prestate for Cannon Kona games.
    Claim cannonKonaPrestate = Claim.wrap(bytes32(keccak256("cannonKonaPrestate")));

    /// @notice Name of the chain being forked.
    string public opChain = Config.forkOpChain();

    /// @notice Default v2 upgrade input.
    IOPContractsManagerV2.UpgradeInput v2UpgradeInput;

    /// @notice Special string constant used to indicate that we expect a revert without any data.
    bytes public constant EXPECT_REVERT_WITHOUT_DATA = bytes("EXPECT_REVERT_WITHOUT_DATA");

    /// @notice Thrown when trying to run past upgrades on an unsupported chain.
    error UnsupportedChainId();

    /// @notice Sets up the test suite.
    function setUp() public virtual override {
        super.disableUpgradedFork();
        super.setUp();

        skipIfDevFeatureDisabled(DevFeatures.OPCM_V2);
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
        address initialChallengerForV2 = permissionedGameChallenger(disputeGameFactory);
        address initialProposerForV2 = permissionedGameProposer(disputeGameFactory);
        v2UpgradeInput.systemConfig = systemConfig;
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerV2.DisputeGameConfig({
                enabled: true,
                initBond: disputeGameFactory.initBonds(GameTypes.CANNON),
                gameType: GameTypes.CANNON,
                gameArgs: abi.encode(IOPContractsManagerV2.FaultDisputeGameConfig({ absolutePrestate: cannonPrestate }))
            })
        );
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerV2.DisputeGameConfig({
                enabled: true,
                initBond: disputeGameFactory.initBonds(GameTypes.PERMISSIONED_CANNON),
                gameType: GameTypes.PERMISSIONED_CANNON,
                gameArgs: abi.encode(
                    IOPContractsManagerV2.PermissionedDisputeGameConfig({
                        absolutePrestate: cannonPrestate,
                        proposer: initialProposerForV2,
                        challenger: initialChallengerForV2
                    })
                )
            })
        );
        v2UpgradeInput.disputeGameConfigs.push(
            IOPContractsManagerV2.DisputeGameConfig({
                enabled: isDevFeatureEnabled(DevFeatures.CANNON_KONA),
                initBond: disputeGameFactory.initBonds(GameTypes.CANNON_KONA),
                gameType: GameTypes.CANNON_KONA,
                gameArgs: abi.encode(IOPContractsManagerV2.FaultDisputeGameConfig({ absolutePrestate: cannonKonaPrestate }))
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
        address initialChallenger = permissionedGameChallenger(disputeGameFactory);
        address initialProposer = permissionedGameProposer(disputeGameFactory);

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

        // Less than 90% of the gas target of 2**24 (EIP-7825) to account for the gas used by using Safe.
        uint256 fusakaLimit = 2 ** 24;
        VmSafe.Gas memory gas = vm.lastCallGas();
        assertLt(gas.gasTotalUsed, fusakaLimit * 9 / 10, "Upgrade exceeds gas target of 90% of 2**24 (EIP-7825)");

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
        IOPContractsManagerStandardValidator validator = _opcm.standardValidator();

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
        if (isDevFeatureEnabled(DevFeatures.CANNON_KONA)) {
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
        } else {
            validator.validateWithOverrides(
                IOPContractsManagerStandardValidator.ValidationInput({
                    sysCfg: v2UpgradeInput.systemConfig,
                    absolutePrestate: cannonPrestate.raw(),
                    l2ChainID: l2ChainId,
                    proposer: initialProposer
                }),
                false,
                validationOverrides
            );
        }
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
        IOPContractsManagerV2.DisputeGameConfig memory temp = v2UpgradeInput.disputeGameConfigs[0];
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
            abi.encodeWithSelector(IOPContractsManagerV2.OPContractsManagerV2_InvalidUpgradeInstruction.selector)
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
            opcmV2.implementations().faultDisputeGameV2Impl,
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
            opcmV2.implementations().faultDisputeGameV2Impl,
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
            abi.encode(IOPContractsManagerV2.FaultDisputeGameConfig({ absolutePrestate: newPrestate }));
        v2UpgradeInput.disputeGameConfigs[1].gameArgs = abi.encode(
            IOPContractsManagerV2.PermissionedDisputeGameConfig({
                absolutePrestate: newPrestate,
                proposer: permissionedGameProposer(disputeGameFactory),
                challenger: permissionedGameChallenger(disputeGameFactory)
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
