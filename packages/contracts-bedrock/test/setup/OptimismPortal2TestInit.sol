// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { DisputeGameFactory_TestInit } from "test/setup/DisputeGameFactoryTestInit.sol";
import { DisputeGames } from "test/setup/DisputeGames.sol";

// Scripts
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Features } from "src/libraries/Features.sol";
import { Claim, GameType } from "src/dispute/lib/Types.sol";

// Interfaces
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";

abstract contract OptimismPortal2_TestInit is DisputeGameFactory_TestInit {
    address depositor;

    Types.WithdrawalTransaction _defaultTx;
    IFaultDisputeGame game;
    uint256 _proposedGameIndex;
    uint256 _proposedBlockNumber;
    bytes32 _stateRoot;
    bytes32 _storageRoot;
    bytes32 _outputRoot;
    bytes32 _withdrawalHash;
    bytes[] _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;
    GameType internal respectedGameType;

    /// @notice The configured proof maturity delay. Read once in setUp so tests can pass it back to
    ///         initialize() without an external call consuming a pending prank.
    uint256 internal portalProofMaturityDelay;

    event ProofMaturityDelaySecondsSet(uint256 proofMaturityDelaySeconds);

    // Use a constructor to set the storage vars above, so as to minimize the number of ffi calls.
    constructor() {
        super.setUp();

        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0,
            sender: alice,
            target: bob,
            value: 100,
            gasLimit: 100_000,
            data: hex"aa" // includes calldata for ERC20 withdrawal test
        });

        if (isUsingCustomGasToken()) {
            _defaultTx.value = 0;
        }

        // Get withdrawal proof data we can use for testing.
        (_stateRoot, _storageRoot, _outputRoot, _withdrawalHash, _withdrawalProof) =
            ffi.getProveWithdrawalTransactionInputs(_defaultTx);

        // Setup a dummy output root proof for reuse.
        _outputRootProof = Types.OutputRootProof({
            version: bytes32(uint256(0)),
            stateRoot: _stateRoot,
            messagePasserStorageRoot: _storageRoot,
            latestBlockhash: bytes32(uint256(0))
        });
    }

    /// @dev Setup the system for a ready-to-use state.
    function setUp() public virtual override {
        if (isL1ForkTest()) {
            // Set the proposed block number to be the next block number on the forked network
            (, _proposedBlockNumber) = anchorStateRegistry.getAnchorRoot();
            _proposedBlockNumber += 1;

            // Set the init bond of anchor game type 0 to be 0.
            // It is a mapping so the storage slot is calculated as keccak256(abi.encode(key, slot)).
            // The storage slot for the initBond mapping is 102, see `snapshots/storageLayout/DisputeGameFactory.json`.
            vm.store(
                address(disputeGameFactory), keccak256(abi.encode(GameType.wrap(0), uint256(102))), bytes32(uint256(0))
            );
        } else {
            // Set up the dummy game.
            _proposedBlockNumber = 0xFF;
        }

        depositor = makeAddr("depositor");
        portalProofMaturityDelay = optimismPortal2.proofMaturityDelaySeconds();

        setupFaultDisputeGame(Claim.wrap(_outputRoot));

        // Warp forward in time to ensure that the game is created after the retirement timestamp.
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        respectedGameType = optimismPortal2.respectedGameType();
        game = IFaultDisputeGame(
            payable(address(_createDisputeGame(respectedGameType, _outputRoot, _proposedBlockNumber)))
        );

        // Grab the index of the game we just created.
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the chess clocks and finalize the game.
        vm.warp(block.timestamp + game.maxClockDuration().raw() + 1 seconds);

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(optimismPortal2), 0xFFFFFFFF);
        if (isUsingLockbox()) {
            vm.deal(address(ethLockbox), 0xFFFFFFFF);
        }
    }

    /// @notice Prepares a funded, proven withdrawal that is eligible for finalization.
    function _prepareWithdrawalForFinalization(Types.WithdrawalTransaction memory _tx) internal returns (bytes32) {
        vm.deal(address(optimismPortal2), _tx.value);
        if (isUsingLockbox()) {
            vm.deal(address(ethLockbox), _tx.value);
        }

        (
            bytes32 stateRoot,
            bytes32 storageRoot,
            bytes32 outputRoot,
            bytes32 withdrawalHash,
            bytes[] memory withdrawalProof
        ) = ffi.getProveWithdrawalTransactionInputs(_tx);
        Types.OutputRootProof memory proof = Types.OutputRootProof({
            version: bytes32(0),
            stateRoot: stateRoot,
            messagePasserStorageRoot: storageRoot,
            latestBlockhash: bytes32(0)
        });
        if (DisputeGames.isSuperGame(game.gameType())) {
            vm.mockCall(
                address(game),
                abi.encodeCall(game.rootClaimByChainId, (systemConfig.l2ChainId())),
                abi.encode(outputRoot)
            );
        } else {
            vm.mockCall(address(game), abi.encodeCall(game.rootClaim, ()), abi.encode(outputRoot));
        }

        optimismPortal2.proveWithdrawalTransaction(_tx, _proposedGameIndex, proof, withdrawalProof);
        game.resolveClaim(0, 0);
        game.resolve();
        vm.warp(
            block.timestamp + optimismPortal2.proofMaturityDelaySeconds()
                + optimismPortal2.disputeGameFinalityDelaySeconds() + 1
        );
        optimismPortal2.checkWithdrawal(withdrawalHash, address(this));
        return withdrawalHash;
    }

    /// @notice Creates a dispute game with proper extraData for both regular and super game types.
    function _createDisputeGame(
        GameType _gameType,
        bytes32 _outputRoot_,
        uint256 _blockNumber
    )
        internal
        returns (IDisputeGame)
    {
        Claim claim;
        bytes memory extra;
        if (DisputeGames.isSuperGame(_gameType)) {
            Types.OutputRootWithChainId[] memory roots = new Types.OutputRootWithChainId[](1);
            roots[0] = Types.OutputRootWithChainId({ chainId: systemConfig.l2ChainId(), root: _outputRoot_ });
            Types.SuperRootProof memory proof = Types.SuperRootProof({
                version: bytes1(uint8(1)), timestamp: uint64(_blockNumber), outputRoots: roots
            });
            claim = Claim.wrap(Hashing.hashSuperRootProof(proof));
            extra = Encoding.encodeSuperRootProof(proof);
        } else {
            claim = Claim.wrap(_outputRoot_);
            extra = abi.encode(_blockNumber);
        }
        return disputeGameFactory.create{ value: disputeGameFactory.initBonds(_gameType) }(_gameType, claim, extra);
    }

    /// @notice Asserts that the reentrant call will revert.
    function callPortalAndExpectRevert() external payable {
        vm.expectRevert(IOptimismPortal.OptimismPortal_NoReentrancy.selector);

        // Arguments here don't matter, as the require check is the first thing that happens.
        // We assume that this has already been proven.
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);

        // Assert that the withdrawal was not finalized.
        assertFalse(optimismPortal2.finalizedWithdrawals(Hashing.hashWithdrawal(_defaultTx)));
    }

    /// @notice Checks if the ETHLockbox feature is enabled.
    /// @return bool True if the ETHLockbox feature is enabled.
    function isUsingLockbox() public view returns (bool) {
        return
            systemConfig.isFeatureEnabled(Features.ETH_LOCKBOX) && address(optimismPortal2.ethLockbox()) != address(0);
    }

    /// @notice Checks if the Custom Gas Token feature is enabled.
    /// @return bool True if the Custom Gas Token feature is enabled.
    function isUsingCustomGasToken() public view returns (bool) {
        return systemConfig.isFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
    }

    /// @notice Enables the ETHLockbox feature if not enabled.
    /// @param _lockbox Address of the lockbox to enable.
    function forceEnableLockbox(address _lockbox) public {
        if (!isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            vm.prank(address(proxyAdmin));
            systemConfig.setFeature(Features.ETH_LOCKBOX, true);
        }

        // Overwrite the lockbox either way.
        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "ethLockbox");
        vm.store(address(optimismPortal2), bytes32(slot.slot), bytes32(uint256(uint160(address(_lockbox)))));

        // If the recipient address has no code, store STOP so we don't get reverts.
        if (address(_lockbox).code.length == 0) {
            vm.etch(address(_lockbox), hex"00");
        }
    }

    /// @notice Sets the bridge hook.
    /// @param _hook Address of the bridge hook to set.
    function forceSetBridgeHook(address _hook) public {
        StorageSlot memory slot = ForgeArtifacts.getSlot("OptimismPortal2", "bridgeHook");
        vm.store(address(optimismPortal2), bytes32(slot.slot), bytes32(uint256(uint160(_hook))));
    }

    /// @notice Enables the runtime interop features used by portal migration methods.
    function forceEnableInterop() public {
        forceEnableLockbox(address(ethLockbox));

        if (!isSysFeatureEnabled(Features.INTEROP)) {
            vm.prank(address(proxyAdmin));
            systemConfig.setFeature(Features.INTEROP, true);
        }
    }

    /// @notice Sets the useCustomGasToken variable
    function setUseCustomGasToken(bool _useCustomGasToken) public {
        vm.prank(address(proxyAdmin));
        systemConfig.setFeature(Features.CUSTOM_GAS_TOKEN, _useCustomGasToken);
    }
}
