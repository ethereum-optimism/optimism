// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Foundry
import "forge-std/Test.sol";

// Contracts
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { OwnerLivenessModule } from "src/safe/OwnerLivenessModule.sol";

// Libraries
import "test/safe-tools/SafeTestTools.sol";
import { ECDSA } from "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

/// @notice Harness for testing claim failure.
contract OwnerLivenessModule_ClaimingReverter_Harness {
    /// @notice Calls claim on the provided module. Reverts on receiving ETH.
    function triggerClaim(address _module) external {
        OwnerLivenessModule(_module).claim();
    }

    /// @notice Reverts on fallback.
    fallback() external payable {
        revert("RR: revert on receive");
    }

    /// @notice Reverts on receive.
    receive() external payable {
        revert("RR: revert on receive");
    }
}

/// @title OwnerLivenessModule_TestInit
/// @notice Provides common setup utilities for OwnerLivenessModule tests.
contract OwnerLivenessModule_TestInit is Test, SafeTestTools {
    using SafeTestLib for SafeInstance;

    event ChallengeCreated(Safe indexed safe, address indexed owner, address indexed challenger, uint256 timeout);
    event ChallengeClosed(
        Safe indexed safe, address indexed owner, address indexed challenger, OwnerLivenessModule.ChallengeResult result
    );
    event OwnerRemoved(Safe indexed safe, address indexed owner, uint256 threshold);
    event OwnerSwapped(Safe indexed safe, address indexed oldOwner, address indexed newOwner);

    uint256 internal constant INIT_TIME = 10;
    uint256 internal constant CHALLENGE_PERIOD = 7 days;
    uint256 internal constant CHALLENGE_BOND = 1 ether;
    uint256 internal constant MIN_OWNERS = 6;
    uint256 internal constant THRESHOLD_PERCENTAGE = 75;

    OwnerLivenessModule internal ownerLivenessModule;
    OwnerLivenessModule internal unconfiguredModule;
    SafeInstance internal safeInstance;
    address internal fallbackOwner;
    address internal nonOwner;
    uint256 internal nonOwnerKey;
    address internal challenger;

    /// @notice Configures the module for the current Safe.
    function _configureModule() internal {
        OwnerLivenessModule.ModuleConfig memory cfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        ownerLivenessModule.configure(cfg);
    }

    /// @notice Creates a challenge against the first owner and returns the challenged owner address.
    function _createChallenge() internal returns (address challengedOwner) {
        challengedOwner = safeInstance.owners[0];
        vm.deal(challenger, CHALLENGE_BOND);
        vm.prank(challenger);
        ownerLivenessModule.attack{ value: CHALLENGE_BOND }(Safe(payable(address(safeInstance.safe))), challengedOwner);
    }

    /// @notice Creates a signature for defend() from the given private key and timestamp.
    /// @param _privateKey The private key to sign with.
    /// @param _timestamp The timestamp to include in the signature.
    /// @return The encoded signature bytes.
    function _createDefendSignature(uint256 _privateKey, uint256 _timestamp) internal pure returns (bytes memory) {
        bytes32 digest =
            ECDSA.toEthSignedMessageHash(abi.encodePacked("OwnerLivenessModule Reply ", Strings.toString(_timestamp)));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(_privateKey, digest);
        return abi.encodePacked(r, s, v);
    }

    /// @notice Sets up the test environment.
    function setUp() public virtual {
        // Consistent starting timestamp so tests using block.timestamp work deterministically.
        vm.warp(INIT_TIME);

        // Deploy a Safe with 10 owners and threshold 8 to keep above MIN_OWNERS.
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("olmTest", 10);
        safeInstance = _setupSafe(keys, 8);

        // Deploy configured module and enable it on the Safe.
        ownerLivenessModule = new OwnerLivenessModule();
        safeInstance.enableModule(address(ownerLivenessModule));

        // Deploy unconfigured module for testing NotConfigured errors.
        unconfiguredModule = new OwnerLivenessModule();
        safeInstance.enableModule(address(unconfiguredModule));

        // Create test addresses.
        fallbackOwner = makeAddr("fallbackOwner");
        (nonOwner, nonOwnerKey) = makeAddrAndKey("nonOwner");
        challenger = makeAddr("challenger");

        // Give challenger funds for multiple challenges.
        vm.deal(challenger, CHALLENGE_BOND * 10);

        // Configure the main module.
        _configureModule();
    }
}

contract OwnerLivenessModule_Configure_Test is OwnerLivenessModule_TestInit {
    /// @notice Tests that configure succeeds with various valid parameter combinations.
    /// @param _thresholdPercentage Threshold percentage for Safe operations (1-100).
    /// @param _challengePeriod Time period for challenges in seconds (1 to max uint256).
    /// @param _challengeBond ETH bond required for challenges in wei (1 to max uint256).
    /// @param _minOwners Minimum number of owners before fallback (1 to current owner count).
    function testFuzz_configure_validParameters_succeeds(
        uint256 _thresholdPercentage,
        uint256 _challengePeriod,
        uint256 _challengeBond,
        uint256 _minOwners
    )
        external
    {
        _thresholdPercentage = bound(_thresholdPercentage, 1, 100);
        _challengePeriod = bound(_challengePeriod, 1, type(uint256).max);
        _challengeBond = bound(_challengeBond, 1, type(uint256).max);
        _minOwners = bound(_minOwners, 1, safeInstance.owners.length);

        OwnerLivenessModule.ModuleConfig memory cfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: _challengePeriod,
            challengeBond: _challengeBond,
            minOwners: _minOwners,
            thresholdPercentage: _thresholdPercentage,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        ownerLivenessModule.configure(cfg);

        (uint256 ts, uint256 cp, uint256 cb, uint256 min, uint256 th, address fb) =
            ownerLivenessModule.configs(Safe(payable(address(safeInstance.safe))));
        assertEq(cp, _challengePeriod);
        assertEq(cb, _challengeBond);
        assertEq(min, _minOwners);
        assertEq(th, _thresholdPercentage);
        assertEq(fb, fallbackOwner);
        assertGt(ts, 0);
    }

    /// @notice Tests that configure reverts when minOwners is zero.
    function test_configure_minOwnersZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: 0,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_TooLowMinOwners.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when minOwners is greater than owners length.
    function test_configure_minOwnersTooHigh_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: safeInstance.owners.length + 1,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_TooHighMinOwners.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when threshold percentage is zero.
    function test_configure_thresholdZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: 0,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_TooLowThreshold.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when threshold percentage is over 100.
    function test_configure_thresholdTooHigh_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: 101,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_TooHighThreshold.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when challenge period is zero.
    function test_configure_challengePeriodZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: 0,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidChallengePeriod.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when challenge bond is zero.
    function test_configure_challengeBondZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: 0,
            minOwners: MIN_OWNERS,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidChallengeBond.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when fallback owner is zero address.
    function test_configure_fallbackOwnerZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 0,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: address(0)
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidFallback.selector);
        ownerLivenessModule.configure(badCfg);
    }

    /// @notice Tests that configure reverts when timestamp is not zero.
    function test_configure_timestampNotZero_reverts() external {
        OwnerLivenessModule.ModuleConfig memory badCfg = OwnerLivenessModule.ModuleConfig({
            timestamp: 1,
            challengePeriod: CHALLENGE_PERIOD,
            challengeBond: CHALLENGE_BOND,
            minOwners: MIN_OWNERS,
            thresholdPercentage: THRESHOLD_PERCENTAGE,
            fallbackOwner: fallbackOwner
        });

        vm.prank(address(safeInstance.safe));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidConfigTimestamp.selector);
        ownerLivenessModule.configure(badCfg);
    }
}

contract OwnerLivenessModule_Attack_Test is OwnerLivenessModule_TestInit {
    /// @notice Tests that attack reverts when module not configured.
    function test_attack_notConfigured_reverts() external {
        vm.prank(challenger);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_NotConfigured.selector);
        unconfiguredModule.attack{ value: CHALLENGE_BOND }(
            Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]
        );
    }

    /// @notice Tests that attack reverts when incorrect bond value sent.
    function test_attack_incorrectBond_reverts() external {
        vm.prank(challenger);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_IncorrectBond.selector);
        ownerLivenessModule.attack{ value: CHALLENGE_BOND - 1 }(
            Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]
        );
    }

    /// @notice Tests that attack reverts when challenging a non-owner.
    function test_attack_invalidOwner_reverts() external {
        vm.prank(challenger);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidOwner.selector);
        ownerLivenessModule.attack{ value: CHALLENGE_BOND }(Safe(payable(address(safeInstance.safe))), nonOwner);
    }

    /// @notice Tests that attack reverts on duplicate challenge for same owner.
    function test_attack_duplicateChallenge_reverts() external {
        address challengedOwner = _createChallenge();
        vm.prank(challenger);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_DuplicateChallenge.selector);
        ownerLivenessModule.attack{ value: CHALLENGE_BOND }(Safe(payable(address(safeInstance.safe))), challengedOwner);
    }

    /// @notice Tests that attack succeeds against any valid owner in the Safe.
    /// @param _ownerIndex Index of owner to challenge (0 to owner count - 1).
    function testFuzz_attack_anyValidOwner_succeeds(uint256 _ownerIndex) external {
        _ownerIndex = bound(_ownerIndex, 0, safeInstance.owners.length - 1);
        address ownerToChallenge = safeInstance.owners[_ownerIndex];

        vm.expectEmit(true, true, true, true);
        emit ChallengeCreated(
            Safe(payable(address(safeInstance.safe))), ownerToChallenge, challenger, block.timestamp + CHALLENGE_PERIOD
        );

        vm.prank(challenger);
        ownerLivenessModule.attack{ value: CHALLENGE_BOND }(Safe(payable(address(safeInstance.safe))), ownerToChallenge);

        (uint256 ts, address chal, uint256 bond) =
            ownerLivenessModule.challenges(Safe(payable(address(safeInstance.safe))), ownerToChallenge);
        assertEq(chal, challenger);
        assertEq(bond, CHALLENGE_BOND);
        assertEq(ts, block.timestamp);
    }
}

contract OwnerLivenessModule_Defend_Test is OwnerLivenessModule_TestInit {
    /// @notice Tests that defend reverts when module not configured.
    function test_defend_notConfigured_reverts() external {
        address challengedOwner = safeInstance.owners[0];
        vm.prank(challengedOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_NotConfigured.selector);
        unconfiguredModule.defend(
            Safe(payable(address(safeInstance.safe))), challengedOwner, block.timestamp, bytes("")
        );
    }

    /// @notice Tests that defend reverts when no active challenge exists.
    function test_defend_noActiveChallenge_reverts() external {
        address challengedOwner = safeInstance.owners[0];
        vm.prank(challengedOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_NoActiveChallenge.selector);
        ownerLivenessModule.defend(
            Safe(payable(address(safeInstance.safe))), challengedOwner, block.timestamp, bytes("")
        );
    }

    /// @notice Tests that defend reverts when called after expiry.
    function test_defend_expired_reverts() external {
        address challengedOwner = _createChallenge();
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        vm.prank(challengedOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_ChallengeExpired.selector);
        ownerLivenessModule.defend(
            Safe(payable(address(safeInstance.safe))), challengedOwner, block.timestamp, bytes("")
        );
    }

    /// @notice Tests that defend reverts when timestamp is before challenge start.
    function test_defend_timestampTooEarly_reverts() external {
        address challengedOwner = _createChallenge();
        vm.prank(challengedOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidSignatureTimestamp.selector);
        ownerLivenessModule.defend(
            Safe(payable(address(safeInstance.safe))), challengedOwner, block.timestamp - 1, bytes("")
        );
    }

    /// @notice Tests that defend reverts when timestamp is after challenge end.
    function test_defend_timestampTooLate_reverts() external {
        address challengedOwner = _createChallenge();
        vm.prank(challengedOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidSignatureTimestamp.selector);
        ownerLivenessModule.defend(
            Safe(payable(address(safeInstance.safe))),
            challengedOwner,
            block.timestamp + CHALLENGE_PERIOD + 1,
            bytes("")
        );
    }

    /// @notice Tests that defend reverts with invalid signature from non-owner.
    function test_defend_invalidSignature_reverts() external {
        address challengedOwner = _createChallenge();
        uint256 timestamp = block.timestamp;
        bytes memory invalidSig = _createDefendSignature(nonOwnerKey, timestamp);

        vm.prank(nonOwner);
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_InvalidSignature.selector);
        ownerLivenessModule.defend(Safe(payable(address(safeInstance.safe))), challengedOwner, timestamp, invalidSig);
    }

    /// @notice Tests that defend succeeds with any valid timestamp within the challenge period.
    /// @param _timeOffset Time offset from challenge start (0 to CHALLENGE_PERIOD seconds).
    function testFuzz_defend_validTimestamp_succeeds(uint256 _timeOffset) external {
        address challengedOwner = _createChallenge();
        uint256 challengeStart = block.timestamp;

        _timeOffset = bound(_timeOffset, 0, CHALLENGE_PERIOD);
        uint256 defendTimestamp = challengeStart + _timeOffset;

        vm.expectEmit(true, true, true, true);
        emit ChallengeClosed(
            Safe(payable(address(safeInstance.safe))),
            challengedOwner,
            challenger,
            OwnerLivenessModule.ChallengeResult.FAILURE
        );

        vm.prank(challengedOwner);
        ownerLivenessModule.defend(
            Safe(payable(address(safeInstance.safe))), challengedOwner, defendTimestamp, bytes("")
        );

        // Challenge should be deleted and Safe rewarded.
        (uint256 ts, address chal, uint256 bond) =
            ownerLivenessModule.challenges(Safe(payable(address(safeInstance.safe))), challengedOwner);
        assertEq(ts, 0);
        assertEq(chal, address(0));
        assertEq(bond, 0);

        uint256 reward = ownerLivenessModule.rewards(address(safeInstance.safe));
        assertEq(reward, CHALLENGE_BOND);
    }
}

contract OwnerLivenessModule_Finalize_Test is OwnerLivenessModule_TestInit {
    using SafeTestLib for SafeInstance;

    /// @notice Tests that finalize reverts when module not configured.
    function test_finalize_notConfigured_reverts() external {
        vm.prank(makeAddr("caller"));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_NotConfigured.selector);
        unconfiguredModule.finalize(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]);
    }

    /// @notice Tests that finalize reverts when no active challenge exists.
    function test_finalize_noActiveChallenge_reverts() external {
        vm.prank(makeAddr("caller"));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_NoActiveChallenge.selector);
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]);
    }

    /// @notice Tests that finalize reverts if challenge still pending.
    function test_finalize_stillPending_reverts() external {
        _createChallenge();
        vm.prank(makeAddr("caller"));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_ChallengeStillPending.selector);
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]);
    }

    /// @notice Tests that finalize invalidates challenge when target is no longer owner.
    function test_finalize_ownerNoLongerValid_succeeds() external {
        address challengedOwner = _createChallenge();

        // Manually remove the owner from the Safe to test invalidation.
        address[] memory owners = safeInstance.safe.getOwners();
        address prevOwner = address(1);
        for (uint256 i = 0; i < owners.length; i++) {
            if (owners[i] == challengedOwner && i > 0) {
                prevOwner = owners[i - 1];
                break;
            }
        }
        safeInstance.removeOwner(prevOwner, challengedOwner, owners.length - 1);

        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        vm.expectEmit(true, true, true, true);
        emit ChallengeClosed(
            Safe(payable(address(safeInstance.safe))),
            challengedOwner,
            challenger,
            OwnerLivenessModule.ChallengeResult.INVALIDATED
        );

        vm.prank(makeAddr("caller"));
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), challengedOwner);

        // Challenge should be deleted and challenger gets bond back.
        (uint256 ts, address chal, uint256 bond) =
            ownerLivenessModule.challenges(Safe(payable(address(safeInstance.safe))), challengedOwner);
        assertEq(ts, 0);
        assertEq(chal, address(0));
        assertEq(bond, 0);
        assertEq(ownerLivenessModule.rewards(challenger), CHALLENGE_BOND);
    }

    /// @notice Tests that finalize succeeds and removes owner and pays challenger when above minOwners.
    function test_finalize_success_succeeds() external {
        address challengedOwner = _createChallenge();
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        uint256 ownersBefore = safeInstance.safe.getOwners().length;

        vm.expectEmit(true, true, true, true);
        emit ChallengeClosed(
            Safe(payable(address(safeInstance.safe))),
            challengedOwner,
            challenger,
            OwnerLivenessModule.ChallengeResult.SUCCESS
        );

        vm.prank(makeAddr("caller2"));
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), challengedOwner);

        // Owner removed and challenger rewarded.
        assertFalse(safeInstance.safe.isOwner(challengedOwner));
        assertEq(safeInstance.safe.getOwners().length, ownersBefore - 1);
        assertEq(ownerLivenessModule.rewards(challenger), CHALLENGE_BOND);
    }

    /// @notice Tests that finalize triggers fallback path when owner count drops below minOwners.
    function test_finalize_fallbackOwnerPath_succeeds() external {
        // Deploy new Safe with MIN_OWNERS so removing one triggers fallback.
        (, uint256[] memory keys) = SafeTestLib.makeAddrsAndKeys("olmSmall", MIN_OWNERS);
        safeInstance = _setupSafe(keys, MIN_OWNERS);
        ownerLivenessModule = new OwnerLivenessModule();
        safeInstance.enableModule(address(ownerLivenessModule));
        fallbackOwner = makeAddr("fbOwnerAlt");
        _configureModule();

        address challengedOwner = _createChallenge();
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);

        vm.expectEmit(true, true, true, true);
        emit OwnerSwapped(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0], fallbackOwner);

        vm.prank(makeAddr("caller3"));
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), challengedOwner);

        // Safe should now have exactly one owner – the fallbackOwner.
        address[] memory owners = safeInstance.safe.getOwners();
        assertEq(owners.length, 1);
        assertEq(owners[0], fallbackOwner);
        assertEq(safeInstance.safe.getThreshold(), 1);
        assertEq(ownerLivenessModule.rewards(challenger), CHALLENGE_BOND);
    }
}

contract OwnerLivenessModule_Claim_Test is OwnerLivenessModule_TestInit {
    /// @notice Tests that claim succeeds and rewards the caller.
    function test_claim_success_succeeds() external {
        address challenger = _createChallenge();
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]);

        uint256 balBefore = challenger.balance;
        vm.prank(challenger);
        ownerLivenessModule.claim();
        assertEq(challenger.balance, balBefore + CHALLENGE_BOND);
        assertEq(ownerLivenessModule.rewards(challenger), 0);
    }

    /// @notice Tests that claim succeeds with zero balance (no-op).
    function test_claim_zeroBalance_succeeds() external {
        address claimer = makeAddr("claimer");
        uint256 balBefore = claimer.balance;
        vm.prank(claimer);
        ownerLivenessModule.claim();
        assertEq(claimer.balance, balBefore);
        assertEq(ownerLivenessModule.rewards(claimer), 0);
    }

    /// @notice Tests that multiple claims in sequence work correctly.
    function test_claim_multipleClaims_succeeds() external {
        address challenger = _createChallenge();
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), safeInstance.owners[0]);

        // First claim should transfer the bond.
        uint256 balBefore = challenger.balance;
        vm.prank(challenger);
        ownerLivenessModule.claim();
        assertEq(challenger.balance, balBefore + CHALLENGE_BOND);

        // Second claim should be no-op.
        uint256 balAfterFirst = challenger.balance;
        vm.prank(challenger);
        ownerLivenessModule.claim();
        assertEq(challenger.balance, balAfterFirst);
        assertEq(ownerLivenessModule.rewards(challenger), 0);
    }

    /// @notice Tests that claim reverts when ETH transfer to msg.sender fails inside claim.
    function test_claim_transferFailed_reverts() external {
        OwnerLivenessModule_ClaimingReverter_Harness recv = new OwnerLivenessModule_ClaimingReverter_Harness();
        vm.deal(address(recv), CHALLENGE_BOND);

        address challengedOwner = safeInstance.owners[0];

        // Reverting contract initiates a challenge.
        vm.prank(address(recv));
        ownerLivenessModule.attack{ value: CHALLENGE_BOND }(Safe(payable(address(safeInstance.safe))), challengedOwner);

        // Finalize to credit reward to reverting contract.
        vm.warp(block.timestamp + CHALLENGE_PERIOD + 1);
        ownerLivenessModule.finalize(Safe(payable(address(safeInstance.safe))), challengedOwner);

        // Expect claim to revert due to failed ETH transfer.
        vm.prank(address(recv));
        vm.expectRevert(OwnerLivenessModule.OwnerLivenessModule_ClaimFailed.selector);
        recv.triggerClaim(address(ownerLivenessModule));
    }
}
