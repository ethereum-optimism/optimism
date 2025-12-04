// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import "forge-std/Test.sol";
import { ERC1967Proxy } from "@openzeppelin/contracts/proxy/ERC1967/ERC1967Proxy.sol";

// Libraries
import { Claim, Duration, GameStatus, GameType, Hash, Timestamp, Proposal } from "src/dispute/lib/Types.sol";
import {
    BadAuth,
    IncorrectBondAmount,
    UnexpectedRootClaim,
    NoCreditToClaim,
    GameNotFinalized,
    ParentGameNotResolved,
    InvalidParentGame,
    ClaimAlreadyChallenged,
    GameOver,
    GameNotOver,
    IncorrectDisputeGameFactory
} from "src/dispute/lib/Errors.sol";
import { OP_SUCCINCT_FAULT_DISPUTE_GAME_TYPE } from "src/dispute/lib/Types.sol";
import { Constants } from "src/libraries/Constants.sol";

// Contracts
import { DisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { OPSuccinctFaultDisputeGame } from "src/dispute/zk/OPSuccinctFaultDisputeGame.sol";
import { AnchorStateRegistry } from "src/dispute/AnchorStateRegistry.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { AccessManager } from "src/dispute/zk/AccessManager.sol";
import { SP1MockVerifier } from "./mocks/SP1MockVerifier.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISP1Verifier } from "src/dispute/zk/ISP1Verifier.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

/// @title OPSuccinctFaultDisputeGame_TestInit
/// @notice Base test contract with shared setup for OPSuccinctFaultDisputeGame tests.
abstract contract OPSuccinctFaultDisputeGame_TestInit is Test {
    // Events
    event Challenged(address indexed challenger);
    event Proved(address indexed prover);
    event Resolved(GameStatus indexed status);

    DisputeGameFactory factory;
    ERC1967Proxy factoryProxy;

    OPSuccinctFaultDisputeGame gameImpl;
    OPSuccinctFaultDisputeGame parentGame;
    OPSuccinctFaultDisputeGame game;

    AnchorStateRegistry anchorStateRegistry;
    AccessManager accessManager;

    address proposer = address(0x123);
    address challenger = address(0x456);
    address prover = address(0x789);
    address guardian = address(0xABCD);

    uint256 disputeGameFinalityDelaySeconds = 1000;

    // Fixed parameters.
    GameType gameType = GameType.wrap(OP_SUCCINCT_FAULT_DISPUTE_GAME_TYPE);
    Duration maxChallengeDuration = Duration.wrap(12 hours);
    Duration maxProveDuration = Duration.wrap(3 days);
    Claim rootClaim = Claim.wrap(keccak256("rootClaim"));

    // Child game creation parameters.
    uint256 l2BlockNumber = 2000;
    uint32 parentIndex = 0;

    // For a new parent game that we manipulate separately in some tests.
    OPSuccinctFaultDisputeGame separateParentGame;

    function setUp() public virtual {
        // Deploy the implementation contract for DisputeGameFactory.
        DisputeGameFactory factoryImpl = new DisputeGameFactory();

        // Deploy a proxy pointing to the factory implementation.
        factoryProxy = new ERC1967Proxy(address(factoryImpl), new bytes(0));

        // Set the ProxyAdmin in the reserved storage slot so ProxyAdminOwnedBase can find it.
        // The test contract itself acts as the ProxyAdmin for simplicity.
        vm.store(address(factoryProxy), Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(address(this)))));

        // Now initialize the factory.
        factory = DisputeGameFactory(address(factoryProxy));
        factory.initialize(address(this));

        // Create a mock verifier.
        SP1MockVerifier sp1Verifier = new SP1MockVerifier();

        // Deploy real SuperchainConfig.
        SuperchainConfig superchainConfigImpl = new SuperchainConfig();
        ERC1967Proxy superchainConfigProxy = new ERC1967Proxy(address(superchainConfigImpl), new bytes(0));
        vm.store(
            address(superchainConfigProxy), Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(address(this))))
        );
        ISuperchainConfig superchainConfig = ISuperchainConfig(address(superchainConfigProxy));
        SuperchainConfig(address(superchainConfigProxy)).initialize(guardian);

        // Deploy real SystemConfig.
        SystemConfig systemConfigImpl = new SystemConfig();
        ERC1967Proxy systemConfigProxy = new ERC1967Proxy(address(systemConfigImpl), new bytes(0));
        vm.store(address(systemConfigProxy), Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(address(this)))));
        ISystemConfig systemConfig = ISystemConfig(address(systemConfigProxy));
        SystemConfig(address(systemConfigProxy)).initialize(
            address(this), // owner
            1368, // basefeeScalar
            810949, // blobbasefeeScalar
            bytes32(uint256(uint160(makeAddr("batcher")))), // batcherHash
            30_000_000, // gasLimit
            makeAddr("sequencer"), // unsafeBlockSigner
            Constants.DEFAULT_RESOURCE_CONFIG(),
            makeAddr("batchInbox"), // batchInbox
            SystemConfig.Addresses({
                l1CrossDomainMessenger: makeAddr("l1CrossDomainMessenger"),
                l1ERC721Bridge: makeAddr("l1ERC721Bridge"),
                l1StandardBridge: makeAddr("l1StandardBridge"),
                optimismPortal: makeAddr("optimismPortal"),
                optimismMintableERC20Factory: makeAddr("optimismMintableERC20Factory"),
                delayedWETH: address(0)
            }),
            901, // l2ChainId
            superchainConfig
        );

        // Deploy real AnchorStateRegistry with dependencies.
        // First deploy the real AnchorStateRegistry implementation.
        AnchorStateRegistry anchorStateRegistryImpl = new AnchorStateRegistry(disputeGameFinalityDelaySeconds);

        // Deploy a proxy for AnchorStateRegistry.
        ERC1967Proxy anchorStateRegistryProxy = new ERC1967Proxy(address(anchorStateRegistryImpl), new bytes(0));

        // Set the ProxyAdmin in the reserved storage slot.
        vm.store(
            address(anchorStateRegistryProxy), Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(address(this))))
        );

        // Initialize the real AnchorStateRegistry.
        anchorStateRegistry = AnchorStateRegistry(address(anchorStateRegistryProxy));
        anchorStateRegistry.initialize(
            systemConfig,
            IDisputeGameFactory(address(factory)),
            Proposal({ root: Hash.wrap(keccak256("genesis")), l2SequenceNumber: 0 }),
            gameType
        );

        // Create a new access manager with 1 hour permissionless timeout.
        accessManager = new AccessManager(2 weeks, IDisputeGameFactory(address(factory)));
        accessManager.setProposer(proposer, true);
        accessManager.setChallenger(challenger, true);

        // Parameters for the OPSuccinctFaultDisputeGame.
        bytes32 rollupConfigHash = bytes32(0);
        bytes32 aggregationVkey = bytes32(0);
        bytes32 rangeVkeyCommitment = bytes32(0);
        uint256 proofReward = 1 ether;

        // Deploy the reference implementation of OPSuccinctFaultDisputeGame.
        gameImpl = new OPSuccinctFaultDisputeGame(
            maxChallengeDuration,
            maxProveDuration,
            IDisputeGameFactory(address(factory)),
            ISP1Verifier(address(sp1Verifier)),
            rollupConfigHash,
            aggregationVkey,
            rangeVkeyCommitment,
            proofReward,
            IAnchorStateRegistry(address(anchorStateRegistry)),
            accessManager
        );

        // Set the init bond on the factory for the OPSuccinctFDG specific GameType.
        factory.setInitBond(gameType, 1 ether);

        // Register our reference implementation under the specified gameType.
        factory.setImplementation(gameType, IDisputeGame(address(gameImpl)));

        // Create the first (parent) game – it uses uint32.max as parent index.
        vm.startPrank(proposer);
        vm.deal(proposer, 2 ether); // extra funds for testing.

        // Warp time forward to ensure the parent game is created after the respectedGameTypeUpdatedAt timestamp.
        vm.warp(block.timestamp + 1000);

        // This parent game will be at index 0.
        parentGame = OPSuccinctFaultDisputeGame(
            address(
                factory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("genesis")),
                    // encode l2BlockNumber = 1000, parentIndex = uint32.max.
                    abi.encodePacked(uint256(1000), type(uint32).max)
                )
            )
        );

        // We want the parent game to finalize. We'll skip its challenge period.
        (,,,,, Timestamp parentGameDeadline) = parentGame.claimData();
        vm.warp(parentGameDeadline.raw() + 1 seconds);
        parentGame.resolve();

        vm.warp(parentGame.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        parentGame.claimCredit(proposer);

        // Create the child game referencing parent index = 0.
        // The child game is at index 1.
        game = OPSuccinctFaultDisputeGame(
            address(
                factory.create{ value: 1 ether }(
                    gameType,
                    rootClaim,
                    // encode l2BlockNumber = 2000, parentIndex = 0.
                    abi.encodePacked(l2BlockNumber, parentIndex)
                )
            )
        );

        vm.stopPrank();
    }
}

/// @title OPSuccinctFaultDisputeGame_Initialize_Test
/// @notice Tests for initialization of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_Initialize_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_initialize_succeeds() public view {
        // Test that the factory is correctly initialized.
        assertEq(address(factory.owner()), address(this));
        assertEq(address(factory.gameImpls(gameType)), address(gameImpl));
        // We expect two games so far (parentGame at index 0, game at index 1).
        assertEq(factory.gameCount(), 2);

        // Check that the second game (our child game) matches the 'gameAtIndex(1)'.
        (,, IDisputeGame proxy_) = factory.gameAtIndex(1);
        assertEq(address(game), address(proxy_));

        // Check the child game fields.
        assertEq(game.gameType().raw(), gameType.raw());
        assertEq(game.rootClaim().raw(), rootClaim.raw());
        assertEq(game.maxChallengeDuration().raw(), maxChallengeDuration.raw());
        assertEq(game.maxProveDuration().raw(), maxProveDuration.raw());
        assertEq(address(game.disputeGameFactory()), address(factory));
        assertEq(game.l2SequenceNumber(), l2BlockNumber);

        // The parent's block number was 1000.
        assertEq(game.startingBlockNumber(), 1000);

        // The parent's root was keccak256("genesis").
        assertEq(game.startingRootHash().raw(), keccak256("genesis"));

        assertEq(address(game).balance, 1 ether);

        // Check the claimData.
        (
            uint32 parentIndex_,
            address counteredBy_,
            address prover_,
            Claim claim_,
            OPSuccinctFaultDisputeGame.ProposalStatus status_,
            Timestamp deadline_
        ) = game.claimData();

        assertEq(parentIndex_, 0);
        assertEq(counteredBy_, address(0));
        assertEq(game.gameCreator(), proposer);
        assertEq(prover_, address(0));
        assertEq(claim_.raw(), rootClaim.raw());

        // Initially, the status is Unchallenged.
        assertEq(uint8(status_), uint8(OPSuccinctFaultDisputeGame.ProposalStatus.Unchallenged));

        // The child's initial deadline is block.timestamp + maxChallengeDuration.
        uint256 currentTime = block.timestamp;
        uint256 expectedDeadline = currentTime + maxChallengeDuration.raw();
        assertEq(deadline_.raw(), expectedDeadline);
    }

    function test_initialize_blockNumberTooSmall_reverts() public {
        // The parent game used L2 block 1234567890.
        // Try to create a child game that references l2BlockNumber = 1.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        // We expect revert
        vm.expectRevert(
            abi.encodeWithSelector(
                UnexpectedRootClaim.selector,
                Claim.wrap(keccak256("rootClaim")) // The rootClaim we pass.
            )
        );

        factory.create{ value: 1 ether }(
            gameType,
            rootClaim,
            abi.encodePacked(uint256(1), uint32(0)) // L2 block is smaller than parent's block.
        );
        vm.stopPrank();
    }

    function test_initialize_parentBlacklisted_reverts() public {
        // Blacklist the game on the anchor state registry (which is what's actually used for validation).
        vm.prank(guardian);
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(game)));

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("blacklisted-parent-game")), abi.encodePacked(uint256(3000), uint32(1))
        );
        vm.stopPrank();
    }

    function test_initialize_parentNotRespected_reverts() public {
        // Create a new game at index 2 which will be the parent.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        OPSuccinctFaultDisputeGame parentNotRespected = OPSuccinctFaultDisputeGame(
            address(
                factory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("not-respected-parent-game")),
                    abi.encodePacked(uint256(3000), uint32(1))
                )
            )
        );
        vm.stopPrank();

        // Blacklist the parent game to make it invalid.
        vm.prank(guardian);
        anchorStateRegistry.blacklistDisputeGame(IDisputeGame(address(parentNotRespected)));

        // Try to create a game with a parent game that is not valid.
        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);
        vm.expectRevert(InvalidParentGame.selector);
        factory.create{ value: 1 ether }(
            gameType,
            Claim.wrap(keccak256("child-with-not-respected-parent")),
            abi.encodePacked(uint256(4000), uint32(2))
        );
        vm.stopPrank();
    }

    function test_initialize_noPermission_reverts() public {
        address maliciousProposer = address(0x1234);

        vm.startPrank(maliciousProposer);
        vm.deal(maliciousProposer, 1 ether);

        vm.expectRevert(BadAuth.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim")), abi.encodePacked(uint256(3000), uint32(1))
        );

        vm.stopPrank();
    }

    function test_initialize_wrongFactory_reverts() public {
        // Deploy the implementation contract for new DisputeGameFactory.
        DisputeGameFactory newFactoryImpl = new DisputeGameFactory();

        // Deploy a proxy pointing to the new factory implementation.
        ERC1967Proxy newFactoryProxy = new ERC1967Proxy(address(newFactoryImpl), new bytes(0));

        // Set the ProxyAdmin in the reserved storage slot so ProxyAdminOwnedBase can find it.
        vm.store(address(newFactoryProxy), Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(address(this)))));

        // Cast the proxy to the DisputeGameFactory interface and initialize it.
        DisputeGameFactory newFactory = DisputeGameFactory(address(newFactoryProxy));
        newFactory.initialize(address(this));

        // Set the implementation with the same implementation as the old factory.
        newFactory.setImplementation(gameType, IDisputeGame(address(gameImpl)));
        newFactory.setInitBond(gameType, 1 ether);

        vm.startPrank(proposer);
        vm.deal(proposer, 1 ether);

        vm.expectRevert(IncorrectDisputeGameFactory.selector);
        newFactory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim")), abi.encodePacked(uint256(3000), uint32(1))
        );

        vm.stopPrank();
    }
}

/// @title OPSuccinctFaultDisputeGame_Resolve_Test
/// @notice Tests for resolve functionality of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_Resolve_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_resolve_unchallenged_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Should revert if we try to resolve before deadline.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Warp forward past the challenge deadline.
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Expect the Resolved event.
        vm.expectEmit(true, false, false, false, address(game));
        emit Resolved(GameStatus.DEFENDER_WINS);

        // Now we can resolve successfully.
        game.resolve();

        // Proposer gets the bond back.
        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        game.claimCredit(proposer);

        // Check final state
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        // The contract should have paid back the proposer.
        assertEq(address(game).balance, 0);
        // Proposer posted 1 ether, so they get it back.
        assertEq(proposer.balance, 2 ether);
        assertEq(challenger.balance, 0);
    }

    function test_resolve_unchallengedWithProof_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Should revert if we try to resolve before the first challenge deadline.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Prover proves the claim while unchallenged.
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.stopPrank();

        // Now the proposal is UnchallengedAndValidProofProvided; we can resolve immediately.
        game.resolve();

        // Prover does not get any credit.
        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        vm.expectRevert(NoCreditToClaim.selector);
        game.claimCredit(prover);

        // Proposer gets the bond back.
        game.claimCredit(proposer);

        // Final status: DEFENDER_WINS.
        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);

        // Proposer gets their 1 ether back.
        assertEq(proposer.balance, 2 ether);
        // Prover does NOT get the reward because no challenger posted a bond.
        assertEq(prover.balance, 0 ether);
        assertEq(challenger.balance, 0);
    }

    function test_resolve_challengedWithProof_succeeds() public {
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));
        assertEq(address(game).balance, 1 ether);

        // Try to resolve too early.
        vm.expectRevert(GameNotOver.selector);
        game.resolve();

        // Challenger posts the bond incorrectly.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);

        // Must pay exactly the required bond.
        vm.expectRevert(IncorrectBondAmount.selector);
        game.challenge{ value: 0.5 ether }();

        // Correctly challenge the game.
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Now the contract holds 2 ether total.
        assertEq(address(game).balance, 2 ether);

        // Confirm the proposal is in Challenged state.
        (, address counteredBy_,,, OPSuccinctFaultDisputeGame.ProposalStatus challStatus,) = game.claimData();
        assertEq(counteredBy_, challenger);
        assertEq(uint8(challStatus), uint8(OPSuccinctFaultDisputeGame.ProposalStatus.Challenged));

        // Prover proves the claim in time.
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.stopPrank();

        // Confirm the proposal is now ChallengedAndValidProofProvided.
        (,,,, challStatus,) = game.claimData();
        assertEq(uint8(challStatus), uint8(OPSuccinctFaultDisputeGame.ProposalStatus.ChallengedAndValidProofProvided));
        assertEq(uint8(game.status()), uint8(GameStatus.IN_PROGRESS));

        // Resolve the game.
        game.resolve();

        // Prover gets the proof reward.
        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        game.claimCredit(prover);

        // Proposer gets the bond back.
        game.claimCredit(proposer);

        assertEq(uint8(game.status()), uint8(GameStatus.DEFENDER_WINS));
        assertEq(address(game).balance, 0);

        // Final balances:
        // - The proposer recovers their 1 ether stake.
        // - The prover gets 1 ether reward.
        // - The challenger gets nothing.
        assertEq(proposer.balance, 2 ether);
        assertEq(prover.balance, 1 ether);
        assertEq(challenger.balance, 0);
    }

    function test_resolve_challengedNoProof_succeeds() public {
        // Challenge the game.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // The contract now has 2 ether total.
        assertEq(address(game).balance, 2 ether);

        // We must wait for the prove deadline to pass.
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);

        // Now we can resolve, resulting in CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond.
        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        game.claimCredit(challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // The challenger receives the entire 3 ether.
        assertEq(challenger.balance, 3 ether); // started with 2, spent 1, got 2 from the game.

        // The proposer loses their 1 ether stake.
        assertEq(proposer.balance, 1 ether); // started with 2, lost 1.
        // The contract balance is zero.
        assertEq(address(game).balance, 0);
    }

    function test_resolve_parentGameInProgress_reverts() public {
        vm.startPrank(proposer);

        // Create a new game with parentIndex = 1.
        OPSuccinctFaultDisputeGame childGame = OPSuccinctFaultDisputeGame(
            address(
                factory.create{ value: 1 ether }(
                    gameType,
                    Claim.wrap(keccak256("new-claim")),
                    // encode l2BlockNumber = 3000, parentIndex = 1.
                    abi.encodePacked(uint256(3000), uint32(1))
                )
            )
        );

        vm.stopPrank();

        // The parent game is still in progress, not resolved.
        // So, if we try to resolve the childGame, it should revert with ParentGameNotResolved.
        vm.expectRevert(ParentGameNotResolved.selector);
        childGame.resolve();
    }

    function test_resolve_parentGameInvalid_succeeds() public {
        // 1) Now create a child game referencing that losing parent at index 1.
        vm.startPrank(proposer);
        OPSuccinctFaultDisputeGame childGame = OPSuccinctFaultDisputeGame(
            address(
                factory.create{ value: 1 ether }(
                    gameType, Claim.wrap(keccak256("child-of-loser")), abi.encodePacked(uint256(10000), uint32(1))
                )
            )
        );
        vm.stopPrank();

        // 2) Challenge the parent game so that it ends up CHALLENGER_WINS when proof is not provided within the prove
        // deadline.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // 3) Warp past the prove deadline.
        (,,,,, Timestamp gameDeadline) = game.claimData();
        vm.warp(gameDeadline.raw() + 1);

        // 4) The game resolves as CHALLENGER_WINS.
        game.resolve();

        // Challenger gets the bond back and wins proposer's bond.
        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        game.claimCredit(challenger);

        assertEq(uint8(game.status()), uint8(GameStatus.CHALLENGER_WINS));

        // 5) If we try to resolve the child game, it should be resolved as CHALLENGER_WINS
        // because parent's claim is invalid.
        // The child's bond is lost since there is no challenger for the child game.
        childGame.resolve();

        // Challenger hasn't challenged the child game, so it gets nothing.
        vm.warp(childGame.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);

        vm.expectRevert(NoCreditToClaim.selector);
        childGame.claimCredit(challenger);

        assertEq(uint8(childGame.status()), uint8(GameStatus.CHALLENGER_WINS));

        assertEq(address(childGame).balance, 1 ether);
        assertEq(address(challenger).balance, 3 ether);
        assertEq(address(proposer).balance, 0 ether);
    }
}

/// @title OPSuccinctFaultDisputeGame_Challenge_Test
/// @notice Tests for challenge functionality of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_Challenge_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_challenge_alreadyChallenged_reverts() public {
        // Initially unchallenged.
        (, address counteredBy_,,, OPSuccinctFaultDisputeGame.ProposalStatus status_,) = game.claimData();
        assertEq(counteredBy_, address(0));
        assertEq(uint8(status_), uint8(OPSuccinctFaultDisputeGame.ProposalStatus.Unchallenged));

        // The first challenge is valid.
        vm.startPrank(challenger);
        vm.deal(challenger, 2 ether);
        game.challenge{ value: 1 ether }();

        // A second challenge from any party should revert because the proposal is no longer "Unchallenged".
        vm.expectRevert(ClaimAlreadyChallenged.selector);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();
    }

    function test_challenge_noPermission_reverts() public {
        address maliciousChallenger = address(0x1234);

        vm.startPrank(maliciousChallenger);
        vm.deal(maliciousChallenger, 1 ether);

        vm.expectRevert(BadAuth.selector);
        game.challenge{ value: 1 ether }();

        vm.stopPrank();
    }
}

/// @title OPSuccinctFaultDisputeGame_Prove_Test
/// @notice Tests for prove functionality of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_Prove_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_prove_afterDeadline_reverts() public {
        // Challenge first.
        vm.startPrank(challenger);
        vm.deal(challenger, 1 ether);
        game.challenge{ value: 1 ether }();
        vm.stopPrank();

        // Move time forward beyond the prove period.
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);

        vm.startPrank(prover);
        // Attempting to prove after the deadline is exceeded.
        vm.expectRevert(GameOver.selector);
        game.prove(bytes(""));
        vm.stopPrank();
    }

    function test_prove_alreadyProved_reverts() public {
        vm.startPrank(prover);
        game.prove(bytes(""));
        vm.expectRevert(GameOver.selector);
        game.prove(bytes(""));
        vm.stopPrank();
    }
}

/// @title OPSuccinctFaultDisputeGame_ClaimCredit_Test
/// @notice Tests for claimCredit functionality of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_ClaimCredit_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_claimCredit_notFinalized_reverts() public {
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.expectRevert(GameNotFinalized.selector);
        game.claimCredit(proposer);
    }
}

/// @title OPSuccinctFaultDisputeGame_CloseGame_Test
/// @notice Tests for closeGame functionality of OPSuccinctFaultDisputeGame.
contract OPSuccinctFaultDisputeGame_CloseGame_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_closeGame_notResolved_reverts() public {
        vm.expectRevert(GameNotFinalized.selector);
        game.closeGame();
    }

    function test_closeGame_updatesAnchorGame_succeeds() public {
        (,,,,, Timestamp deadline) = game.claimData();
        vm.warp(deadline.raw() + 1);
        game.resolve();

        vm.warp(game.resolvedAt().raw() + disputeGameFinalityDelaySeconds + 1 seconds);
        game.closeGame();

        assertEq(address(anchorStateRegistry.anchorGame()), address(game));
    }
}

/// @title OPSuccinctFaultDisputeGame_AccessManager_Test
/// @notice Tests for AccessManager permissionless fallback functionality.
contract OPSuccinctFaultDisputeGame_AccessManager_Test is OPSuccinctFaultDisputeGame_TestInit {
    function test_accessManager_permissionlessAfterTimeout_succeeds() public {
        // Initially, unauthorized user should not be allowed
        address unauthorizedUser = address(0x9999);

        // Try to create a game as unauthorized user - should fail
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-1")), abi.encodePacked(uint256(3000), uint32(1))
        );

        vm.prank(proposer);
        vm.deal(proposer, 1 ether);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-2")), abi.encodePacked(l2BlockNumber, parentIndex)
        );

        // Warp time forward past the timeout
        vm.warp(block.timestamp + 2 weeks + 1);

        // Now unauthorized user should be allowed due to timeout
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-3")), abi.encodePacked(uint256(4000), uint32(1))
        );

        // After the new game, timeout resets - unauthorized user should not be allowed immediately
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-4")), abi.encodePacked(uint256(5000), uint32(1))
        );
    }

    function test_accessManager_permissionlessNoGamesAfterTimeout_succeeds() public {
        // Initially, unauthorized user should not be allowed
        address unauthorizedUser = address(0x9999);

        // Try to create a game as unauthorized user - should fail
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-1")), abi.encodePacked(uint256(3000), uint32(1))
        );

        // Warp time forward past the timeout
        vm.warp(block.timestamp + 2 weeks + 1 hours);

        // Now unauthorized user should be allowed due to timeout
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-3")), abi.encodePacked(uint256(4000), uint32(1))
        );

        // After the new game, timeout resets - unauthorized user should not be allowed immediately
        vm.prank(unauthorizedUser);
        vm.deal(unauthorizedUser, 1 ether);
        vm.expectRevert(BadAuth.selector);
        factory.create{ value: 1 ether }(
            gameType, Claim.wrap(keccak256("new-claim-4")), abi.encodePacked(uint256(5000), uint32(1))
        );
    }
}
