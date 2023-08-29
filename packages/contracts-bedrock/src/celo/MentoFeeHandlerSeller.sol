// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import "../../lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

import "./interfaces/IStableTokenMento.sol";

import "./common/interfaces/IFeeHandlerSeller.sol";
import "./stability/interfaces/ISortedOracles.sol";
import "./common/FixidityLib.sol";
import "./common/Initializable.sol";

import "./FeeHandlerSeller.sol";

// An implementation of FeeHandlerSeller supporting interfaces compatible with
// Mento
// See https://github.com/celo-org/celo-proposals/blob/master/CIPs/cip-0052.md
contract MentoFeeHandlerSeller is FeeHandlerSeller {
    using FixidityLib for FixidityLib.Fraction;

    /**
     * @notice Sets initialized == true on implementation contracts.
     * @param test Set to true to skip implementation initialisation.
     */
    constructor(bool test) FeeHandlerSeller(test) { }

    // without this line the contract can't receive native Celo transfers
    receive() external payable { }

    /**
     * @notice Returns the storage, major, minor, and patch version of the contract.
     * @return Storage version of the contract.
     * @return Major version of the contract.
     * @return Minor version of the contract.
     * @return Patch version of the contract.
     */
    function getVersionNumber() external pure returns (uint256, uint256, uint256, uint256) {
        return (1, 1, 0, 0);
    }

    function sell(
        address sellTokenAddress,
        address buyTokenAddress,
        uint256 amount,
        uint256 maxSlippage // as fraction,
    )
        external
        returns (uint256)
    {
        require(
            buyTokenAddress == registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID), "Buy token can only be gold token"
        );

        IStableTokenMento stableToken = IStableTokenMento(sellTokenAddress);
        require(amount <= stableToken.balanceOf(address(this)), "Balance of token to burn not enough");

        address exchangeAddress = registry.getAddressForOrDie(stableToken.getExchangeRegistryId());

        IExchange exchange = IExchange(exchangeAddress);

        uint256 minAmount = 0;

        ISortedOracles sortedOracles = getSortedOracles();

        require(
            sortedOracles.numRates(sellTokenAddress) >= minimumReports[sellTokenAddress],
            "Number of reports for token not enough"
        );

        (uint256 rateNumerator, uint256 rateDenominator) = sortedOracles.medianRate(sellTokenAddress);
        minAmount = calculateMinAmount(rateNumerator, rateDenominator, amount, maxSlippage);

        // TODO an upgrade would be to compare using routers as well
        stableToken.approve(exchangeAddress, amount);
        exchange.sell(amount, minAmount, false);

        IERC20 goldToken = getGoldToken();
        uint256 celoAmount = goldToken.balanceOf(address(this));
        goldToken.transfer(msg.sender, celoAmount);

        emit TokenSold(sellTokenAddress, buyTokenAddress, amount);
        return celoAmount;
    }
}
