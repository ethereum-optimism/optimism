// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { DisputeGameFactory_TestInit } from "test/dispute/DisputeGameFactory.t.sol";
import { ForgeArtifacts } from "scripts/libraries/ForgeArtifacts.sol";

// Contracts
import { ResourceMetering } from "src/L1/ResourceMetering.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "src/libraries/Types.sol";
import { Features } from "src/libraries/Features.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import "src/dispute/lib/Types.sol";

// Interfaces
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IResourceMetering } from "interfaces/L1/IResourceMetering.sol";
import { IFaultDisputeGame } from "interfaces/dispute/IFaultDisputeGame.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";

contract OptimismPortal2_Depositor is StdUtils, ResourceMetering {
    Vm internal vm;
    IOptimismPortal2 internal portal;
    bool public failedToComplete;

    constructor(Vm _vm, IOptimismPortal2 _portal) {
        vm = _vm;
        portal = _portal;
        initialize();
    }

    function initialize() internal initializer {
        __ResourceMetering_init();
    }

    function resourceConfig() public pure returns (ResourceMetering.ResourceConfig memory) {
        return _resourceConfig();
    }

    function _resourceConfig() internal pure override returns (ResourceMetering.ResourceConfig memory config_) {
        IResourceMetering.ResourceConfig memory rcfg = Constants.DEFAULT_RESOURCE_CONFIG();
        assembly ("memory-safe") {
            config_ := rcfg
        }
    }

    // A test intended to identify any unexpected halting conditions
    function depositTransactionCompletes(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        public
        payable
    {
        vm.assume((!_isCreation || _to == address(0)) && _data.length <= 120_000);

        uint256 preDepositvalue = bound(_value, 0, type(uint128).max);
        // Give the depositor some ether
        vm.deal(address(this), preDepositvalue);
        // cache the contract's eth balance
        uint256 preDepositBalance = address(this).balance;
        uint256 value = bound(preDepositvalue, 0, preDepositBalance);

        // If custom gas token is enabled, set deposit value to 0
        if (portal.systemConfig().isFeatureEnabled(Features.CUSTOM_GAS_TOKEN)) {
            value = 0;
        }

        (, uint64 cachedPrevBoughtGas,) = ResourceMetering(address(portal)).params();
        ResourceMetering.ResourceConfig memory rcfg = resourceConfig();
        uint256 maxResourceLimit = uint64(rcfg.maxResourceLimit);
        uint64 gasLimit = uint64(
            bound(_gasLimit, portal.minimumGasLimit(uint64(_data.length)), maxResourceLimit - cachedPrevBoughtGas)
        );

        try portal.depositTransaction{ value: value }(_to, value, gasLimit, _isCreation, _data) {
        // Do nothing; Call succeeded
        }
        catch {
            failedToComplete = true;
        }
    }
}

contract OptimismPortal2_Invariant_Harness is DisputeGameFactory_TestInit {
    // Reusable default values for a test withdrawal
    Types.WithdrawalTransaction _defaultTx;

    uint256 _proposedGameIndex;
    uint256 _proposedBlockNumber;
    bytes32 _stateRoot;
    bytes32 _storageRoot;
    bytes32 _outputRoot;
    bytes32 _withdrawalHash;
    bytes[] _withdrawalProof;
    Types.OutputRootProof internal _outputRootProof;

    function setUp() public virtual override {
        super.setUp();

        _defaultTx = Types.WithdrawalTransaction({
            nonce: 0, sender: alice, target: bob, value: 100, gasLimit: 100_000, data: hex""
        });

        // If custom gas token is enabled, set deposit value to 0
        if (systemConfig.isFeatureEnabled(Features.CUSTOM_GAS_TOKEN)) {
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

        // Warp forward in time to ensure that the game is created after the retirement timestamp.
        vm.warp(anchorStateRegistry.retirementTimestamp() + 1);

        setupFaultDisputeGame(Claim.wrap(bytes32(0)));

        // Create a dispute game with the output root we've proposed.
        _proposedBlockNumber = 0xFF;
        IFaultDisputeGame game = IFaultDisputeGame(
            payable(address(
                    disputeGameFactory.create{
                        value: disputeGameFactory.initBonds(optimismPortal2.respectedGameType())
                    }(
                        optimismPortal2.respectedGameType(), Claim.wrap(_outputRoot), abi.encode(_proposedBlockNumber)
                    )
                ))
        );
        _proposedGameIndex = disputeGameFactory.gameCount() - 1;

        // Warp beyond the finalization period for the dispute game and resolve it.
        vm.warp(block.timestamp + (game.maxClockDuration().raw() * 2) + 1 seconds);
        game.resolveClaim(0, 0);
        game.resolve();

        // Fund the portal so that we can withdraw ETH.
        vm.deal(address(ethLockbox), 0xFFFFFFFF);
        vm.deal(address(optimismPortal2), 0xFFFFFFFF);
    }

    function _targetOptimismPortal2() internal {
        targetContract(address(optimismPortal2));

        string[] memory artifacts = new string[](1);
        artifacts[0] = "OptimismPortal2";
        targetInterface(FuzzInterface({ addr: address(optimismPortal2), artifacts: artifacts }));

        excludeSender(EIP1967Helper.getAdmin(address(optimismPortal2)));
        // The ProxyAdmin owner could move the proof maturity delay and shift the finalization
        // window under the invariants.
        excludeSender(proxyAdminOwner);
        // The lockbox cannot originate arbitrary ETH transfers.
        excludeSender(address(ethLockbox));
    }
}

contract OptimismPortal2_Deposit_Invariant is CommonTest {
    OptimismPortal2_Depositor internal actor;

    function setUp() public override {
        super.setUp();
        // Create a deposit actor.
        actor = new OptimismPortal2_Depositor(vm, optimismPortal2);

        targetContract(address(actor));

        bytes4[] memory selectors = new bytes4[](1);
        selectors[0] = actor.depositTransactionCompletes.selector;
        FuzzSelector memory selector = FuzzSelector({ addr: address(actor), selectors: selectors });
        targetSelector(selector);
    }

    /// @custom:invariant Deposits of any value should always succeed unless
    ///                   `_to` = `address(0)` or `_isCreation` = `true`.
    ///
    ///                   All deposits, barring creation transactions and transactions
    ///                   sent to `address(0)`, should always succeed.
    function invariant_deposit_completes() external view {
        assertEq(actor.failedToComplete(), false);
    }
}

contract OptimismPortal2_CannotTimeTravel is OptimismPortal2_Invariant_Harness {
    function setUp() public override {
        super.setUp();

        // Prove the withdrawal transaction
        optimismPortal2.proveWithdrawalTransaction(_defaultTx, _proposedGameIndex, _outputRootProof, _withdrawalProof);

        _targetOptimismPortal2();
    }

    /// @custom:invariant `finalizeWithdrawalTransaction` should revert if the proof maturity period has not elapsed.
    ///
    ///                   A withdrawal that has been proven should not be able to be finalized
    ///                   until after the proof maturity period has elapsed.
    function invariant_cannotFinalizeBeforePeriodHasPassed() external {
        vm.expectRevert(IOptimismPortal2.OptimismPortal_ProofNotOldEnough.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);
    }
}

contract OptimismPortal2_CannotFinalizeTwice is OptimismPortal2_Invariant_Harness {
    function setUp() public override {
        super.setUp();

        // Prove the withdrawal transaction
        optimismPortal2.proveWithdrawalTransaction(_defaultTx, _proposedGameIndex, _outputRootProof, _withdrawalProof);

        // Warp past the proof maturity period.
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        // Finalize the withdrawal transaction.
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);

        _targetOptimismPortal2();
    }

    /// @custom:invariant `finalizeWithdrawalTransaction` should revert if the withdrawal has already been finalized.
    ///
    ///                   Ensures that there is no chain of calls that can be made that allows a withdrawal to be
    ///                   finalized twice.
    function invariant_cannotFinalizeTwice() external {
        vm.expectRevert(IOptimismPortal2.OptimismPortal_AlreadyFinalized.selector);
        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);
    }
}

contract OptimismPortal_CanAlwaysFinalizeAfterWindow is OptimismPortal2_Invariant_Harness {
    function setUp() public override {
        super.setUp();

        // Prove the withdrawal transaction
        optimismPortal2.proveWithdrawalTransaction(_defaultTx, _proposedGameIndex, _outputRootProof, _withdrawalProof);

        // Warp past the proof maturity period.
        vm.warp(block.timestamp + optimismPortal2.proofMaturityDelaySeconds() + 1);

        _targetOptimismPortal2();
    }

    /// @custom:invariant A withdrawal should **always** be able to be finalized `PROOF_MATURITY_DELAY_SECONDS` after
    ///                   it was successfully proven, if the game has resolved and passed the air-gap.
    ///
    ///                   This invariant asserts that there is no chain of calls that can be made that will prevent a
    ///                   withdrawal from being finalized exactly `PROOF_MATURITY_DELAY_SECONDS` after it was
    ///                   successfully proven and the game has resolved and passed the air-gap.
    function invariant_canAlwaysFinalize() external {
        uint256 bobBalanceBefore = address(bob).balance;

        optimismPortal2.finalizeWithdrawalTransaction(_defaultTx);

        assertEq(address(bob).balance, bobBalanceBefore + _defaultTx.value);
    }
}

/// @notice Bridge hook that also deposits. Tracks the gas the Portal publishes in each block.
contract OptimismPortal2_HookDepositor is StdUtils {
    Vm internal vm;
    IOptimismPortal2 internal portal;
    uint256 internal initializedSlot;

    /// @notice Verdict that the next screen returns.
    bool internal pass;

    /// @notice Deposits this hook holds.
    BridgeHookTypes.Item[] internal held;

    /// @notice Gas limits of the TransactionDeposited events emitted in each block.
    mapping(uint256 => uint256) public published;

    constructor(Vm _vm, IOptimismPortal2 _portal, uint256 _initializedSlot) {
        vm = _vm;
        portal = _portal;
        initializedSlot = _initializedSlot;
    }

    /// @notice Adds the gas of each TransactionDeposited event emitted by the call to this block.
    modifier recordPublished() {
        vm.recordLogs();
        _;
        Vm.Log[] memory logs = vm.getRecordedLogs();
        for (uint256 i; i < logs.length; i++) {
            if (
                logs[i].emitter == address(portal)
                    && logs[i].topics[0] == keccak256("TransactionDeposited(address,address,uint256,bytes)")
            ) {
                // opaqueData packs mint (32 bytes), value (32 bytes), then the gas limit (8 bytes).
                bytes memory opaqueData = abi.decode(logs[i].data, (bytes));
                published[block.number] += uint64(bytes8(Bytes.slice(opaqueData, 64, 8)));
            }
        }
    }

    function screenDeposit(BridgeHookTypes.Item calldata) external view returns (bool) {
        return pass;
    }

    function holdDeposit(BridgeHookTypes.Item calldata _item) external payable {
        held.push(_item);
    }

    function deposit(uint64 _gasLimit, bool _hold) external recordPublished {
        pass = !_hold;
        portal.depositTransaction(address(1), 0, _boundGasLimit(_gasLimit), false, hex"");
    }

    function depositAsL1PAO(uint64 _gasLimit) external recordPublished {
        uint64 gasLimit = _boundGasLimit(_gasLimit);
        vm.prank(portal.proxyAdminOwner());
        portal.depositTransaction(address(1), 0, gasLimit, false, hex"");
    }

    function complete(uint256 _index) external recordPublished {
        if (held.length == 0) return;
        _index = bound(_index, 0, held.length - 1);
        portal.completeDepositTransaction(held[_index]);
        held[_index] = held[held.length - 1];
        held.pop();
    }

    function setHook() external {
        // initialize only runs on a cleared initialized slot, as in an upgrade.
        vm.store(address(portal), bytes32(initializedSlot), bytes32(0));
        // vm.prank would apply to the first getter call in the arguments, not to initialize.
        vm.startPrank(portal.proxyAdminOwner());
        portal.initialize(
            portal.systemConfig(),
            portal.anchorStateRegistry(),
            IETHLockbox(address(0)),
            portal.proofMaturityDelaySeconds(),
            IBridgeHook(address(this))
        );
        vm.stopPrank();
    }

    function removeHook() external {
        vm.prank(portal.proxyAdminOwner());
        portal.removeHook();
    }

    function roll() external {
        vm.roll(block.number + 1);
    }

    function _boundGasLimit(uint64 _gasLimit) internal view returns (uint64) {
        return
            uint64(bound(_gasLimit, portal.minimumGasLimit(0), portal.systemConfig().resourceConfig().maxResourceLimit));
    }
}

contract OptimismPortal2_PublishedGas_Invariant is CommonTest {
    OptimismPortal2_HookDepositor internal actor;

    function setUp() public override {
        super.setUp();
        actor = new OptimismPortal2_HookDepositor(
            vm, optimismPortal2, ForgeArtifacts.getSlot("OptimismPortal2", "_initialized").slot
        );

        targetContract(address(actor));

        bytes4[] memory selectors = new bytes4[](6);
        selectors[0] = actor.deposit.selector;
        selectors[1] = actor.depositAsL1PAO.selector;
        selectors[2] = actor.complete.selector;
        selectors[3] = actor.setHook.selector;
        selectors[4] = actor.removeHook.selector;
        selectors[5] = actor.roll.selector;
        targetSelector(FuzzSelector({ addr: address(actor), selectors: selectors }));
    }

    /// @custom:invariant A block never publishes more deposit gas than `maxResourceLimit`.
    ///
    ///                   Calls mix deposits, holds, completions, L1PAO deposits and hook changes.
    ///                   Checks only the current block. Its total grows only while it is current,
    ///                   and this check runs after every call.
    function invariant_publishedGas_neverAboveLimit() external view {
        assertLe(actor.published(block.number), systemConfig.resourceConfig().maxResourceLimit);
    }
}
