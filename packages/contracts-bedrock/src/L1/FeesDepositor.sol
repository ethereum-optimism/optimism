// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ProxyAdminOwnedBase } from "src/L1/ProxyAdminOwnedBase.sol";
import { ReinitializableBase } from "src/universal/ReinitializableBase.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

/// @custom:proxied true
/// @title FeesDepositor
/// @notice A contract that deposits fees to the L2 recipient when the deposit threshold is reached.
contract FeesDepositor is ProxyAdminOwnedBase, Initializable, ReinitializableBase, ISemver {
    /// @notice The L1CrossDomainMessenger contract.
    IL1CrossDomainMessenger public messenger;

    /// @notice The threshold at which fees are deposited.
    uint96 public minDepositAmount;

    /// @notice The L2 recipient of the fees.
    address public l2Recipient;

    /// @notice The gas limit for the deposit transaction.
    uint32 public gasLimit;

    /// @notice Emitted when fees are received.
    /// @param sender The sender of the fees.
    /// @param amount The amount of fees received.
    /// @param newBalance The new balance after receiving fees.
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);

    /// @notice Emitted when fees are deposited.
    /// @param amount The amount of fees deposited.
    /// @param l2Recipient The L2 recipient of the fees.
    event FeesDeposited(address indexed l2Recipient, uint256 amount);

    /// @notice Emitted when the deposit threshold is updated.
    /// @param oldMinDepositAmount The old deposit threshold.
    /// @param newMinDepositAmount The new deposit threshold.
    event MinDepositAmountUpdated(uint96 oldMinDepositAmount, uint96 newMinDepositAmount);

    /// @notice Emitted when the L2 recipient is updated.
    /// @param oldL2Recipient The old L2 recipient.
    /// @param newL2Recipient The new L2 recipient.
    event L2RecipientUpdated(address oldL2Recipient, address newL2Recipient);

    /// @notice Emitted when the gas limit is updated.
    /// @param oldGasLimit The old gas limit.
    /// @param newGasLimit The new gas limit.
    event GasLimitUpdated(uint32 oldGasLimit, uint32 newGasLimit);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Constructs the FeesDepositor contract.
    constructor() ReinitializableBase(1) {
        _disableInitializers();
    }

    /// @notice Initializes the FeesDepositor contract.
    /// @param _minDepositAmount The threshold at which fees are deposited.
    /// @param _l2Recipient The L2 recipient of the fees.
    /// @param _messenger The L1CrossDomainMessenger contract.
    /// @param _gasLimit The gas limit for the deposit transaction.
    function initialize(
        uint96 _minDepositAmount,
        address _l2Recipient,
        IL1CrossDomainMessenger _messenger,
        uint32 _gasLimit
    )
        external
        reinitializer(initVersion())
    {
        // Initialization transactions must come from the ProxyAdmin or its owner.
        _assertOnlyProxyAdminOrProxyAdminOwner();

        messenger = _messenger;
        minDepositAmount = _minDepositAmount;
        l2Recipient = _l2Recipient;
        gasLimit = _gasLimit;
    }

    /// @notice Receives ETH and sends it to the L2 recipient via CrossDomainMessenger when the threshold is reached.
    receive() external payable {
        uint256 balance = address(this).balance;
        emit FundsReceived(msg.sender, msg.value, balance);

        if (balance >= minDepositAmount) {
            address recipient = l2Recipient;
            messenger.sendMessage{ value: balance }(recipient, hex"", gasLimit);
            emit FeesDeposited(recipient, balance);
        }
    }

    /// @notice Updates the deposit threshold.
    /// @param _newMinDepositAmount The new deposit threshold.
    function setMinDepositAmount(uint96 _newMinDepositAmount) external {
        _assertOnlyProxyAdminOwner();
        uint96 oldMinDepositAmount = minDepositAmount;
        minDepositAmount = _newMinDepositAmount;
        emit MinDepositAmountUpdated(oldMinDepositAmount, _newMinDepositAmount);
    }

    /// @notice Updates the L2 recipient for the deposit transaction.
    /// @dev The L2 recipient MUST be able to receive ether or the deposit on L2 will fail.
    /// @param _newL2Recipient The new L2 recipient.
    function setL2Recipient(address _newL2Recipient) external {
        _assertOnlyProxyAdminOwner();
        address oldL2Recipient = l2Recipient;
        l2Recipient = _newL2Recipient;
        emit L2RecipientUpdated(oldL2Recipient, _newL2Recipient);
    }

    /// @notice Updates the gas limit for the deposit transaction.
    /// @param _newGasLimit The new gas limit.
    function setGasLimit(uint32 _newGasLimit) external {
        _assertOnlyProxyAdminOwner();
        uint32 oldGasLimit = gasLimit;
        gasLimit = _newGasLimit;
        emit GasLimitUpdated(oldGasLimit, _newGasLimit);
    }
}
