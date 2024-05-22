// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { ERC20PermitUpgradeable } from
    "@openzeppelin/contracts-upgradeable/token/ERC20/extensions/draft-ERC20PermitUpgradeable.sol";
import { ERC20Upgradeable } from "@openzeppelin/contracts-upgradeable/token/ERC20/ERC20Upgradeable.sol";
import { OwnableUpgradeable } from "@openzeppelin/contracts-upgradeable/access/OwnableUpgradeable.sol";

import { IStableTokenV2 } from "./interfaces/IStableToken.sol";
import { CalledByVm } from "./CalledByVm.sol";

/**
 * @title ERC20 token with minting and burning permissioned to a broker and validators.
 */
contract StableTokenV2 is IStableTokenV2, ERC20PermitUpgradeable, CalledByVm, OwnableUpgradeable {
    address public validators;
    address public broker;
    address public exchange;

    event TransferComment(string comment);
    event BrokerUpdated(address broker);
    event ValidatorsUpdated(address validators);
    event ExchangeUpdated(address exchange);

    /**
     * @dev Restricts a function so it can only be executed by an address that's allowed to mint.
     * Currently that's the broker, validators, or exchange.
     */
    modifier onlyMinter() {
        address sender = _msgSender();
        require(sender == broker || sender == validators || sender == exchange, "StableTokenV2: not allowed to mint");
        _;
    }

    /**
     * @dev Restricts a function so it can only be executed by an address that's allowed to burn.
     * Currently that's the broker or exchange.
     */
    modifier onlyBurner() {
        address sender = _msgSender();
        require(sender == broker || sender == exchange, "StableTokenV2: not allowed to burn");
        _;
    }

    /**
     * @notice The constructor for the StableTokenV2 contract.
     * @dev Should be called with disable=true in deployments when
     * it's accessed through a Proxy.
     * Call this with disable=false during testing, when used
     * without a proxy.
     * @param disable Set to true to run `_disableInitializers()` inherited from
     * openzeppelin-contracts-upgradeable/Initializable.sol
     */
    constructor(bool disable) {
        if (disable) {
            _disableInitializers();
        }
    }

    /**
     * @notice Initializes a StableTokenV2.
     * It keeps the same signature as the original initialize() function
     * in legacy/StableToken.sol
     * @param _name The name of the stable token (English)
     * @param _symbol A short symbol identifying the token (e.g. "cUSD")
     * @param initialBalanceAddresses Array of addresses with an initial balance.
     * @param initialBalanceValues Array of balance values corresponding to initialBalanceAddresses.
     * deprecated-param exchangeIdentifier String identifier of exchange in registry (for specific fiat pairs)
     */
    function initialize(
        // slither-disable-start shadowing-local
        string calldata _name,
        string calldata _symbol,
        // slither-disable-end shadowing-local
        address[] calldata initialBalanceAddresses,
        uint256[] calldata initialBalanceValues
    )
        external
        initializer
    {
        __ERC20_init_unchained(_name, _symbol);
        __ERC20Permit_init(_symbol);
        _transferOwnership(_msgSender());

        require(initialBalanceAddresses.length == initialBalanceValues.length, "Array length mismatch");
        for (uint256 i = 0; i < initialBalanceAddresses.length; i += 1) {
            _mint(initialBalanceAddresses[i], initialBalanceValues[i]);
        }
    }

    /**
     * @notice Initializes a StableTokenV2 contract
     * when upgrading from legacy/StableToken.sol.
     * It sets the addresses that were previously read from the Registry.
     * It runs the ERC20PermitUpgradeable initializer.
     * @dev This function is only callable once.
     * @param _broker The address of the Broker contract.
     * @param _validators The address of the Validators contract.
     * @param _exchange The address of the Exchange contract.
     */
    function initializeV2(
        address _broker,
        address _validators,
        address _exchange
    )
        external
        reinitializer(2)
        onlyOwner
    {
        _setBroker(_broker);
        _setValidators(_validators);
        _setExchange(_exchange);
        __ERC20Permit_init(symbol());
    }

    /**
     * @notice Sets the address of the Broker contract.
     * @dev This function is only callable by the owner.
     * @param _broker The address of the Broker contract.
     */
    function setBroker(address _broker) external onlyOwner {
        _setBroker(_broker);
    }

    /**
     * @notice Sets the address of the Validators contract.
     * @dev This function is only callable by the owner.
     * @param _validators The address of the Validators contract.
     */
    function setValidators(address _validators) external onlyOwner {
        _setValidators(_validators);
    }

    /**
     * @notice Sets the address of the Exchange contract.
     * @dev This function is only callable by the owner.
     * @param _exchange The address of the Exchange contract.
     */
    function setExchange(address _exchange) external onlyOwner {
        _setExchange(_exchange);
    }

    /**
     * @notice Transfer token for a specified address
     * @param to The address to transfer to.
     * @param value The amount to be transferred.
     * @param comment The transfer comment.
     * @return True if the transaction succeeds.
     */
    function transferWithComment(address to, uint256 value, string calldata comment) external returns (bool) {
        emit TransferComment(comment);
        return transfer(to, value);
    }

    /**
     * @notice Mints new StableToken and gives it to 'to'.
     * @param to The account for which to mint tokens.
     * @param value The amount of StableToken to mint.
     */
    function mint(address to, uint256 value) external onlyMinter returns (bool) {
        _mint(to, value);
        return true;
    }

    /**
     * @notice Burns StableToken from the balance of msg.sender.
     * @param value The amount of StableToken to burn.
     */
    function burn(uint256 value) external onlyBurner returns (bool) {
        _burn(msg.sender, value);
        return true;
    }

    /**
     * @notice Set the address of the Broker contract and emit an event
     * @param _broker The address of the Broker contract.
     */
    function _setBroker(address _broker) internal {
        broker = _broker;
        emit BrokerUpdated(_broker);
    }

    /**
     * @notice Set the address of the Validators contract and emit an event
     * @param _validators The address of the Validators contract.
     */
    function _setValidators(address _validators) internal {
        validators = _validators;
        emit ValidatorsUpdated(_validators);
    }

    /**
     * @notice Set the address of the Exchange contract and emit an event
     * @param _exchange The address of the Exchange contract.
     */
    function _setExchange(address _exchange) internal {
        exchange = _exchange;
        emit ExchangeUpdated(_exchange);
    }

    /// @inheritdoc ERC20Upgradeable
    function transferFrom(
        address from,
        address to,
        uint256 amount
    )
        public
        override(ERC20Upgradeable, IStableTokenV2)
        returns (bool)
    {
        return ERC20Upgradeable.transferFrom(from, to, amount);
    }

    /// @inheritdoc ERC20Upgradeable
    function transfer(address to, uint256 amount) public override(ERC20Upgradeable, IStableTokenV2) returns (bool) {
        return ERC20Upgradeable.transfer(to, amount);
    }

    /// @inheritdoc ERC20Upgradeable
    function balanceOf(address account) public view override(ERC20Upgradeable, IStableTokenV2) returns (uint256) {
        return ERC20Upgradeable.balanceOf(account);
    }

    /// @inheritdoc ERC20Upgradeable
    function approve(
        address spender,
        uint256 amount
    )
        public
        override(ERC20Upgradeable, IStableTokenV2)
        returns (bool)
    {
        return ERC20Upgradeable.approve(spender, amount);
    }

    /// @inheritdoc ERC20Upgradeable
    function allowance(
        address owner,
        address spender
    )
        public
        view
        override(ERC20Upgradeable, IStableTokenV2)
        returns (uint256)
    {
        return ERC20Upgradeable.allowance(owner, spender);
    }

    /// @inheritdoc ERC20Upgradeable
    function totalSupply() public view override(ERC20Upgradeable, IStableTokenV2) returns (uint256) {
        return ERC20Upgradeable.totalSupply();
    }

    /// @inheritdoc ERC20PermitUpgradeable
    function permit(
        address owner,
        address spender,
        uint256 value,
        uint256 deadline,
        uint8 v,
        bytes32 r,
        bytes32 s
    )
        public
        override(ERC20PermitUpgradeable, IStableTokenV2)
    {
        ERC20PermitUpgradeable.permit(owner, spender, value, deadline, v, r, s);
    }

    /**
     * @notice Reserve balance for making payments for gas in this StableToken currency.
     * @param from The account to reserve balance from
     * @param value The amount of balance to reserve
     * @dev Note that this function is called by the protocol when paying for tx fees in this
     * currency. After the tx is executed, gas is refunded to the sender and credited to the
     * various tx fee recipients via a call to `creditGasFees`.
     */
    function debitGasFees(address from, uint256 value) external onlyVm {
        _burn(from, value);
    }

    /**
     * @notice Alternative function to credit balance after making payments
     * for gas in this StableToken currency.
     * @param from The account to debit balance from
     * @param feeRecipient Coinbase address
     * @param gatewayFeeRecipient Gateway address
     * @param communityFund Community fund address
     * @param refund amount to be refunded by the VM
     * @param tipTxFee Coinbase fee
     * @param baseTxFee Community fund fee
     * @param gatewayFee Gateway fee
     * @dev Note that this function is called by the protocol when paying for tx fees in this
     * currency. Before the tx is executed, gas is debited from the sender via a call to
     * `debitGasFees`.
     */
    function creditGasFees(
        address from,
        address feeRecipient,
        address gatewayFeeRecipient,
        address communityFund,
        uint256 refund,
        uint256 tipTxFee,
        uint256 gatewayFee,
        uint256 baseTxFee
    )
        external
        onlyVm
    {
        // slither-disable-next-line uninitialized-local
        uint256 amountToBurn;
        _mint(from, refund + tipTxFee + gatewayFee + baseTxFee);

        if (feeRecipient != address(0)) {
            _transfer(from, feeRecipient, tipTxFee);
        } else if (tipTxFee > 0) {
            amountToBurn += tipTxFee;
        }

        if (gatewayFeeRecipient != address(0)) {
            _transfer(from, gatewayFeeRecipient, gatewayFee);
        } else if (gatewayFee > 0) {
            amountToBurn += gatewayFee;
        }

        if (communityFund != address(0)) {
            _transfer(from, communityFund, baseTxFee);
        } else if (baseTxFee > 0) {
            amountToBurn += baseTxFee;
        }

        if (amountToBurn > 0) {
            _burn(from, amountToBurn);
        }
    }
}
