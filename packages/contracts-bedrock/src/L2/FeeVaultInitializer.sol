// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { BaseFeeVault } from "src/L2/BaseFeeVault.sol";
import { SequencerFeeVault } from "src/L2/SequencerFeeVault.sol";
import { L1FeeVault } from "src/L2/L1FeeVault.sol";
import { OperatorFeeVault } from "src/L2/OperatorFeeVault.sol";

// Libraries
import { Types } from "src/libraries/Types.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

/// @title FeeVaultInitializer
/// @notice This contract deploys new fee vault implementations with current configurations as immutables.
///         It reads the current configuration from existing fee vault proxies and deploys new implementations
///         with those values set as immutable parameters, ensuring consistent behavior across deployments.
contract FeeVaultInitializer is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Emitted when a fee vault implementation is deployed.
    /// @param vaultType The type of fee vault being deployed.
    /// @param newImplementation The deployed implementation address.
    /// @param recipient The recipient address for the implementation.
    /// @param network The withdrawal network for the implementation.
    /// @param minWithdrawalAmount The minimum withdrawal amount for the implementation.
    event FeeVaultDeployed(
        string indexed vaultType,
        address indexed newImplementation,
        address recipient,
        Types.WithdrawalNetwork network,
        uint256 minWithdrawalAmount
    );

    /// @notice Constructor that deploys new fee vault implementations with current values as immutables.
    constructor() {
        _deployBaseFeeVault();
        _deploySequencerFeeVault();
        _deployL1FeeVault();
        _deployOperatorFeeVault();
    }

    /// @notice Helper function to get current fee vault configuration.
    /// @param _feeVaultAddress The address of the fee vault to get configuration from.
    /// @return recipient_ The recipient address.
    /// @return network_ The withdrawal network.
    /// @return minWithdrawalAmount_ The minimum withdrawal amount.
    function _getFeeVaultConfig(address _feeVaultAddress)
        internal
        view
        returns (address recipient_, Types.WithdrawalNetwork network_, uint256 minWithdrawalAmount_)
    {
        // Make sure to use legacy functions to avoid failure on upgrade.
        recipient_ = IFeeVault(payable(_feeVaultAddress)).RECIPIENT();
        minWithdrawalAmount_ = IFeeVault(payable(_feeVaultAddress)).MIN_WITHDRAWAL_AMOUNT();
        // Use low level call to check for WITHDRAWAL_NETWORK, default to L2 if it doesn't exist
        (bool success, bytes memory data) =
            _feeVaultAddress.staticcall(abi.encodeCall(IFeeVault.WITHDRAWAL_NETWORK, ()));
        network_ =
            success && data.length >= 32 ? abi.decode(data, (Types.WithdrawalNetwork)) : Types.WithdrawalNetwork.L2;
    }

    /// @notice Deploys a new Base Fee Vault implementation with current configuration as immutables.
    ///         Reads the current configuration from the existing proxy and deploys a new implementation
    ///         with those values set as immutable parameters.
    function _deployBaseFeeVault() internal {
        (address recipient, Types.WithdrawalNetwork network, uint256 minWithdrawalAmount) =
            _getFeeVaultConfig(Predeploys.BASE_FEE_VAULT);

        // Deploy new implementation with current values as immutables
        BaseFeeVault newBaseFeeVault = new BaseFeeVault(recipient, minWithdrawalAmount, network);

        emit FeeVaultDeployed("BaseFeeVault", address(newBaseFeeVault), recipient, network, minWithdrawalAmount);
    }

    /// @notice Deploys a new Sequencer Fee Vault implementation with current configuration as immutables.
    ///         Reads the current configuration from the existing proxy and deploys a new implementation
    ///         with those values set as immutable parameters.
    function _deploySequencerFeeVault() internal {
        (address recipient, Types.WithdrawalNetwork network, uint256 minWithdrawalAmount) =
            _getFeeVaultConfig(Predeploys.SEQUENCER_FEE_WALLET);

        // Deploy new implementation with current values as immutables
        SequencerFeeVault newSequencerFeeVault = new SequencerFeeVault(recipient, minWithdrawalAmount, network);

        emit FeeVaultDeployed(
            "SequencerFeeVault", address(newSequencerFeeVault), recipient, network, minWithdrawalAmount
        );
    }

    /// @notice Deploys a new L1 Fee Vault implementation with current configuration as immutables.
    ///         Reads the current configuration from the existing proxy and deploys a new implementation
    ///         with those values set as immutable parameters.
    function _deployL1FeeVault() internal {
        (address recipient, Types.WithdrawalNetwork network, uint256 minWithdrawalAmount) =
            _getFeeVaultConfig(Predeploys.L1_FEE_VAULT);

        // Deploy new implementation with current values as immutables
        L1FeeVault newL1FeeVault = new L1FeeVault(recipient, minWithdrawalAmount, network);

        emit FeeVaultDeployed("L1FeeVault", address(newL1FeeVault), recipient, network, minWithdrawalAmount);
    }

    /// @notice Deploys a new Operator Fee Vault implementation with current configuration as immutables.
    ///         Reads the current configuration from the existing proxy and deploys a new implementation
    ///         with those values set as immutable parameters.
    function _deployOperatorFeeVault() internal {
        (address recipient, Types.WithdrawalNetwork network, uint256 minWithdrawalAmount) =
            _getFeeVaultConfig(Predeploys.OPERATOR_FEE_VAULT);

        // Deploy new implementation with current values as immutables
        OperatorFeeVault newOperatorFeeVault = new OperatorFeeVault(recipient, minWithdrawalAmount, network);

        emit FeeVaultDeployed("OperatorFeeVault", address(newOperatorFeeVault), recipient, network, minWithdrawalAmount);
    }
}
