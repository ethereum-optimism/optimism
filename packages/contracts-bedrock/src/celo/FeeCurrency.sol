// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

abstract contract FeeCurrency is ERC20 {
    modifier onlyVm() {
        require(msg.sender == address(0), "Only VM can call");
        _;
    }

    function debitGasFees(address from, uint256 value) external onlyVm {
        _burn(from, value);
    }

    // New function signature, will be used when all fee currencies have migrated
    function creditGasFees(address[] calldata recipients, uint256[] calldata amounts) public onlyVm {
        require(recipients.length == amounts.length, "Recipients and amounts must be the same length.");

        for (uint256 i = 0; i < recipients.length; i++) {
            _mint(recipients[i], amounts[i]);
        }
    }

    // Old function signature for backwards compatibility
    function creditGasFees(
        address from,
        address feeRecipient,
        address, // gatewayFeeRecipient, unused
        address communityFund,
        uint256 refund,
        uint256 tipTxFee,
        uint256, // gatewayFee, unused
        uint256 baseTxFee
    )
        public
        onlyVm
    {
        // Calling the new creditGasFees would make sense here, but that is not
        // possible due to its calldata arguments.
        _mint(from, refund);
        _mint(feeRecipient, tipTxFee);
        _mint(communityFund, baseTxFee);
    }
}
