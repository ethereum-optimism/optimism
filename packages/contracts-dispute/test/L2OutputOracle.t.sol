// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "forge-std/Test.sol";

import "src/types/Errors.sol";
import "src/types/Types.sol";

import { LibClock } from "src/lib/LibClock.sol";
import { LibHashing } from "src/lib/LibHashing.sol";
import { LibPosition } from "src/lib/LibPosition.sol";

import { ResourceMetering } from "contracts-bedrock/L1/ResourceMetering.sol";
import { SystemConfig } from "contracts-bedrock/L1/SystemConfig.sol";
import { L2OutputOracle } from "contracts-bedrock/L1/L2OutputOracle.sol";

import { AttestationDisputeGame } from "src/AttestationDisputeGame.sol";
import { IDisputeGameFactory } from "src/interfaces/IDisputeGameFactory.sol";
import { IDisputeGame } from "src/interfaces/IDisputeGame.sol";
import { IBondManager } from "src/interfaces/IBondManager.sol";
import { BondManager } from "src/BondManager.sol";
import { DisputeGameFactory } from "src/DisputeGameFactory.sol";

/// @title L2OutputOracle Tests
contract L2OutputOracle_Test is Test {
    bytes32 constant TYPE_HASH = 0x2676994b0652bcdf7968635d15b78aac9aaf797cc94c5adeb94376cc28f987d6;

    DisputeGameFactory factory;
    BondManager bm;
    AttestationDisputeGame disputeGameImplementation;
    SystemConfig systemConfig;
    L2OutputOracle l2oo;
    AttestationDisputeGame disputeGameProxy;

    // L2OutputOracle Constructor arguments
    address internal proposer = 0x000000000000000000000000000000000000AbBa;
    address internal owner = 0x000000000000000000000000000000000000ACDC;
    uint256 internal submissionInterval = 1800;
    uint256 internal l2BlockTime = 1;
    uint256 internal startingBlockNumber = 200;
    uint256 internal startingTimestamp = 2;

    // SystemConfig `signerSet` keys
    uint256[] signerKeys;

    /// @notice Emitted when a new dispute game is created by the [DisputeGameFactory]
    event DisputeGameCreated(address indexed disputeProxy, GameType indexed gameType, Claim indexed rootClaim);

    function setUp() public {
        // vm.warp(startingTimestamp);

        factory = new DisputeGameFactory(address(this));
        vm.label(address(factory), "DisputeGameFactory");
        bm = new BondManager(factory);
        vm.label(address(bm), "BondManager");

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

        l2oo = new L2OutputOracle({
            _l2BlockTime: l2BlockTime,
            _startingBlockNumber: startingBlockNumber,
            _startingTimestamp: block.timestamp,
            _finalizationPeriodSeconds: 7 days,
            _bondManager: IBondManager(address(bm)),
            _disputeGameFactory: IDisputeGameFactory(address(factory))
        });
        vm.label(address(l2oo), "L2OutputOracle");

        // Create the dispute game implementation
        disputeGameImplementation = new AttestationDisputeGame(IBondManager(address(bm)), systemConfig, l2oo);
        vm.label(address(disputeGameImplementation), "AttestationDisputeGame_Implementation");

        // Set the implementation in the factory
        GameType gt = GameType.ATTESTATION;
        factory.setImplementation(gt, IDisputeGame(address(disputeGameImplementation)));

        // Create the attestation dispute game in the factory
        bytes memory extraData = hex"";
        Claim rootClaim = Claim.wrap(bytes32(0));
        vm.expectEmit(false, true, true, false);
        emit DisputeGameCreated(address(0), gt, rootClaim);
        disputeGameProxy = AttestationDisputeGame(address(factory.create(gt, rootClaim, extraData)));
        assertEq(address(factory.games(gt, rootClaim, extraData)), address(disputeGameProxy));
        vm.label(address(disputeGameProxy), "AttestationDisputeGame_Proxy");
    }

    /*****************************
     * Delete Tests - Happy Path *
     *****************************/

    function test_deleteOutputs_singleOutput_succeeds() external {
        test_proposeL2Output_proposeAnotherOutput_succeeds();
        test_proposeL2Output_proposeAnotherOutput_succeeds();

        uint256 highestL2BlockNumber = oracle.latestBlockNumber() + 1;
        Types.OutputProposal memory newLatestOutput = oracle.getL2Output(highestL2BlockNumber - 1);

        vm.prank(owner);
        vm.expectEmit(true, true, false, false);
        emit OutputsDeleted(0, highestL2BlockNumber);
        oracle.deleteL2Output(highestL2BlockNumber);

        // validate that the new latest output is as expected.
        Types.OutputProposal memory proposal = oracle.getL2Output(highestL2BlockNumber);
        assertEq(newLatestOutput.outputRoot, proposal.outputRoot);
        assertEq(newLatestOutput.timestamp, proposal.timestamp);
    }


    /***************************
     * Delete Tests - Sad Path *
     ***************************/

    function testFuzz_deleteL2Outputs_nonDisputeGame_reverts(address game) external {
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();

        vm.prank(game);
        vm.expectRevert();
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    function test_deleteL2Outputs_unauthorized_reverts() external {
        uint256 highestL2BlockNumber = oracle.startingBlockNumber();

        // Create the correct dispute game
        address proxy = createMockAttestationGame();
        Claim rootClaim = Claim.wrap(bytes32(""));
        bytes memory extraData = bytes("");
        GameType gt = GameType.ATTESTATION;
        assertEq(address(disputeGameFactory.games(gt, rootClaim, extraData)), proxy);

        // Call delete from an unauthorized game
        address badGame = createEmptyAttestationGame();
        vm.prank(badGame);
        vm.expectRevert("L2OutputOracle: Unauthorized output deletion.");
        oracle.deleteL2Outputs(highestL2BlockNumber);
    }

    function test_deleteL2Outputs_gameIncomplete_reverts() external {

    }

    function test_deleteL2Outputs_finalized_reverts() external {
        // TODO:

        vm.warp(block.timestamp + oracle.FINALIZATION_PERIOD_SECONDS() + 1);
        uint256 highestL2BlockNumber = oracle.startingBlockNumber() + 1;


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
    receive() external payable { }
    fallback() external payable { }

    /// @dev Resolve the game with a split
    function splitResolve() public {
        challengers = [address(1), address(2)];
        bm.seizeAndSplit(bondId, challengers);
    }

    /// -------------------------------------------
    /// IInitializable Functions
    /// -------------------------------------------

    function initialize() external { /* noop */ }

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
