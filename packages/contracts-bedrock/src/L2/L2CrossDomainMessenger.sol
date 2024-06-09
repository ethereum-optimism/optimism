// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { CrossDomainMessenger } from "src/universal/CrossDomainMessenger.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
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
    function _xDomainRelayMessageValidationGas(uint64) internal pure override returns (uint64) {
        return RELAY_MESSAGE_VALIDATOR_CONFIG_NOOP_GAS + RELAY_MESSAGE_VALIDATOR_CALL_NOOOP_GAS;
    }

    /// @inheritdoc CrossDomainMessenger
    function _relayMessageValidatorConfig() internal view override returns (address) {
        return L1Block(Predeploys.L1_BLOCK_ATTRIBUTES).l2MessageValidator();
    }

    /// @inheritdoc CrossDomainMessenger
    function _isRelayMessageValidated(
        address _messageValidator,
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        bytes calldata _message
    )
        internal
        view
        override
        returns (bool)
    {
        // Early exit in case this a replay transaction OR the _messageValidator is the zero address (for extra sanity
        // and safety).
        // NOTE: the `_isOtherMessenger` check assumes the corresponding L1 message validator constrains forced
        // arbitrary execution
        // to go through the CrossDomainMessenger contracts.
        if (_messageValidator == address(0) || !_isOtherMessenger()) {
            return true;
        }
        // Perform the relay message validation call
        bytes memory callData = abi.encodeWithSelector(
            IL2MessageValidator(_messageValidator).validateMessage.selector, _nonce, _sender, _target, _value, _message
        );
        (bool success, bytes memory returnData) =
            _messageValidator.staticcall{ gas: RELAY_MESSAGE_VALIDATOR_CALL_GAS }(callData);
        // The static call must not have reverted and returned true for validation.
        return success && abi.decode(returnData, (bool));
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
