// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { GameType } from "src/dispute/lib/Types.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

// Libraries
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { GameType } from "src/dispute/lib/Types.sol";
import { LibString } from "@solady/utils/LibString.sol";

// Contracts
import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { DeployDisputeGame } from "scripts/deploy/DeployDisputeGame.s.sol";

contract DeployDisputeGame_Test is Test {
    DeployDisputeGame deployDisputeGame;

    PreimageOracle preimageOracle;

    DeployDisputeGameBigStepper bigStepper;

    // Define default input variables for testing.
    IDelayedWETH defaultDelayedWethProxy = IDelayedWETH(payable(makeAddr("IDelayedWETH")));
    IAnchorStateRegistry defaultAnchorStateRegistryProxy = IAnchorStateRegistry(makeAddr("IAnchorStateRegistry"));
    address defaultProposer = makeAddr("Proposer");
    address defaultChallenger = makeAddr("Challenger");

    function setUp() public {
        deployDisputeGame = new DeployDisputeGame();
        preimageOracle = new PreimageOracle(0, 0);
        bigStepper = new DeployDisputeGameBigStepper(preimageOracle);

        vm.label(address(deployDisputeGame), "DeployDisputeGame");
        vm.label(address(preimageOracle), "PreimageOracle");
        vm.label(address(bigStepper), "BigStepper");
    }

    function testFuzz_run_withFaultDisputeGame_succeeds(
        DeployDisputeGame.Input memory _input,
        uint32 _gameType,
        uint64 _clockExtension,
        uint64 _maxClockDuration,
        uint8 _splitDepth,
        uint8 _maxGameDepth
    )
        public
    {
        vm.assume(_input.absolutePrestate != bytes32(0));
        vm.assume(_input.l2ChainId != 0);
        vm.assume(_gameType != 0);
        vm.assume(_clockExtension != 0);
        vm.assume(!LibString.eq(_input.release, ""));
        vm.assume(address(_input.anchorStateRegistryProxy) != address(0));
        vm.assume(address(_input.delayedWethProxy) != address(0));
        vm.assume(_input.challenger != address(0));
        vm.assume(_input.proposer != address(0));

        // These come from the constructor or FaultDisputeGame
        vm.assume(_gameType != type(uint32).max);
        vm.assume(_maxClockDuration >= 2);
        vm.assume(_maxGameDepth >= 4);
        vm.assume(_maxGameDepth <= LibPosition.MAX_POSITION_BITLEN - 1);

        _input.gameKind = "FaultDisputeGame";
        _input.gameType = GameType.wrap(_gameType);
        _input.clockExtension = uint64(bound(_clockExtension, 1, _maxClockDuration / 2));
        _input.maxClockDuration = _maxClockDuration;
        _input.maxGameDepth = _maxGameDepth;
        _input.splitDepth = bound(_splitDepth, 2, _maxGameDepth - 2);
        _input.vmAddress = bigStepper;

        // Run the deployment script.
        deployDisputeGame.run(_input);
    }

    function testFuzz_run_withPermissionedDisputeGame_succeeds(
        DeployDisputeGame.Input memory _input,
        uint32 _gameType,
        uint64 _clockExtension,
        uint64 _maxClockDuration,
        uint8 _splitDepth,
        uint8 _maxGameDepth
    )
        public
    {
        vm.assume(_input.absolutePrestate != bytes32(0));
        vm.assume(_input.l2ChainId != 0);
        vm.assume(_gameType != 0);
        vm.assume(_clockExtension != 0);
        vm.assume(!LibString.eq(_input.release, ""));
        vm.assume(address(_input.anchorStateRegistryProxy) != address(0));
        vm.assume(address(_input.delayedWethProxy) != address(0));
        vm.assume(_input.challenger != address(0));
        vm.assume(_input.proposer != address(0));

        // These come from the constructor or FaultDisputeGame
        vm.assume(_gameType != type(uint32).max);
        vm.assume(_maxClockDuration >= 2);
        vm.assume(_maxGameDepth >= 4);
        vm.assume(_maxGameDepth <= LibPosition.MAX_POSITION_BITLEN - 1);

        _input.gameKind = "PermissionedDisputeGame";
        _input.gameType = GameType.wrap(_gameType);
        _input.clockExtension = uint64(bound(_clockExtension, 1, _maxClockDuration / 2));
        _input.maxClockDuration = _maxClockDuration;
        _input.maxGameDepth = _maxGameDepth;
        _input.splitDepth = bound(_splitDepth, 2, _maxGameDepth - 2);
        _input.vmAddress = bigStepper;

        // Run the deployment script.
        deployDisputeGame.run(_input);
    }

    function test_run_nullInputsWithFaultDisputeGame_reverts() public {
        DeployDisputeGame.Input memory input;

        // Test case: release not set
        input = defaultFaultDisputeGameInput();
        input.release = "";
        vm.expectRevert("DeployDisputeGame: release not set");
        deployDisputeGame.run(input);

        // Test case: l2ChainId not set
        input = defaultFaultDisputeGameInput();
        input.l2ChainId = 0;
        vm.expectRevert("DeployDisputeGame: l2ChainId not set");
        deployDisputeGame.run(input);

        // Test case: maxGameDepth not set
        input = defaultFaultDisputeGameInput();
        input.maxGameDepth = 0;
        vm.expectRevert("DeployDisputeGame: maxGameDepth not set");
        deployDisputeGame.run(input);

        // Test case: delayedWethProxy not set
        input = defaultFaultDisputeGameInput();
        input.delayedWethProxy = IDelayedWETH(payable(address(0)));
        vm.expectRevert("DeployDisputeGame: delayedWethProxy not set");
        deployDisputeGame.run(input);

        // Test case: anchorStateRegistryProxy not set
        input = defaultFaultDisputeGameInput();
        input.anchorStateRegistryProxy = IAnchorStateRegistry(payable(address(0)));
        vm.expectRevert("DeployDisputeGame: anchorStateRegistryProxy not set");
        deployDisputeGame.run(input);
    }

    function test_run_nullInputsWithPermissionedDisputeGame_reverts() public {
        DeployDisputeGame.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.proposer = address(0);
        vm.expectRevert("DeployDisputeGame: proposer not set");
        deployDisputeGame.run(input);

        input = defaultPermissionedDisputeGameInput();
        input.challenger = address(0);
        vm.expectRevert("DeployDisputeGame: challenger not set");
        deployDisputeGame.run(input);
    }

    function test_run_withUnknownGameKind_reverts(string memory _gameKind) public {
        vm.assume(!LibString.eq(_gameKind, "PermissionedDisputeGame"));
        vm.assume(!LibString.eq(_gameKind, "FaultDisputeGame"));

        DeployDisputeGame.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.gameKind = _gameKind;
        vm.expectRevert("DeployDisputeGame: unknown game kind");
        deployDisputeGame.run(input);
    }

    function defaultFaultDisputeGameInput() private view returns (DeployDisputeGame.Input memory input_) {
        input_ = DeployDisputeGame.Input({
            release: "op-contracts",
            gameKind: "FaultDisputeGame",
            gameType: GameType.wrap(1),
            absolutePrestate: bytes32(uint256(1)),
            maxGameDepth: 10,
            splitDepth: 2,
            clockExtension: 1,
            maxClockDuration: 1000,
            l2ChainId: 1,
            delayedWethProxy: defaultDelayedWethProxy,
            anchorStateRegistryProxy: defaultAnchorStateRegistryProxy,
            vmAddress: bigStepper,
            proposer: defaultProposer,
            challenger: defaultChallenger
        });
    }

    function defaultPermissionedDisputeGameInput() private view returns (DeployDisputeGame.Input memory input_) {
        input_ = DeployDisputeGame.Input({
            release: "op-contracts",
            gameKind: "PermissionedDisputeGame",
            gameType: GameType.wrap(1),
            absolutePrestate: bytes32(uint256(1)),
            maxGameDepth: 10,
            splitDepth: 2,
            clockExtension: 1,
            maxClockDuration: 1000,
            l2ChainId: 1,
            delayedWethProxy: defaultDelayedWethProxy,
            anchorStateRegistryProxy: defaultAnchorStateRegistryProxy,
            vmAddress: bigStepper,
            proposer: defaultProposer,
            challenger: defaultChallenger
        });
    }
}

contract DeployDisputeGameBigStepper is IBigStepper {
    PreimageOracle private immutable mockOracle;

    constructor(PreimageOracle _oracle) {
        mockOracle = _oracle;
    }

    function step(bytes calldata, bytes calldata, bytes32) external pure returns (bytes32) {
        return bytes32(0);
    }

    function oracle() external view override returns (IPreimageOracle) {
        return IPreimageOracle(address(mockOracle));
    }
}
