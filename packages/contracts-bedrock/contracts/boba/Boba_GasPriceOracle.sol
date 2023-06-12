// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Predeploys } from "../libraries/Predeploys.sol";
import { SafeMath } from "@openzeppelin/contracts/utils/math/SafeMath.sol";

/* Contract Imports */
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { L2GovernanceERC20 } from "./L2GovernanceERC20.sol";
import { GasPriceOracle } from "../L2/GasPriceOracle.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

/* Contract Imports */
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/**
 * @title Boba_GasPriceOracle
 */
contract Boba_GasPriceOracle {
    using SafeERC20 for IERC20;
    using SafeMath for uint256;

    /*************
     * Constants *
     *************/

    // Minimum BOBA balance that can be withdrawn in a single withdrawal.
    uint256 public constant MIN_WITHDRAWAL_AMOUNT = 150e18;

    /*************
     * Variables *
     *************/

    // Owner address
    address private _owner;

    // Address that will hold the fees once withdrawn. Dynamically initialized within l2geth.
    address public feeWallet;

    // L2 Boba token address
    address public l2BobaAddress;

    // The maximum value of ETH and BOBA
    uint256 public maxPriceRatio = 5000;

    // The minimum value of ETH and BOBA
    uint256 public minPriceRatio = 500;

    // The price ratio of ETH and BOBA
    // This price ratio considers the saving percentage of using BOBA as the fee token
    uint256 public priceRatio;

    // Gas price oracle address
    address public gasPriceOracleAddress = 0x420000000000000000000000000000000000000F;

    // Record the wallet address that wants to use boba as fee token
    mapping(address => bool) public bobaFeeTokenUsers;

    // Boba fee for the meta transaction
    uint256 public metaTransactionFee = 3e18;

    // Received ETH amount for the swap - 0.005
    uint256 public receivedETHAmount = 5e15;

    // Price ratio without discount
    uint256 public marketPriceRatio;

    /*************
     *  Events   *
     *************/

    event TransferOwnership(address, address);
    event UseBobaAsFeeToken(address);
    event SwapBOBAForETHMetaTransaction(address);
    event UseETHAsFeeToken(address);
    event UpdatePriceRatio(address, uint256, uint256);
    event UpdateMaxPriceRatio(address, uint256);
    event UpdateMinPriceRatio(address, uint256);
    event UpdateGasPriceOracleAddress(address, address);
    event UpdateMetaTransactionFee(address, uint256);
    event UpdateReceivedETHAmount(address, uint256);
    event WithdrawBOBA(address, address);
    event WithdrawETH(address, address);

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyNotInitialized() {
        require(address(feeWallet) == address(0), "Contract has been initialized");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == _owner, "caller is not the owner");
        _;
    }

    /********************
     * Fall back Functions *
     ********************/

    /**
     * Receive ETH
     */
    receive() external payable {}

    /********************
     * Public Functions *
     ********************/

    /**
     * transfer ownership
     * @param _newOwner new owner address
     */
    function transferOwnership(address _newOwner) public onlyOwner {
        require(_newOwner != address(0), "Ownable: new owner is the zero address");
        address oldOwner = _owner;
        _owner = _newOwner;
        emit TransferOwnership(oldOwner, _newOwner);
    }

    /**
     * Returns the address of the current owner.
     */
    function owner() public view returns (address) {
        return _owner;
    }

    /**
     * Initialize feeWallet and l2BobaAddress.
     */
    function initialize(
        address payable _feeWallet,
        address _l2BobaAddress
    ) public onlyNotInitialized {
        require(_feeWallet != address(0) && _l2BobaAddress != address(0));
        feeWallet = _feeWallet;
        l2BobaAddress = _l2BobaAddress;

        // Initialize the parameters
        _owner = msg.sender;
        gasPriceOracleAddress = 0x420000000000000000000000000000000000000F;
        metaTransactionFee = 3e18;
        maxPriceRatio = 5000;
        priceRatio = 2000;
        minPriceRatio = 500;
        marketPriceRatio = 2000;
    }

    /**
     * Add the users that want to use BOBA as the fee token
     */
    function useBobaAsFeeToken() public {
        require(!Address.isContract(msg.sender), "Account not EOA");
        // Users should have more than 3 BOBA
        require(
            L2GovernanceERC20(l2BobaAddress).balanceOf(msg.sender) >= 3e18,
            "Insufficient Boba balance"
        );
        bobaFeeTokenUsers[msg.sender] = true;
        emit UseBobaAsFeeToken(msg.sender);
    }

    /**
     * Add the users that want to use BOBA as the fee token
     * using the Meta Transaction
     * NOTE: Only works for the mainnet and local testnet
     */
    function swapBOBAForETHMetaTransaction(
        address tokenOwner,
        address spender,
        uint256 value,
        uint256 deadline,
        uint8 v,
        bytes32 r,
        bytes32 s
    ) public {
        require(!Address.isContract(tokenOwner), "Account not EOA");
        require(spender == address(this), "Spender is not this contract");
        uint256 totalCost = receivedETHAmount.mul(marketPriceRatio).add(metaTransactionFee);
        require(value >= totalCost, "Value is not enough");
        L2GovernanceERC20 bobaToken = L2GovernanceERC20(l2BobaAddress);
        bobaToken.permit(tokenOwner, spender, value, deadline, v, r, s);
        IERC20(l2BobaAddress).safeTransferFrom(tokenOwner, address(this), totalCost);
        (bool sent, ) = address(tokenOwner).call{ value: receivedETHAmount }("");
        require(sent, "Failed to send ETH");
        emit SwapBOBAForETHMetaTransaction(tokenOwner);
    }

    /**
     * Add the users that want to use ETH as the fee token
     */
    function useETHAsFeeToken() public {
        require(!Address.isContract(msg.sender), "Account not EOA");
        // Users should have more than 0.002 ETH
        require(address(msg.sender).balance >= 2e15, "Insufficient ETH balance");
        bobaFeeTokenUsers[msg.sender] = false;
        emit UseETHAsFeeToken(msg.sender);
    }

    /**
     * Update the price ratio of ETH and BOBA
     * @param _priceRatio the price ratio of ETH and BOBA
     * @param _marketPriceRatio tha market price ratio of ETH and BOBA
     */
    function updatePriceRatio(uint256 _priceRatio, uint256 _marketPriceRatio) public onlyOwner {
        require(_priceRatio <= maxPriceRatio && _priceRatio >= minPriceRatio);
        require(_marketPriceRatio <= maxPriceRatio && _marketPriceRatio >= minPriceRatio);
        priceRatio = _priceRatio;
        marketPriceRatio = _marketPriceRatio;
        emit UpdatePriceRatio(owner(), _priceRatio, _marketPriceRatio);
    }

    /**
     * Update the maximum price ratio of ETH and BOBA
     * @param _maxPriceRatio the maximum price ratio of ETH and BOBA
     */
    function updateMaxPriceRatio(uint256 _maxPriceRatio) public onlyOwner {
        require(_maxPriceRatio >= minPriceRatio && _maxPriceRatio > 0);
        maxPriceRatio = _maxPriceRatio;
        emit UpdateMaxPriceRatio(owner(), _maxPriceRatio);
    }

    /**
     * Update the minimum price ratio of ETH and BOBA
     * @param _minPriceRatio the minimum price ratio of ETH and BOBA
     */
    function updateMinPriceRatio(uint256 _minPriceRatio) public onlyOwner {
        require(_minPriceRatio <= maxPriceRatio && _minPriceRatio > 0);
        minPriceRatio = _minPriceRatio;
        emit UpdateMinPriceRatio(owner(), _minPriceRatio);
    }

    /**
     * Update the gas oracle address
     * @param _gasPriceOracleAddress gas oracle address
     */
    function updateGasPriceOracleAddress(address _gasPriceOracleAddress) public onlyOwner {
        require(Address.isContract(_gasPriceOracleAddress), "Account is EOA");
        require(_gasPriceOracleAddress != address(0));
        gasPriceOracleAddress = _gasPriceOracleAddress;
        emit UpdateGasPriceOracleAddress(owner(), _gasPriceOracleAddress);
    }

    /**
     * Update the fee for the meta transaction
     * @param _metaTransactionFee the fee for the meta transaction
     */
    function updateMetaTransactionFee(uint256 _metaTransactionFee) public onlyOwner {
        require(_metaTransactionFee > 0);
        metaTransactionFee = _metaTransactionFee;
        emit UpdateMetaTransactionFee(owner(), _metaTransactionFee);
    }

    /**
     * Update the received ETH amount
     * @param _receivedETHAmount the received ETH amount
     */
    function updateReceivedETHAmount(uint256 _receivedETHAmount) public onlyOwner {
        require(_receivedETHAmount > 1e15 && _receivedETHAmount < 10e15);
        receivedETHAmount = _receivedETHAmount;
        emit UpdateReceivedETHAmount(owner(), _receivedETHAmount);
    }

    /**
     * Get the price for swapping BOBA for ETH
     */
    function getBOBAForSwap() public view returns (uint256) {
        return receivedETHAmount.mul(marketPriceRatio).add(metaTransactionFee);
    }

    /**
     * Get L1 Boba fee for fee estimation
     * @param _txData the data payload
     */
    function getL1BobaFee(bytes memory _txData) public view returns (uint256) {
        GasPriceOracle gasPriceOracleContract = GasPriceOracle(gasPriceOracleAddress);
        return gasPriceOracleContract.getL1Fee(_txData) * priceRatio;
    }

    /**
     * withdraw BOBA tokens to l1 fee wallet
     */
    function withdrawBOBA() public {
        require(
            L2GovernanceERC20(l2BobaAddress).balanceOf(address(this)) >= MIN_WITHDRAWAL_AMOUNT,
            // solhint-disable-next-line max-line-length
            "Boba_GasPriceOracle: withdrawal amount must be greater than minimum withdrawal amount"
        );

        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).withdrawTo(
            l2BobaAddress,
            feeWallet,
            L2GovernanceERC20(l2BobaAddress).balanceOf(address(this)),
            0,
            bytes("")
        );
        emit WithdrawBOBA(owner(), feeWallet);
    }

    /**
     * withdraw ETH tokens to l2 fee wallet
     */
    function withdrawETH() public onlyOwner {
        (bool sent, ) = feeWallet.call{ value: address(this).balance }("");
        require(sent, "Failed to send ETH to fee wallet");
        emit WithdrawETH(owner(), feeWallet);
    }
}
