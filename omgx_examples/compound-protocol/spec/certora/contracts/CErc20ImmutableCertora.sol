pragma solidity ^0.5.16;

import "../../../contracts/CErc20Immutable.sol";
import "../../../contracts/EIP20Interface.sol";

import "./CTokenCollateral.sol";

contract CErc20ImmutableCertora is CErc20Immutable {
    CTokenCollateral public otherToken;

    constructor(address underlying_,
                ComptrollerInterface comptroller_,
                InterestRateModel interestRateModel_,
                uint initialExchangeRateMantissa_,
                string memory name_,
                string memory symbol_,
                uint8 decimals_,
                address payable admin_) public CErc20Immutable(underlying_, comptroller_, interestRateModel_, initialExchangeRateMantissa_, name_, symbol_, decimals_, admin_) {
    }

    function balanceOfInOther(address account) public view returns (uint) {
        return otherToken.balanceOf(account);
    }

    function borrowBalanceStoredInOther(address account) public view returns (uint) {
        return otherToken.borrowBalanceStored(account);
    }

    function exchangeRateStoredInOther() public view returns (uint) {
        return otherToken.exchangeRateStored();
    }

    function getCashInOther() public view returns (uint) {
        return otherToken.getCash();
    }

    function getCashOf(address account) public view returns (uint) {
        return EIP20Interface(underlying).balanceOf(account);
    }

    function getCashOfInOther(address account) public view returns (uint) {
        return otherToken.getCashOf(account);
    }

    function totalSupplyInOther() public view returns (uint) {
        return otherToken.totalSupply();
    }

    function totalBorrowsInOther() public view returns (uint) {
        return otherToken.totalBorrows();
    }

    function totalReservesInOther() public view returns (uint) {
        return otherToken.totalReserves();
    }

    function underlyingInOther() public view returns (address) {
        return otherToken.underlying();
    }

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
