pragma solidity ^0.5.16;

import "../../../contracts/CErc20Delegate.sol";
import "../../../contracts/EIP20Interface.sol";

import "./CTokenCollateral.sol";

contract CErc20DelegateCertora is CErc20Delegate {
    CTokenCollateral public otherToken;

    function mintFreshPub(address minter, uint mintAmount) public returns (uint) {
        (uint error,) = mintFresh(minter, mintAmount);
        return error;
    }

    function redeemFreshPub(address payable redeemer, uint redeemTokens, uint redeemUnderlying) public returns (uint) {
        return redeemFresh(redeemer, redeemTokens, redeemUnderlying);
    }

    function borrowFreshPub(address payable borrower, uint borrowAmount) public returns (uint) {
        return borrowFresh(borrower, borrowAmount);
    }

    function repayBorrowFreshPub(address payer, address borrower, uint repayAmount) public returns (uint) {
        (uint error,) = repayBorrowFresh(payer, borrower, repayAmount);
        return error;
    }

    function liquidateBorrowFreshPub(address liquidator, address borrower, uint repayAmount) public returns (uint) {
        (uint error,) = liquidateBorrowFresh(liquidator, borrower, repayAmount, otherToken);
        return error;
    }
}
