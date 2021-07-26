pragma solidity ^0.5.16;

import "../../../contracts/Exponential.sol";
import "../../../contracts/InterestRateModel.sol";

contract InterestRateModelModel is InterestRateModel {
    uint borrowDummy;
    uint supplyDummy;

    function isInterestRateModel() external pure returns (bool) {
        return true;
    }

    function getBorrowRate(uint _cash, uint _borrows, uint _reserves) external view returns (uint) {
        return borrowDummy;
    }

    function getSupplyRate(uint _cash, uint _borrows, uint _reserves, uint _reserveFactorMantissa) external view returns (uint) {
        return supplyDummy;
    }
}
