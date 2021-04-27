// SPDX-License-Identifier: MIT
// @unsupported: evm
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_ExecutionManagerWrapper } from "../../optimistic-ethereum/libraries/wrappers/Lib_ExecutionManagerWrapper.sol";
import { Lib_MerkleTree } from "../../optimistic-ethereum/libraries/utils/Lib_MerkleTree.sol";

/**
 * @title ChugSplashDeployer
 */
contract ChugSplashDeployer {
    
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
        uint256 gasLimit;
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
    bytes32 public currentBundleHash;
    uint256 public currentBundleSize;
    uint256 public currentBundleTxsExecuted;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Initial owner address.
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

        currentBundleHash = _bundleHash;
        currentBundleSize = _bundleSize;
        currentBundleTxsExecuted = 0;

        // TODO: Set system status to "upgrading".
    }

    function executeAction(
        ChugSplashAction memory _action,
        ChugSplashActionProof memory _proof
    )
        public
    {
        // TODO: Do we need to validate enums or does solidity do it for us?

        require(
            hasActiveBundle() == true,
            "ChugSplashDeployer: there is no active bundle"
        );

        // Make sure the user has provided enough gas to perform this action successfully.
        require(
            gasleft() > _action.gasLimit,
            "ChugSplashDeployer: sender didn't supply enough gas"
        );

        // Make sure that the owner did actually sign off on this action.
        require(
            Lib_MerkleTree.verify(
                currentBundleHash,
                keccak256(
                    _action.actionType,
                    _action.gasLimit,
                    _action.target,
                    _action.data
                ),
                _proof.actionIndex,
                _proof.siblings,
                currentBundleSize
            ),
            "ChugSplashDeployer: invalid action proof"
        );

        if (_type == ActionType.SET_CODE) {
            Lib_ExecutionManagerWrapper.ovmSETCODE(_target, _data);
        } else {
            (bytes32 key, bytes32 val) = abi.decode(_data, (bytes32, bytes32));
            Lib_ExecutionManagerWrapper.ovmSETSTORAGE(_target, key, val);
        }

        currentBundleTxsExecuted++;
        if (currentBundleSize == currentBundleTxsExecuted) {
            currentBundleHash = bytes32(0);
            // TODO: Set system status to "done upgrading/active".
        }
    }
}
