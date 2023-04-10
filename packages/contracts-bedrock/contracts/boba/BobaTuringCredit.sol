// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import "@openzeppelin/contracts/utils/math/SafeMath.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";

/* Interface */
//import "@boba/turing-hybrid-compute/contracts/ITuringHelper.sol";
import "contracts/boba/ITuringHelper.sol";

/**
 * @title BobaTuringCredit
 * @dev The credit system for Boba Turing
 */
contract BobaTuringCredit {
    using SafeMath for uint256;
    using SafeERC20 for IERC20;

    /**********************
     * Contract Variables *
     **********************/
    address public owner;

    mapping(address => uint256) public prepaidBalance;

    address public turingToken;
    uint256 public turingPrice;
    uint256 public ownerRevenue;

    /********************
     *      Events      *
     ********************/

    event TransferOwnership(address oldOwner, address newOwner);

    event AddBalanceTo(address sender, uint256 balanceAmount, address helperContractAddress);

    event WithdrawRevenue(address sender, uint256 withdrawAmount);

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyNotInitialized() {
        require(address(turingToken) == address(0), "Contract has been initialized");
        _;
    }

    modifier onlyInitialized() {
        require(address(turingToken) != address(0), "Contract has not yet been initialized");
        _;
    }

    modifier onlyOwner() {
        require(msg.sender == owner || owner == address(0), "caller is not the owner");
        _;
    }

    /********************
     *    Constructor   *
     ********************/

    constructor(uint256 _turingPrice) {
        turingPrice = _turingPrice;
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * @dev Update turing token
     *
     * @param _turingToken credit token address
     */
    function updateTuringToken(address _turingToken) public onlyOwner onlyNotInitialized {
        turingToken = _turingToken;
    }

    /**
     * @dev transfer ownership
     *
     * @param _newOwner new owner address
     */
    function transferOwnership(address _newOwner) public onlyOwner {
        require(_newOwner != address(0));
        owner = _newOwner;
        emit TransferOwnership(msg.sender, _newOwner);
    }

    /**
     * @dev Update turing price
     *
     * @param _turingPrice turing price for each off-chain computation
     */
    function updateTuringPrice(uint256 _turingPrice) public onlyOwner {
        turingPrice = _turingPrice;
    }

    /**
     * @dev Add credit for a Turing helper contract
     *
     * @param _addBalanceAmount the prepaid amount that the user want to add
     * @param _helperContractAddress the address of the turing helper contract
     */
    function addBalanceTo(uint256 _addBalanceAmount, address _helperContractAddress)
        public
        onlyInitialized
    {
        require(_addBalanceAmount != 0, "Invalid amount");
        require(Address.isContract(_helperContractAddress), "Address is EOA");
        require(
            ERC165Checker.supportsInterface(_helperContractAddress, 0x2f7adf43),
            "Invalid Helper Contract"
        );

        prepaidBalance[_helperContractAddress] += _addBalanceAmount;

        emit AddBalanceTo(msg.sender, _addBalanceAmount, _helperContractAddress);

        // Transfer token to this contract
        IERC20(turingToken).safeTransferFrom(msg.sender, address(this), _addBalanceAmount);
    }

    /**
     * @dev Return the credit of a specific helper contract
     */
    function getCreditAmount(address _helperContractAddress) public view returns (uint256) {
        require(turingPrice != 0, "Unlimited credit");
        return prepaidBalance[_helperContractAddress].div(turingPrice);
    }

    /**
     * @dev Owner withdraws revenue
     *
     * @param _withdrawAmount the revenue amount that the owner wants to withdraw
     */
    function withdrawRevenue(uint256 _withdrawAmount) public onlyOwner onlyInitialized {
        require(_withdrawAmount <= ownerRevenue, "Invalid Amount");

        ownerRevenue -= _withdrawAmount;

        emit WithdrawRevenue(msg.sender, _withdrawAmount);

        IERC20(turingToken).safeTransfer(owner, _withdrawAmount);
    }
}
