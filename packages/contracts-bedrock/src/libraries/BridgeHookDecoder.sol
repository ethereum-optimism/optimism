// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Bytes } from "src/libraries/Bytes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

/// @title BridgeHookDecoder
/// @notice Reads the messenger and standard bridge calls inside a bridge hook item. No function
///         reverts, since the Portal reads every item and a revert would block its deposits and
///         withdrawals. Input that does not decode returns as not valid. Only bounds are checked,
///         because a call is read only after its sender is checked to be the messenger or the
///         bridge. A later version must still accept every item already held, or a held token
///         transfer can never complete.
library BridgeHookDecoder {
    /// @notice Recognizes a token transfer between the standard bridges, ignoring the item's asset.
    /// @param _item                   Item to recognize.
    /// @param _l1CrossDomainMessenger Address of the L1CrossDomainMessenger.
    /// @param _l1StandardBridge       Address of the L1StandardBridge.
    /// @return isTokenTransfer_ Whether the item is a token transfer.
    /// @return message_         The bridge's call, or empty if not a transfer.
    function recognizeTokenTransfer(
        BridgeHookTypes.Item memory _item,
        address _l1CrossDomainMessenger,
        address _l1StandardBridge
    )
        internal
        pure
        returns (bool isTokenTransfer_, bytes memory message_)
    {
        // This is the only value check: the messenger always sets l2Value and its message's value
        // to ethAmount. A contract creation has target zero, so it fails the messenger check.
        if (_item.ethAmount != 0) return (false, message_);

        (,, bytes memory bridgeMessage) = _unwrap(_item, _l1CrossDomainMessenger, _l1StandardBridge);
        if (!_isFinalizeBridgeERC20(bridgeMessage)) return (false, message_);

        return (true, bridgeMessage);
    }

    /// @notice Resolves the item's initiator, on the source chain, and recipient, on the
    ///         destination chain. These are the bridge call's sender and recipient, or else the
    ///         sender and target of the innermost authenticated layer. Callers must not assume the
    ///         array has exactly 2 entries.
    /// @param _item                   Item to resolve.
    /// @param _l1CrossDomainMessenger Address of the L1CrossDomainMessenger.
    /// @param _l1StandardBridge       Address of the L1StandardBridge.
    /// @return parties_ The initiator and recipient, in that order.
    function resolveParties(
        BridgeHookTypes.Item memory _item,
        address _l1CrossDomainMessenger,
        address _l1StandardBridge
    )
        internal
        pure
        returns (address[] memory parties_)
    {
        (address initiator, address recipient, bytes memory bridgeMessage) =
            _unwrap(_item, _l1CrossDomainMessenger, _l1StandardBridge);

        (bool isERC20,,, address erc20From, address erc20To,) = _decodeFinalizeBridgeERC20(bridgeMessage);
        if (isERC20) {
            (initiator, recipient) = (erc20From, erc20To);
        } else {
            (bool isETH, address ethFrom, address ethTo) = _decodeFinalizeBridgeETH(bridgeMessage);
            if (isETH) (initiator, recipient) = (ethFrom, ethTo);
        }

        parties_ = new address[](2);
        parties_[0] = initiator;
        parties_[1] = recipient;
    }

    /// @notice Extracts the bridge's message from a token transfer item. Skips the route checks,
    ///         because the Portal sets the asset only after it recognizes the transfer.
    ///         decodeTokenTransfer checks the selector.
    /// @param _item Item to extract from.
    /// @return valid_   Whether the item's asset is ERC20 and its messenger call decodes.
    /// @return message_ The bridge's message, or empty if not valid.
    function extractTokenMessage(BridgeHookTypes.Item memory _item)
        internal
        pure
        returns (bool valid_, bytes memory message_)
    {
        if (_item.asset != BridgeHookTypes.Asset.ERC20) return (false, message_);
        (valid_,,, message_) = _decodeRelayMessage(_item.data);
    }

    /// @notice Decodes the tokens and amount of a finalizeBridgeERC20 call. Checks no sender, so
    ///         use it only on a message from recognizeTokenTransfer or extractTokenMessage.
    /// @param _direction Direction of the item the message came from.
    /// @param _message   The bridge's finalizeBridgeERC20 call.
    /// @return valid_   Whether the message is a finalizeBridgeERC20 call.
    /// @return l1Token_ Address of the L1 token.
    /// @return l2Token_ Address of the L2 token.
    /// @return amount_  Amount of the token transferred.
    function decodeTokenTransfer(
        BridgeHookTypes.Direction _direction,
        bytes memory _message
    )
        internal
        pure
        returns (bool valid_, address l1Token_, address l2Token_, uint256 amount_)
    {
        (bool valid, address localToken, address remoteToken,,, uint256 amount) = _decodeFinalizeBridgeERC20(_message);
        if (!valid) return (false, l1Token_, l2Token_, amount_);

        // The bridge call runs on the destination chain, so for a deposit its local token is the
        // L2 token.
        if (_direction == BridgeHookTypes.Direction.Deposit) return (true, remoteToken, localToken, amount);
        return (true, localToken, remoteToken, amount);
    }

    /// @notice Unwraps the item's messenger and bridge layers while each one is authenticated, and
    ///         returns the sender and target of the innermost authenticated layer.
    /// @param _item                   Item to unwrap.
    /// @param _l1CrossDomainMessenger Address of the L1CrossDomainMessenger.
    /// @param _l1StandardBridge       Address of the L1StandardBridge.
    /// @return sender_        Sender of the innermost authenticated layer.
    /// @return target_        Target of the innermost authenticated layer.
    /// @return bridgeMessage_ The bridge's call if the item reached the bridge, otherwise empty.
    function _unwrap(
        BridgeHookTypes.Item memory _item,
        address _l1CrossDomainMessenger,
        address _l1StandardBridge
    )
        private
        pure
        returns (address sender_, address target_, bytes memory bridgeMessage_)
    {
        sender_ = _item.aliased ? AddressAliasHelper.undoL1ToL2Alias(_item.from) : _item.from;
        target_ = _item.to;

        // The L1 messenger is a contract, so its deposits are always aliased.
        if (_item.direction == BridgeHookTypes.Direction.Deposit && !_item.aliased) {
            return (sender_, target_, bridgeMessage_);
        }
        if (!_isCallToOther(
                _item.direction, sender_, target_, _l1CrossDomainMessenger, Predeploys.L2_CROSS_DOMAIN_MESSENGER
            )) {
            return (sender_, target_, bridgeMessage_);
        }

        (bool valid, address sender, address target, bytes memory message) = _decodeRelayMessage(_item.data);
        if (!valid) return (sender_, target_, bridgeMessage_);
        (sender_, target_) = (sender, target);

        if (!_isCallToOther(_item.direction, sender_, target_, _l1StandardBridge, Predeploys.L2_STANDARD_BRIDGE)) {
            return (sender_, target_, bridgeMessage_);
        }
        bridgeMessage_ = message;
    }

    /// @notice Whether the sender and target are the given L1 contract and its L2 counterpart, in
    ///         the item's direction.
    /// @param _direction  Direction of the item.
    /// @param _sender     Sender of the call.
    /// @param _target     Target of the call.
    /// @param _l1Contract Address of the L1 contract.
    /// @param _l2Contract Address of the L2 contract.
    /// @return Whether the call is from one contract to the other.
    function _isCallToOther(
        BridgeHookTypes.Direction _direction,
        address _sender,
        address _target,
        address _l1Contract,
        address _l2Contract
    )
        private
        pure
        returns (bool)
    {
        if (_direction == BridgeHookTypes.Direction.Deposit) return _sender == _l1Contract && _target == _l2Contract;
        return _sender == _l2Contract && _target == _l1Contract;
    }

    /// @notice Decodes the sender, target and message of a relayMessage call.
    /// @param _data Call data to decode.
    /// @return valid_   Whether the data is a relayMessage call with an in-bounds message.
    /// @return sender_  Sender of the message.
    /// @return target_  Target of the message.
    /// @return message_ Copy of the message.
    function _decodeRelayMessage(bytes memory _data)
        private
        pure
        returns (bool valid_, address sender_, address target_, bytes memory message_)
    {
        if (!_hasCallHead(_data, ICrossDomainMessenger.relayMessage.selector, 6)) {
            return (false, sender_, target_, message_);
        }

        // No subtraction underflows: the head check makes argsLength at least 192, and each check
        // keeps the next subtraction in range.
        uint256 argsLength = _data.length - 4;
        uint256 offset = _headWord(_data, 5);
        if (offset > argsLength - 32) return (false, sender_, target_, message_);
        uint256 length = _wordAt(_data, 4 + offset);
        if (length > argsLength - 32 - offset) return (false, sender_, target_, message_);

        return (true, _headAddress(_data, 1), _headAddress(_data, 2), Bytes.slice(_data, 36 + offset, length));
    }

    /// @notice Whether the data is a finalizeBridgeERC20 call.
    /// @param _data Call data to check.
    /// @return Whether the data is a finalizeBridgeERC20 call.
    function _isFinalizeBridgeERC20(bytes memory _data) private pure returns (bool) {
        return _hasCallHead(_data, IStandardBridge.finalizeBridgeERC20.selector, 6);
    }

    /// @notice Decodes the tokens, sender, recipient and amount of a finalizeBridgeERC20 call.
    /// @param _data Call data to decode.
    /// @return valid_       Whether the data is a finalizeBridgeERC20 call.
    /// @return localToken_  Token on the chain the call runs on.
    /// @return remoteToken_ Token on the other chain.
    /// @return from_        Sender of the transfer.
    /// @return to_          Recipient of the transfer.
    /// @return amount_      Amount of the token transferred.
    function _decodeFinalizeBridgeERC20(bytes memory _data)
        private
        pure
        returns (bool valid_, address localToken_, address remoteToken_, address from_, address to_, uint256 amount_)
    {
        if (!_isFinalizeBridgeERC20(_data)) return (false, localToken_, remoteToken_, from_, to_, amount_);
        return (
            true,
            _headAddress(_data, 0),
            _headAddress(_data, 1),
            _headAddress(_data, 2),
            _headAddress(_data, 3),
            _headWord(_data, 4)
        );
    }

    /// @notice Decodes the sender and recipient of a finalizeBridgeETH call.
    /// @param _data Call data to decode.
    /// @return valid_ Whether the data is a finalizeBridgeETH call.
    /// @return from_  Sender of the transfer.
    /// @return to_    Recipient of the transfer.
    function _decodeFinalizeBridgeETH(bytes memory _data)
        private
        pure
        returns (bool valid_, address from_, address to_)
    {
        if (!_hasCallHead(_data, IStandardBridge.finalizeBridgeETH.selector, 4)) return (false, from_, to_);
        return (true, _headAddress(_data, 0), _headAddress(_data, 1));
    }

    /// @notice Whether the data has the selector and at least the given number of head words.
    /// @param _data      Call data to check.
    /// @param _selector  Expected selector.
    /// @param _headWords Number of head words the call needs.
    /// @return Whether the data has the selector and head.
    function _hasCallHead(bytes memory _data, bytes4 _selector, uint256 _headWords) private pure returns (bool) {
        return _data.length >= 4 + 32 * _headWords && bytes4(bytes32(_wordAt(_data, 0))) == _selector;
    }

    /// @notice Reads the head word at the index as an address.
    /// @param _data  Call data to read.
    /// @param _index Index of the head word.
    /// @return The head word's low 20 bytes.
    function _headAddress(bytes memory _data, uint256 _index) private pure returns (address) {
        return address(uint160(_headWord(_data, _index)));
    }

    /// @notice Reads the head word at the index.
    /// @param _data  Call data to read.
    /// @param _index Index of the head word.
    /// @return The head word.
    function _headWord(bytes memory _data, uint256 _index) private pure returns (uint256) {
        return _wordAt(_data, 4 + 32 * _index);
    }

    /// @notice Reads the word at the byte offset into the data. The caller checks it is in bounds.
    /// @param _data   Data to read.
    /// @param _offset Byte offset into the data.
    /// @return word_ The word.
    function _wordAt(bytes memory _data, uint256 _offset) private pure returns (uint256 word_) {
        assembly ("memory-safe") {
            word_ := mload(add(add(_data, 0x20), _offset))
        }
    }
}
