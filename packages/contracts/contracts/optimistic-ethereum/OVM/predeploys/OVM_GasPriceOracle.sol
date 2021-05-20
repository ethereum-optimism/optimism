// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title OVM_GasPriceOracle
 * @dev This contract exposes the current execution price, a measure of how congested the network
 * currently is. This measure is used by the Sequencer to determine what fee to charge for
 * transactions. When the system is more congested, the execution price will increase and fees
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
}
