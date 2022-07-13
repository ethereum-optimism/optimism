// SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";

/**
 * @custom:legacy
 * @custom:proxied
 * @custom:predeploy 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000
 * @title LegacyERC20ETH
 * @notice LegacyERC20ETH is a legacy contract that held ETH balances before the Bedrock upgrade.
 *         All ETH balances held within this contract were migrated to the state trie as part of
 *         the Bedrock upgrade. Functions within this contract that mutate state were already
 *         disabled as part of the EVM equivalence upgrade.
 */
contract LegacyERC20ETH is OptimismMintableERC20 {
    /**
     * @notice Initializes the contract as an Optimism Mintable ERC20.
     */
    constructor()
        OptimismMintableERC20(PredeployAddresses.L2_STANDARD_BRIDGE, address(0), "Ether", "ETH")
    {}

    /**
     * @notice Mints some amount of ETH.
     *
     * @param _to     Address of the recipient.
     * @param _amount Amount of ETH to mint.
     */
    function mint(address _to, uint256 _amount) public virtual override {
        revert("LegacyERC20ETH: mint is disabled");
    }

    /**
     * @notice Burns some amount of ETH.
     *
     * @param _from   Address to burn from.
     * @param _amount Amount of ETH to burn.
     */
    function burn(address _from, uint256 _amount) public virtual override {
        revert("LegacyERC20ETH: burn is disabled");
    }

    /**
     * @notice Transfers some amount of ETH.
     *
     * @param _recipient Address to send to.
     * @param _amount    Amount of ETH to send.
     */
    function transfer(address _recipient, uint256 _amount) public virtual override returns (bool) {
        revert("LegacyERC20ETH: transfer is disabled");
    }

    /**
     * @notice Approves a spender to spend some amount of ETH.
     *
     * @param _spender Address of the spender.
     * @param _amount  Amount of ETH to approve.
     */
    function approve(address _spender, uint256 _amount) public virtual override returns (bool) {
        revert("LegacyERC20ETH: approve is disabled");
    }

    /**
     * @notice Transfers funds from some sender account.
     *
     * @param _sender    Address of the sender.
     * @param _recipient Address of the recipient.
     * @param _amount    Amount of ETH to transfer.
     */
    function transferFrom(
        address _sender,
        address _recipient,
        uint256 _amount
    ) public virtual override returns (bool) {
        revert("LegacyERC20ETH: transferFrom is disabled");
    }

    /**
     * @notice Increases the allowance of a spender.
     *
     * @param _spender    Address of the spender.
     * @param _addedValue Amount of ETH to increase the allowance by.
     */
    function increaseAllowance(address _spender, uint256 _addedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("LegacyERC20ETH: increaseAllowance is disabled");
    }

    /**
     * @notice Decreases the allowance of a spender.
     *
     * @param _spender         Address of the spender.
     * @param _subtractedValue Amount of ETH to decrease the allowance by.
     */
    function decreaseAllowance(address _spender, uint256 _subtractedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("LegacyERC20ETH: decreaseAllowance is disabled");
    }
}
