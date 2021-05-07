// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Library Imports */
import { Lib_ExecutionManagerWrapper } from "../optimistic-ethereum/libraries/wrappers/Lib_ExecutionManagerWrapper.sol";
import { Lib_MerkleTree } from "../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";

/**
 * @title L2ChugSplashDeployer
 */
contract L2ChugSplashDeployer {
    
    /*********
     * Enums *
     *********/

    enum ActionType {
        SET_CODE,
        SET_STORAGE
    }


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

    // Address that can approve new transaction bundles.
    address public owner;
    uint256 public currentBundleNonce;
    bytes32 public currentBundleHash;
    uint256 public currentBundleSize;
    uint256 public currentBundleTxsExecuted;
    mapping (uint256 => mapping (uint256 => bool)) internal completedBundleActions;


    /***************
     * Constructor *
     ***************/
    
    /**
     * @param _owner Address that will initially own the L2ChugSplashDeployer.
     */
    constructor(
        address _owner
    ) {
        owner = _owner;
    }


    /**********************
     * Function Modifiers *
     **********************/

    /**
     * Marks a function as only callable by the owner.
     */
    modifier onlyOwner() {
        require(
            msg.sender == owner,
            "ChugSplashDeployer: sender is not owner"
        );
        _;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Changes the owner. Only callable by the current owner.
     * @param _owner New owner address.
     */
    function setOwner(
        address _owner
    )
        public
        onlyOwner
    {
        owner = _owner;
    }

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
            hasActiveBundle() == false,
            "ChugSplashDeployer: previous bundle has not yet been fully executed"
        );

        currentBundleNonce += 1;
        currentBundleHash = _bundleHash;
        currentBundleSize = _bundleSize;
        currentBundleTxsExecuted = 0;
        _setUpgradeStatus(true);
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

        currentBundleHash = bytes32(0);
        currentBundleSize = 0;
        currentBundleTxsExecuted = 0;
        _setUpgradeStatus(false);
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
        // TODO: Do we need to check gas limit?

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

        currentBundleTxsExecuted++;
        if (currentBundleSize == currentBundleTxsExecuted) {
            currentBundleHash = bytes32(0);
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
