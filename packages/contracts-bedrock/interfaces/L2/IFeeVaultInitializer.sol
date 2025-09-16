// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Types } from "src/libraries/Types.sol";

interface IFeeVaultInitializer {
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

    function version() external view returns (string memory);
}