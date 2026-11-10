// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { AlphabetVM } from "test/mocks/AlphabetVM.sol";
import { ZKMockVerifier } from "test/dispute/zk/ZKMockVerifier.sol";

// Scripts
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Contracts
import { ZKDisputeGame } from "src/dispute/zk/ZKDisputeGame.sol";

// Libraries
import { Claim, Duration, GameType, GameTypes, VMStatus } from "src/dispute/lib/Types.sol";
import { Types } from "src/libraries/Types.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";

// Interfaces
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { ISuperFaultDisputeGame } from "interfaces/dispute/ISuperFaultDisputeGame.sol";
import { IPermissionedDisputeGame } from "interfaces/dispute/IPermissionedDisputeGame.sol";
import { ISuperPermissionedDisputeGame } from "interfaces/dispute/ISuperPermissionedDisputeGame.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

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

    /// @notice Sets up a super cannon game implementation at the default SUPER_CANNON slot.
    function setupSuperFaultDisputeGame(Claim _absolutePrestate)
        internal
        returns (address gameImpl_, AlphabetVM vm_, IPreimageOracle preimageOracle_)
    {
        return setupSuperFaultDisputeGame(_absolutePrestate, GameTypes.SUPER_CANNON);
    }

    /// @notice Sets up a super cannon game implementation registered at the given game type.
    /// @dev The SuperFaultDisputeGame impl is type-agnostic (the game type is appended by the
    ///      factory), so the same impl can be registered at SUPER_CANNON or SUPER_CANNON_KONA.
    function setupSuperFaultDisputeGame(
        Claim _absolutePrestate,
        GameType _gameType
    )
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

        _setGame(gameImpl_, _gameType, immutableArgs);
    }

    /// @notice Sets up a super permissioned game implementation
    function setupSuperPermissionedDisputeGame(address _proposer) internal returns (address gameImpl_) {
        bytes memory implArgs = getSuperPermissionedDisputeGameImmutableArgs(_proposer);
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
        _setGame(gameImpl_, GameTypes.CANNON_KONA, immutableArgs);
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
    function getSuperPermissionedDisputeGameImmutableArgs(address _proposer)
        internal
        view
        returns (bytes memory implArgs_)
    {
        implArgs_ = abi.encodePacked(anchorStateRegistry, _proposer);
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
            _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperPermissionedDisputeGame.__constructor__, ()))
        });
        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(GameTypes.SUPER_PERMISSIONED, IDisputeGame(gameImpl_), _implArgs);
        disputeGameFactory.setInitBond(GameTypes.SUPER_PERMISSIONED, 0);
        vm.stopPrank();
    }

    /// @notice Parameters for ZKDisputeGame setup
    struct ZKDisputeGameParams {
        Duration maxChallengeDuration;
        Duration maxProveDuration;
        bytes32 absolutePrestate;
        uint256 challengerBond;
    }

    /// @notice Builds a canonical SuperRootProof from `_pairs` at `_timestamp` and returns the
    ///         encoded ZK extraData (4-byte parentIndex prefix + SuperRootProof) together with the
    ///         `rootClaim` that hashes to that proof. Use this everywhere a ZK game is created so
    ///         the init-time hash binding check (`hashSuperRootProof(decode(extraData)) == rootClaim`)
    ///         passes.
    function _makeZKExtraDataAndClaim(
        uint32 _parentIndex,
        uint64 _timestamp,
        Types.OutputRootWithChainId[] memory _pairs
    )
        internal
        pure
        returns (bytes memory extraData_, Claim rootClaim_)
    {
        Types.SuperRootProof memory proof =
            Types.SuperRootProof({ version: bytes1(0x01), timestamp: _timestamp, outputRoots: _pairs });
        bytes memory superRootBytes = Encoding.encodeSuperRootProof(proof);
        extraData_ = abi.encodePacked(_parentIndex, superRootBytes);
        rootClaim_ = Claim.wrap(Hashing.hashSuperRootProof(proof));
    }

    /// @notice Convenience: single-chain SuperRootProof using this test fixture's `l2ChainId` and
    ///         a caller-supplied output root.
    function _makeZKExtraDataAndClaim(
        uint32 _parentIndex,
        uint64 _timestamp,
        bytes32 _outputRoot
    )
        internal
        view
        returns (bytes memory extraData_, Claim rootClaim_)
    {
        Types.OutputRootWithChainId[] memory pairs = new Types.OutputRootWithChainId[](1);
        pairs[0] = Types.OutputRootWithChainId({ chainId: l2ChainId, root: _outputRoot });
        return _makeZKExtraDataAndClaim(_parentIndex, _timestamp, pairs);
    }

    /// @notice Sets up a ZKDisputeGame implementation with gameArgs
    function setupZKDisputeGame(ZKDisputeGameParams memory _params)
        internal
        returns (address gameImpl_, IZKVerifier zkVerifier_)
    {
        // Deploy mock verifier
        zkVerifier_ = IZKVerifier(address(new ZKMockVerifier()));

        // Deploy game implementation (no constructor args)
        gameImpl_ = address(new ZKDisputeGame());

        GameType zkGameType = GameTypes.ZK_DISPUTE_GAME;

        // Encode the gameArgs for CWIA (tightly packed). The super-root ZKDisputeGame derives
        // chain scoping from the SuperRootProof preimage committed to via rootClaim, so no
        // l2ChainId field is included in the layout.
        bytes memory gameArgs = abi.encodePacked(
            _params.absolutePrestate, // 32 bytes
            zkVerifier_, // 20 bytes
            _params.maxChallengeDuration, // 8 bytes
            _params.maxProveDuration, // 8 bytes
            _params.challengerBond, // 32 bytes
            anchorStateRegistry, // 20 bytes
            delayedWeth // 20 bytes
        );

        // Set respected game type
        vm.prank(superchainConfig.guardian());
        anchorStateRegistry.setRespectedGameType(zkGameType);

        // Register with factory using 3-arg setImplementation
        vm.startPrank(disputeGameFactory.owner());
        disputeGameFactory.setImplementation(zkGameType, IDisputeGame(gameImpl_), gameArgs);
        disputeGameFactory.setInitBond(zkGameType, _params.challengerBond);
        vm.stopPrank();
    }
}
