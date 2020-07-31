pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { ExecutionManager } from "./ExecutionManager.sol";
import { DataTypes } from "../utils/libraries/DataTypes.sol";
import { StateManager } from "./StateManager.sol";

/**
 * @title L2ExecutionManager
 * @notice This extension of ExecutionManager that should only run in L2 because it has optimistic
 *         execution details that are unnecessary and inefficient to run in L1.
 */
contract L2ExecutionManager is ExecutionManager {
    /*
     * Contract Variables
     */

    mapping(bytes32 => bytes32) private ovmHashToEvmHash;
    mapping(bytes32 => bytes32) private evmHashToOvmHash;
    mapping(bytes32 => bytes) private ovmHashToOvmTx;


    /*
     * Constructor
     */

    constructor(
        address _addressResolver,
        address _owner,
        DataTypes.GasMeterConfig memory _gasMeterConfig
    )
        public
        ExecutionManager(
            _addressResolver,
            _owner,
            _gasMeterConfig
        )
    {}


    /*
     * Public Functions
     */

    /**
     * Increments the provided address's nonce. This is only used by the
     * sequencer to correct nonces when transactions fail.
     * @param _addr The address of the nonce to increment.
     */
    function incrementNonce(
        address _addr
    )
        public
    {
        StateManager stateManager = resolveStateManager();
        stateManager.incrementOvmContractNonce(_addr);
    }

    /**
     * @notice Stores the provided OVM transaction, mapping its hash to its value and its hash to the EVM tx hash
            with which it's associated.
     * @param ovmTransactionHash The OVM transaction hash, used publicly as the reference to the transaction.
     * @param internalTransactionHash The internal transaction hash of the transaction actually executed.
     * @param signedOvmTx The signed OVM tx that we received
     */
    function storeOvmTransaction(
        bytes32 ovmTransactionHash,
        bytes32 internalTransactionHash,
        bytes memory signedOvmTx
    )
        public
    {
        evmHashToOvmHash[internalTransactionHash] = ovmTransactionHash;
        ovmHashToEvmHash[ovmTransactionHash] = internalTransactionHash;
        ovmHashToOvmTx[ovmTransactionHash] = signedOvmTx;
    }

    /**
     * @notice Gets the OVM transaction hash associated with the provided EVM transaction hash.
     * @param evmTransactionHash The EVM transaction hash.
     * @return The associated OVM transaction hash.
     */
    function getOvmTransactionHash(
        bytes32 evmTransactionHash
    )
        public
        view
        returns (bytes32)
    {
        return evmHashToOvmHash[evmTransactionHash];
    }

    /**
     * @notice Gets the EVM transaction hash associated with the provided OVM transaction hash.
     * @param ovmTransactionHash The OVM transaction hash.
     * @return The associated EVM transaction hash.
     */
    function getInternalTransactionHash(
        bytes32 ovmTransactionHash
    )
        public
        view
        returns (bytes32)
    {
        return ovmHashToEvmHash[ovmTransactionHash];
    }

    /**
     * @notice Gets the OVM transaction associated with the provided OVM transaction hash.
     * @param ovmTransactionHash The OVM transaction hash.
     * @return The associated signed OVM transaction.
     */
    function getOvmTransaction(
        bytes32 ovmTransactionHash
    )
        public
        view
        returns (bytes memory)
    {
        return ovmHashToOvmTx[ovmTransactionHash];
    }
}
