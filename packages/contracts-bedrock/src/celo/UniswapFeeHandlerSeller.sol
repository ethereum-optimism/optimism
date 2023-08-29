// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../lib/openzeppelin-contracts/contracts/utils/math/Math.sol";
import "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import "../../lib/openzeppelin-contracts/contracts/utils/structs/EnumerableSet.sol";
import "../../lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

import "./UsingRegistry.sol";

import "./common/interfaces/IFeeHandlerSeller.sol";
import "./stability/interfaces/ISortedOracles.sol";
import "./common/FixidityLib.sol";
import "./common/Initializable.sol";
import "./FeeHandlerSeller.sol";

import "./uniswap/interfaces/IUniswapV2RouterMin.sol";
import "./uniswap/interfaces/IUniswapV2FactoryMin.sol";

// An implementation of FeeHandlerSeller supporting interfaces compatible with
// Uniswap V2 API
// See https://github.com/celo-org/celo-proposals/blob/master/CIPs/cip-0052.md
contract UniswapFeeHandlerSeller is FeeHandlerSeller {
    using FixidityLib for FixidityLib.Fraction;
    using EnumerableSet for EnumerableSet.AddressSet;

    uint256 constant MAX_TIMESTAMP_BLOCK_EXCHANGE = 20;
    uint256 constant MAX_NUMBER_ROUTERS_PER_TOKEN = 3;
    mapping(address => EnumerableSet.AddressSet) private routerAddresses;

    event ReceivedQuote(address indexed tokneAddress, address indexed router, uint256 quote);
    event RouterUsed(address router);
    event RouterAddressSet(address token, address router);
    event RouterAddressRemoved(address token, address router);

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

    /**
     * @notice Allows owner to set the router for a token.
     * @param token Address of the token to set.
     * @param router The new router.
     */
    function setRouter(address token, address router) external onlyOwner {
        _setRouter(token, router);
    }

    function _setRouter(address token, address router) private {
        require(router != address(0), "Router can't be address zero");
        routerAddresses[token].add(router);
        require(routerAddresses[token].values().length <= MAX_NUMBER_ROUTERS_PER_TOKEN, "Max number of routers reached");
        emit RouterAddressSet(token, router);
    }

    /**
     * @notice Allows owner to remove a router for a token.
     * @param token Address of the token.
     * @param router Address of the router to remove.
     */
    function removeRouter(address token, address router) external onlyOwner {
        routerAddresses[token].remove(router);
        emit RouterAddressRemoved(token, router);
    }

    /**
     * @notice Get the list of routers for a token.
     * @param token The address of the token to query.
     * @return An array of all the allowed router.
     */
    function getRoutersForToken(address token) external view returns (address[] memory) {
        return routerAddresses[token].values();
    }

    /**
     * @dev Calculates the minimum amount of tokens that can be received for a given amount of sell tokens,
     *       taking into account the slippage and the rates of the sell token and CELO token on the Uniswap V2 pair.
     * @param sellTokenAddress The address of the sell token.
     * @param maxSlippage The maximum slippage allowed.
     * @param amount The amount of sell tokens to be traded.
     * @param bestRouter The Uniswap V2 router with the best price.
     * @return The minimum amount of tokens that can be received.
     */
    function calculateAllMinAmount(
        address sellTokenAddress,
        uint256 maxSlippage,
        uint256 amount,
        IUniswapV2RouterMin bestRouter
    )
        private
        view
        returns (uint256)
    {
        ISortedOracles sortedOracles = getSortedOracles();
        uint256 minReports = minimumReports[sellTokenAddress];

        require(sortedOracles.numRates(sellTokenAddress) >= minReports, "Number of reports for token not enough");

        uint256 minimalSortedOracles = 0;
        // if minimumReports for this token is zero, assume the check is not needed
        if (minReports > 0) {
            (uint256 rateNumerator, uint256 rateDenominator) = sortedOracles.medianRate(sellTokenAddress);

            minimalSortedOracles = calculateMinAmount(rateNumerator, rateDenominator, amount, maxSlippage);
        }

        IERC20 celoToken = getGoldToken();
        address pair = IUniswapV2FactoryMin(bestRouter.factory()).getPair(sellTokenAddress, address(celoToken));
        uint256 minAmountPair =
            calculateMinAmount(IERC20(sellTokenAddress).balanceOf(pair), celoToken.balanceOf(pair), amount, maxSlippage);

        return Math.max(minAmountPair, minimalSortedOracles);
    }

    // This function explicitly defines few variables because it was getting error "stack too deep"
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

        require(routerAddresses[sellTokenAddress].values().length > 0, "routerAddresses should be non empty");

        // An improvement to this function would be to allow the user to pass a path as argument
        // and if it generates a better outcome that the ones enabled that gets used
        // and the user gets a reward

        IERC20 celoToken = getGoldToken();

        IUniswapV2RouterMin bestRouter;
        uint256 bestRouterQuote = 0;

        address[] memory path = new address[](2);

        path[0] = sellTokenAddress;
        path[1] = address(celoToken);

        for (uint256 i = 0; i < routerAddresses[sellTokenAddress].values().length; i++) {
            address poolAddress = routerAddresses[sellTokenAddress].at(i);
            IUniswapV2RouterMin router = IUniswapV2RouterMin(poolAddress);

            // Using the second return value becuase it's the last argument,
            // the previous values show how many tokens are exchanged in each path
            // so the first value would be equivalent to balanceToBurn
            uint256 wouldGet = router.getAmountsOut(amount, path)[1];

            emit ReceivedQuote(sellTokenAddress, poolAddress, wouldGet);
            if (wouldGet > bestRouterQuote) {
                bestRouterQuote = wouldGet;
                bestRouter = router;
            }
        }

        require(bestRouterQuote != 0, "Can't exchange with zero quote");

        uint256 minAmount = 0;
        minAmount = calculateAllMinAmount(sellTokenAddress, maxSlippage, amount, bestRouter);

        IERC20(sellTokenAddress).approve(address(bestRouter), amount);
        bestRouter.swapExactTokensForTokens(
            amount, minAmount, path, address(this), block.timestamp + MAX_TIMESTAMP_BLOCK_EXCHANGE
        );

        uint256 celoAmount = celoToken.balanceOf(address(this));
        celoToken.transfer(msg.sender, celoAmount);
        emit RouterUsed(address(bestRouter));
        emit TokenSold(sellTokenAddress, buyTokenAddress, amount);
        return celoAmount;
    }
}
