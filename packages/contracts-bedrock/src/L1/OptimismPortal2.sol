// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { DisputeGameFactory, IDisputeGame } from "src/dispute/DisputeGameFactory.sol";
import { SystemConfig } from "src/L1/SystemConfig.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";
import { Constants } from "src/libraries/Constants.sol";
import { Types } from "src/libraries/Types.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { SecureMerkleTrie } from "src/libraries/trie/SecureMerkleTrie.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { ResourceMetering } from "src/L1/ResourceMetering.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { Constants } from "src/libraries/Constants.sol";

import "src/libraries/DisputeTypes.sol";

/// @custom:proxied
/// @title OptimismPortal2
/// @notice The OptimismPortal is a low-level contract responsible for passing messages between L1
///         and L2. Messages sent directly to the OptimismPortal have no form of replayability.
///         Users are encouraged to use the L1CrossDomainMessenger for a higher-level interface.
contract OptimismPortal2 is Initializable, ResourceMetering, ISemver {
    /// @notice Represents a proven withdrawal.
    /// @custom:field disputeGameProxy The address of the dispute game proxy that the withdrawal was proven against.
    /// @custom:field timestamp        Timestamp at whcih the withdrawal was proven.
    struct ProvenWithdrawal {
        IDisputeGame disputeGameProxy;
        uint64 timestamp;
    }

    /// @dev Remove this in favor of a configurable sauron role. This should probably live in the superchain config,
    ///      but need to confirm with security.
    address internal constant SAURON = address(0xdead);

    /// @notice The delay between when a withdrawal transaction is proven and when it may be finalized.
    uint256 internal immutable PROOF_MATURITY_DELAY_SECONDS;

    /// @notice The delay between when a dispute game is resolved and when a withdrawal proven against it may be
    ///         finalized.
    uint256 internal immutable DISPUTE_GAME_FINALITY_DELAY_SECONDS;

    /// @notice Version of the deposit event.
    uint256 internal constant DEPOSIT_VERSION = 0;

    /// @notice The L2 gas limit set when eth is deposited using the receive() function.
    uint64 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 100_000;

    /// @notice Address of the L2 account which initiated a withdrawal in this transaction.
    ///         If the of this variable is the default L2 sender address, then we are NOT inside of
    ///         a call to finalizeWithdrawalTransaction.
    address public l2Sender;

    /// @notice A list of withdrawal hashes which have been successfully finalized.
    mapping(bytes32 => bool) public finalizedWithdrawals;

    /// @custom:legacy
    /// @custom:spacer provenWithdrawals
    /// @notice Spacer taking up the legacy `provenWithdrawals` mapping slot.
    bytes32 private spacer_52_0_32;

    /// @custom:legacy
    /// @custom:spacer paused
    /// @notice Spacer for backwards compatibility.
    bool private spacer_53_0_1;

    /// @notice Contract of the Superchain Config.
    SuperchainConfig public superchainConfig;

    /// @custom:legacy
    /// @custom:spacer l2Oracle
    /// @notice Spacer taking up the legacy `l2Oracle` address slot.
    address private spacer_54_0_20;

    /// @notice Contract of the SystemConfig.
    /// @custom:network-specific
    SystemConfig public systemConfig;

    /// @notice Address of the DisputeGameFactory.
    /// @custom:network-specific
    DisputeGameFactory public disputeGameFactory;

    /// @notice A mapping of withdrawal hashes to `ProvenWithdrawal` data.
    mapping(bytes32 => ProvenWithdrawal) public provenWithdrawals;

    /// @notice A mapping of dispute game addresses to whether or not they are blacklisted.
    mapping(IDisputeGame => bool) public disputeGameBlacklist;

    /// @notice The game type that the OptimismPortal consults for output proposals.
    GameType public respectedGameType;

    /// @notice Emitted when a transaction is deposited from L1 to L2.
    ///         The parameters of this event are read by the rollup node and used to derive deposit
    ///         transactions on L2.
    /// @param from       Address that triggered the deposit transaction.
    /// @param to         Address that the deposit transaction is directed to.
    /// @param version    Version of this deposit transaction event.
    /// @param opaqueData ABI encoded deposit data to be parsed off-chain.
    event TransactionDeposited(address indexed from, address indexed to, uint256 indexed version, bytes opaqueData);

    /// @notice Emitted when a withdrawal transaction is proven.
    /// @param withdrawalHash Hash of the withdrawal transaction.
    /// @param from           Address that triggered the withdrawal transaction.
    /// @param to             Address that the withdrawal transaction is directed to.
    event WithdrawalProven(bytes32 indexed withdrawalHash, address indexed from, address indexed to);

    /// @notice Emitted when a withdrawal transaction is finalized.
    /// @param withdrawalHash Hash of the withdrawal transaction.
    /// @param success        Whether the withdrawal transaction was successful.
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);

    /// @notice Reverts when paused.
    modifier whenNotPaused() {
        require(!paused(), "OptimismPortal: paused");
        _;
    }

    /// @notice Semantic version.
    /// @custom:semver 3.0.0
    string public constant version = "3.0.0";

    /// @notice Constructs the OptimismPortal contract.
    constructor(
        uint256 _proofMaturityDelaySeconds,
        uint256 _disputeGameFinalityDelaySeconds,
        GameType _initialRespectedGameType
    ) {
        PROOF_MATURITY_DELAY_SECONDS = _proofMaturityDelaySeconds;
        DISPUTE_GAME_FINALITY_DELAY_SECONDS = _disputeGameFinalityDelaySeconds;
        respectedGameType = _initialRespectedGameType;

        initialize({
            _disputeGameFactory: DisputeGameFactory(address(0)),
            _systemConfig: SystemConfig(address(0)),
            _superchainConfig: SuperchainConfig(address(0))
        });
    }

    /// @notice Initializer.
    /// @param _disputeGameFactory Contract of the DisputeGameFactory.
    /// @param _systemConfig Contract of the SystemConfig.
    /// @param _superchainConfig Contract of the SuperchainConfig.
    function initialize(
        DisputeGameFactory _disputeGameFactory,
        SystemConfig _systemConfig,
        SuperchainConfig _superchainConfig
    )
        public
        initializer
    {
        disputeGameFactory = _disputeGameFactory;
        systemConfig = _systemConfig;
        superchainConfig = _superchainConfig;
        if (l2Sender == address(0)) {
            l2Sender = Constants.DEFAULT_L2_SENDER;
        }
        __ResourceMetering_init();
    }

    /// @notice Getter function for the contract of the SystemConfig on this chain.
    ///         Public getter is legacy and will be removed in the future. Use `systemConfig()` instead.
    /// @return Contract of the SystemConfig on this chain.
    /// @custom:legacy
    function SYSTEM_CONFIG() external view returns (SystemConfig) {
        return systemConfig;
    }

    /// @notice Getter function for the address of the guardian.
    ///         Public getter is legacy and will be removed in the future. Use `SuperchainConfig.guardian()` instead.
    /// @return Address of the guardian.
    /// @custom:legacy
    function GUARDIAN() external view returns (address) {
        return guardian();
    }

    /// @notice Getter function for the address of the guardian.
    ///         Public getter is legacy and will be removed in the future. Use `SuperchainConfig.guardian()` instead.
    /// @return Address of the guardian.
    /// @custom:legacy
    function guardian() public view returns (address) {
        return superchainConfig.guardian();
    }

    /// @notice Getter for the current paused status.
    /// @return paused_ Whether or not the contract is paused.
    function paused() public view returns (bool paused_) {
        paused_ = superchainConfig.paused();
    }

    /// @notice Computes the minimum gas limit for a deposit.
    ///         The minimum gas limit linearly increases based on the size of the calldata.
    ///         This is to prevent users from creating L2 resource usage without paying for it.
    ///         This function can be used when interacting with the portal to ensure forwards
    ///         compatibility.
    /// @param _byteCount Number of bytes in the calldata.
    /// @return The minimum gas limit for a deposit.
    function minimumGasLimit(uint64 _byteCount) public pure returns (uint64) {
        return _byteCount * 16 + 21000;
    }

    /// @notice Accepts value so that users can send ETH directly to this contract and have the
    ///         funds be deposited to their address on L2. This is intended as a convenience
    ///         function for EOAs. Contracts should call the depositTransaction() function directly
    ///         otherwise any deposited funds will be lost due to address aliasing.
    // solhint-disable-next-line ordering
    receive() external payable {
        depositTransaction(msg.sender, msg.value, RECEIVE_DEFAULT_GAS_LIMIT, false, bytes(""));
    }

    /// @notice Accepts ETH value without triggering a deposit to L2.
    ///         This function mainly exists for the sake of the migration between the legacy
    ///         Optimism system and Bedrock.
    function donateETH() external payable {
        // Intentionally empty.
    }

    /// @notice Getter for the resource config.
    ///         Used internally by the ResourceMetering contract.
    ///         The SystemConfig is the source of truth for the resource config.
    /// @return ResourceMetering ResourceConfig
    function _resourceConfig() internal view override returns (ResourceMetering.ResourceConfig memory) {
        return systemConfig.resourceConfig();
    }

    /// @notice Proves a withdrawal transaction.
    /// @param _tx               Withdrawal transaction to finalize.
    /// @param _disputeGameIndex Index of the dispute game to prove the withdrawal against.
    /// @param _outputRootProof  Inclusion proof of the L2ToL1MessagePasser contract's storage root.
    /// @param _withdrawalProof  Inclusion proof of the withdrawal in L2ToL1MessagePasser contract.
    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256 _disputeGameIndex,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    )
        external
        whenNotPaused
    {
        // Prevent users from creating a deposit transaction where this address is the message
        // sender on L2. Because this is checked here, we do not need to check again in
        // `finalizeWithdrawalTransaction`.
        require(_tx.target != address(this), "OptimismPortal: you cannot send messages to the portal contract");

        // Fetch the dispute game proxy from the `DisputeGameFactory` contract.
        (GameType gameType,, IDisputeGame gameProxy) = disputeGameFactory.gameAtIndex(_disputeGameIndex);
        Claim outputRoot = gameProxy.rootClaim();

        // The game type of the dispute game must be the respected game type.
        require(gameType.raw() == respectedGameType.raw(), "OptimismPortal: invalid game type");

        // Verify that the output root can be generated with the elements in the proof.
        require(
            outputRoot.raw() == Hashing.hashOutputRootProof(_outputRootProof),
            "OptimismPortal: invalid output root proof"
        );

        // Load the ProvenWithdrawal into memory, using the withdrawal hash as a unique identifier.
        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);
        ProvenWithdrawal memory provenWithdrawal = provenWithdrawals[withdrawalHash];

        // We generally want to prevent users from proving the same withdrawal multiple times
        // because each successive proof will update the timestamp. A malicious user can take
        // advantage of this to prevent other users from finalizing their withdrawal. However,
        // in the case that an honest user proves their withdrawal against a dispute game that
        // resolves against the root claim, or the dispute game is blacklisted, we allow
        // re-proving the withdrawal against a new proposal.
        require(
            provenWithdrawal.timestamp == 0 || gameProxy.status() == GameStatus.CHALLENGER_WINS
                || disputeGameBlacklist[gameProxy],
            "OptimismPortal: withdrawal hash has already been proven, and dispute game is not invalid"
        );

        // Compute the storage slot of the withdrawal hash in the L2ToL1MessagePasser contract.
        // Refer to the Solidity documentation for more information on how storage layouts are
        // computed for mappings.
        bytes32 storageKey = keccak256(
            abi.encode(
                withdrawalHash,
                uint256(0) // The withdrawals mapping is at the first slot in the layout.
            )
        );

        // Verify that the hash of this withdrawal was stored in the L2toL1MessagePasser contract
        // on L2. If this is true, under the assumption that the SecureMerkleTrie does not have
        // bugs, then we know that this withdrawal was actually triggered on L2 and can therefore
        // be relayed on L1.
        require(
            SecureMerkleTrie.verifyInclusionProof(
                abi.encode(storageKey), hex"01", _withdrawalProof, _outputRootProof.messagePasserStorageRoot
            ),
            "OptimismPortal: invalid withdrawal inclusion proof"
        );

        // Designate the withdrawalHash as proven by storing the `disputeGameProxy` & `timestamp` in the
        // `provenWithdrawals` mapping. A `withdrawalHash` can only be proven once unless the dispute game it proved
        // against resolves against the favor of the root claim.
        provenWithdrawals[withdrawalHash] =
            ProvenWithdrawal({ disputeGameProxy: gameProxy, timestamp: uint64(block.timestamp) });

        // Emit a `WithdrawalProven` event.
        emit WithdrawalProven(withdrawalHash, _tx.sender, _tx.target);
    }

    /// @notice Finalizes a withdrawal transaction.
    /// @param _tx Withdrawal transaction to finalize.
    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external whenNotPaused {
        // Make sure that the l2Sender has not yet been set. The l2Sender is set to a value other
        // than the default value when a withdrawal transaction is being finalized. This check is
        // a defacto reentrancy guard.
        require(
            l2Sender == Constants.DEFAULT_L2_SENDER, "OptimismPortal: can only trigger one withdrawal per transaction"
        );

        // Compute the withdrawal hash.
        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);

        // Check that the withdrawal can be finalized.
        checkWithdrawal(withdrawalHash);

        // Mark the withdrawal as finalized so it can't be replayed.
        finalizedWithdrawals[withdrawalHash] = true;

        // Set the l2Sender so contracts know who triggered this withdrawal on L2.
        l2Sender = _tx.sender;

        // Trigger the call to the target contract. We use a custom low level method
        // SafeCall.callWithMinGas to ensure two key properties
        //   1. Target contracts cannot force this call to run out of gas by returning a very large
        //      amount of data (and this is OK because we don't care about the returndata here).
        //   2. The amount of gas provided to the execution context of the target is at least the
        //      gas limit specified by the user. If there is not enough gas in the current context
        //      to accomplish this, `callWithMinGas` will revert.
        bool success = SafeCall.callWithMinGas(_tx.target, _tx.gasLimit, _tx.value, _tx.data);

        // Reset the l2Sender back to the default value.
        l2Sender = Constants.DEFAULT_L2_SENDER;

        // All withdrawals are immediately finalized. Replayability can
        // be achieved through contracts built on top of this contract
        emit WithdrawalFinalized(withdrawalHash, success);

        // Reverting here is useful for determining the exact gas cost to successfully execute the
        // sub call to the target contract if the minimum gas limit specified by the user would not
        // be sufficient to execute the sub call.
        if (!success && tx.origin == Constants.ESTIMATION_ADDRESS) {
            revert("OptimismPortal: withdrawal failed");
        }
    }

    /// @notice Accepts deposits of ETH and data, and emits a TransactionDeposited event for use in
    ///         deriving deposit transactions. Note that if a deposit is made by a contract, its
    ///         address will be aliased when retrieved using `tx.origin` or `msg.sender`. Consider
    ///         using the CrossDomainMessenger contracts for a simpler developer experience.
    /// @param _to         Target address on L2.
    /// @param _value      ETH value to send to the recipient.
    /// @param _gasLimit   Amount of L2 gas to purchase by burning gas on L1.
    /// @param _isCreation Whether or not the transaction is a contract creation.
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
        metered(_gasLimit)
    {
        // Just to be safe, make sure that people specify address(0) as the target when doing
        // contract creations.
        if (_isCreation) {
            require(_to == address(0), "OptimismPortal: must send to address(0) when creating a contract");
        }

        // Prevent depositing transactions that have too small of a gas limit. Users should pay
        // more for more resource usage.
        require(_gasLimit >= minimumGasLimit(uint64(_data.length)), "OptimismPortal: gas limit too small");

        // Prevent the creation of deposit transactions that have too much calldata. This gives an
        // upper limit on the size of unsafe blocks over the p2p network. 120kb is chosen to ensure
        // that the transaction can fit into the p2p network policy of 128kb even though deposit
        // transactions are not gossipped over the p2p network.
        require(_data.length <= 120_000, "OptimismPortal: data too large");

        // Transform the from-address to its alias if the caller is a contract.
        address from = msg.sender;
        if (msg.sender != tx.origin) {
            from = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        // Compute the opaque data that will be emitted as part of the TransactionDeposited event.
        // We use opaque data so that we can update the TransactionDeposited event in the future
        // without breaking the current interface.
        bytes memory opaqueData = abi.encodePacked(msg.value, _value, _gasLimit, _isCreation, _data);

        // Emit a TransactionDeposited event so that the rollup node can derive a deposit
        // transaction for this deposit.
        emit TransactionDeposited(from, _to, DEPOSIT_VERSION, opaqueData);
    }

    /// @notice Blacklists a dispute game. Should only be used in the event that a dispute game resolves incorrectly.
    /// @param _disputeGame Dispute game to blacklist.
    function blacklistDisputeGame(IDisputeGame _disputeGame) external {
        require(msg.sender == SAURON, "OptimismPortal: only sauron can blacklist dispute games");
        disputeGameBlacklist[_disputeGame] = true;
    }

    /// @notice Deletes a proven withdrawal from the `provenWithdrawals` mapping in the case that a MPT proof was
    ///         incorrectly verified by the `MerkleTrie` verifier contract.
    /// @param _withdrawalHash Hash of the withdrawal transaction to delete from the `pendingWithdrawals` mapping.
    function deleteProvenWithdrawal(bytes32 _withdrawalHash) external {
        require(msg.sender == SAURON, "OptimismPortal: only sauron can delete proven withdrawals");
        delete provenWithdrawals[_withdrawalHash];
    }

    /// @notice Sets the respected game type. Changing this value can alter the security properties of the system,
    ///         depending on the new game's behavior.
    /// @param _gameType The game type to consult for output proposals.
    function setRespectedGameType(GameType _gameType) external {
        require(msg.sender == SAURON, "OptimismPortal: only sauron can set the respected game type");
        respectedGameType = _gameType;
    }

    /// @notice Checks if a withdrawal can be finalized. NOTE: Decision was made to have this
    ///         function revert rather than returning a boolean so that was more obvious why the
    ///         function failed.
    /// @param _withdrawalHash Hash of the withdrawal to check.
    function checkWithdrawal(bytes32 _withdrawalHash) public view {
        ProvenWithdrawal memory provenWithdrawal = provenWithdrawals[_withdrawalHash];
        IDisputeGame disputeGameProxy = provenWithdrawal.disputeGameProxy;

        // The dispute game must not be blacklisted.
        require(!disputeGameBlacklist[disputeGameProxy], "OptimismPortal: dispute game has been blacklisted");

        // A withdrawal can only be finalized if it has been proven. We know that a withdrawal has
        // been proven at least once when its timestamp is non-zero. Unproven withdrawals will have
        // a timestamp of zero.
        require(provenWithdrawal.timestamp != 0, "OptimismPortal: withdrawal has not been proven yet");

        // As a sanity check, we make sure that the proven withdrawal's timestamp is greater than
        // starting timestamp inside the Dispute Game. Not strictly necessary but extra layer of
        // safety against weird bugs in the proving step.
        require(
            provenWithdrawal.timestamp > disputeGameProxy.createdAt().raw(),
            "OptimismPortal: withdrawal timestamp less than L2 Oracle starting timestamp"
        );

        // A proven withdrawal must wait at least `PROOF_MATURITY_DELAY_SECONDS` before finalizing.
        require(
            block.timestamp - provenWithdrawal.timestamp > PROOF_MATURITY_DELAY_SECONDS,
            "OptimismPortal: proven withdrawal has not matured yet"
        );

        // A proven withdrawal must wait until the dispute game it was proven against has been
        // resolved in favor of the root claim (the output proposal). This is to prevent users
        // from finalizing withdrawals proven against non-finalized output roots.
        require(
            disputeGameProxy.status() == GameStatus.DEFENDER_WINS,
            "OptimismPortal: output proposal has not been finalized yet"
        );

        // Before a withdrawal can be finalized, the dispute game it was proven against must have been
        // resolved for at least `DISPUTE_GAME_FINALITY_DELAY_SECONDS`. This is to allow for manual
        // intervention in the event that a dispute game is resolved incorrectly.
        require(
            block.timestamp - disputeGameProxy.resolvedAt().raw() > DISPUTE_GAME_FINALITY_DELAY_SECONDS,
            "OptimismPortal: output proposal in air-gap"
        );

        // Check that this withdrawal has not already been finalized, this is replay protection.
        require(!finalizedWithdrawals[_withdrawalHash], "OptimismPortal: withdrawal has already been finalized");
    }
}
