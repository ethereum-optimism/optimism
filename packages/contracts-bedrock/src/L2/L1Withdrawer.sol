// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title L1Withdrawer
/// @notice A contract that receives ETH and automatically initiates withdrawals to L1 when a
///         minimum balance threshold is reached. This contract is designed to be used as a
///         recipient for the FeeSplitter contract and is part of the revenue sharing standard contracts.
contract L1Withdrawer is ISemver {
    /// @notice Thrown when the caller is not the ProxyAdmin owner.
    error L1Withdrawer_OnlyProxyAdminOwner();

    /// @notice The minimum amount of ETH that must be accumulated before a withdrawal is initiated.
    uint256 public minWithdrawalAmount;

    /// @notice The L1 address that will receive the withdrawn ETH.
    address public recipient;

    /// @notice The L1 gas limit set when initiating withdrawals.
    /// @dev withdrawalGasLimit should be overestimated to account for expensive receive()
    uint32 public withdrawalGasLimit;

    /// @notice Emitted when a withdrawal to L1 is initiated.
    /// @param recipient The L1 address receiving the withdrawal.
    /// @param amount The amount of ETH being withdrawn.
    event WithdrawalInitiated(address indexed recipient, uint256 amount);

    /// @notice Emitted when the contract receives funds.
    /// @param sender The address that sent the funds.
    /// @param amount The amount of ETH received.
    /// @param newBalance The new balance after receiving funds.
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);

    /// @notice Emitted when the minimum withdrawal amount is updated.
    /// @param oldMinWithdrawalAmount The previous minimum withdrawal amount.
    /// @param newMinWithdrawalAmount The new minimum withdrawal amount.
    event MinWithdrawalAmountUpdated(uint256 oldMinWithdrawalAmount, uint256 newMinWithdrawalAmount);

    /// @notice Emitted when the recipient is updated.
    /// @param oldRecipient The previous recipient address.
    /// @param newRecipient The new recipient address.
    event RecipientUpdated(address oldRecipient, address newRecipient);

    /// @notice Emitted when the withdrawal gas limit is updated.
    /// @param oldWithdrawalGasLimit The previous withdrawal gas limit.
    /// @param newWithdrawalGasLimit The new withdrawal gas limit.
    event WithdrawalGasLimitUpdated(uint32 oldWithdrawalGasLimit, uint32 newWithdrawalGasLimit);

    /// @notice Semantic version.
    /// @custom:semver 1.0.1
    string public constant version = "1.0.1";

    /// @notice Constructs the L1Withdrawer contract.
    /// @param _minWithdrawalAmount The minimum amount of ETH required to trigger a withdrawal.
    /// @param _recipient The L1 address that will receive withdrawals.
    /// @param _withdrawalGasLimit The gas limit for the L1 withdrawal transaction.
    /// @dev If target on L1 is `FeesDepositor`, the gas limit should be at or above 800k gas.
    constructor(uint256 _minWithdrawalAmount, address _recipient, uint32 _withdrawalGasLimit) {
        minWithdrawalAmount = _minWithdrawalAmount;
        recipient = _recipient;
        withdrawalGasLimit = _withdrawalGasLimit;
    }

    /// @notice Receives ETH and initiates a withdrawal to L1 if the balance meets the threshold.
    receive() external payable {
        uint256 balance = address(this).balance;
        emit FundsReceived(msg.sender, msg.value, balance);

        if (balance >= minWithdrawalAmount) {
            IL2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage{ value: balance }(
                recipient, hex"", withdrawalGasLimit
            );

            emit WithdrawalInitiated(recipient, balance);
        }
    }

    /// @notice Updates the minimum withdrawal amount. Only callable by the ProxyAdmin owner.
    /// @param _newMinWithdrawalAmount The new minimum withdrawal amount.
    function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert L1Withdrawer_OnlyProxyAdminOwner();
        }
        uint256 oldMinWithdrawalAmount = minWithdrawalAmount;
        minWithdrawalAmount = _newMinWithdrawalAmount;
        emit MinWithdrawalAmountUpdated(oldMinWithdrawalAmount, _newMinWithdrawalAmount);
    }

    /// @notice Updates the recipient address. Only callable by the ProxyAdmin owner.
    /// @dev The recipient MUST be able to receive ether or L1Withdrawer#receive will fail
    /// when the withdrawal is finalized.
    /// @param _newRecipient The new recipient address.
    function setRecipient(address _newRecipient) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert L1Withdrawer_OnlyProxyAdminOwner();
        }
        address oldRecipient = recipient;
        recipient = _newRecipient;
        emit RecipientUpdated(oldRecipient, _newRecipient);
    }

    /// @notice Updates the withdrawal gas limit. Only callable by the ProxyAdmin owner.
    /// @param _newWithdrawalGasLimit The new withdrawal gas limit.
    /// @dev If target on L1 is `FeesDepositor`, the gas limit should be at or above 800k gas.
    function setWithdrawalGasLimit(uint32 _newWithdrawalGasLimit) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert L1Withdrawer_OnlyProxyAdminOwner();
        }
        uint32 oldWithdrawalGasLimit = withdrawalGasLimit;
        withdrawalGasLimit = _newWithdrawalGasLimit;
        emit WithdrawalGasLimitUpdated(oldWithdrawalGasLimit, _newWithdrawalGasLimit);
    }
}
