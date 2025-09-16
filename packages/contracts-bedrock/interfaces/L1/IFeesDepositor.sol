// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";
import { IReinitializableBase } from "interfaces/universal/IReinitializableBase.sol";
import { IOptimismPortal2 as IOptimismPortal } from "interfaces/L1/IOptimismPortal2.sol";

interface IFeesDepositor is ISemver, IProxyAdminOwnedBase, IReinitializableBase {
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event FeesDeposited(address indexed l2Recipient, uint256 amount);
    event MinDepositAmountUpdated(uint96 oldminDepositAmount, uint96 newminDepositAmount);
    event L2RecipientUpdated(address oldL2Recipient, address newL2Recipient);
    event GasLimitUpdated(uint64 oldGasLimit, uint64 newGasLimit);
    event DepositDataUpdated(bytes oldDepositData, bytes newDepositData);

    function minDepositAmount() external view returns (uint256);
    function portal() external view returns (IOptimismPortal);
    function l2Recipient() external view returns (address);
    function gasLimit() external view returns (uint64);
    function depositData() external view returns (bytes memory);
    function initialize(
        uint96 _minDepositAmount,
        address _l2Recipient,
        IOptimismPortal _portal,
        uint64 _gasLimit,
        bytes memory _depositData
    )
        external;

    function setMinDepositAmount(uint96 _minDepositAmount) external;
    function setL2Recipient(address _l2Recipient) external;
    function setGasLimit(uint64 _gasLimit) external;
    function setDepositData(bytes memory _depositData) external;

    function __constructor__() external;
}
