// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Predeploys } from "../libraries/Predeploys.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";

/// @custom:legacy
/// @custom:proxied
/// @custom:predeploy 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000
/// @title LegacyERC20ETH
/// @notice LegacyERC20ETH is a legacy contract that held ETH balances before the Bedrock upgrade.
///         All ETH balances held within this contract were migrated to the state trie as part of
///         the Bedrock upgrade. Functions within this contract that mutate state were already
///         disabled as part of the EVM equivalence upgrade.
contract LegacyERC20ETH is OptimismMintableERC20 {
    /// @notice Initializes the contract as an Optimism Mintable ERC20.
    constructor()
        OptimismMintableERC20(Predeploys.L2_STANDARD_BRIDGE, address(0), "Ether", "ETH")
    {}

    /// @notice Returns the ETH balance of the target account. Overrides the base behavior of the
    ///         contract to preserve the invariant that the balance within this contract always
    ///         matches the balance in the state trie.
    /// @param _who Address of the account to query.
    /// @return The ETH balance of the target account.
    function balanceOf(address _who) public view virtual override returns (uint256) {
        return address(_who).balance;
    }

    /// @custom:blocked
    /// @notice Mints some amount of ETH.
    function mint(address, uint256) public virtual override {
        revert("LegacyERC20ETH: mint is disabled");
    }

    /// @custom:blocked
    /// @notice Burns some amount of ETH.
    function burn(address, uint256) public virtual override {
        revert("LegacyERC20ETH: burn is disabled");
    }

    /// @custom:blocked
    /// @notice Transfers some amount of ETH.
    function transfer(address, uint256) public virtual override returns (bool) {
        revert("LegacyERC20ETH: transfer is disabled");
    }

    /// @custom:blocked
    /// @notice Approves a spender to spend some amount of ETH.
    function approve(address, uint256) public virtual override returns (bool) {
        revert("LegacyERC20ETH: approve is disabled");
    }

    /// @custom:blocked
    /// @notice Transfers funds from some sender account.
    function transferFrom(
        address,
        address,
        uint256
    ) public virtual override returns (bool) {
        revert("LegacyERC20ETH: transferFrom is disabled");
    }

    /// @custom:blocked
    /// @notice Increases the allowance of a spender.
    function increaseAllowance(address, uint256) public virtual override returns (bool) {
        revert("LegacyERC20ETH: increaseAllowance is disabled");
    }

    /// @custom:blocked
    /// @notice Decreases the allowance of a spender.
    function decreaseAllowance(address, uint256) public virtual override returns (bool) {
        revert("LegacyERC20ETH: decreaseAllowance is disabled");
    }
}
