pragma solidity ^0.5.16;

import "../../contracts/ComptrollerG1.sol";
import "../../contracts/PriceOracle.sol";

// XXX we should delete G1 everything...
//  requires fork/deploy bytecode tests

contract ComptrollerScenarioG1 is ComptrollerG1 {
    uint public blockNumber;

    constructor() ComptrollerG1() public {}

    function membershipLength(CToken cToken) public view returns (uint) {
        return accountAssets[address(cToken)].length;
    }

    function fastForward(uint blocks) public returns (uint) {
        blockNumber += blocks;

        return blockNumber;
    }

    function setBlockNumber(uint number) public {
        blockNumber = number;
    }

    function _become(
        Unitroller unitroller,
        PriceOracle _oracle,
        uint _closeFactorMantissa,
        uint _maxAssets,
        bool reinitializing) public {
        super._become(unitroller, _oracle, _closeFactorMantissa, _maxAssets, reinitializing);
    }

    function getHypotheticalAccountLiquidity(
        address account,
        address cTokenModify,
        uint redeemTokens,
        uint borrowAmount) public view returns (uint, uint, uint) {
        (Error err, uint liquidity, uint shortfall) =
            super.getHypotheticalAccountLiquidityInternal(account, CToken(cTokenModify), redeemTokens, borrowAmount);
        return (uint(err), liquidity, shortfall);
    }

    function unlist(CToken cToken) public {
        markets[address(cToken)].isListed = false;
    }
}
