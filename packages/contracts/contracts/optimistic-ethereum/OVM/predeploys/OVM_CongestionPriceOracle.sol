// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* External Imports */
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title OVM_CongestionPriceOracle
 * @dev This contract exposes the current congestion price, a measure of how congested the network
 * currently is. This measure is used by the Sequencer to determine what fee to charge for
 * transactions. When the system is more congested, the congestion price will increase and fees
 * will also increase as a result.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_CongestionPriceOracle is Ownable {
    
    /*************
     * Variables *
     *************/
    
    // Current congestion price
    uint256 internal congestionPrice;
    

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
     * @return Current congestion price.
     */
    function getCongestionPrice()
        public
        view
        returns (
            uint256
        )
    {
        return congestionPrice;
    }

    /**
     * Allows the owner to modify the congestion price.
     * @param _congestionPrice New congestion price.
     */
    function setCongestionPrice(
        uint256 _congestionPrice
    )
        public
        onlyOwner
    {
        congestionPrice = _congestionPrice;
    }
}
