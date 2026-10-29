// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { ProxyAdminOwnedBase } from "src/universal/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";

// Libraries
import { EOA } from "src/libraries/EOA.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Features } from "src/libraries/Features.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookDecoder } from "src/libraries/BridgeHookDecoder.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IBridgeHook } from "interfaces/L1/IBridgeHook.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";

/// @custom:proxied true
/// @title MockOptimismPortal2
/// @notice Devnet-only OptimismPortal for the bridge-screener mock stack, selected by the
///         BRIDGE_HOOK_MOCKS dev feature. It reproduces the deposit and withdrawal event
///         choreography, including the bridge hook paths that the real Portal only declares as
///         empty stubs, and verifies nothing: no merkle proofs, no dispute games, no resource
///         metering.
/// @dev    NOT FOR PRODUCTION. Two divergences from the spec are deliberate and load-bearing.
///         First, setBridgeHook exists because OPCM passes a zero hook and ChainAssertions
///         requires the hook to still be zero when DeployOPChain finishes, so the hook can only
///         be wired afterwards. The real Portal must keep initialize as the only way to set it.
///         Second, deposits are not metered, so the deposit gas accounting the spec requires is
///         absent entirely.
contract MockOptimismPortal2 is Initializable, ReinitializableBase, ProxyAdminOwnedBase, ISemver {
    /// @notice Represents a proven withdrawal. The mock verifies no dispute game, but keeps the
    ///         field so the struct packs exactly as the real Portal's does. Dropping it would
    ///         shift timestamp to offset 0 and misread every withdrawal already proven against
    ///         the live proxy.
    /// @custom:field disputeGameProxy Unused. Present only to match the real layout.
    /// @custom:field timestamp        Timestamp at which the withdrawal was proven.
    struct ProvenWithdrawal {
        address disputeGameProxy;
        uint64 timestamp;
    }

    /// @notice Version of the deposit event.
    uint256 internal constant DEPOSIT_VERSION = 0;

    /// @notice The L2 gas limit set when ETH is deposited using the receive() function.
    uint64 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 100_000;

    /// @notice The lowest value in seconds that `proofMaturityDelaySeconds` may be initialized to.
    uint256 internal immutable MIN_PROOF_MATURITY_DELAY_SECONDS;

    /// @notice The highest value in seconds that `proofMaturityDelaySeconds` may be initialized
    ///         to.
    uint256 internal immutable MAX_PROOF_MATURITY_DELAY_SECONDS;

    // The mock runs behind the real Portal's proxy, so its storage has to line up with the
    // real Portal's exactly. Slots the mock never reads are declared as spacers rather than
    // omitted, which keeps the upgrade safe in both directions: the mock reads the live values,
    // and upgrading back to the real Portal finds its state where it left it. Slots and offsets
    // mirror snapshots/storageLayout/OptimismPortal2.json.

    /// @custom:spacer ResourceMetering params
    bytes32 private spacer_1_0_32;

    /// @custom:spacer ResourceMetering __gap
    uint256[48] private spacer_2_0_1536;

    /// @notice Address of the L2 account which initiated a withdrawal in this transaction.
    address public l2Sender;

    /// @notice Withdrawals that have been finalized, keyed by withdrawal hash.
    mapping(bytes32 => bool) public finalizedWithdrawals;

    /// @custom:spacer
    bytes32 private spacer_52_0_32;

    /// @custom:spacer
    bool private spacer_53_0_1;

    /// @custom:spacer
    address private spacer_53_1_20;

    /// @custom:spacer
    address private spacer_54_0_20;

    /// @notice Address of the SystemConfig contract.
    ISystemConfig public systemConfig;

    /// @custom:spacer
    address private spacer_56_0_20;

    /// @notice Proven withdrawals, keyed by withdrawal hash and proof submitter.
    mapping(bytes32 => mapping(address => ProvenWithdrawal)) public provenWithdrawals;

    /// @custom:spacer
    bytes32 private spacer_58_0_32;

    /// @custom:spacer
    uint32 private spacer_59_0_4;

    /// @custom:spacer
    uint64 private spacer_59_4_8;

    /// @notice Addresses that have submitted a proof for a withdrawal hash.
    mapping(bytes32 => address[]) public proofSubmitters;

    /// @custom:spacer
    uint256 private spacer_61_0_32;

    /// @notice Address of the AnchorStateRegistry contract.
    IAnchorStateRegistry public anchorStateRegistry;

    /// @notice Address of the ETHLockbox contract.
    IETHLockbox public ethLockbox;

    /// @custom:spacer
    bool private spacer_63_20_1;

    /// @notice The delay between when a withdrawal is proven and when it may be finalized.
    uint256 public proofMaturityDelaySeconds;

    /// @notice Bridge hook that screens deposits and withdrawals, or the zero address if unset.
    IBridgeHook public bridgeHook;

    /// @notice Deposits the hook is holding, keyed by item ID.
    mapping(bytes32 => bool) public heldDeposits;

    /// @notice Withdrawals the hook is holding, keyed by item ID.
    mapping(bytes32 => bool) public heldWithdrawals;

    /// @notice Counter that makes each screened deposit's item ID unique. Mock-only, so it is
    ///         appended after the real Portal's layout ends rather than placed among it.
    uint256 public depositCount;

    /// @notice Emitted when a transaction is deposited from L1 to L2.
    event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);

    /// @notice Emitted when a withdrawal is proven.
    event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);

    /// @notice Emitted when a withdrawal is proven, naming the proof submitter.
    event WithdrawalProvenExtension1(bytes32 indexed withdrawalHash, address indexed proofSubmitter);

    /// @notice Emitted when a withdrawal is finalized. A held withdrawal emits this only when it
    ///         completes, never when it is held.
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);

    /// @notice Emitted when the ETH balance is migrated to the lockbox.
    event ETHMigrated(address indexed lockbox, uint256 balance);

    /// @notice Emitted when the proof maturity delay is set.
    event ProofMaturityDelaySecondsSet(uint256 proofMaturityDelaySeconds);

    /// @notice Emitted when the bridge hook is set.
    event BridgeHookSet(IBridgeHook indexed bridgeHook);

    /// @notice Emitted when the bridge hook is removed.
    event BridgeHookRemoved(IBridgeHook indexed bridgeHook);

    /// @notice Thrown when a deposit is made in custom gas token mode with value attached.
    error OptimismPortal_NotAllowedOnCGTMode();

    /// @notice Thrown when a contract creation deposit names a non-zero target.
    error OptimismPortal_BadTarget();

    /// @notice Thrown when a deposit's gas limit is below the minimum.
    error OptimismPortal_GasLimitTooLow();

    /// @notice Thrown when a deposit's calldata is too large.
    error OptimismPortal_CalldataTooLarge();

    /// @notice Thrown when removing or setting a bridge hook that leaves the state invalid.
    error OptimismPortal_InvalidBridgeHookState();

    /// @notice Thrown when a call is made while the system is paused.
    error OptimismPortal_CallPaused();

    /// @notice Thrown when finalizing a withdrawal that was never proven.
    error OptimismPortal_Unproven();

    /// @notice Thrown when a proof has not matured.
    error OptimismPortal_ProofNotOldEnough();

    /// @notice Thrown when finalizing a withdrawal that is already finalized.
    error OptimismPortal_AlreadyFinalized();

    /// @notice Thrown when a finalization is attempted inside another withdrawal's execution.
    error OptimismPortal_NoReentrancy();

    /// @notice Thrown when the forwarded withdrawal call fails during a completion.
    error OptimismPortal_CompletionFailed();

    /// @notice Thrown by a failed finalization sent from the gas estimation address.
    error OptimismPortal_GasEstimation();

    /// @notice Thrown when the caller is not the bridge hook.
    error OptimismPortal_NotBridgeHook();

    /// @notice Thrown when completing an item that is not held.
    error OptimismPortal_NotHeld();

    /// @notice Thrown when the ETH sent with a completion does not match the held item.
    error OptimismPortal_ValueMismatch();

    /// @notice Semantic version.
    /// @custom:semver 7.2.0-mock.1
    function version() public pure virtual returns (string memory) {
        return "7.2.0-mock.1";
    }

    /// @param _minProofMaturityDelaySeconds The lowest proof maturity delay a chain may use.
    /// @param _maxProofMaturityDelaySeconds The highest proof maturity delay a chain may use.
    constructor(uint256 _minProofMaturityDelaySeconds, uint256 _maxProofMaturityDelaySeconds) ReinitializableBase(4) {
        MIN_PROOF_MATURITY_DELAY_SECONDS = _minProofMaturityDelaySeconds;
        MAX_PROOF_MATURITY_DELAY_SECONDS = _maxProofMaturityDelaySeconds;
        _disableInitializers();
    }

    /// @notice Initializer. Mirrors the real Portal's signature exactly, because OPCM encodes
    ///         this call and passes a zero bridge hook.
    /// @param _systemConfig Address of the SystemConfig.
    /// @param _anchorStateRegistry Address of the AnchorStateRegistry.
    /// @param _ethLockbox Address of the ETHLockbox.
    /// @param _proofMaturityDelaySeconds The proof maturity delay in seconds.
    /// @param _bridgeHook Bridge hook to set, or the zero address to keep the current one.
    function initialize(
        ISystemConfig _systemConfig,
        IAnchorStateRegistry _anchorStateRegistry,
        IETHLockbox _ethLockbox,
        uint256 _proofMaturityDelaySeconds,
        IBridgeHook _bridgeHook
    )
        external
        reinitializer(initVersion())
    {
        _assertOnlyProxyAdminOrProxyAdminOwner();

        systemConfig = _systemConfig;
        anchorStateRegistry = _anchorStateRegistry;
        if (address(_ethLockbox) != address(0)) ethLockbox = _ethLockbox;
        if (address(_bridgeHook) != address(0)) {
            bridgeHook = _bridgeHook;
            emit BridgeHookSet(_bridgeHook);
        }

        proofMaturityDelaySeconds = _proofMaturityDelaySeconds;
        emit ProofMaturityDelaySecondsSet(_proofMaturityDelaySeconds);

        if (l2Sender == address(0)) l2Sender = Constants.DEFAULT_L2_SENDER;
    }

    /// @notice Accepts ETH and deposits it to the sender's address on L2.
    receive() external payable {
        depositTransaction(msg.sender, msg.value, RECEIVE_DEFAULT_GAS_LIMIT, false, bytes(""));
    }

    /// @notice Accepts ETH without triggering a deposit. Used by the ETHLockbox on unlock.
    function donateETH() external payable {
        // Intentionally empty.
    }

    /// @notice Getter for the current paused status.
    function paused() public view returns (bool) {
        return ethLockbox.paused();
    }

    /// @notice Getter for the SuperchainConfig, served by the lockbox as on the real Portal.
    function superchainConfig() external view returns (ISuperchainConfig) {
        return ethLockbox.superchainConfig();
    }

    /// @notice Getter for the guardian, served by the lockbox as on the real Portal.
    function guardian() external view returns (address) {
        return ethLockbox.guardian();
    }

    /// @notice Getter for the DisputeGameFactory, which withdrawal tooling reads.
    function disputeGameFactory() public view returns (IDisputeGameFactory) {
        return anchorStateRegistry.disputeGameFactory();
    }

    /// @notice Computes the minimum gas limit for a deposit of the given calldata size.
    function minimumGasLimit(uint64 _byteCount) public pure returns (uint64) {
        return _byteCount * 40 + 21000;
    }

    /// @notice Getter for the lowest proof maturity delay this Portal accepts at initialization.
    function minProofMaturityDelaySeconds() external view returns (uint256) {
        return MIN_PROOF_MATURITY_DELAY_SECONDS;
    }

    /// @notice Getter for the highest proof maturity delay this Portal accepts at initialization.
    function maxProofMaturityDelaySeconds() external view returns (uint256) {
        return MAX_PROOF_MATURITY_DELAY_SECONDS;
    }

    /// @notice Sets the proof maturity delay. Unlike the real Portal this accepts any value,
    ///         including zero, so a devnet withdrawal can be finalized without waiting.
    /// @param _proofMaturityDelaySeconds The new delay in seconds.
    function setProofMaturityDelaySeconds(uint256 _proofMaturityDelaySeconds) external {
        _assertOnlyProxyAdminOwner();
        proofMaturityDelaySeconds = _proofMaturityDelaySeconds;
        emit ProofMaturityDelaySecondsSet(_proofMaturityDelaySeconds);
    }

    /// @notice Locks the Portal's whole ETH balance into the lockbox. OPCM calls this on an
    ///         initial deployment.
    function migrateLiquidity() public {
        _assertOnlyProxyAdminOwner();
        uint256 ethBalance = address(this).balance;
        if (_isUsingLockbox()) _lockETH(ethBalance);
        emit ETHMigrated(address(ethLockbox), ethBalance);
    }

    /// @notice Sets the bridge hook after deployment.
    /// @dev    MOCK ONLY, and a deliberate divergence from the spec, which makes initialize the
    ///         only way to set a hook. OPCM passes a zero hook and ChainAssertions requires the
    ///         hook to still be zero when DeployOPChain finishes, so there is no other moment to
    ///         wire a module that itself needs the Portal's address. Do not copy this into the
    ///         real Portal.
    /// @param _bridgeHook The hook to set.
    function setBridgeHook(IBridgeHook _bridgeHook) external {
        _assertOnlyProxyAdminOwner();
        if (address(_bridgeHook) == address(0)) revert OptimismPortal_InvalidBridgeHookState();
        bridgeHook = _bridgeHook;
        emit BridgeHookSet(_bridgeHook);
    }

    /// @notice Removes the bridge hook. The removed hook keeps the value it holds.
    function removeHook() external {
        _assertOnlyProxyAdminOwner();
        if (address(bridgeHook) == address(0)) revert OptimismPortal_InvalidBridgeHookState();
        IBridgeHook hook = bridgeHook;
        bridgeHook = IBridgeHook(address(0));
        emit BridgeHookRemoved(hook);
    }

    /// @notice Accepts deposits of ETH and data, and emits a TransactionDeposited event unless
    ///         the bridge hook holds the deposit.
    /// @param _to         Target address on L2.
    /// @param _value      ETH value to send to the recipient.
    /// @param _gasLimit   Minimum L2 gas limit.
    /// @param _isCreation Whether the deposit is a contract creation.
    /// @param _data       Data to trigger the recipient with.
    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        public
        payable
    {
        if (_isUsingCustomGasToken() && msg.value > 0) revert OptimismPortal_NotAllowedOnCGTMode();
        if (_isCreation && _to != address(0)) revert OptimismPortal_BadTarget();
        if (_gasLimit < minimumGasLimit(uint64(_data.length))) revert OptimismPortal_GasLimitTooLow();
        if (_data.length > 120_000) revert OptimismPortal_CalldataTooLarge();

        address from = msg.sender;
        bool aliased = !EOA.isSenderEOA();
        if (aliased) from = AddressAliasHelper.applyL1ToL2Alias(msg.sender);

        // A deposit the hook holds does not reach L2 now. Its value moves to the hook and it
        // emits no TransactionDeposited until it completes.
        if (_isUsingBridgeHook() && !_screenDeposit(from, aliased, _to, _value, _gasLimit, _isCreation, _data)) {
            return;
        }

        // Held deposits never lock, so the lock happens only once the hook has let it proceed.
        if (_isUsingLockbox() && msg.value > 0) _lockETH(msg.value);

        emit TransactionDeposited(
            from, _to, DEPOSIT_VERSION, abi.encodePacked(msg.value, _value, _gasLimit, _isCreation, _data)
        );
    }

    /// @notice Sends a held deposit to L2 on its original terms. Only callable by the bridge hook,
    ///         which must attach the item's ETH and approve the bridge for its tokens.
    /// @param _item The item as it was held. Its identifier is rebuilt from this preimage.
    function completeDepositTransaction(BridgeHookTypes.Item memory _item) external payable {
        _releaseHeld(_item, heldDeposits);

        if (_isUsingLockbox() && msg.value > 0) _lockETH(msg.value);

        emit TransactionDeposited(
            _item.from,
            _item.to,
            DEPOSIT_VERSION,
            abi.encodePacked(_item.ethAmount, _item.l2Value, uint64(_item.gasLimit), _item.isCreation, _item.data)
        );
    }

    /// @notice Proves a withdrawal transaction. The mock verifies no merkle proof and consults
    ///         no dispute game, but keeps the real four-argument shape because existing
    ///         withdrawal tooling encodes all four. Re-proving resets the proof timer.
    /// @param _tx Withdrawal transaction to prove.
    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256,
        Types.OutputRootProof calldata,
        bytes[] calldata
    )
        external
    {
        if (_isUnsafeTarget(_tx.target)) revert OptimismPortal_BadTarget();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);
        provenWithdrawals[withdrawalHash][msg.sender] =
            ProvenWithdrawal({ disputeGameProxy: address(0), timestamp: uint64(block.timestamp) });
        proofSubmitters[withdrawalHash].push(msg.sender);

        emit WithdrawalProven(withdrawalHash, _tx.sender, _tx.target);
        emit WithdrawalProvenExtension1(withdrawalHash, msg.sender);
    }

    /// @notice Checks that a withdrawal has been proven and its proof has matured.
    /// @param _withdrawalHash Hash of the withdrawal.
    /// @param _proofSubmitter Address of the proof submitter.
    function checkWithdrawal(bytes32 _withdrawalHash, address _proofSubmitter) public view {
        if (finalizedWithdrawals[_withdrawalHash]) revert OptimismPortal_AlreadyFinalized();

        uint64 provenAt = provenWithdrawals[_withdrawalHash][_proofSubmitter].timestamp;
        if (provenAt == 0) revert OptimismPortal_Unproven();
        if (block.timestamp - provenAt <= proofMaturityDelaySeconds) revert OptimismPortal_ProofNotOldEnough();
    }

    /// @notice Returns the number of proof submitters for a withdrawal hash.
    function numProofSubmitters(bytes32 _withdrawalHash) external view returns (uint256) {
        return proofSubmitters[_withdrawalHash].length;
    }

    /// @notice Finalizes a withdrawal transaction using the caller's own proof.
    /// @param _tx Withdrawal transaction to finalize.
    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external {
        finalizeWithdrawalTransactionExternalProof(_tx, msg.sender);
    }

    /// @notice Finalizes a withdrawal transaction against a proof submitted by another address.
    /// @param _tx             Withdrawal transaction to finalize.
    /// @param _proofSubmitter Address that submitted the proof.
    function finalizeWithdrawalTransactionExternalProof(
        Types.WithdrawalTransaction memory _tx,
        address _proofSubmitter
    )
        public
    {
        _assertNotPaused();
        if (_isUsingCustomGasToken() && _tx.value > 0) revert OptimismPortal_NotAllowedOnCGTMode();
        if (l2Sender != Constants.DEFAULT_L2_SENDER) revert OptimismPortal_NoReentrancy();
        if (_isUnsafeTarget(_tx.target)) revert OptimismPortal_BadTarget();

        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);
        checkWithdrawal(withdrawalHash, _proofSubmitter);

        // The withdrawal is consumed whether it pays out or is held, so the commitment is the
        // only way a held withdrawal ever executes.
        finalizedWithdrawals[withdrawalHash] = true;

        // A withdrawal the hook holds does not forward its call and emits no WithdrawalFinalized.
        if (_isUsingBridgeHook() && !_screenWithdrawal(_tx, withdrawalHash, _proofSubmitter)) return;

        if (_isUsingLockbox() && _tx.value > 0) _unlockETH(_tx.value);

        l2Sender = _tx.sender;
        bool success = SafeCall.callWithMinGas(_tx.target, _tx.gasLimit, _tx.value, _tx.data);
        l2Sender = Constants.DEFAULT_L2_SENDER;

        // Fires even when the relay swallowed a failure of its own, because the Portal only
        // knows that its own call returned.
        emit WithdrawalFinalized(withdrawalHash, success);

        if (_isUsingLockbox() && !success && _tx.value > 0) _lockETH(_tx.value);
        if (!success && tx.origin == Constants.ESTIMATION_ADDRESS) revert OptimismPortal_GasEstimation();
    }

    /// @notice Runs a held withdrawal on its committed terms. Only callable by the bridge hook,
    ///         which must attach the item's ETH and approve the bridge for its tokens.
    /// @param _item The item as it was held. Its identifier is rebuilt from this preimage.
    function completeWithdrawalTransaction(BridgeHookTypes.Item memory _item) external payable {
        _assertNotPaused();
        if (l2Sender != Constants.DEFAULT_L2_SENDER) revert OptimismPortal_NoReentrancy();

        _releaseHeld(_item, heldWithdrawals);

        // The ETH arrives as the value of this call, so nothing is unlocked here. It was already
        // unlocked when the withdrawal was held.
        l2Sender = _item.from;
        bool success = SafeCall.callWithMinGas(_item.to, _item.gasLimit, _item.ethAmount, _item.data);
        l2Sender = Constants.DEFAULT_L2_SENDER;

        // A completion deletes the commitment before it runs, so a failed call cannot be retried.
        // It reverts instead, leaving the item held.
        if (!success) revert OptimismPortal_CompletionFailed();

        emit WithdrawalFinalized(_itemWithdrawalHash(_item), true);
    }

    /// @notice Offers a withdrawal to the bridge hook and, if the hook holds it, commits its
    ///         terms and moves its value to the hook.
    /// @return pass_ True if the withdrawal proceeds, false if the hook holds it.
    function _screenWithdrawal(
        Types.WithdrawalTransaction memory _tx,
        bytes32 _withdrawalHash,
        address _proofSubmitter
    )
        internal
        returns (bool pass_)
    {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ETH,
            from: _tx.sender,
            aliased: false,
            to: _tx.target,
            ethAmount: _tx.value,
            l2Value: 0,
            gasLimit: _tx.gasLimit,
            isCreation: false,
            data: _tx.data,
            nonce: _tx.nonce
        });
        bytes memory message = _recognize(item);

        pass_ = bridgeHook.screenWithdrawal(item, _finalizableAt(_withdrawalHash, _proofSubmitter));
        if (pass_) return pass_;

        _commitHold(item, message, heldWithdrawals);

        // Unlike a held deposit, a held withdrawal does unlock, exactly as the paying path would.
        if (_isUsingLockbox() && item.ethAmount > 0) _unlockETH(item.ethAmount);
        bridgeHook.holdWithdrawal{ value: item.ethAmount }(item);
    }

    /// @notice Returns when a withdrawal became finalizable, so a policy can decide how long to
    ///         wait before holding an uncleared one.
    function _finalizableAt(bytes32 _withdrawalHash, address _proofSubmitter) internal view returns (uint256) {
        return provenWithdrawals[_withdrawalHash][_proofSubmitter].timestamp + proofMaturityDelaySeconds;
    }

    /// @notice Rebuilds a held withdrawal's hash from the item, which carries every field of the
    ///         original withdrawal.
    function _itemWithdrawalHash(BridgeHookTypes.Item memory _item) internal pure returns (bytes32) {
        return Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: _item.nonce,
                sender: _item.from,
                target: _item.to,
                value: _item.ethAmount,
                gasLimit: _item.gasLimit,
                data: _item.data
            })
        );
    }

    /// @notice Reverts when the system is paused.
    function _assertNotPaused() internal view {
        if (paused()) revert OptimismPortal_CallPaused();
    }

    /// @notice Returns whether the given target may not be called by a withdrawal.
    function _isUnsafeTarget(address _target) internal view returns (bool) {
        if (_target == address(this) || _target == address(ethLockbox)) return true;
        if (_target == address(_l1StandardBridge())) return true;
        return _isUsingBridgeHook() && _target == address(bridgeHook);
    }

    /// @notice Builds the item a deposit with these terms would be screened as, so an off-chain
    ///         service can check its own identifier computation against the chain.
    /// @dev    MOCK ONLY. Uses the counter the next deposit would be assigned.
    function buildDepositItem(
        address _from,
        bool _aliased,
        address _to,
        uint256 _ethAmount,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external
        view
        returns (BridgeHookTypes.Item memory item_, bytes32 id_)
    {
        item_ = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: _from,
            aliased: _aliased,
            to: _to,
            ethAmount: _ethAmount,
            l2Value: _value,
            gasLimit: _gasLimit,
            isCreation: _isCreation,
            data: _data,
            nonce: depositCount
        });
        _recognize(item_);
        id_ = BridgeHookHashing.hashItem(item_);
    }

    /// @notice Offers a deposit to the bridge hook and, if the hook holds it, commits its terms
    ///         and moves its value to the hook.
    /// @return pass_ True if the deposit proceeds, false if the hook holds it.
    function _screenDeposit(
        address _from,
        bool _aliased,
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        internal
        returns (bool pass_)
    {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Deposit,
            asset: BridgeHookTypes.Asset.ETH,
            from: _from,
            aliased: _aliased,
            to: _to,
            ethAmount: msg.value,
            l2Value: _value,
            gasLimit: _gasLimit,
            isCreation: _isCreation,
            data: _data,
            nonce: depositCount++
        });
        bytes memory message = _recognize(item);

        pass_ = bridgeHook.screenDeposit(item);
        if (pass_) return pass_;

        _commitHold(item, message, heldDeposits);
        bridgeHook.holdDeposit{ value: item.ethAmount }(item);
    }

    /// @notice Marks the item as a token transfer if it is one.
    /// @return message_ The bridge's inner call, or empty if not a token transfer.
    function _recognize(BridgeHookTypes.Item memory _item) internal view returns (bytes memory message_) {
        bool isTokenTransfer;
        (isTokenTransfer, message_) = BridgeHookDecoder.recognizeTokenTransfer(
            _item, systemConfig.l1CrossDomainMessenger(), address(_l1StandardBridge())
        );
        if (isTokenTransfer) _item.asset = BridgeHookTypes.Asset.ERC20;
    }

    /// @notice Commits a held item and moves its escrowed tokens to the hook. The bridge moves
    ///         first, so the hook has custody of the tokens before it is asked to take the item.
    function _commitHold(
        BridgeHookTypes.Item memory _item,
        bytes memory _message,
        mapping(bytes32 => bool) storage _held
    )
        internal
    {
        _held[BridgeHookHashing.hashItem(_item)] = true;
        if (_item.asset == BridgeHookTypes.Asset.ERC20) {
            _l1StandardBridge().holdERC20(_item.direction, _message);
        }
    }

    /// @notice Deletes a held item's commitment and returns its escrowed tokens to the bridge.
    ///         Reverts unless the caller is the hook and sends exactly the item's ETH.
    function _releaseHeld(BridgeHookTypes.Item memory _item, mapping(bytes32 => bool) storage _held) internal {
        if (msg.sender != address(bridgeHook)) revert OptimismPortal_NotBridgeHook();

        bytes32 id = BridgeHookHashing.hashItem(_item);
        if (!_held[id]) revert OptimismPortal_NotHeld();
        delete _held[id];

        if (msg.value != _item.ethAmount) revert OptimismPortal_ValueMismatch();

        if (_item.asset == BridgeHookTypes.Asset.ERC20) {
            (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(_item);
            _l1StandardBridge().restoreERC20(_item.direction, message);
        }
    }

    /// @notice Checks if a bridge hook is set.
    function _isUsingBridgeHook() internal view returns (bool) {
        return address(bridgeHook) != address(0);
    }

    /// @notice Checks if the chain uses the ETHLockbox.
    function _isUsingLockbox() internal view returns (bool) {
        return _isFeatureEnabled(Features.ETH_LOCKBOX) && address(ethLockbox) != address(0);
    }

    /// @notice Checks if the chain uses a custom gas token.
    function _isUsingCustomGasToken() internal view returns (bool) {
        return _isFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
    }

    /// @notice Checks if a SystemConfig feature is enabled.
    function _isFeatureEnabled(bytes32 _feature) internal view returns (bool) {
        return systemConfig.isFeatureEnabled(_feature);
    }

    /// @notice Resolves the L1StandardBridge through the SystemConfig.
    function _l1StandardBridge() internal view returns (IL1StandardBridge) {
        return IL1StandardBridge(payable(systemConfig.l1StandardBridge()));
    }

    /// @notice Locks ETH in the lockbox.
    function _lockETH(uint256 _amount) internal {
        ethLockbox.lockETH{ value: _amount }();
    }

    /// @notice Unlocks ETH from the lockbox.
    function _unlockETH(uint256 _amount) internal {
        ethLockbox.unlockETH(_amount);
    }
}
