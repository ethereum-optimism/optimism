// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { IReinitializableBase } from "interfaces/universal/IReinitializableBase.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";

interface IFeesDepositor is ISemver, IProxyAdminOwnedBase, IReinitializableBase {
    event Initialized(uint8 version);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event FeesDeposited(address indexed l2Recipient, uint256 amount);
    event MinDepositAmountUpdated(uint96 oldMinDepositAmount, uint96 newMinDepositAmount);
    event L2RecipientUpdated(address oldL2Recipient, address newL2Recipient);
    event GasLimitUpdated(uint32 oldGasLimit, uint32 newGasLimit);

    function minDepositAmount() external view returns (uint96);
    function messenger() external view returns (IL1CrossDomainMessenger);
    function l2Recipient() external view returns (address);
    function gasLimit() external view returns (uint32);
    function initialize(
        uint96 _minDepositAmount,
        address _l2Recipient,
        IL1CrossDomainMessenger _messenger,
        uint32 _gasLimit
    )
        external;

    function setMinDepositAmount(uint96 _newMinDepositAmount) external;
    function setL2Recipient(address _newL2Recipient) external;
    function setGasLimit(uint32 _newGasLimit) external;

    receive() external payable;

    function __constructor__() external;
}
