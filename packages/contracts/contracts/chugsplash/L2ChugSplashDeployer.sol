// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_ExecutionManagerWrapper } from "../optimistic-ethereum/libraries/wrappers/Lib_ExecutionManagerWrapper.sol";
import { Lib_MerkleTree } from "../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";
import { Lib_EIP155Tx } from "../optimistic-ethereum/libraries/codec/Lib_EIP155Tx.sol";

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title L2ChugSplashDeployer
 * @dev Contract responsible for managing and executing "ChugSplash" upgrades. The ChugSplash
 * scheme involves "upgrade bundles" containing a set of "actions." An action can take one of two
 * types, SET_CODE or SET_STORAGE. A SET_CODE action will modify the code of a given account,
 * whereas a SET_STORAGE action will modify the value of a given storage slot for an account.
 * An owner (a multisig on mainnet) can approve a bundle by specifying the root of a Merkle tree
 * generated from the actions within the bundle. Any other account is then allowed to execute any
 * actions (in any order) by demonstrating with a Merkle proof that the action was approved by the
 * contract owner. Only a single upgrade may be active at any given time.
 */
contract L2ChugSplashDeployer is Ownable {

    /*************
     * Libraries *
     *************/

    using Lib_EIP155Tx for Lib_EIP155Tx.EIP155Tx;


    /*********
     * Enums *
     *********/

    enum ActionType {
        SET_CODE,
        SET_STORAGE
    }


    /**********
     * Events *
     **********/
    
    event BundleApproved(
        bytes32 indexed bundleHash,
        uint256 indexed bundleNonce,
        address indexed who,
        uint256 bundleSize
    );
    
    event BundleCompleted(
        bytes32 indexed bundleHash,
        uint256 indexed bundleNonce
    );

    event ActionExecuted(
        bytes32 indexed bundleHash,
        uint256 indexed bundleNonce,
        uint256 actionIndex
    );


    /***********
     * Structs *
     ***********/

    struct ChugSplashAction {
        ActionType actionType;
        address target;
        bytes data;
    }

    struct ChugSplashActionProof {
        uint256 actionIndex;
        bytes32[] siblings;
    }


    /*************
     * Constants *
     *************/

    // bytes32(uint256(keccak256("upgrading")) - 1);
    bytes32 public constant SLOT_KEY_IS_UPGRADING = 0xac04bb17f7be83a1536e4b894c20a9b8acafb7c35cd304dfa3dabeee91e3c4c2;


    /*************
     * Variables *
     *************/

    // Unique number for each bundle, incremented whenever a new bundle is activated.
    uint256 public currentBundleNonce;

    // A merkle root which commits to all steps in the currently active bundle.
    bytes32 public currentBundleHash;

    // The total number of actions in the currently active bundle.
    uint256 public currentBundleSize;

    // The number of actions in the bundle which have been completed so far.
    uint256 public currentBundleTxsExecuted;

    // A boolean for whether or not an action has been completed, across all bundles and their actions.
    mapping (uint256 => mapping (uint256 => bool)) internal completedBundleActions;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Address that will initially own the L2ChugSplashDeployer.
     */
    constructor(
        address _owner
    )
        Ownable()
    {
        transferOwnership(_owner);
    }


    /*********************
     * Fallback Function *
     *********************/

    fallback()
        external
    {
        // Fallback function is used as a way to gracefully handle upgrades. When
        // `isUpgrading() == true`, all transactions are automatically routed to this contract.
        // msg.data may (or may not) be a properly encoded EIP-155 transaction. However, it’s ok to
        // assume that the data *is* properly encoded because any improperly encoded transactions
        // will simply trigger a revert when we try to decode them below. Since L1 => L2 messages
        // are *not* EIP-155 transactions, they’ll revert here (meaning those messages will fail).
        // As a result, any L1 => L2 messages executed during an upgrade will have to be replayed.

        // We use this twice, so it’s more gas efficient to store a copy of it (barely).
        bytes memory encodedTx = msg.data;

        // Decode the tx with the correct chain ID.
        Lib_EIP155Tx.EIP155Tx memory decodedTx = Lib_EIP155Tx.decode(
            encodedTx,
            Lib_ExecutionManagerWrapper.ovmCHAINID()
        );

        // Make sure that the transaction target is the L2ChugSplashDeployer. Any transactions that
        // were not intended to be sent to this contract will revert at this point.
        require(
            decodedTx.to == address(this),
            "L2ChugSplashDeployer: the system is currently undergoing an upgrade"
        );

        // Call into this contract with the decoded transaction data. Of course this means that
        // any functions with onlyOwner cannot be triggered via this fallback, but that’s ok
        // because we only need to be able to trigger executeAction.
        (bool success, bytes memory returndata) = address(this).call(decodedTx.data);

        if (success) {
            assembly {
                return(add(returndata, 0x20), mload(returndata))
            }
        } else {
            assembly {
                revert(add(returndata, 0x20), mload(returndata))
            }
        }
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * @return boolean, whether or not an upgrade is currently being executed.
     */
    function isUpgrading()
        public
        view
        returns (
            bool
        )
    {
        bool status;
        assembly {
            status := sload(SLOT_KEY_IS_UPGRADING)
        }
        return status;
    }

    /**
     * Allows the owner to approve a new upgrade bundle.
     * @param _bundleHash Root of the Merkle tree of actions in this bundle.
     * @param _bundleSize Total number of elements in the bundle.
     */
    function approveTransactionBundle(
        bytes32 _bundleHash,
        uint256 _bundleSize
    )
        public
        onlyOwner
    {
        require(
            _bundleHash != bytes32(0),
            "ChugSplashDeployer: bundle hash must not be the empty hash"
        );

        require(
            _bundleSize > 0,
            "ChugSplashDeployer: bundle must include at least one action"
        );

        require(
            isUpgrading() == false,
            "ChugSplashDeployer: previous bundle is still active"
        );

        currentBundleNonce += 1;
        currentBundleHash = _bundleHash;
        currentBundleSize = _bundleSize;
        currentBundleTxsExecuted = 0;
        _setUpgradeStatus(true);

        emit BundleApproved(
            _bundleHash,
            currentBundleNonce,
            msg.sender,
            _bundleSize
        );
    }

    /**
     * Allows anyone to execute an action that has been approved as part of an upgrade bundle.
     * @param _action ChugSplashAction to execute.
     * @param _proof Proof that the given action was included in the upgrade bundle.
     */
    function executeAction(
        ChugSplashAction memory _action,
        ChugSplashActionProof memory _proof
    )
        public
    {
        require(
            isUpgrading() == true,
            "ChugSplashDeployer: there is no active bundle"
        );

        require(
            completedBundleActions[currentBundleNonce][_proof.actionIndex] == false,
            "ChugSplashDeployer: action has already been executed"
        );

        bytes32 actionHash = keccak256(
            abi.encode(
                _action.actionType,
                _action.target,
                _action.data
            )
        );

        // Make sure that the owner did actually sign off on this action.
        require(
            Lib_MerkleTree.verify(
                currentBundleHash,
                actionHash,
                _proof.actionIndex,
                _proof.siblings,
                currentBundleSize
            ),
            "ChugSplashDeployer: invalid action proof"
        );

        if (_action.actionType == ActionType.SET_CODE) {
            // When the action is SET_CODE, we expect that the data is exactly the bytecode that
            // the user wants to set the code to.
            Lib_ExecutionManagerWrapper.ovmSETCODE(
                _action.target,
                _action.data
            );
        } else {
            // When the action is SET_STORAGE, we expect that the data is actually an ABI encoded
            // key/value pair. So we'll need to decode that first.
            (bytes32 key, bytes32 value) = abi.decode(
                _action.data,
                (bytes32, bytes32)
            );

            Lib_ExecutionManagerWrapper.ovmSETSTORAGE(
                _action.target,
                key,
                value
            );
        }

        // Mark the action as complete.
        completedBundleActions[currentBundleNonce][_proof.actionIndex] = true;

        emit ActionExecuted(
            currentBundleHash,
            currentBundleNonce,
            _proof.actionIndex
        );

        currentBundleTxsExecuted++;
        if (currentBundleSize == currentBundleTxsExecuted) {
            emit BundleCompleted(
                currentBundleHash,
                currentBundleNonce
            );

            currentBundleHash = bytes32(0);
            currentBundleSize = 0;
            currentBundleTxsExecuted = 0;
            _setUpgradeStatus(false);
        }
    }


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Sets the system status to "upgrading" or "done upgrading" depending on the boolean input.
     * @param _upgrading `true` sets status to "upgrading", `false` to "done upgrading."
     */
    function _setUpgradeStatus(
        bool _upgrading
    )
        internal
    {
        assembly {
            sstore(SLOT_KEY_IS_UPGRADING, _upgrading)
        }
    }
}
