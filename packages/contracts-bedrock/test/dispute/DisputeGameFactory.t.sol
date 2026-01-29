// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Libraries
import "src/dispute/lib/Types.sol";
import "src/dispute/lib/Errors.sol";

// Interfaces
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";
// Mocks
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { SP1MockVerifier } from "test/dispute/zk/mocks/SP1MockVerifier.sol";

// OptimisticZk
import { OptimisticZkGame } from "src/dispute/zk/OptimisticZkGame.sol";
import { AccessManager } from "src/dispute/zk/AccessManager.sol";
import { ISP1Verifier } from "src/dispute/zk/ISP1Verifier.sol";

/// @notice A fake clone used for testing the `DisputeGameFactory` contract's `create` function.
contract DisputeGameFactory_FakeClone_Harness {
    function initialize() external payable {
        // noop
    }

    function extraData() external pure returns (bytes memory) {
        return hex"FF0420";
    }

    function parentHash() external pure returns (bytes32) {
        return bytes32(0);
    }

    function rootClaim() external pure returns (Claim) {
        return Claim.wrap(bytes32(0));
    }
}

/// @title DisputeGameFactory_TestInit
/// @notice Reusable test initialization for `DisputeGameFactory` tests.
abstract contract DisputeGameFactory_TestInit is CommonTest {
    DisputeGameFactory_FakeClone_Harness fakeClone;

    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);
    event ImplementationSet(address indexed impl, GameType indexed gameType);
    event ImplementationArgsSet(GameType indexed gameType, bytes args);
    event InitBondUpdated(GameType indexed gameType, uint256 indexed newBond);

    uint256 l2ChainId = 111;

    function setUp() public virtual override {
        super.setUp();
        fakeClone = new DisputeGameFactory_FakeClone_Harness();

        // Transfer ownership of the factory to the test contract.
        vm.prank(disputeGameFactory.owner());
        disputeGameFactory.transferOwnership(address(this));
    }

    /// @notice Creates a new VM instance with the given absolute prestate
    function _createVM(Claim _absolutePrestate) internal returns (AlphabetVM vm_, IPreimageOracle preimageOracle_) {
        // Set preimage oracle challenge period to something arbitrary (4 seconds) just so we can
        // actually test the clock extensions later on. This is not a realistic value.
        preimageOracle_ = IPreimageOracle(
            DeployUtils.create1({
                _name: "PreimageOracle",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IPreimageOracle.__constructor__, (0, 4)))
            })
        );
        vm_ = new AlphabetVM(_absolutePrestate, preimageOracle_);
    }

    function _getGameConstructorParams()
        internal
        pure
        returns (IFaultDisputeGame.GameConstructorParams memory params_)
    {
        return IFaultDisputeGame.GameConstructorParams({
            maxGameDepth: 2 ** 3,
            splitDepth: 2 ** 2,
            clockExtension: Duration.wrap(3 hours),
            maxClockDuration: Duration.wrap(3.5 days)
        });
    }

    function _getSuperGameConstructorParams()
        internal
        pure
        returns (ISuperFaultDisputeGame.GameConstructorParams memory params_)
    {
        return ISuperFaultDisputeGame.GameConstructorParams({
            maxGameDepth: 2 ** 3,
            splitDepth: 2 ** 2,
            clockExtension: Duration.wrap(3 hours),
            maxClockDuration: Duration.wrap(3.5 days)
        });
    }

    function _setGame(address _gameImpl, GameType _gameType) internal {
        _setGame(_gameImpl, _gameType, false, "");
    }

    function _setGame(address _gameImpl, GameType _gameType, bytes memory _implArgs) internal {
        _setGame(_gameImpl, _gameType, true, _implArgs);
    }

    function _setGame(address _gameImpl, GameType _gameType, bool _hasImplArgs, bytes memory _implArgs) internal {
        vm.startPrank(disputeGameFactory.owner());
        if (_hasImplArgs) {
            disputeGameFactory.setImplementation(_gameType, IDisputeGame(_gameImpl), _implArgs);
        } else {
            disputeGameFactory.setImplementation(_gameType, IDisputeGame(_gameImpl));
        }
        disputeGameFactory.setInitBond(_gameType, DEFAULT_DISPUTE_GAME_INIT_BOND);
        vm.stopPrank();
    }

    /// @notice Sets up a super cannon game implementation
    function setupSuperFaultDisputeGame(Claim _absolutePrestate)
        internal
        returns (address gameImpl_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        bytes memory immutableArgs;
        (immutableArgs, vm_, preimageOracle_) = getSuperFaultDisputeGameImmutableArgs(_absolutePrestate);

        gameImpl_ = DeployUtils.create1({
            _name: "SuperFaultDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(ISuperFaultDisputeGame.__constructor__, (_getSuperGameConstructorParams()))
            )
        });

        _setGame(gameImpl_, GameTypes.SUPER_CANNON, immutableArgs);
    }

    /// @notice Sets up a super permissioned game implementation
    function setupSuperPermissionedDisputeGame(
        Claim _absolutePrestate,
        address _proposer,
        address _challenger
    )
        internal
        returns (address gameImpl_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        bytes memory implArgs;
        (implArgs, vm_, preimageOracle_) =
            getSuperPermissionedDisputeGameImmutableArgs(_absolutePrestate, _proposer, _challenger);
        gameImpl_ = setupSuperPermissionedDisputeGame(implArgs);
    }

    /// @notice Sets up immutable data for fault game implementation
    function getFaultDisputeGameImmutableArgs(Claim _absolutePrestate)
        internal
        returns (bytes memory immutableArgs_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        (vm_, preimageOracle_) = _createVM(_absolutePrestate);
        // Encode the implementation args for CWIA (tightly packed)
        immutableArgs_ = abi.encodePacked(
            _absolutePrestate, // 32 bytes
            vm_, // 20 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth, // 20 bytes
            l2ChainId // 32 bytes (l2ChainId)
        );
    }

    /// @notice Sets up immutable data for super fault dispute game implementation
    function getSuperFaultDisputeGameImmutableArgs(Claim _absolutePrestate)
        internal
        returns (bytes memory immutableArgs_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        (vm_, preimageOracle_) = _createVM(_absolutePrestate);
        // Encode the implementation args for CWIA (tightly packed)
        immutableArgs_ = abi.encodePacked(
            _absolutePrestate, // 32 bytes
            vm_, // 20 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth, // 20 bytes
            uint256(0) // 32 bytes (l2ChainId)
        );
    }

    /// @notice Sets up a fault game v2 implementation
    function setupFaultDisputeGame(Claim _absolutePrestate)
        internal
        returns (address gameImpl_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        bytes memory immutableArgs;
        (immutableArgs, vm_, preimageOracle_) = getFaultDisputeGameImmutableArgs(_absolutePrestate);
        gameImpl_ = setupFaultDisputeGame(immutableArgs);
    }

    function setupFaultDisputeGame(bytes memory immutableArgs) internal returns (address gameImpl_) {
        gameImpl_ = DeployUtils.create1({
            _name: "FaultDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(IFaultDisputeGame.__constructor__, (_getGameConstructorParams()))
            )
        });

        _setGame(gameImpl_, GameTypes.CANNON, immutableArgs);
    }

    function changeClaimStatus(Claim _claim, VMStatus _status) public pure returns (Claim out_) {
        assembly {
            out_ := or(and(not(shl(248, 0xFF)), _claim), shl(248, _status))
        }
    }

    /// @notice Sets up immutable args for PDG v2 implementation
    function getPermissionedDisputeGameImmutableArgs(
        Claim _absolutePrestate,
        address _proposer,
        address _challenger
    )
        internal
        returns (bytes memory implArgs_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        (vm_, preimageOracle_) = _createVM(_absolutePrestate);

        // Encode the implementation args for CWIA (tightly packed)
        implArgs_ = abi.encodePacked(
            _absolutePrestate, // 32 bytes
            vm_, // 20 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth, // 20 bytes
            l2ChainId, // 32 bytes (l2ChainId),
            _proposer, // 20 bytes
            _challenger // 20 bytes
        );
    }

    /// @notice Sets up immutable args for Super PDG implementation
    function getSuperPermissionedDisputeGameImmutableArgs(
        Claim _absolutePrestate,
        address _proposer,
        address _challenger
    )
        internal
        returns (bytes memory implArgs_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        (vm_, preimageOracle_) = _createVM(_absolutePrestate);

        // Encode the implementation args for CWIA (tightly packed)
        implArgs_ = abi.encodePacked(
            _absolutePrestate, // 32 bytes
            vm_, // 20 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth, // 20 bytes
            uint256(0), // 32 bytes (l2ChainId),
            _proposer, // 20 bytes
            _challenger // 20 bytes
        );
    }

    /// @notice Deploys PDG v2 implementation and sets it on the DGF
    function setupPermissionedDisputeGame(
        Claim _absolutePrestate,
        address _proposer,
        address _challenger
    )
        internal
        returns (address gameImpl_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        bytes memory implArgs;
        (implArgs, vm_, preimageOracle_) =
            getPermissionedDisputeGameImmutableArgs(_absolutePrestate, _proposer, _challenger);

        gameImpl_ = setupPermissionedDisputeGame(implArgs);
    }

    /// @notice Deploys PDG v2 implementation and sets it on the DGF
    function setupPermissionedDisputeGame(bytes memory _implArgs) internal returns (address gameImpl_) {
        gameImpl_ = DeployUtils.create1({
            _name: "PermissionedDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(IPermissionedDisputeGame.__constructor__, (_getGameConstructorParams()))
            )
        });

        _setGame(gameImpl_, GameTypes.PERMISSIONED_CANNON, _implArgs);
    }

    /// @notice Deploys Super PDG implementation and sets it on the DGF
    function setupSuperPermissionedDisputeGame(bytes memory _implArgs) internal returns (address gameImpl_) {
        gameImpl_ = DeployUtils.create1({
            _name: "SuperPermissionedDisputeGame",
            _args: DeployUtils.encodeConstructor(
                abi.encodeCall(ISuperPermissionedDisputeGame.__constructor__, (_getSuperGameConstructorParams()))
            )
        });
        _setGame(gameImpl_, GameTypes.SUPER_PERMISSIONED_CANNON, _implArgs);
    }

    /// @notice Parameters for OptimisticZk game setup
    struct OptimisticZkGameParams {
        Duration maxChallengeDuration;
        Duration maxProveDuration;
        address proposer;
        address challenger;
        bytes32 rollupConfigHash;
        bytes32 aggregationVkey;
        bytes32 rangeVkeyCommitment;
        uint256 challengerBond;
    }

    /// @notice Sets up an OptimisticZk game implementation
    function setupOptimisticZkGame(OptimisticZkGameParams memory _params)
        internal
        returns (address gameImpl_, AccessManager accessManager_, ISP1Verifier sp1Verifier_)
    {
        // Deploy mock verifier
        sp1Verifier_ = ISP1Verifier(address(new SP1MockVerifier()));

        // Deploy access manager
        accessManager_ = new AccessManager(2 weeks, disputeGameFactory);
        accessManager_.setProposer(_params.proposer, true);
        accessManager_.setChallenger(_params.challenger, true);

        // Deploy game implementation
        gameImpl_ = address(
            new OptimisticZkGame(
                _params.maxChallengeDuration,
                _params.maxProveDuration,
                disputeGameFactory,
                sp1Verifier_,
                _params.rollupConfigHash,
                _params.aggregationVkey,
                _params.rangeVkeyCommitment,
                _params.challengerBond,
                anchorStateRegistry,
                accessManager_
            )
        );

        // Set respected game type for OptimisticZk
        GameType gameType = GameTypes.OPTIMISTIC_ZK_GAME_TYPE;
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(gameType);

        // Register with factory
        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(gameType, IDisputeGame(gameImpl_));
        disputeGameFactory.setInitBond(gameType, _params.challengerBond);
        vm.stopPrank();
    }
}

/// @title DisputeGameFactory_Initialize_Test
/// @notice Tests the `initialize` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_Initialize_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that initialization reverts if called by a non-proxy admin or proxy admin
    ///         owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(
            _sender != address(disputeGameFactory.proxyAdmin()) && _sender != disputeGameFactory.proxyAdminOwner()
        );

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("DisputeGameFactory", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(disputeGameFactory), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector.
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender.
        vm.prank(_sender);
        disputeGameFactory.initialize(address(1234));
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal initialization
    ///         but confirms that the initValue is not incremented incorrectly if an upgrade
    ///         function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("DisputeGameFactory", "_initialized");

        // Get the initializer value.
        bytes32 slotVal = vm.load(address(disputeGameFactory), bytes32(slot.slot));
        uint8 val = uint8(uint256(slotVal) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, disputeGameFactory.initVersion());
    }
}

/// @title DisputeGameFactory_Create_Test
/// @notice Tests the `create` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_Create_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that the `create` function succeeds when creating a new dispute game with a
    ///         `GameType` that has an implementation set.
    function testFuzz_create_succeeds(
        uint32 gameType,
        Claim rootClaim,
        bytes calldata extraData,
        uint256 _value
    )
        public
    {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible
        // values.
        uint32 maxGameType = 8;
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, maxGameType)));
        // Ensure the rootClaim has a VMStatus that disagrees with the validity.
        rootClaim = changeClaimStatus(rootClaim, VMStatuses.INVALID);

        // Set all three implementations to the same `FakeClone` contract.
        for (uint8 i; i < maxGameType + 1; i++) {
            GameType lgt = GameType.wrap(i);
            disputeGameFactory.setImplementation(lgt, IDisputeGame(address(fakeClone)));
            disputeGameFactory.setInitBond(lgt, _value);
        }

        vm.deal(address(this), _value);

        uint256 gameCountBefore = disputeGameFactory.gameCount();

        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        IDisputeGame proxy = disputeGameFactory.create{ value: _value }(gt, rootClaim, extraData);

        (IDisputeGame game, Timestamp timestamp) = disputeGameFactory.games(gt, rootClaim, extraData);

        // Ensure that the dispute game was assigned to the `disputeGames` mapping.
        assertEq(address(game), address(proxy));
        assertEq(Timestamp.unwrap(timestamp), block.timestamp);
        assertEq(disputeGameFactory.gameCount(), gameCountBefore + 1);

        (, Timestamp timestamp2, IDisputeGame game2) = disputeGameFactory.gameAtIndex(gameCountBefore);
        assertEq(address(game2), address(proxy));
        assertEq(Timestamp.unwrap(timestamp2), block.timestamp);

        // Ensure that the game proxy received the bonded ETH.
        assertEq(address(proxy).balance, _value);
    }

    /// @notice Tests that the `create` function reverts when creating a new dispute game with an
    ///         incorrect bond amount.
    function testFuzz_create_incorrectBondAmount_reverts(
        uint32 gameType,
        Claim rootClaim,
        bytes calldata extraData
    )
        public
    {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible
        // values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));
        // Ensure the rootClaim has a VMStatus that disagrees with the validity.
        rootClaim = changeClaimStatus(rootClaim, VMStatuses.INVALID);

        // Set all three implementations to the same `FakeClone` contract.
        for (uint8 i; i < 3; i++) {
            GameType lgt = GameType.wrap(i);
            disputeGameFactory.setImplementation(lgt, IDisputeGame(address(fakeClone)));
            disputeGameFactory.setInitBond(lgt, 1 ether);
        }

        vm.expectRevert(IncorrectBondAmount.selector);
        disputeGameFactory.create(gt, rootClaim, extraData);
    }

    /// @notice Tests that the `create` function reverts when there is no implementation set for
    ///         the given `GameType`.
    function testFuzz_create_noImpl_reverts(uint32 gameType, Claim rootClaim, bytes calldata extraData) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible
        // values. We skip over game type = 0, since the deploy script set the implementation for
        // that game type.
        uint32 maxGameType = 8;
        GameType gt = GameType.wrap(uint32(bound(gameType, maxGameType + 1, type(uint32).max)));
        // Ensure the rootClaim has a VMStatus that disagrees with the validity.
        rootClaim = changeClaimStatus(rootClaim, VMStatuses.INVALID);

        vm.expectRevert(abi.encodeWithSelector(NoImplementation.selector, gt));
        disputeGameFactory.create(gt, rootClaim, extraData);
    }

    /// @notice Tests that the `create` function reverts when there exists a dispute game with the
    ///         same UUID.
    function testFuzz_create_sameUUID_reverts(uint32 gameType, Claim rootClaim, bytes calldata extraData) public {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible
        // values.
        uint32 maxGameType = 8;
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, maxGameType)));
        // Ensure the rootClaim has a VMStatus that disagrees with the validity.
        rootClaim = changeClaimStatus(rootClaim, VMStatuses.INVALID);

        // Set all three implementations to the same `FakeClone` contract.
        for (uint8 i; i < maxGameType + 1; i++) {
            disputeGameFactory.setImplementation(GameType.wrap(i), IDisputeGame(address(fakeClone)));
        }

        uint256 bondAmount = disputeGameFactory.initBonds(gt);

        // Create our first dispute game - this should succeed.
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        IDisputeGame proxy = disputeGameFactory.create{ value: bondAmount }(gt, rootClaim, extraData);

        (IDisputeGame game, Timestamp timestamp) = disputeGameFactory.games(gt, rootClaim, extraData);
        // Ensure that the dispute game was assigned to the `disputeGames` mapping.
        assertEq(address(game), address(proxy));
        assertEq(Timestamp.unwrap(timestamp), block.timestamp);

        // Ensure that the `create` function reverts when called with parameters that would result
        // in the same UUID.
        vm.expectRevert(
            abi.encodeWithSelector(GameAlreadyExists.selector, disputeGameFactory.getGameUUID(gt, rootClaim, extraData))
        );
        disputeGameFactory.create{ value: bondAmount }(gt, rootClaim, extraData);
    }

    function test_create_implArgs_succeeds() public {
        Claim absolutePrestate = Claim.wrap(bytes32(hex"dead"));
        (, AlphabetVM vm_,) = setupFaultDisputeGame(absolutePrestate);

        Claim rootClaim = changeClaimStatus(Claim.wrap(bytes32(hex"beef")), VMStatuses.INVALID);
        // extraData should contain the l2BlockNumber as first 32 bytes
        bytes memory extraData = bytes.concat(bytes32(uint256(type(uint32).max)));

        uint256 bondAmount = disputeGameFactory.initBonds(GameTypes.CANNON);
        vm.deal(address(this), bondAmount);

        // Create the game
        IDisputeGame proxy = disputeGameFactory.create{ value: bondAmount }(GameTypes.CANNON, rootClaim, extraData);

        // Verify the game was created and stored
        (IDisputeGame game, Timestamp timestamp) = disputeGameFactory.games(GameTypes.CANNON, rootClaim, extraData);

        assertEq(address(game), address(proxy));
        assertEq(Timestamp.unwrap(timestamp), block.timestamp);

        // Verify the game has the correct parameters via CWIA
        IFaultDisputeGame game_ = IFaultDisputeGame(address(proxy));

        // Test CWIA getters
        assertEq(Claim.unwrap(game_.absolutePrestate()), Claim.unwrap(absolutePrestate));
        assertEq(Claim.unwrap(game_.rootClaim()), Claim.unwrap(rootClaim));
        assertEq(game_.extraData(), extraData);
        assertEq(game_.l2ChainId(), l2ChainId);
        assertEq(address(game_.gameCreator()), address(this));
        assertEq(game_.l2BlockNumber(), uint256(type(uint32).max));
        assertEq(address(game_.vm()), address(vm_));
        assertEq(address(game_.weth()), address(delayedWeth));
        assertEq(address(game_.anchorStateRegistry()), address(anchorStateRegistry));
        // Test Constructor args
        assertEq(GameType.unwrap(game_.gameType()), GameType.unwrap(GameTypes.CANNON));
        assertEq(game_.maxGameDepth(), 2 ** 3);
        assertEq(game_.splitDepth(), 2 ** 2);
        assertEq(Duration.unwrap(game_.clockExtension()), Duration.unwrap(Duration.wrap(3 hours)));
        assertEq(Duration.unwrap(game_.maxClockDuration()), Duration.unwrap(Duration.wrap(3.5 days)));
    }

    /// @notice Tests that games get unique addresses based on their inputs (gameType, rootClaim, extraData)
    ///         even when the factory's nonce is unchanged. This test would fail if CREATE was used instead
    ///         of CREATE2, since CREATE only depends on deployer address and nonce.
    function test_create_uniqueAddressFromInputs_succeeds() public {
        // Set up the implementation for the game type
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(address(fakeClone)));
        disputeGameFactory.setInitBond(GameTypes.CANNON, 0);

        Claim rootClaim1 = changeClaimStatus(Claim.wrap(bytes32(uint256(1))), VMStatuses.INVALID);
        Claim rootClaim2 = changeClaimStatus(Claim.wrap(bytes32(uint256(2))), VMStatuses.INVALID);

        // Take a snapshot of the state before creating any games
        uint256 snapshot = vm.snapshotState();

        // Create game with first set of inputs and store address
        address addr1 = address(disputeGameFactory.create(GameTypes.CANNON, rootClaim1, abi.encode(uint256(100))));

        // Revert to the snapshot - this resets the factory's nonce to what it was before
        vm.revertTo(snapshot);

        // Create game with different rootClaim at the "same nonce"
        address addr2 = address(disputeGameFactory.create(GameTypes.CANNON, rootClaim2, abi.encode(uint256(100))));

        // The addresses should be different because CREATE2 uses the inputs (via UUID salt)
        // With CREATE, these would be the same address since nonce is the same
        assertTrue(addr1 != addr2, "Different rootClaim should produce different address");

        // Revert again and test with different extraData
        vm.revertTo(snapshot);
        address addr3 = address(disputeGameFactory.create(GameTypes.CANNON, rootClaim1, abi.encode(uint256(200))));
        assertTrue(addr1 != addr3, "Different extraData should produce different address");

        // Revert again and test with different gameType
        vm.revertTo(snapshot);
        disputeGameFactory.setImplementation(GameTypes.PERMISSIONED_CANNON, IDisputeGame(address(fakeClone)));
        disputeGameFactory.setInitBond(GameTypes.PERMISSIONED_CANNON, 0);
        address addr4 =
            address(disputeGameFactory.create(GameTypes.PERMISSIONED_CANNON, rootClaim1, abi.encode(uint256(100))));
        assertTrue(addr1 != addr4, "Different gameType should produce different address");

        // Finally, verify that same inputs at the "same nonce" produce the same address (deterministic)
        vm.revertTo(snapshot);
        address addr5 = address(disputeGameFactory.create(GameTypes.CANNON, rootClaim1, abi.encode(uint256(100))));
        assertEq(addr1, addr5, "Same inputs should produce same address (CREATE2 is deterministic)");
    }
}

/// @title DisputeGameFactory_SetImplementation_Test
/// @notice Tests the `setImplementation` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_SetImplementation_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that the `setImplementation` function properly sets the implementation for a
    ///         given `GameType`.
    function test_setImplementation_succeeds() public {
        vm.expectEmit(true, true, true, true, address(disputeGameFactory));
        emit ImplementationSet(address(1), GameTypes.CANNON);

        // Set the implementation for the `GameTypes.CANNON` enum value.
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(address(1)));

        // Ensure that the implementation for the `GameTypes.CANNON` enum value is set.
        assertEq(address(disputeGameFactory.gameImpls(GameTypes.CANNON)), address(1));
    }

    /// @notice Tests that the `setImplementation` function reverts when called by a non-owner.
    function test_setImplementation_notOwner_reverts() public {
        // Ensure that the `setImplementation` function reverts when called by a non-owner.
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(address(1)));
    }

    /// @notice Tests that the `setImplementation` function with args properly sets the implementation
    ///         and args for a given `GameType`.
    function test_setImplementation_withArgs_succeeds() public {
        address fakeGame = address(1);
        Claim absolutePrestate = Claim.wrap(bytes32(hex"dead"));
        AlphabetVM vm_;
        IPreimageOracle preimageOracle_;
        (vm_, preimageOracle_) = _createVM(absolutePrestate);

        bytes memory args = abi.encodePacked(
            absolutePrestate, // 32 bytes
            vm_, // 20 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth, // 20 bytes
            l2ChainId // 32 bytes (l2ChainId)
        );

        vm.expectEmit(true, true, true, true, address(disputeGameFactory));
        emit ImplementationSet(address(1), GameTypes.CANNON);
        vm.expectEmit(true, true, true, true, address(disputeGameFactory));
        emit ImplementationArgsSet(GameTypes.CANNON, args);

        // Set the implementation and args for the `GameTypes.CANNON` enum value.
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(fakeGame), args);

        // Ensure that the implementation for the `GameTypes.CANNON` enum value is set.
        assertEq(address(disputeGameFactory.gameImpls(GameTypes.CANNON)), address(1));
        // Ensure that the args for the `GameTypes.CANNON` enum value are set.
        assertEq(disputeGameFactory.gameArgs(GameTypes.CANNON), args);
    }

    /// @notice Tests that the `setImplementation` function with args reverts when called by a non-owner.
    function test_setImplementationArgs_notOwner_reverts() public {
        bytes memory args = abi.encode(uint256(123), address(0xdead));

        // Ensure that the `setImplementation` function reverts when called by a non-owner.
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        disputeGameFactory.setImplementation(GameTypes.CANNON, IDisputeGame(address(1)), args);
    }
}

/// @title DisputeGameFactory_SetInitBond_Test
/// @notice Tests the `setInitBond` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_SetInitBond_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that the `setInitBond` function properly sets the init bond for a given
    ///         `GameType`.
    function test_setInitBond_succeeds() public {
        vm.expectEmit(true, true, true, true, address(disputeGameFactory));
        emit InitBondUpdated(GameTypes.CANNON, 1 ether);

        // Set the init bond for the `GameTypes.CANNON` enum value.
        disputeGameFactory.setInitBond(GameTypes.CANNON, 1 ether);

        // Ensure that the init bond for the `GameTypes.CANNON` enum value is set.
        assertEq(disputeGameFactory.initBonds(GameTypes.CANNON), 1 ether);

        vm.expectEmit(true, true, true, true, address(disputeGameFactory));
        emit InitBondUpdated(GameTypes.CANNON, 2 ether);

        // Set the init bond for the `GameTypes.CANNON` enum value.
        disputeGameFactory.setInitBond(GameTypes.CANNON, 2 ether);

        // Ensure that the init bond for the `GameTypes.CANNON` enum value is set.
        assertEq(disputeGameFactory.initBonds(GameTypes.CANNON), 2 ether);
    }

    /// @notice Tests that the `setInitBond` function reverts when called by a non-owner.
    function test_setInitBond_notOwner_reverts() public {
        // Ensure that the `setInitBond` function reverts when called by a non-owner.
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        disputeGameFactory.setInitBond(GameTypes.CANNON, 1 ether);
    }
}

/// @title DisputeGameFactory_GetGameUUID_Test
/// @notice Tests the `getGameUUID` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_GetGameUUID_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that the `getGameUUID` function returns the correct hash when comparing
    ///         against the keccak256 hash of the abi-encoded parameters.
    function testDiff_getGameUUID_succeeds(uint32 gameType, Claim rootClaim, bytes calldata extraData) public view {
        // Ensure that the `gameType` is within the bounds of the `GameType` enum's possible
        // values.
        GameType gt = GameType.wrap(uint8(bound(gameType, 0, 2)));

        assertEq(
            Hash.unwrap(disputeGameFactory.getGameUUID(gt, rootClaim, extraData)),
            keccak256(abi.encode(gt, rootClaim, extraData))
        );
    }
}

/// @title DisputeGameFactory_FindLatestGames_Test
/// @notice Tests the `findLatestGames` function of the `DisputeGameFactory` contract.
contract DisputeGameFactory_FindLatestGames_Test is DisputeGameFactory_TestInit {
    function setUp() public override {
        super.setUp();

        // Set three implementations to the same `FakeClone` contract.
        for (uint8 i; i < 3; i++) {
            GameType lgt = GameType.wrap(i);
            disputeGameFactory.setImplementation(lgt, IDisputeGame(address(fakeClone)));
            disputeGameFactory.setInitBond(lgt, 0);
        }
    }

    /// @notice Tests that `findLatestGames` returns an empty array when the passed starting index
    ///         is greater than or equal to the game count.
    function testFuzz_findLatestGames_greaterThanLength_succeeds(uint256 _start) public {
        // Creation count should be 32 for normal tests, 5 for upgrade tests.
        uint256 creationCount = isForkTest() ? 5 : 32;

        // Create some dispute games of varying game types.
        for (uint256 i; i < creationCount; i++) {
            disputeGameFactory.create(GameType.wrap(uint8(i % 2)), Claim.wrap(bytes32(i)), abi.encode(i));
        }

        // Bound the starting index to a number greater than the length of the game list.
        uint256 gameCount = disputeGameFactory.gameCount();
        _start = bound(_start, gameCount, type(uint256).max);

        // The array's length should always be 0.
        IDisputeGameFactory.GameSearchResult[] memory games =
            disputeGameFactory.findLatestGames(GameTypes.CANNON, _start, 1);
        assertEq(games.length, 0);
    }

    /// @notice Tests that `findLatestGames` returns the correct games.
    function test_findLatestGames_static_succeeds() public {
        // Creation count should be 32 for normal tests, 5 for upgrade tests.
        uint256 creationCount = isForkTest() ? 5 : 32;

        // Create some dispute games of varying game types, repeatedly iterating over the game
        // types 0, 1, 2.
        for (uint256 i; i < creationCount; i++) {
            disputeGameFactory.create(GameType.wrap(uint8(i % 3)), Claim.wrap(bytes32(i)), abi.encode(i));
        }

        uint256 gameCount = disputeGameFactory.gameCount();

        IDisputeGameFactory.GameSearchResult[] memory games;

        uint256 start = gameCount - 1;

        // Find type 1 games.
        games = disputeGameFactory.findLatestGames(GameType.wrap(1), start, 1);
        assertEq(games.length, 1);

        // The type 1 game should be the last one added.
        assertEq(games[0].index, start);
        (GameType gameType, Timestamp createdAt, address game) = games[0].metadata.unpack();
        assertEq(gameType.raw(), 1);
        assertEq(createdAt.raw(), block.timestamp);

        // Find type 0 games.
        games = disputeGameFactory.findLatestGames(GameType.wrap(0), start, 1);
        assertEq(games.length, 1);

        // The type 0 game should be the second to last one added.
        assertEq(games[0].index, start - 1);
        (gameType, createdAt, game) = games[0].metadata.unpack();
        assertEq(gameType.raw(), 0);
        assertEq(createdAt.raw(), block.timestamp);

        // Find type 2 games.
        games = disputeGameFactory.findLatestGames(GameType.wrap(2), start, 1);
        assertEq(games.length, 1);

        // The type 2 game should be the third to last one added.
        assertEq(games[0].index, start - 2);
        (gameType, createdAt, game) = games[0].metadata.unpack();
        assertEq(gameType.raw(), 2);
        assertEq(createdAt.raw(), block.timestamp);
    }

    /// @notice Tests that `findLatestGames` returns the correct games, if there are less than `_n`
    ///         games of the given type available.
    function test_findLatestGames_lessThanNAvailable_succeeds() public {
        // Need to clear out the length of the game list on forked list to avoid massive iteration.
        if (isForkTest()) {
            vm.store(
                address(disputeGameFactory),
                bytes32(ForgeArtifacts.getSlot("DisputeGameFactory", "_disputeGameList").slot),
                bytes32(0)
            );
        }

        // Create some dispute games of varying game types.
        disputeGameFactory.create(GameType.wrap(1), Claim.wrap(bytes32(0)), abi.encode(0));
        disputeGameFactory.create(GameType.wrap(1), Claim.wrap(bytes32(uint256(1))), abi.encode(1));
        for (uint256 i; i < 1 << 3; i++) {
            disputeGameFactory.create(GameType.wrap(0), Claim.wrap(bytes32(i)), abi.encode(i));
        }

        // Grab the existing game count.
        uint256 gameCount = disputeGameFactory.gameCount();

        // Try to find 5 games of type 2, but there are none.
        IDisputeGameFactory.GameSearchResult[] memory games;
        games = disputeGameFactory.findLatestGames(GameType.wrap(2), gameCount - 1, 5);
        assertEq(games.length, 0);

        // Try to find 2 games of type 1, but there are only 2.
        games = disputeGameFactory.findLatestGames(GameType.wrap(1), gameCount - 1, 5);
        assertEq(games.length, 2);
        assertEq(games[0].index, 1);
        assertEq(games[1].index, 0);
    }

    /// @notice Tests that the expected number of games are returned when `findLatestGames` is
    ///         called.
    function testFuzz_findLatestGames_correctAmount_succeeds(
        uint256 _numGames,
        uint256 _numSearchedGames,
        uint256 _n
    )
        public
    {
        _numGames = bound(_numGames, 0, isForkTest() ? 5 : 256);
        _numSearchedGames = bound(_numSearchedGames, 0, _numGames);
        _n = bound(_n, 0, _numSearchedGames);

        // Create `_numGames` dispute games, with at least `_numSearchedGames` games.
        for (uint256 i; i < _numGames; i++) {
            uint32 gameType = i < _numSearchedGames ? 0 : 1;
            disputeGameFactory.create(GameType.wrap(gameType), Claim.wrap(bytes32(i)), abi.encode(i));
        }

        // Ensure that the correct number of games are returned.
        uint256 start = _numGames == 0 ? 0 : _numGames - 1;
        IDisputeGameFactory.GameSearchResult[] memory games =
            disputeGameFactory.findLatestGames(GameType.wrap(0), start, _n);
        assertEq(games.length, _n);
    }
}

/// @title DisputeGameFactory_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `DisputeGameFactory`
///         contract or are testing multiple functions at once.
contract DisputeGameFactory_Uncategorized_Test is DisputeGameFactory_TestInit {
    /// @notice Tests that the `owner` function returns the correct address after deployment.
    function test_owner_succeeds() public view {
        assertEq(disputeGameFactory.owner(), address(this));
    }

    /// @notice Tests that the `transferOwnership` function succeeds when called by the owner.
    function test_transferOwnership_succeeds() public {
        disputeGameFactory.transferOwnership(address(1));
        assertEq(disputeGameFactory.owner(), address(1));
    }

    /// @notice Tests that the `transferOwnership` function reverts when called by a non-owner.
    function test_transferOwnership_notOwner_reverts() public {
        vm.prank(address(0));
        vm.expectRevert("Ownable: caller is not the owner");
        disputeGameFactory.transferOwnership(address(1));
    }
}
