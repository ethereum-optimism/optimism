// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../lib/openzeppelin-contracts/contracts/utils/math/Math.sol";
import "../../lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import "../../lib/openzeppelin-contracts/contracts/utils/structs/EnumerableSet.sol";
import "../../lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

import "./UsingRegistry.sol";
import "./common/Freezable.sol";
import "./common/FixidityLib.sol";
import "./common/Initializable.sol";

import "./common/interfaces/IFeeHandler.sol";
import "./common/interfaces/IFeeHandlerSeller.sol";

// TODO move to IStableToken when it adds method getExchangeRegistryId
import "./interfaces/IStableTokenMento.sol";
import "./common/interfaces/ICeloVersionedContract.sol";
import "./common/interfaces/ICeloToken.sol";
import "./stability/interfaces/ISortedOracles.sol";

// Using the minimal required signatures in the interfaces so more contracts could be compatible
import { ReentrancyGuard } from "@openzeppelin/contracts/security/ReentrancyGuard.sol";

// An implementation of FeeHandler as described in CIP-52
// See https://github.com/celo-org/celo-proposals/blob/master/CIPs/cip-0052.md
contract FeeHandler is
    Ownable,
    Initializable,
    UsingRegistry,
    ICeloVersionedContract,
    Freezable,
    IFeeHandler,
    ReentrancyGuard
{
    using FixidityLib for FixidityLib.Fraction;
    using EnumerableSet for EnumerableSet.AddressSet;

    uint256 public constant FIXED1_UINT = 1000000000000000000000000; // TODO move to FIX and add check

    // Min units that can be burned
    uint256 public constant MIN_BURN = 200;

    // last day the daily limits were updated
    uint256 public lastLimitDay;

    FixidityLib.Fraction public burnFraction; // 80%

    address public feeBeneficiary;

    uint256 public celoToBeBurned;

    // This mapping can not be public because it contains a FixidityLib.Fraction
    // and that'd be only supported with experimental features in this
    // compiler version
    mapping(address => TokenState) private tokenStates;

    struct TokenState {
        address handler;
        FixidityLib.Fraction maxSlippage;
        // Max amounts that can be burned in a day for a token
        uint256 dailySellLimit;
        // Max amounts that can be burned today for a token
        uint256 currentDaySellLimit;
        uint256 toDistribute;
        // Historical amounts burned by this contract
        uint256 pastBurn;
    }

    EnumerableSet.AddressSet private activeTokens;

    event SoldAndBurnedToken(address token, uint256 value);
    event DailyLimitSet(address tokenAddress, uint256 newLimit);
    event DailyLimitHit(address token, uint256 burning);
    event MaxSlippageSet(address token, uint256 maxSlippage);
    event DailySellLimitUpdated(uint256 amount);
    event FeeBeneficiarySet(address newBeneficiary);
    event BurnFractionSet(uint256 fraction);
    event TokenAdded(address tokenAddress, address handlerAddress);
    event TokenRemoved(address tokenAddress);

    /**
     * @notice Sets initialized == true on implementation contracts.
     * @param test Set to true to skip implementation initialisation.
     */
    constructor(bool test) Initializable(test) { }

    /**
     * @notice Used in place of the constructor to allow the contract to be upgradable via proxy.
     */
    function initialize(
        address _registryAddress,
        address newFeeBeneficiary,
        uint256 newBurnFraction,
        address[] calldata tokens,
        address[] calldata handlers,
        uint256[] calldata newLimits,
        uint256[] calldata newMaxSlippages
    )
        external
        initializer
    {
        require(tokens.length == handlers.length, "handlers length should match tokens length");
        require(tokens.length == newLimits.length, "limits length should match tokens length");
        require(tokens.length == newMaxSlippages.length, "maxSlippage length should match tokens length");

        _transferOwnership(msg.sender);
        setRegistry(_registryAddress);
        _setFeeBeneficiary(newFeeBeneficiary);
        _setBurnFraction(newBurnFraction);

        for (uint256 i = 0; i < tokens.length; i++) {
            _addToken(tokens[i], handlers[i]);
            _setDailySellLimit(tokens[i], newLimits[i]);
            _setMaxSplippage(tokens[i], newMaxSlippages[i]);
        }
    }

    // Without this the contract cant receive Celo as native transfer
    receive() external payable { }

    /**
     * @dev Returns the handler address for the specified token.
     * @param tokenAddress The address of the token for which to return the handler.
     * @return The address of the handler contract for the specified token.
     */
    function getTokenHandler(address tokenAddress) external view returns (address) {
        return tokenStates[tokenAddress].handler;
    }

    /**
     * @dev Returns a boolean indicating whether the specified token is active or not.
     * @param tokenAddress The address of the token for which to retrieve the active status.
     * @return A boolean representing the active status of the specified token.
     */
    function getTokenActive(address tokenAddress) external view returns (bool) {
        return activeTokens.contains(tokenAddress);
    }

    /**
     * @dev Returns the maximum slippage percentage for the specified token.
     * @param tokenAddress The address of the token for which to retrieve the maximum
     *  slippage percentage.
     * @return The maximum slippage percentage as a uint256 value.
     */
    function getTokenMaxSlippage(address tokenAddress) external view returns (uint256) {
        return FixidityLib.unwrap(tokenStates[tokenAddress].maxSlippage);
    }

    /**
     * @dev Returns the daily burn limit for the specified token.
     * @param tokenAddress The address of the token for which to retrieve the daily burn limit.
     * @return The daily burn limit as a uint256 value.
     */
    function getTokenDailySellLimit(address tokenAddress) external view returns (uint256) {
        return tokenStates[tokenAddress].dailySellLimit;
    }

    /**
     * @dev Returns the current daily sell limit for the specified token.
     * @param tokenAddress The address of the token for which to retrieve the current daily limit.
     * @return The current daily limit as a uint256 value.
     */
    function getTokenCurrentDaySellLimit(address tokenAddress) external view returns (uint256) {
        return tokenStates[tokenAddress].currentDaySellLimit;
    }

    /**
     * @dev Returns the amount of tokens available to distribute for the specified token.
     * @param tokenAddress The address of the token for which to retrieve the amount of
     * tokens available to distribute.
     * @return The amount of tokens available to distribute as a uint256 value.
     */
    function getTokenToDistribute(address tokenAddress) external view returns (uint256) {
        return tokenStates[tokenAddress].toDistribute;
    }

    function getActiveTokens() public view returns (address[] memory) {
        return activeTokens.values();
    }

    /**
     * @dev Sets the fee beneficiary address to the specified address.
     * @param beneficiary The address to set as the fee beneficiary.
     */
    function setFeeBeneficiary(address beneficiary) external onlyOwner {
        return _setFeeBeneficiary(beneficiary);
    }

    function _setFeeBeneficiary(address beneficiary) private {
        feeBeneficiary = beneficiary;
        emit FeeBeneficiarySet(beneficiary);
    }

    /**
     * @dev Sets the burn fraction to the specified value.
     * @param fraction The value to set as the burn fraction.
     */
    function setBurnFraction(uint256 fraction) external onlyOwner {
        return _setBurnFraction(fraction);
    }

    function _setBurnFraction(uint256 newFraction) private {
        FixidityLib.Fraction memory fraction = FixidityLib.wrap(newFraction);
        require(FixidityLib.lte(fraction, FixidityLib.fixed1()), "Burn fraction must be less than or equal to 1");
        burnFraction = fraction;
        emit BurnFractionSet(newFraction);
    }

    /**
     * @dev Sets the burn fraction to the specified value. Token has to have a handler set.
     * @param tokenAddress The address of the token to sell
     */
    function sell(address tokenAddress) external {
        return _sell(tokenAddress);
    }

    /**
     * @dev Adds a new token to the contract with the specified token and handler addresses.
     * @param tokenAddress The address of the token to add.
     * @param handlerAddress The address of the handler contract for the specified token.
     */
    function addToken(address tokenAddress, address handlerAddress) external onlyOwner {
        _addToken(tokenAddress, handlerAddress);
    }

    function _addToken(address tokenAddress, address handlerAddress) private {
        require(handlerAddress != address(0), "Can't set handler to zero");
        TokenState storage tokenState = tokenStates[tokenAddress];
        tokenState.handler = handlerAddress;

        activeTokens.add(tokenAddress);
        emit TokenAdded(tokenAddress, handlerAddress);
    }

    /**
     * @notice Allows the owner to activate a specified token.
     * @param tokenAddress The address of the token to be activated.
     */
    function activateToken(address tokenAddress) external onlyOwner {
        _activateToken(tokenAddress);
    }

    function _activateToken(address tokenAddress) private {
        TokenState storage tokenState = tokenStates[tokenAddress];
        require(
            tokenState.handler != address(0) || tokenAddress == registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID),
            "Handler has to be set to activate token"
        );
        activeTokens.add(tokenAddress);
    }

    /**
     * @dev Deactivates the specified token by marking it as inactive.
     * @param tokenAddress The address of the token to deactivate.
     */
    function deactivateToken(address tokenAddress) external onlyOwner {
        _deactivateToken(tokenAddress);
    }

    function _deactivateToken(address tokenAddress) private {
        activeTokens.remove(tokenAddress);
    }

    /**
     * @notice Allows the owner to set a handler contract for a specified token.
     * @param tokenAddress The address of the token to set the handler for.
     * @param handlerAddress The address of the handler contract to be set.
     */
    function setHandler(address tokenAddress, address handlerAddress) external onlyOwner {
        _setHandler(tokenAddress, handlerAddress);
    }

    function _setHandler(address tokenAddress, address handlerAddress) private {
        require(handlerAddress != address(0), "Can't set handler to zero, use deactivateToken");
        TokenState storage tokenState = tokenStates[tokenAddress];
        tokenState.handler = handlerAddress;
    }

    function removeToken(address tokenAddress) external onlyOwner {
        _removeToken(tokenAddress);
    }

    function _removeToken(address tokenAddress) private {
        _deactivateToken(tokenAddress);
        TokenState storage tokenState = tokenStates[tokenAddress];
        tokenState.handler = address(0);
        emit TokenRemoved(tokenAddress);
    }

    function _sell(address tokenAddress) private onlyWhenNotFrozen nonReentrant {
        IERC20 token = IERC20(tokenAddress);

        TokenState storage tokenState = tokenStates[tokenAddress];
        require(tokenState.handler != address(0), "Handler has to be set to sell token");
        require(FixidityLib.unwrap(tokenState.maxSlippage) != 0, "Max slippage has to be set to sell token");
        FixidityLib.Fraction memory balanceToProcess =
            FixidityLib.newFixed(token.balanceOf(address(this)) - tokenState.toDistribute);

        uint256 balanceToBurn = (burnFraction.multiply(balanceToProcess).fromFixed());

        tokenState.toDistribute = tokenState.toDistribute + balanceToProcess.fromFixed() - balanceToBurn;

        // small numbers cause rounding errors and zero case should be skipped
        if (balanceToBurn < MIN_BURN) {
            return;
        }

        if (dailySellLimitHit(tokenAddress, balanceToBurn)) {
            // in case the limit is hit, burn the max possible
            balanceToBurn = tokenState.currentDaySellLimit;
            emit DailyLimitHit(tokenAddress, balanceToBurn);
        }

        token.transfer(tokenState.handler, balanceToBurn);
        IFeeHandlerSeller handler = IFeeHandlerSeller(tokenState.handler);

        uint256 celoReceived = handler.sell(
            tokenAddress,
            registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID),
            balanceToBurn,
            FixidityLib.unwrap(tokenState.maxSlippage)
        );

        celoToBeBurned = celoToBeBurned + celoReceived;
        tokenState.pastBurn = tokenState.pastBurn + balanceToBurn;
        updateLimits(tokenAddress, balanceToBurn);

        emit SoldAndBurnedToken(tokenAddress, balanceToBurn);
    }

    /**
     * @dev Distributes the available tokens for the specified token address to the fee beneficiary.
     * @param tokenAddress The address of the token for which to distribute the available tokens.
     */
    function distribute(address tokenAddress) external {
        return _distribute(tokenAddress);
    }

    function _distribute(address tokenAddress) private onlyWhenNotFrozen nonReentrant {
        require(feeBeneficiary != address(0), "Can't distribute to the zero address");
        IERC20 token = IERC20(tokenAddress);
        uint256 tokenBalance = token.balanceOf(address(this));

        TokenState storage tokenState = tokenStates[tokenAddress];
        require(
            tokenState.handler != address(0) || tokenAddress == registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID),
            "Handler has to be set to sell token"
        );

        // safty check to avoid a revert due balance
        uint256 balanceToDistribute = Math.min(tokenBalance, tokenState.toDistribute);

        if (balanceToDistribute == 0) {
            // don't distribute with zero balance
            return;
        }

        token.transfer(feeBeneficiary, balanceToDistribute);
        tokenState.toDistribute = tokenState.toDistribute - balanceToDistribute;
    }

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
     * @notice Allows owner to set max slippage for a token.
     * @param token Address of the token to set.
     * @param newMax New sllipage to set, as Fixidity fraction.
     */
    function setMaxSplippage(address token, uint256 newMax) external onlyOwner {
        _setMaxSplippage(token, newMax);
    }

    function _setMaxSplippage(address token, uint256 newMax) private {
        TokenState storage tokenState = tokenStates[token];
        require(newMax != 0, "Cannot set max slippage to zero");
        tokenState.maxSlippage = FixidityLib.wrap(newMax);
        require(
            FixidityLib.lte(tokenState.maxSlippage, FixidityLib.fixed1()), "Splippage must be less than or equal to 1"
        );
        emit MaxSlippageSet(token, newMax);
    }

    /**
     * @notice Allows owner to set the daily burn limit for a token.
     * @param token Address of the token to set.
     * @param newLimit The new limit to set, in the token units.
     */
    function setDailySellLimit(address token, uint256 newLimit) external onlyOwner {
        _setDailySellLimit(token, newLimit);
    }

    function _setDailySellLimit(address token, uint256 newLimit) private {
        TokenState storage tokenState = tokenStates[token];
        tokenState.dailySellLimit = newLimit;
        emit DailyLimitSet(token, newLimit);
    }

    /**
     * @dev Burns CELO tokens according to burnFraction.
     */
    function burnCelo() external {
        return _burnCelo();
    }

    /**
     * @dev Distributes the available tokens for all registered tokens to the feeBeneficiary.
     */
    function distributeAll() external {
        return _distributeAll();
    }

    function _distributeAll() private {
        for (uint256 i = 0; i < EnumerableSet.length(activeTokens); i++) {
            address token = activeTokens.at(i);
            _distribute(token);
        }
        // distribute Celo
        _distribute(registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID));
    }

    /**
     * @dev Distributes the available tokens for all registered tokens to the feeBeneficiary.
     */
    function handleAll() external {
        return _handleAll();
    }

    function _handleAll() private {
        for (uint256 i = 0; i < EnumerableSet.length(activeTokens); i++) {
            // calling _handle would trigger may burn Celo and distributions
            // that can be just batched at the end
            address token = activeTokens.at(i);
            _sell(token);
        }
        _distributeAll(); // distributes Celo as well
        _burnCelo();
    }

    /**
     * @dev Distributes the the token for to the feeBeneficiary.
     */
    function handle(address tokenAddress) external {
        return _handle(tokenAddress);
    }

    function _handle(address tokenAddress) private {
        // Celo doesn't have to be exchanged for anything
        if (tokenAddress != registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID)) {
            _sell(tokenAddress);
        }
        _burnCelo();
        _distribute(tokenAddress);
        _distribute(registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID));
    }

    /**
     * @notice Burns all the Celo balance of this contract.
     */
    function _burnCelo() private {
        TokenState storage tokenState = tokenStates[registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID)];
        ICeloToken celo = ICeloToken(registry.getAddressForOrDie(GOLD_TOKEN_REGISTRY_ID));

        uint256 balanceOfCelo = address(this).balance;

        uint256 balanceToProcess = balanceOfCelo - tokenState.toDistribute - celoToBeBurned;
        uint256 currentBalanceToBurn = FixidityLib.newFixed(balanceToProcess).multiply(burnFraction).fromFixed();
        uint256 totalBalanceToBurn = currentBalanceToBurn + celoToBeBurned;
        celo.burn(totalBalanceToBurn);

        celoToBeBurned = 0;
        tokenState.toDistribute = tokenState.toDistribute + balanceToProcess - currentBalanceToBurn;
    }

    /**
     * @param token The address of the token to query.
     * @return The amount burned for a token.
     */
    function getPastBurnForToken(address token) external view returns (uint256) {
        return tokenStates[token].pastBurn;
    }

    /**
     * @param token The address of the token to query.
     * @param amountToBurn The amount of the token to burn.
     * @return Returns true if burning amountToBurn would exceed the daily limit.
     */
    function dailySellLimitHit(address token, uint256 amountToBurn) public returns (bool) {
        TokenState storage tokenState = tokenStates[token];

        if (tokenState.dailySellLimit == 0) {
            // if no limit set, assume uncapped
            return false;
        }

        uint256 currentDay = block.timestamp / 1 days;
        // Pattern borrowed from Reserve.sol
        if (currentDay > lastLimitDay) {
            lastLimitDay = currentDay;
            tokenState.currentDaySellLimit = tokenState.dailySellLimit;
        }

        return amountToBurn >= tokenState.currentDaySellLimit;
    }

    /**
     * @notice Updates the current day limit for a token.
     * @param token The address of the token to query.
     * @param amountBurned the amount of the token that was burned.
     */
    function updateLimits(address token, uint256 amountBurned) private {
        TokenState storage tokenState = tokenStates[token];

        if (tokenState.dailySellLimit == 0) {
            // if no limit set, assume uncapped
            return;
        }
        tokenState.currentDaySellLimit = tokenState.currentDaySellLimit - amountBurned;
        emit DailySellLimitUpdated(amountBurned);
    }

    /**
     * @notice Allows owner to transfer tokens of this contract. It's meant for governance to
     *   trigger use cases not contemplated in this contract.
     *   @param token The address of the token to transfer.
     *   @param recipient The address of the recipient to transfer the tokens to.
     *   @param value The amount of tokens to transfer.
     *   @return A boolean indicating whether the transfer was successful or not.
     */
    function transfer(address token, address recipient, uint256 value) external onlyOwner returns (bool) {
        return IERC20(token).transfer(recipient, value);
    }
}
