// SPDX-License-Identifier: MIT
pragma solidity >=0.5.13 <0.9.0;

/**
 * @title Breaker Box Interface
 * @notice Defines the basic interface for the Breaker Box
 */
interface IBreakerBox {
    /**
     * @dev Used to keep track of the status of a breaker for a specific rate feed.
     *
     * - TradingMode: Represents the trading mode the breaker is in for a rate feed.
     *                This uses a bitmask approach, meaning each bit represents a
     *                different trading mode. The final trading mode of the rate feed
     *                is obtained by applying a logical OR operation to the TradingMode
     *                of all breakers associated with that rate feed. This allows multiple
     *                breakers to contribute to the final trading mode simultaneously.
     *                Possible values:
     *                0: bidirectional trading.
     *                1: inflow only.
     *                2: outflow only.
     *                3: trading halted.
     *
     * - LastUpdatedTime: Records the last time the breaker status was updated. This is
     *                    used to manage cooldown periods before the breaker can be reset.
     *
     * - Enabled:     Indicates whether the breaker is enabled for the associated rate feed.
     */
    struct BreakerStatus {
        uint8 tradingMode;
        uint64 lastUpdatedTime;
        bool enabled;
    }

    /**
     * @notice Emitted when a new breaker is added to the breaker box.
     * @param breaker The address of the breaker.
     */
    event BreakerAdded(address indexed breaker);

    /**
     * @notice Emitted when a breaker is removed from the breaker box.
     * @param breaker The address of the breaker.
     */
    event BreakerRemoved(address indexed breaker);

    /**
     * @notice Emitted when a breaker is tripped by a rate feed.
     * @param breaker The address of the breaker.
     * @param rateFeedID The address of the rate feed.
     */
    event BreakerTripped(address indexed breaker, address indexed rateFeedID);

    /**
     * @notice Emitted when a new rate feed is added to the breaker box.
     * @param rateFeedID The address of the rate feed.
     */
    event RateFeedAdded(address indexed rateFeedID);

    /**
     * @notice Emitted when dependencies for a rate feed are set.
     * @param rateFeedID The address of the rate feed.
     * @param dependencies The addresses of the dependendent rate feeds.
     */
    event RateFeedDependenciesSet(address indexed rateFeedID, address[] indexed dependencies);

    /**
     * @notice Emitted when a rate feed is removed from the breaker box.
     * @param rateFeedID The address of the rate feed.
     */
    event RateFeedRemoved(address indexed rateFeedID);

    /**
     * @notice Emitted when the trading mode for a rate feed is updated
     * @param rateFeedID The address of the rate feed.
     * @param tradingMode The new trading mode.
     */
    event TradingModeUpdated(address indexed rateFeedID, uint256 tradingMode);

    /**
     * @notice Emitted after a reset attempt is successful.
     * @param rateFeedID The address of the rate feed.
     * @param breaker The address of the breaker.
     */
    event ResetSuccessful(address indexed rateFeedID, address indexed breaker);

    /**
     * @notice  Emitted after a reset attempt fails when the
     *          rate feed fails the breakers reset criteria.
     * @param rateFeedID The address of the rate feed.
     * @param breaker The address of the breaker.
     */
    event ResetAttemptCriteriaFail(address indexed rateFeedID, address indexed breaker);

    /**
     * @notice Emitted after a reset attempt fails when cooldown time has not elapsed.
     * @param rateFeedID The address of the rate feed.
     * @param breaker The address of the breaker.
     */
    event ResetAttemptNotCool(address indexed rateFeedID, address indexed breaker);

    /**
     * @notice Emitted when the sortedOracles address is updated.
     * @param newSortedOracles The address of the new sortedOracles.
     */
    event SortedOraclesUpdated(address indexed newSortedOracles);

    /**
     * @notice Emitted when the breaker is enabled or disabled for a rate feed.
     * @param breaker The address of the breaker.
     * @param rateFeedID The address of the rate feed.
     * @param status Indicating the status.
     */
    event BreakerStatusUpdated(address breaker, address rateFeedID, bool status);

    /**
     * @notice Checks breakers for the rateFeedID and sets correct trading mode
     * if any breakers are tripped or need to be reset.
     * @param rateFeedID The address of the rate feed to run checks for.
     */
    function checkAndSetBreakers(address rateFeedID) external;

    /**
     * @notice Retrives an array of all breaker addresses.
     */
    function getBreakers() external view returns (address[] memory);

    /**
     * @notice Checks if a breaker with the specified address has been added to the breaker box.
     * @param breaker The address of the breaker to check;
     * @return A bool indicating whether or not the breaker has been added.
     */
    function isBreaker(address breaker) external view returns (bool);

    /**
     * @notice Gets the trading mode for the specified rateFeedID.
     * @param rateFeedID The address of the rate feed to retrieve the trading mode for.
     */
    function getRateFeedTradingMode(address rateFeedID) external view returns (uint8 tradingMode);
}
