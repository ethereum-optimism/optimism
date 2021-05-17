// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_ExecutionManagerWrapper } from "../optimistic-ethereum/libraries/wrappers/Lib_ExecutionManagerWrapper.sol";
import { Lib_MerkleTree } from "../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";

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

    event BundleCancelled(
        bytes32 indexed bundleHash,
        uint256 indexed bundleNonce,
        address indexed who
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


    /********************
     * Public Functions *
     ********************/

    /**
     * @return boolean, whether or not an upgrade is currently being executed.
     */
    function hasActiveBundle()
        public
        view
        returns (
            bool
        )
    {
        return (
            currentBundleHash != bytes32(0)
            && currentBundleTxsExecuted < currentBundleSize
        );
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
            hasActiveBundle() == false,
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
     * Allows the owner to cancel the current active upgrade bundle.
     */
    function cancelTransactionBundle()
        public
        onlyOwner
    {
        require(
            hasActiveBundle() == true,
            "ChugSplashDeployer: cannot cancel when there is no active bundle"
        );

        emit BundleCancelled(
            currentBundleHash,
            currentBundleNonce,
            msg.sender
        );

        currentBundleHash = bytes32(0);
        currentBundleSize = 0;
        currentBundleTxsExecuted = 0;
        _setUpgradeStatus(false);
    }

    /**
     * Allows the owner to cancel a transaction bundle and immediately approve a new one.
     * @param _bundleHash Root of the Merkle tree of actions in the new bundle.
     * @param _bundleSize Total number of elements in the new bundle.
     */
    function overrideTransactionBundle(
        bytes32 _bundleHash,
        uint256 _bundleSize
    )
        public
        onlyOwner
    {
        cancelTransactionBundle();
        approveTransactionBundle(_bundleHash, _bundleSize);
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
            hasActiveBundle() == true,
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
        // TODO: Requires system status work planned for Ben.
    }
}
