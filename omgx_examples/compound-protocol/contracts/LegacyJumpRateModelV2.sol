pragma solidity ^0.5.16;

import "./BaseJumpRateModelV2.sol";
import "./LegacyInterestRateModel.sol";


/**
  * @title Compound's JumpRateModel Contract V2 for legacy cTokens
  * @author Arr00
  * @notice Supports only legacy cTokens
  */
contract LegacyJumpRateModelV2 is LegacyInterestRateModel, BaseJumpRateModelV2  {

	/**
     * @notice Calculates the current borrow rate per block, with the error code expected by the market
     * @param cash The amount of cash in the market
     * @param borrows The amount of borrows in the market
     * @param reserves The amount of reserves in the market
     * @return (Error, The borrow rate percentage per block as a mantissa (scaled by 1e18))
     */
    function getBorrowRate(uint cash, uint borrows, uint reserves) external view returns (uint, uint) {
        return (0,getBorrowRateInternal(cash, borrows, reserves));
    }
    
    constructor(uint baseRatePerYear, uint multiplierPerYear, uint jumpMultiplierPerYear, uint kink_, address owner_) 
    	BaseJumpRateModelV2(baseRatePerYear,multiplierPerYear,jumpMultiplierPerYear,kink_,owner_) public {}
}
