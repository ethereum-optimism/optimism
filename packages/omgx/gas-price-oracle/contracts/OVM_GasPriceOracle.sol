// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title OVM_GasPriceOracle
 * @dev This contract exposes the current l2 gas price, a measure of how congested the network
 * currently is. This measure is used by the Sequencer to determine what fee to charge for
 * transactions. When the system is more congested, the l2 gas price will increase and fees
 * will also increase as a result.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_GasPriceOracle is Ownable {

    /*************
     * Variables *
     *************/

    // Current l2 gas price
    uint256 public gasPrice;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _owner Address that will initially own this contract.
     */
    constructor(
        address _owner,
        uint256 _initialGasPrice
    )
        Ownable()
    {
        setGasPrice(_initialGasPrice);
        transferOwnership(_owner);
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Allows the owner to modify the l2 gas price.
     * @param _gasPrice New l2 gas price.
     */
    function setGasPrice(
        uint256 _gasPrice
    )
        public
        onlyOwner
    {
        gasPrice = _gasPrice;
    }
}
