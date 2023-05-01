// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test, StdUtils } from "forge-std/Test.sol";

import { Claim } from "src/types/Types.sol";
import { GameType } from "src/types/Types.sol";
import { GameStatus } from "src/types/Types.sol";
import { Timestamp } from "src/types/Types.sol";

import { BondManager } from "src/BondManager.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";

import { ResourceMetering } from "contracts-bedrock/L1/ResourceMetering.sol";
import { SystemConfig } from "contracts-bedrock/L1/SystemConfig.sol";
import { L2OutputOracle } from "contracts-bedrock/L1/L2OutputOracle.sol";

contract CommonTest is Test {
    address alice = address(128);
    address bob = address(256);
    address multisig = address(512);

    bytes32 constant TYPE_HASH = 0x2676994b0652bcdf7968635d15b78aac9aaf797cc94c5adeb94376cc28f987d6;

    address immutable ZERO_ADDRESS = address(0);
    address immutable NON_ZERO_ADDRESS = address(1);
    uint256 immutable NON_ZERO_VALUE = 100;
    uint256 immutable ZERO_VALUE = 0;
    uint64 immutable NON_ZERO_GASLIMIT = 50000;
    bytes32 nonZeroHash = keccak256(abi.encode("NON_ZERO"));
    bytes NON_ZERO_DATA = hex"0000111122223333444455556666777788889999aaaabbbbccccddddeeeeffff0000";

    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 indexed version,
        bytes opaqueData
    );

    /// @notice Emitted when a new dispute game is created by the [DisputeGameFactory]
    event DisputeGameCreated(
        address indexed disputeProxy,
        GameType indexed gameType,
        Claim indexed rootClaim
    );

    function setUp() public virtual {
        // Give alice and bob some ETH
        vm.deal(alice, 1 << 16);
        vm.deal(bob, 1 << 16);
        vm.deal(multisig, 1 << 16);

        vm.label(alice, "alice");
        vm.label(bob, "bob");
        vm.label(multisig, "multisig");

        // Make sure we have a non-zero base fee
        vm.fee(1000000000);
    }

    function emitTransactionDeposited(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) internal {
        emit TransactionDeposited(
            _from,
            _to,
            0,
            abi.encodePacked(_mint, _value, _gasLimit, _isCreation, _data)
        );
    }
}

contract L2OutputOracle_Initializer is CommonTest {
    L2OutputOracle oracle;
    BondManager bondManager;
    DisputeGameFactory disputeGameFactory;
    SystemConfig systemConfig;
    MockAttestationDisputeGame disputeGameImplementation;
    MockAttestationDisputeGame disputeGameProxy;

    // SystemConfig `signerSet` keys
    uint256[] signerKeys;

    // Constructor arguments
    address guardian;
    uint256 internal minimumProposalCost = 1 ether;
    uint256 internal finalizationPeriodSeconds = 7 days;
    address internal proposer = 0x000000000000000000000000000000000000AbBa;
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal submissionInterval = 1800;
    uint256 internal l2BlockTime = 1;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 2;

    // Test data
    uint256 initL1Time;

    event OutputProposed(
        bytes32 indexed outputRoot,
        uint256 indexed l2OutputIndex,
        uint256 indexed l2BlockNumber,
        uint256 l1Timestamp
    );

    event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);

    // Advance the evm's time to meet the L2OutputOracle's requirements for proposeL2Output
    function warpToProposeTime(uint256 _nextBlockNumber) public {
        vm.warp(oracle.computeL2Timestamp(_nextBlockNumber) + 1);
    }

    function createMockAttestationGame() public returns (address) {
        Claim rootClaim = Claim.wrap(bytes32(""));
        bytes memory extraData = bytes("");
        MockAttestationDisputeGame implementation = new MockAttestationDisputeGame();
        GameType gt = GameType.ATTESTATION;
        disputeGameFactory.setImplementation(gt, IDisputeGame(address(implementation)));

        address proxy = address(disputeGameFactory.create(gt, rootClaim, extraData));

        return proxy;
    }

    function createEmptyAttestationGame() public returns (address) {
        MockAttestationDisputeGame emptyGame = new MockAttestationDisputeGame();
        return address(emptyGame);
    }

    function setUp() public virtual override {
        super.setUp();

        disputeGameFactory = new DisputeGameFactory(address(this));
        vm.label(address(disputeGameFactory), "DisputeGameFactory");
        bondManager = new BondManager(disputeGameFactory);
        vm.label(address(bondManager), "BondManager");

        ResourceMetering.ResourceConfig memory _config = ResourceMetering.ResourceConfig({
            maxResourceLimit: 1000000000,
            elasticityMultiplier: 2,
            baseFeeMaxChangeDenominator: 2,
            minimumBaseFee: 10,
            systemTxMaxGas: 100000000,
            maximumBaseFee: 1000
        });

        systemConfig = new SystemConfig(
            address(this), // _owner,
            100, // _overhead,
            100, // _scalar,
            keccak256("BATCHER.HASH"), // _batcherHash,
            type(uint64).max, // _gasLimit,
            address(0), // _unsafeBlockSigner,
            _config
        );
        vm.label(address(systemConfig), "SystemConfig");

        // Add 5 signers to the signer set
        for (uint256 i = 1; i < 6; i++) {
            signerKeys.push(i);
            systemConfig.authenticateSigner(vm.addr(i), true);
        }
        systemConfig.setSignatureThreshold(5);

        oracle = new L2OutputOracle({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _finalizationPeriodSeconds: finalizationPeriodSeconds,
            _bondManager: IBondManager(address(bondManager)),
            _disputeGameFactory: IDisputeGameFactory(address(disputeGameFactory))
        });
        vm.label(address(oracle), "L2OutputOracleImpl");

        // Create the dispute game implementation
        bytes memory extraData = hex"";
        Claim rootClaim = Claim.wrap(bytes32(0));
        GameType gt = GameType.ATTESTATION;
        disputeGameImplementation = new MockAttestationDisputeGame();
        vm.label(address(disputeGameImplementation), "AttestationDisputeGame_Implementation");

        // Set the implementation in the factory
        disputeGameFactory.setImplementation(gt, IDisputeGame(address(disputeGameImplementation)));

        // Create the attestation dispute game in the factory
        disputeGameProxy = MockAttestationDisputeGame(
            payable(address(disputeGameFactory.create(gt, rootClaim, extraData)))
        );
        assertEq(
            address(disputeGameFactory.games(gt, rootClaim, extraData)),
            address(disputeGameProxy)
        );
        vm.label(address(disputeGameProxy), "AttestationDisputeGame_Proxy");

        // Update the proxy fields
        disputeGameProxy.setBondManager(bondManager);
        disputeGameProxy.setRootClaim(rootClaim);
        disputeGameProxy.setGameStatus(GameStatus.CHALLENGER_WINS);
        // disputeGameProxy.setBondId(bondId);
        // disputeGameProxy.setExtraData(ed);
    }
}

/// @dev A mock dispute game for testing bond seizures.
contract MockAttestationDisputeGame is IDisputeGame {
    GameStatus internal gameStatus;
    BondManager bm;
    Claim internal rc;
    bytes internal ed;
    bytes32 internal bondId;

    address[] internal challengers;

    function getChallengers() public view returns (address[] memory) {
        return challengers;
    }

    function setBondId(bytes32 bid) external {
        bondId = bid;
    }

    function setBondManager(BondManager _bm) external {
        bm = _bm;
    }

    function setGameStatus(GameStatus _gs) external {
        gameStatus = _gs;
    }

    function setRootClaim(Claim _rc) external {
        rc = _rc;
    }

    function setExtraData(bytes memory _ed) external {
        ed = _ed;
    }

    /// @dev Allow the contract to receive ether
    receive() external payable {}

    fallback() external payable {}

    /// @dev Resolve the game with a split
    function splitResolve() public {
        challengers = [address(1), address(2)];
        bm.seizeAndSplit(bondId, challengers);
    }

    /// -------------------------------------------
    /// IInitializable Functions
    /// -------------------------------------------

    function initialize() external {
        /* noop */
    }

    /// -------------------------------------------
    /// IVersioned Functions
    /// -------------------------------------------

    function version() external pure returns (string memory _version) {
        return "0.1.0";
    }

    /// -------------------------------------------
    /// IDisputeGame Functions
    /// -------------------------------------------

    /// @notice Returns the timestamp that the DisputeGame contract was created at.
    function createdAt() external pure override returns (Timestamp _createdAt) {
        return Timestamp.wrap(uint64(0));
    }

    /// @notice Returns the current status of the game.
    function status() external view override returns (GameStatus _status) {
        return gameStatus;
    }

    /// @notice Getter for the game type.
    /// @dev `clones-with-immutable-args` argument #1
    /// @dev The reference impl should be entirely different depending on the type (fault, validity)
    ///      i.e. The game type should indicate the security model.
    /// @return _gameType The type of proof system being used.
    function gameType() external pure returns (GameType _gameType) {
        return GameType.ATTESTATION;
    }

    /// @notice Getter for the root claim.
    /// @return _rootClaim The root claim of the DisputeGame.
    /// @dev `clones-with-immutable-args` argument #2
    function rootClaim() external view override returns (Claim _rootClaim) {
        return rc;
    }

    /// @notice Getter for the extra data.
    /// @dev `clones-with-immutable-args` argument #3
    /// @return _extraData Any extra data supplied to the dispute game contract by the creator.
    function extraData() external view returns (bytes memory _extraData) {
        return ed;
    }

    /// @notice Returns the address of the `BondManager` used
    function bondManager() external view override returns (IBondManager _bondManager) {
        return IBondManager(address(bm));
    }

    /// @notice If all necessary information has been gathered, this function should mark the game
    ///         status as either `CHALLENGER_WINS` or `DEFENDER_WINS` and return the status of
    ///         the resolved game. It is at this stage that the bonds should be awarded to the
    ///         necessary parties.
    /// @dev May only be called if the `status` is `IN_PROGRESS`.
    function resolve() external returns (GameStatus _status) {
        bm.seize(bondId);
        return gameStatus;
    }
}
