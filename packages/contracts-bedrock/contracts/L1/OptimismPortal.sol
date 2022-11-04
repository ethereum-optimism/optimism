// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";
import { SafeCall } from "../libraries/SafeCall.sol";
import { L2OutputOracle } from "./L2OutputOracle.sol";
import { Types } from "../libraries/Types.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { SecureMerkleTrie } from "../libraries/trie/SecureMerkleTrie.sol";
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { ResourceMetering } from "./ResourceMetering.sol";
import { Semver } from "../universal/Semver.sol";

/**
 * @custom:proxied
 * @title OptimismPortal
 * @notice The OptimismPortal is a low-level contract responsible for passing messages between L1
 *         and L2. Messages sent directly to the OptimismPortal have no form of replayability.
 *         Users are encouraged to use the L1CrossDomainMessenger for a higher-level interface.
 */
contract OptimismPortal is Initializable, ResourceMetering, Semver {
    /**
     * @notice Represents a proven withdrawal
     */
    struct ProvenWithdrawal {
        bytes32 outputRoot;
        uint128 timestamp;
        uint128 l2BlockNumber;
    }

    /**
     * @notice Version of the deposit event.
     */
    uint256 internal constant DEPOSIT_VERSION = 0;

    /**
     * @notice Value used to reset the l2Sender, this is more efficient than setting it to zero.
     */
    address internal constant DEFAULT_L2_SENDER = 0x000000000000000000000000000000000000dEaD;

    /**
     * @notice The L2 gas limit set when eth is deposited using the receive() function.
     */
    uint64 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 100_000;

    /**
     * @notice Additional gas reserved for clean up after finalizing a transaction withdrawal.
     */
    uint256 internal constant FINALIZE_GAS_BUFFER = 20_000;

    /**
     * @notice Minimum time (in seconds) that must elapse before a withdrawal can be finalized.
     */
    uint256 public immutable FINALIZATION_PERIOD_SECONDS;

    /**
     * @notice Address of the L2OutputOracle.
     */
    L2OutputOracle public immutable L2_ORACLE;

    /**
     * @notice Address of the L2 account which initiated a withdrawal in this transaction. If the
     *         of this variable is the default L2 sender address, then we are NOT inside of a call
     *         to finalizeWithdrawalTransaction.
     */
    address public l2Sender;

    /**
     * @notice A list of withdrawal hashes which have been successfully finalized.
     */
    mapping(bytes32 => bool) public finalizedWithdrawals;

    /**
     * @notice A mapping of withdrawal hashes to `ProvenWithdrawal` data.
     */
    mapping(bytes32 => ProvenWithdrawal) public provenWithdrawals;

    /**
     * @notice Emitted when a transaction is deposited from L1 to L2. The parameters of this event
     *         are read by the rollup node and used to derive deposit transactions on L2.
     *
     * @param from       Address that triggered the deposit transaction.
     * @param to         Address that the deposit transaction is directed to.
     * @param version    Version of this deposit transaction event.
     * @param opaqueData ABI encoded deposit data to be parsed off-chain.
     */
    event TransactionDeposited(
        address indexed from,
        address indexed to,
        uint256 indexed version,
        bytes opaqueData
    );

    /**
     * @notice Emitted when a withdrawal transaction is proven.
     *
     * @param withdrawalHash Hash of the withdrawal transaction.
     */
    event WithdrawalProven(
        bytes32 indexed withdrawalHash,
        address indexed from,
        address indexed to
    );

    /**
     * @notice Emitted when a withdrawal transaction is finalized.
     *
     * @param withdrawalHash Hash of the withdrawal transaction.
     * @param success        Whether the withdrawal transaction was successful.
     */
    event WithdrawalFinalized(bytes32 indexed withdrawalHash, bool success);

    /**
     * @custom:semver 0.0.1
     *
     * @param _l2Oracle                  Address of the L2OutputOracle contract.
     * @param _finalizationPeriodSeconds Output finalization time in seconds.
     */
    constructor(L2OutputOracle _l2Oracle, uint256 _finalizationPeriodSeconds) Semver(0, 0, 1) {
        L2_ORACLE = _l2Oracle;
        FINALIZATION_PERIOD_SECONDS = _finalizationPeriodSeconds;
        initialize();
    }

    /**
     * @notice Initializer;
     */
    function initialize() public initializer {
        l2Sender = DEFAULT_L2_SENDER;
        __ResourceMetering_init();
    }

    /**
     * @notice Accepts value so that users can send ETH directly to this contract and have the
     *         funds be deposited to their address on L2. This is intended as a convenience
     *         function for EOAs. Contracts should call the depositTransaction() function directly
     *         otherwise any deposited funds will be lost due to address aliasing.
     */
    // solhint-disable-next-line ordering
    receive() external payable {
        depositTransaction(msg.sender, msg.value, RECEIVE_DEFAULT_GAS_LIMIT, false, bytes(""));
    }

    /**
     * @notice Accepts ETH value without triggering a deposit to L2. This function mainly exists
     *         for the sake of the migration between the legacy Optimism system and Bedrock.
     */
    function donateETH() external payable {
        // Intentionally empty.
    }

    /**
     * @notice Proves a withdrawal transaction.
     *
     * @param _tx              Withdrawal transaction to finalize.
     * @param _l2BlockNumber   L2 block number of the outputRoot.
     * @param _outputRootProof Inclusion proof of the L2ToL1MessagePasser contract's storage root.
     * @param _withdrawalProof Inclusion proof of the withdrawal in L2ToL1MessagePasser contract.
     */
    function proveWithdrawalTransaction(
        Types.WithdrawalTransaction memory _tx,
        uint256 _l2BlockNumber,
        Types.OutputRootProof calldata _outputRootProof,
        bytes[] calldata _withdrawalProof
    ) external {
        // Prevent users from creating a deposit transaction where this address is the message
        // sender on L2.
        // In the context of the proxy delegate calling to this implementation,
        // address(this) will return the address of the proxy.
        //
        // Because this is checked here, we do not need to check again in
        // `finalizeWithdrawalTransaction`
        require(
            _tx.target != address(this),
            "OptimismPortal: you cannot send messages to the portal contract"
        );

        // Get the output root and load onto the stack to prevent multiple mloads. This will
        // fail if there is no output root for the given block number.
        bytes32 outputRoot = L2_ORACLE.getL2Output(_l2BlockNumber).outputRoot;

        // Verify that the output root can be generated with the elements in the proof.
        require(
            outputRoot == Hashing.hashOutputRootProof(_outputRootProof),
            "OptimismPortal: invalid output root proof"
        );

        // All withdrawals have a unique hash, we'll use this as the identifier for the withdrawal
        // and to prevent replay attacks.
        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);

        // Load the ProvenWithdrawal into memory
        ProvenWithdrawal memory provenWithdrawal = provenWithdrawals[withdrawalHash];

        // Only allow re-proving a withdrawal transaction if the output root has changed.
        require(
            provenWithdrawal.timestamp == 0 ||
                (_l2BlockNumber == provenWithdrawal.l2BlockNumber &&
                    outputRoot != provenWithdrawal.outputRoot),
            "OptimismPortal: withdrawal hash has already been proven"
        );

        // Verify that the hash of this withdrawal was stored in the L2toL1MessagePasser contract on
        // L2. If this is true, then we know that this withdrawal was actually triggered on L2
        // and can therefore be relayed on L1.
        require(
            _verifyWithdrawalInclusion(
                withdrawalHash,
                _outputRootProof.messagePasserStorageRoot,
                _withdrawalProof
            ),
            "OptimismPortal: invalid withdrawal inclusion proof"
        );

        // Designate the withdrawalHash as proven by storing the `outputRoot`, `timestamp`,
        // and `l2BlockNumber` in the `provenWithdrawals` mapping. A withdrawalHash can only
        // be proven once to prevent a censorship attack unless it is submitted again
        // with a different outputRoot.
        provenWithdrawals[withdrawalHash] = ProvenWithdrawal({
            outputRoot: outputRoot,
            timestamp: uint128(block.timestamp),
            l2BlockNumber: uint128(_l2BlockNumber)
        });

        // Emit a `WithdrawalProven` event.
        emit WithdrawalProven(withdrawalHash, _tx.sender, _tx.target);
    }

    /**
     * @notice Finalizes a withdrawal transaction.
     *
     * @param _tx Withdrawal transaction to finalize.
     */
    function finalizeWithdrawalTransaction(Types.WithdrawalTransaction memory _tx) external {
        // Prevent nested withdrawals within withdrawals.
        require(
            l2Sender == DEFAULT_L2_SENDER,
            "OptimismPortal: can only trigger one withdrawal per transaction"
        );

        // All withdrawals have a unique hash, we'll use this as the identifier for the withdrawal
        // and to prevent replay attacks.
        bytes32 withdrawalHash = Hashing.hashWithdrawal(_tx);

        // Grab the proven withdrawal from the `provenWithdrawals` map.
        ProvenWithdrawal memory provenWithdrawal = provenWithdrawals[withdrawalHash];

        // Ensure that the withdrawal has been proven
        require(provenWithdrawal.timestamp != 0, "OptimismPortal: withdrawal has not been proven");

        // Ensure that the proven withdrawal's timestamp is greater than the
        // L2 Oracle's starting timestamp.
        require(
            provenWithdrawal.timestamp >= L2_ORACLE.startingTimestamp(),
            "OptimismPortal: withdrawal timestamp less than L2 Oracle starting timestamp"
        );

        // Ensure that the withdrawal's finalization period has elapsed.
        require(
            _isFinalizationPeriodElapsed(provenWithdrawal.timestamp),
            "OptimismPortal: proven withdrawal finalization period has not elapsed"
        );

        // Grab the OutputProposal from the L2 Oracle
        Types.OutputProposal memory proposal = L2_ORACLE.getL2Output(
            provenWithdrawal.l2BlockNumber
        );

        // Check that the output proposal hasn't been updated.
        require(
            proposal.outputRoot == provenWithdrawal.outputRoot,
            "OptimismPortal: output root proven is not the same as current output root"
        );

        // Perform second checks on the withdrawal's finalization period, this time with
        // the `OutputProposal`'s timestamp fetched from the L2 Oracle.
        require(
            _isFinalizationPeriodElapsed(proposal.timestamp),
            "OptimismPortal: output proposal finalization period has not elapsed"
        );

        // Check that this withdrawal has not already been finalized, this is replay protection.
        require(
            finalizedWithdrawals[withdrawalHash] == false,
            "OptimismPortal: withdrawal has already been finalized"
        );

        // Mark the withdrawal as finalized so it can't be replayed.
        finalizedWithdrawals[withdrawalHash] = true;

        // We want to maintain the property that the amount of gas supplied to the call to the
        // target contract is at least the gas limit specified by the user. We can do this by
        // enforcing that, at this point in time, we still have gaslimit + buffer gas available.
        require(
            gasleft() >= _tx.gasLimit + FINALIZE_GAS_BUFFER,
            "OptimismPortal: insufficient gas to finalize withdrawal"
        );

        // Set the l2Sender so contracts know who triggered this withdrawal on L2.
        l2Sender = _tx.sender;

        // Trigger the call to the target contract. We use SafeCall because we don't
        // care about the returndata and we don't want target contracts to be able to force this
        // call to run out of gas via a returndata bomb.
        bool success = SafeCall.call(_tx.target, _tx.gasLimit, _tx.value, _tx.data);

        // Reset the l2Sender back to the default value.
        l2Sender = DEFAULT_L2_SENDER;

        // All withdrawals are immediately finalized. Replayability can
        // be achieved through contracts built on top of this contract
        emit WithdrawalFinalized(withdrawalHash, success);
    }

    /**
     * @notice Determine if a given block number is finalized. Reverts if the call to
     *         L2_ORACLE.getL2Output reverts. Returns a boolean otherwise.
     *
     * @param _l2BlockNumber The number of the L2 block.
     */
    function isBlockFinalized(uint256 _l2BlockNumber) external view returns (bool) {
        return _isFinalizationPeriodElapsed(L2_ORACLE.getL2Output(_l2BlockNumber).timestamp);
    }

    /**
     * @notice Accepts deposits of ETH and data, and emits a TransactionDeposited event for use in
     *         deriving deposit transactions. Note that if a deposit is made by a contract, its
     *         address will be aliased when retrieved using `tx.origin` or `msg.sender`. Consider
     *         using the CrossDomainMessenger contracts for a simpler developer experience.
     *
     * @param _to         Target address on L2.
     * @param _value      ETH value to send to the recipient.
     * @param _gasLimit   Minimum L2 gas limit (can be greater than or equal to this value).
     * @param _isCreation Whether or not the transaction is a contract creation.
     * @param _data       Data to trigger the recipient with.
     */
    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    ) public payable metered(_gasLimit) {
        // Just to be safe, make sure that people specify address(0) as the target when doing
        // contract creations.
        if (_isCreation) {
            require(
                _to == address(0),
                "OptimismPortal: must send to address(0) when creating a contract"
            );
        }

        // Transform the from-address to its alias if the caller is a contract.
        address from = msg.sender;
        if (msg.sender != tx.origin) {
            from = AddressAliasHelper.applyL1ToL2Alias(msg.sender);
        }

        bytes memory opaqueData = abi.encodePacked(
            msg.value,
            _value,
            _gasLimit,
            _isCreation,
            _data
        );

        // Emit a TransactionDeposited event so that the rollup node can derive a deposit
        // transaction for this deposit.
        emit TransactionDeposited(from, _to, DEPOSIT_VERSION, opaqueData);
    }

    /**
     * @notice Determine if the finalization period has elapsed with respect to the
     * passed timestamp.
     *
     * @param _timestamp The timestamp to check.
     */
    function _isFinalizationPeriodElapsed(uint256 _timestamp) internal view returns (bool) {
        return block.timestamp > _timestamp + FINALIZATION_PERIOD_SECONDS;
    }

    /**
     * @notice Verifies a Merkle Trie inclusion proof that a given withdrawal hash is present in
     *         the storage of the L2ToL1MessagePasser contract.
     *
     * @param _withdrawalHash Hash of the withdrawal to verify.
     * @param _storageRoot    Root of the storage of the L2ToL1MessagePasser contract.
     * @param _proof          Inclusion proof of the withdrawal hash in the storage root.
     */
    function _verifyWithdrawalInclusion(
        bytes32 _withdrawalHash,
        bytes32 _storageRoot,
        bytes[] memory _proof
    ) internal pure returns (bool) {
        bytes32 storageKey = keccak256(
            abi.encode(
                _withdrawalHash,
                uint256(0) // The withdrawals mapping is at the first slot in the layout.
            )
        );

        return
            SecureMerkleTrie.verifyInclusionProof(
                abi.encode(storageKey),
                hex"01",
                _proof,
                _storageRoot
            );
    }
}
