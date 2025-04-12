// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";

// Interfaces
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IBigStepper } from "interfaces/dispute/IBigStepper.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";
import { IPreimageOracle } from "interfaces/cannon/IPreimageOracle.sol";

// Libraries
import { LibPosition } from "src/dispute/lib/LibPosition.sol";
import { LibString } from "@solady/utils/LibString.sol";

import { PreimageOracle } from "src/cannon/PreimageOracle.sol";
import { DeployDisputeGame2 } from "scripts/deploy/DeployDisputeGame2.s.sol";

contract DeployDisputeGame2_Test is Test {
    DeployDisputeGame2 deployDisputeGame;

    PreimageOracle preimageOracle;

    DeployDisputeGameBigStepper bigStepper;

    // Define default input variables for testing.
    IDelayedWETH defaultDelayedWethProxy = IDelayedWETH(payable(makeAddr("IDelayedWETH")));
    IAnchorStateRegistry defaultAnchorStateRegistryProxy = IAnchorStateRegistry(makeAddr("IAnchorStateRegistry"));
    address defaultProposer = makeAddr("Proposer");
    address defaultChallenger = makeAddr("Challenger");

    function setUp() public {
        deployDisputeGame = new DeployDisputeGame2();
        preimageOracle = new PreimageOracle(0, 0);
        bigStepper = new DeployDisputeGameBigStepper(preimageOracle);

        vm.label(address(deployDisputeGame), "DeployDisputeGame");
        vm.label(address(preimageOracle), "PreimageOracle");
        vm.label(address(bigStepper), "BigStepper");
    }

    function testFuzz_run_withFaultDisputeGame_succeeds(
        DeployDisputeGame2.Input memory _input,
        uint32 _gameType,
        uint32 _clockExtension,
        uint64 _maxClockDuration,
        uint8 _splitDepth,
        uint8 _maxGameDepth
    )
        public
    {
        vm.assume(_input.l2ChainId != 0);
        vm.assume(_gameType != 0);
        vm.assume(_clockExtension != 0);
        vm.assume(!LibString.eq(_input.release, ""));
        vm.assume(!LibString.eq(_input.standardVersionsToml, ""));
        vm.assume(address(_input.anchorStateRegistryProxy) != address(0));
        vm.assume(address(_input.delayedWethProxy) != address(0));

        // These come from the constructor or FaultDisputeGame
        vm.assume(_gameType != type(uint32).max);
        vm.assume(_maxClockDuration >= 2);
        vm.assume(_maxGameDepth >= 4);
        vm.assume(_maxGameDepth <= LibPosition.MAX_POSITION_BITLEN - 1);

        _input.gameKind = "FaultDisputeGame";
        _input.gameType = _gameType;
        _input.clockExtension = bound(_clockExtension, 1, _maxClockDuration / 2);
        _input.maxClockDuration = _maxClockDuration;
        _input.maxGameDepth = _maxGameDepth;
        _input.splitDepth = bound(_splitDepth, 2, _maxGameDepth - 2);
        _input.vm = bigStepper;

        // For FaultDisputeGame, these must be empty
        _input.challenger = address(0);
        _input.proposer = address(0);

        // Run the deployment script.
        deployDisputeGame.run(_input);
    }

    function testFuzz_run_withPermissionedDisputeGame_succeeds(
        DeployDisputeGame2.Input memory _input,
        uint32 _gameType,
        uint32 _clockExtension,
        uint64 _maxClockDuration,
        uint8 _splitDepth,
        uint8 _maxGameDepth
    )
        public
    {
        vm.assume(_input.l2ChainId != 0);
        vm.assume(_gameType != 0);
        vm.assume(_clockExtension != 0);
        vm.assume(!LibString.eq(_input.release, ""));
        vm.assume(!LibString.eq(_input.standardVersionsToml, ""));
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
        _input.gameType = _gameType;
        _input.clockExtension = bound(_clockExtension, 1, _maxClockDuration / 2);
        _input.maxClockDuration = _maxClockDuration;
        _input.maxGameDepth = _maxGameDepth;
        _input.splitDepth = bound(_splitDepth, 2, _maxGameDepth - 2);
        _input.vm = bigStepper;

        // Run the deployment script.
        deployDisputeGame.run(_input);
    }

    function test_run_nullInputsWithFaultDisputeGame_reverts() public {
        DeployDisputeGame2.Input memory input;

        input = defaultFaultDisputeGameInput();
        input.release = "";
        vm.expectRevert("DeployDisputeGame: release not set");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.standardVersionsToml = "";
        vm.expectRevert("DeployDisputeGame: standardVersionsToml not set");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.l2ChainId = 0;
        vm.expectRevert("DeployDisputeGame: l2ChainId not set");
        deployDisputeGame.run(input);

        // Test case: clockExtension not set
        input = defaultFaultDisputeGameInput();
        input.clockExtension = 0;
        vm.expectRevert("DeployDisputeGame: clockExtension not set");
        deployDisputeGame.run(input);

        // Test case: maxClockDuration not set
        input = defaultFaultDisputeGameInput();
        input.maxClockDuration = 0;
        vm.expectRevert("DeployDisputeGame: maxClockDuration not set");
        deployDisputeGame.run(input);

        // Test case: maxGameDepth not set
        input = defaultFaultDisputeGameInput();
        input.maxGameDepth = 0;
        vm.expectRevert("DeployDisputeGame: maxGameDepth not set");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.delayedWethProxy = IDelayedWETH(payable(address(0)));
        vm.expectRevert("DeployDisputeGame: delayedWethProxy not set");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.anchorStateRegistryProxy = IAnchorStateRegistry(payable(address(0)));
        vm.expectRevert("DeployDisputeGame: anchorStateRegistryProxy not set");
        deployDisputeGame.run(input);
    }

    function test_run_proposerWithFaultDisputeGame_reverts(address _proposerOrChallenger) public {
        vm.assume(_proposerOrChallenger != address(0));

        DeployDisputeGame2.Input memory input;

        input = defaultFaultDisputeGameInput();
        input.proposer = _proposerOrChallenger;
        vm.expectRevert("DeployDisputeGame: proposer must be empty");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.challenger = _proposerOrChallenger;
        vm.expectRevert("DeployDisputeGame: challenger must be empty");
        deployDisputeGame.run(input);
    }

    function test_run_nullInputsWithPermissionedDisputeGame_reverts() public {
        DeployDisputeGame2.Input memory input;

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

        DeployDisputeGame2.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.gameKind = _gameKind;
        vm.expectRevert("DeployDisputeGame: unknown game kind");
        deployDisputeGame.run(input);
    }

    function test_run_withClockExtensionTooLarge_reverts(uint256 _clockExtension) public {
        vm.assume(_clockExtension > type(uint64).max);

        DeployDisputeGame2.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.clockExtension = _clockExtension;
        vm.expectRevert("DeployDisputeGame: clockExtension must fit inside uint64");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.clockExtension = _clockExtension;
        vm.expectRevert("DeployDisputeGame: clockExtension must fit inside uint64");
        deployDisputeGame.run(input);
    }

    function test_run_withGameTypeTooLarge_reverts(uint256 _gameType) public {
        vm.assume(_gameType > type(uint32).max);

        DeployDisputeGame2.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.gameType = _gameType;
        vm.expectRevert("DeployDisputeGame: gameType must fit inside uint32");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.gameType = _gameType;
        vm.expectRevert("DeployDisputeGame: gameType must fit inside uint32");
        deployDisputeGame.run(input);
    }

    function test_run_withMaxClockDurationTooLarge_reverts(uint256 _maxClockDuration) public {
        vm.assume(_maxClockDuration > type(uint64).max);

        DeployDisputeGame2.Input memory input;

        input = defaultPermissionedDisputeGameInput();
        input.maxClockDuration = _maxClockDuration;
        vm.expectRevert("DeployDisputeGame: maxClockDuration must fit inside uint64");
        deployDisputeGame.run(input);

        input = defaultFaultDisputeGameInput();
        input.maxClockDuration = _maxClockDuration;
        vm.expectRevert("DeployDisputeGame: maxClockDuration must fit inside uint64");
        deployDisputeGame.run(input);
    }

    function defaultFaultDisputeGameInput() private view returns (DeployDisputeGame2.Input memory input_) {
        input_ = DeployDisputeGame2.Input({
            release: "op-contracts",
            standardVersionsToml: "op-versions.toml",
            gameKind: "FaultDisputeGame",
            gameType: 1,
            absolutePrestate: bytes32(uint256(1)),
            maxGameDepth: 10,
            splitDepth: 2,
            clockExtension: 1,
            maxClockDuration: 1000,
            l2ChainId: 1,
            delayedWethProxy: defaultDelayedWethProxy,
            anchorStateRegistryProxy: defaultAnchorStateRegistryProxy,
            vm: bigStepper,
            proposer: address(0),
            challenger: address(0)
        });
    }

    function defaultPermissionedDisputeGameInput() private view returns (DeployDisputeGame2.Input memory input_) {
        input_ = DeployDisputeGame2.Input({
            release: "op-contracts",
            standardVersionsToml: "op-versions.toml",
            gameKind: "PermissionedDisputeGame",
            gameType: 1,
            absolutePrestate: bytes32(uint256(1)),
            maxGameDepth: 10,
            splitDepth: 2,
            clockExtension: 1,
            maxClockDuration: 1000,
            l2ChainId: 1,
            delayedWethProxy: defaultDelayedWethProxy,
            anchorStateRegistryProxy: defaultAnchorStateRegistryProxy,
            vm: bigStepper,
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
