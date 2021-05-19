// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title OVM_GasPriceOracle
 * @dev This contract exposes the current congestion price, a measure of how congested the network
 * currently is. This measure is used by the Sequencer to determine what fee to charge for
 * transactions. When the system is more congested, the congestion price will increase and fees
 * will also increase as a result.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_GasPriceOracle is Ownable {

    /*************
     * Variables *
     *************/

    // Current execution price
    uint256 internal executionPrice;
    // Current batch overhead
    uint256 internal batchOverhead;
    // Current scalar value
    uint256 internal scalarValue;
    // Current scalar decimals
    uint256 internal scalarDecimals;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Address that will initially own this contract.
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
     * @return Current execution price.
     */
    function getExecutionPrice()
        public
        view
        returns (
            uint256
        )
    {
        return executionPrice;
    }

    /**
     * Allows the owner to modify the execution price.
     * @param _executionPrice New execution price.
     */
    function setExecutionPrice(
        uint256 _executionPrice
    )
        public
        onlyOwner
    {
        executionPrice = _executionPrice;
    }

    /**
     * @return Current batch overhead. Represents the gas
     * used to submit the batch on L1.
     */
    function getBatchOverhead()
        public
        view
        returns (
            uint256
        )
    {
        return batchOverhead;
    }

    /**
     * Allows the owner to modify the congestion price.
     * @param _batchOverhead New congestion price.
     */
    function setBatchOverhead(
        uint256 _batchOverhead
    )
        public
        onlyOwner
    {
        batchOverhead = _batchOverhead;
    }

    /**
     * @return Current scalar value
     */
    function getScalarValue()
        public
        view
        returns (
            uint256
        )
    {
        return scalarValue;
    }

    /**
     * Allows the owner to modify the scalar value.
     * @param _scalarValue New scalar value.
     */
    function setScalarValue(
        uint256 _scalarValue
    )
        public
        onlyOwner
    {
        scalarValue = _scalarValue;
    }

    /**
     * @return Current scalar decimals
     */
    function getScalarDecimals()
        public
        view
        returns (
            uint256
        )
    {
        return scalarDecimals;
    }

    /**
     * Allows the owner to modify the scalar decimals.
     * @param _scalarDecimals New scalar decimals.
     */
    function setScalarDecimals(
        uint256 _scalarDecimals
    )
        public
        onlyOwner
    {
        scalarDecimals = _scalarDecimals;
    }
}
