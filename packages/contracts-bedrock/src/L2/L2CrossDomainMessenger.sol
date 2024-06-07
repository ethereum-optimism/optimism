// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { IL2MessageValidator } from "src/L2/IL2MessageValidator.sol";
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { Constants } from "src/libraries/Constants.sol";
import { L1Block } from "src/L2/L1Block.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000007
/// @title L2CrossDomainMessenger
/// @notice The L2CrossDomainMessenger is a high-level interface for message passing between L1 and
///         L2 on the L2 side. Users are generally encouraged to use this contract instead of lower
///         level message passing contracts.
contract L2CrossDomainMessenger is CrossDomainMessenger, ISemver {
    /// @custom:semver 2.1.0
    string public constant version = "2.1.0";

    /// @notice Gas reserved for message validation within `passesDomainMessageValidator`
    ///         of `relayMessage` in the L2CrossDomainMessenger.
    uint64 public constant RELAY_MESSAGE_VALIDATOR_GAS = 25_500;

    /// @notice Constructs the L2CrossDomainMessenger contract.
    constructor() CrossDomainMessenger() {
        initialize({ _l1CrossDomainMessenger: CrossDomainMessenger(address(0)) });
    }

    /// @notice Initializer.
    /// @param _l1CrossDomainMessenger L1CrossDomainMessenger contract on the other network.
    function initialize(CrossDomainMessenger _l1CrossDomainMessenger) public initializer {
        __CrossDomainMessenger_init({ _otherMessenger: _l1CrossDomainMessenger });
    }

    /// @notice Getter for the remote messenger.
    ///         Public getter is legacy and will be removed in the future. Use `otherMessenger()` instead.
    /// @return L1CrossDomainMessenger contract.
    /// @custom:legacy
    function l1CrossDomainMessenger() public view returns (CrossDomainMessenger) {
        return otherMessenger;
    }

    /// @inheritdoc CrossDomainMessenger
    function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal override {
        L2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).initiateWithdrawal{ value: _value }(
            _to, _gasLimit, _data
        );
    }

    /// @inheritdoc CrossDomainMessenger
    function gasPayingToken() internal view override returns (address addr_, uint8 decimals_) {
        (addr_, decimals_) = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).gasPayingToken();
    }

    /// @inheritdoc CrossDomainMessenger
    function passesDomainMessageValidator(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    )
        internal
        view
        override
        returns (bool)
    {
        // TODO: Remove this comment 5_500 accounted for in this call.
        // Accounting for: Cold Sload (2100) + Cold Static (2600) = 4_700 + 800 (buffer extra logic)
        address l2MessageValidator = L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l2MessageValidator();

        if (l2MessageValidator == address(0)) {
            return true;
        }

        bytes memory callData = abi.encodeWithSelector(
            IL2MessageValidator(l2MessageValidator).validateMessage.selector,
            _nonce,
            _sender,
            _target,
            _value,
            _minGasLimit,
            _message
        );
        // TODO: Decide what's a reasonable gas limit for this call.
        // TODO: Remove comment - TOTAL: RELAY_MESSAGE_VALIDATOR_GAS: 5_500 + 20_000 = 25_500
        (bool success, bytes memory returnData) = l2MessageValidator.staticcall{ gas: 20_000 }(callData);

        return success && abi.decode(returnData, (bool));
    }

    /// @notice Gas reserved for message validation within `passesDomainMessageValidator`
    ///         of `relayMessage` in the L2CrossDomainMessenger.
    function _relayMessageValidationGas() internal view virtual override returns (uint64) {
        return RELAY_MESSAGE_VALIDATOR_GAS;
    }

    /// @inheritdoc CrossDomainMessenger
    function _isOtherMessenger() internal view override returns (bool) {
        return AddressAliasHelper.undoL1ToL2Alias(msg.sender) == address(otherMessenger);
    }

    /// @inheritdoc CrossDomainMessenger
    function _isUnsafeTarget(address _target) internal view override returns (bool) {
        return _target == address(this) || _target == address(Predeploys.L2_TO_L1_MESSAGE_PASSER);
    }
}
