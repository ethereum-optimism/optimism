// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { BridgeHookDecoder } from "src/libraries/BridgeHookDecoder.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";

/// @title BridgeHookDecoder_AbiDecode_Harness
/// @notice Decodes calls with `abi.decode`, the reference for the `BridgeHookDecoder` library.
contract BridgeHookDecoder_AbiDecode_Harness {
    /// @notice Decodes the sender, target and message of a relayMessage call.
    function decodeRelayMessage(bytes calldata _data)
        external
        pure
        returns (address sender_, address target_, bytes memory message_)
    {
        (, sender_, target_,,, message_) = abi.decode(_data[4:], (uint256, address, address, uint256, uint256, bytes));
    }

    /// @notice Decodes the tokens and amount of a finalizeBridgeERC20 call.
    function decodeFinalizeBridgeERC20(bytes calldata _data)
        external
        pure
        returns (address localToken_, address remoteToken_, uint256 amount_)
    {
        (localToken_, remoteToken_,,, amount_,) =
            abi.decode(_data[4:], (address, address, address, address, uint256, bytes));
    }
}

/// @title BridgeHookDecoder_TestInit
/// @notice Reusable setup for the `BridgeHookDecoder` tests.
contract BridgeHookDecoder_TestInit is Test {
    address internal constant L1_MESSENGER = 0x1111111111111111111111111111111111111111;
    address internal constant L1_BRIDGE = 0x2222222222222222222222222222222222222222;

    BridgeHookDecoder_AbiDecode_Harness internal harness;

    function setUp() public {
        harness = new BridgeHookDecoder_AbiDecode_Harness();
    }

    /// @notice Returns the messenger on the source chain and the one on the destination chain.
    function _messengers(BridgeHookTypes.Direction _direction)
        internal
        pure
        returns (address source_, address destination_)
    {
        if (_direction == BridgeHookTypes.Direction.Deposit) {
            return (L1_MESSENGER, Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        }
        return (Predeploys.L2_CROSS_DOMAIN_MESSENGER, L1_MESSENGER);
    }

    /// @notice Returns the standard bridge on the source chain and the one on the destination chain.
    function _bridges(BridgeHookTypes.Direction _direction)
        internal
        pure
        returns (address source_, address destination_)
    {
        if (_direction == BridgeHookTypes.Direction.Deposit) return (L1_BRIDGE, Predeploys.L2_STANDARD_BRIDGE);
        return (Predeploys.L2_STANDARD_BRIDGE, L1_BRIDGE);
    }

    /// @notice Returns a relayMessage call from the sender to the target, with the message.
    function _envelope(address _sender, address _target, bytes memory _message) internal pure returns (bytes memory) {
        return abi.encodeCall(ICrossDomainMessenger.relayMessage, (0, _sender, _target, 0, 0, _message));
    }

    /// @notice Returns a finalizeBridgeERC20 call with every field zero.
    function _tokenMessage() internal pure returns (bytes memory) {
        return
            abi.encodeCall(IStandardBridge.finalizeBridgeERC20, (address(0), address(0), address(0), address(0), 0, ""));
    }

    /// @notice Returns an item from one messenger to the other, in the direction, with the data.
    function _messengerItem(
        BridgeHookTypes.Direction _direction,
        bytes memory _data
    )
        internal
        pure
        returns (BridgeHookTypes.Item memory item_)
    {
        (address source, address destination) = _messengers(_direction);
        item_.direction = _direction;
        // The L1 messenger is a contract, so its deposits are aliased.
        item_.aliased = _direction == BridgeHookTypes.Direction.Deposit;
        item_.from = item_.aliased ? AddressAliasHelper.applyL1ToL2Alias(source) : source;
        item_.to = destination;
        item_.data = _data;
    }

    /// @notice Returns a messenger item whose envelope is from one standard bridge to the other.
    function _bridgeItem(
        BridgeHookTypes.Direction _direction,
        bytes memory _message
    )
        internal
        pure
        returns (BridgeHookTypes.Item memory)
    {
        (address source, address destination) = _bridges(_direction);
        return _messengerItem(_direction, _envelope(source, destination, _message));
    }
}

/// @title BridgeHookDecoder_RecognizeTokenTransfer_Test
/// @notice Tests the `recognizeTokenTransfer` function of the `BridgeHookDecoder` library.
contract BridgeHookDecoder_RecognizeTokenTransfer_Test is BridgeHookDecoder_TestInit {
    /// @notice Tests that a transfer between the standard bridges is recognized, whatever its asset.
    function testFuzz_recognizeTokenTransfer_canonicalTransfer_succeeds(
        BridgeHookTypes.Direction _direction,
        BridgeHookTypes.Asset _asset,
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        external
        pure
    {
        bytes memory message = abi.encodeCall(
            IStandardBridge.finalizeBridgeERC20, (_localToken, _remoteToken, _from, _to, _amount, _extraData)
        );
        BridgeHookTypes.Item memory item = _bridgeItem(_direction, message);
        item.asset = _asset;

        (bool isTokenTransfer, bytes memory recognized) =
            BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertTrue(isTokenTransfer);
        assertEq(recognized, message);
    }

    /// @notice Tests that a transfer carrying ETH is not recognized.
    function testFuzz_recognizeTokenTransfer_nonzeroEthAmount_fails(
        BridgeHookTypes.Direction _direction,
        uint256 _ethAmount
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item = _bridgeItem(_direction, _tokenMessage());
        item.ethAmount = bound(_ethAmount, 1, type(uint256).max);

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a deposit from the L1 messenger itself is not recognized unless aliased.
    function test_recognizeTokenTransfer_depositNotAliased_fails() external pure {
        BridgeHookTypes.Item memory item = _bridgeItem(BridgeHookTypes.Direction.Deposit, _tokenMessage());
        item.from = L1_MESSENGER;
        item.aliased = false;

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that an item not sent by the messenger is not recognized.
    function testFuzz_recognizeTokenTransfer_outerSenderNotMessenger_fails(
        BridgeHookTypes.Direction _direction,
        address _from
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item = _bridgeItem(_direction, _tokenMessage());
        vm.assume(_from != item.from);
        item.from = _from;

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that an item not sent to the other messenger is not recognized.
    function testFuzz_recognizeTokenTransfer_outerTargetNotMessenger_fails(
        BridgeHookTypes.Direction _direction,
        address _to
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item = _bridgeItem(_direction, _tokenMessage());
        vm.assume(_to != item.to);
        item.to = _to;

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a transfer labeled with the opposite direction is not recognized.
    function testFuzz_recognizeTokenTransfer_flippedDirection_fails(BridgeHookTypes.Direction _direction)
        external
        pure
    {
        BridgeHookTypes.Item memory item = _bridgeItem(_direction, _tokenMessage());
        item.direction = _direction == BridgeHookTypes.Direction.Deposit
            ? BridgeHookTypes.Direction.Withdrawal
            : BridgeHookTypes.Direction.Deposit;

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a message not sent by the bridge is not recognized.
    function testFuzz_recognizeTokenTransfer_innerSenderNotBridge_fails(
        BridgeHookTypes.Direction _direction,
        address _sender
    )
        external
        pure
    {
        (address source, address destination) = _bridges(_direction);
        vm.assume(_sender != source);
        BridgeHookTypes.Item memory item = _messengerItem(_direction, _envelope(_sender, destination, _tokenMessage()));

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a message not sent to the other bridge is not recognized.
    function testFuzz_recognizeTokenTransfer_innerTargetNotBridge_fails(
        BridgeHookTypes.Direction _direction,
        address _target
    )
        external
        pure
    {
        (address source, address destination) = _bridges(_direction);
        vm.assume(_target != destination);
        BridgeHookTypes.Item memory item = _messengerItem(_direction, _envelope(source, _target, _tokenMessage()));

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a bridge call other than finalizeBridgeERC20 is not recognized.
    function testFuzz_recognizeTokenTransfer_otherInnerSelector_fails(
        BridgeHookTypes.Direction _direction,
        bytes4 _selector
    )
        external
        pure
    {
        vm.assume(_selector != IStandardBridge.finalizeBridgeERC20.selector);
        BridgeHookTypes.Item memory item = _bridgeItem(
            _direction, bytes.concat(_selector, abi.encode(address(0), address(0), address(0), address(0), 0, ""))
        );

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that a messenger call other than relayMessage is not recognized.
    function testFuzz_recognizeTokenTransfer_otherEnvelopeSelector_fails(
        BridgeHookTypes.Direction _direction,
        bytes4 _selector
    )
        external
        pure
    {
        vm.assume(_selector != ICrossDomainMessenger.relayMessage.selector);
        (address source, address destination) = _bridges(_direction);
        BridgeHookTypes.Item memory item = _messengerItem(
            _direction, bytes.concat(_selector, abi.encode(0, source, destination, 0, 0, _tokenMessage()))
        );

        (bool isTokenTransfer,) = BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }

    /// @notice Tests that an empty message between the bridges is not recognized.
    function testFuzz_recognizeTokenTransfer_emptyBridgeMessage_fails(BridgeHookTypes.Direction _direction)
        external
        pure
    {
        (bool isTokenTransfer,) =
            BridgeHookDecoder.recognizeTokenTransfer(_bridgeItem(_direction, ""), L1_MESSENGER, L1_BRIDGE);
        assertFalse(isTokenTransfer);
    }
}

/// @title BridgeHookDecoder_ExtractTokenMessage_Test
/// @notice Tests the `extractTokenMessage` function of the `BridgeHookDecoder` library.
contract BridgeHookDecoder_ExtractTokenMessage_Test is BridgeHookDecoder_TestInit {
    /// @notice Tests that the message is extracted from an ERC20 item, whatever its route.
    function testFuzz_extractTokenMessage_anyRoute_succeeds(
        BridgeHookTypes.Direction _direction,
        address _from,
        address _to,
        address _sender,
        address _target,
        bytes memory _message
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item;
        item.direction = _direction;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.from = _from;
        item.to = _to;
        item.data = _envelope(_sender, _target, _message);

        (bool valid, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        assertTrue(valid);
        assertEq(message, _message);
    }

    /// @notice Tests that nothing is extracted from an ETH item, even between the bridges.
    function testFuzz_extractTokenMessage_ethAsset_fails(BridgeHookTypes.Direction _direction) external pure {
        (bool valid,) = BridgeHookDecoder.extractTokenMessage(_bridgeItem(_direction, _tokenMessage()));
        assertFalse(valid);
    }

    /// @notice Tests that data one byte shorter than the head is not extracted.
    function test_extractTokenMessage_shorterThanHead_fails() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector, abi.encode(0, address(0), address(0), 0, 0), bytes31(0)
        );

        (bool valid,) = BridgeHookDecoder.extractTokenMessage(item);
        assertFalse(valid);
    }

    /// @notice Tests that the largest offset is not extracted.
    function test_extractTokenMessage_maxOffset_fails() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(0, address(0), address(0), 0, 0, type(uint256).max, 0)
        );

        (bool valid,) = BridgeHookDecoder.extractTokenMessage(item);
        assertFalse(valid);
    }

    /// @notice Tests that an empty message whose offset points at the last word is extracted.
    function test_extractTokenMessage_offsetAtLastWord_succeeds() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        // The six head words, then the length word that the offset 0xc0 points at.
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector, abi.encode(0, address(0), address(0), 0, 0, 0xc0, 0)
        );

        (bool valid, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        assertTrue(valid);
        assertEq(message, "");
    }

    /// @notice Tests that an offset one word past the last word is not extracted.
    function test_extractTokenMessage_offsetPastEnd_fails() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector, abi.encode(0, address(0), address(0), 0, 0, 0xe0, 0)
        );

        (bool valid,) = BridgeHookDecoder.extractTokenMessage(item);
        assertFalse(valid);
    }

    /// @notice Tests that the largest length is not extracted.
    function test_extractTokenMessage_maxLength_fails() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(0, address(0), address(0), 0, 0, 0xc0, type(uint256).max)
        );

        (bool valid,) = BridgeHookDecoder.extractTokenMessage(item);
        assertFalse(valid);
    }

    /// @notice Tests that a message that ends exactly at the end of the data is extracted.
    function test_extractTokenMessage_lengthFillsData_succeeds() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(0, address(0), address(0), 0, 0, 0xc0, 32, bytes32(uint256(1)))
        );

        (bool valid, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        assertTrue(valid);
        assertEq(message, abi.encode(1));
    }

    /// @notice Tests that a message one byte longer than the data is not extracted.
    function test_extractTokenMessage_lengthPastEnd_fails() external pure {
        BridgeHookTypes.Item memory item;
        item.asset = BridgeHookTypes.Asset.ERC20;
        item.data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(0, address(0), address(0), 0, 0, 0xc0, 33, bytes32(uint256(1)))
        );

        (bool valid,) = BridgeHookDecoder.extractTokenMessage(item);
        assertFalse(valid);
    }

    /// @notice Tests that the envelope reader matches abi.decode.
    function testFuzz_extractTokenMessage_matchesAbiDecode_works(
        BridgeHookTypes.Direction _direction,
        address _sender,
        address _target,
        uint256 _gap,
        uint256 _length,
        bytes memory _tail
    )
        external
        view
    {
        // Between the bridges, resolveParties reads the bridge call instead of the envelope.
        (address source, address destination) = _bridges(_direction);
        vm.assume(_sender != source || _target != destination);
        // The offset skips the gap to the length word, so both decoders read the fuzzed length.
        _gap = bound(_gap, 0, 64);
        _length = bound(_length, 0, _tail.length + 64);
        bytes memory data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(0, _sender, _target, 0, 0, 0xc0 + _gap),
            new bytes(_gap),
            abi.encode(_length),
            _tail
        );
        BridgeHookTypes.Item memory item = _messengerItem(_direction, data);
        item.asset = BridgeHookTypes.Asset.ERC20;

        (bool valid, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);

        try harness.decodeRelayMessage(data) returns (address sender_, address target_, bytes memory message_) {
            assertTrue(valid, "valid");
            assertEq(message, message_, "message");
            assertEq(initiator, sender_, "sender");
            assertEq(recipient, target_, "target");
        } catch {
            assertFalse(valid, "valid");
        }
    }

    /// @notice Tests that high bits in the sender and target words are ignored.
    function testFuzz_extractTokenMessage_dirtyAddressWords_succeeds(
        BridgeHookTypes.Direction _direction,
        address _sender,
        address _target,
        uint96 _senderHigh,
        uint96 _targetHigh,
        bytes memory _message
    )
        external
        pure
    {
        vm.assume(_senderHigh != 0 || _targetHigh != 0);
        (address source, address destination) = _bridges(_direction);
        vm.assume(_sender != source || _target != destination);
        bytes memory data = bytes.concat(
            ICrossDomainMessenger.relayMessage.selector,
            abi.encode(
                0,
                uint256(_senderHigh) << 160 | uint160(_sender),
                uint256(_targetHigh) << 160 | uint160(_target),
                0,
                0,
                _message
            )
        );
        BridgeHookTypes.Item memory item = _messengerItem(_direction, data);
        item.asset = BridgeHookTypes.Asset.ERC20;

        (bool valid, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertTrue(valid, "valid");
        assertEq(message, _message, "message");
        assertEq(initiator, _sender, "sender");
        assertEq(recipient, _target, "target");
    }
}

/// @title BridgeHookDecoder_DecodeTokenTransfer_Test
/// @notice Tests the `decodeTokenTransfer` function of the `BridgeHookDecoder` library.
contract BridgeHookDecoder_DecodeTokenTransfer_Test is BridgeHookDecoder_TestInit {
    /// @notice Tests that the local token is the L2 token for a deposit and the L1 token for a
    ///         withdrawal.
    function testFuzz_decodeTokenTransfer_canonicalMessage_succeeds(
        BridgeHookTypes.Direction _direction,
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes memory _extraData
    )
        external
        pure
    {
        bool isDeposit = _direction == BridgeHookTypes.Direction.Deposit;

        (bool valid, address l1Token, address l2Token, uint256 amount) = BridgeHookDecoder.decodeTokenTransfer(
            _direction,
            abi.encodeCall(
                IStandardBridge.finalizeBridgeERC20, (_localToken, _remoteToken, _from, _to, _amount, _extraData)
            )
        );
        assertTrue(valid, "valid");
        assertEq(l1Token, isDeposit ? _remoteToken : _localToken, "l1Token");
        assertEq(l2Token, isDeposit ? _localToken : _remoteToken, "l2Token");
        assertEq(amount, _amount, "amount");
    }

    /// @notice Tests that a call other than finalizeBridgeERC20 is not decoded.
    function testFuzz_decodeTokenTransfer_otherSelector_fails(
        BridgeHookTypes.Direction _direction,
        bytes4 _selector
    )
        external
        pure
    {
        vm.assume(_selector != IStandardBridge.finalizeBridgeERC20.selector);

        (bool valid,,,) = BridgeHookDecoder.decodeTokenTransfer(
            _direction, bytes.concat(_selector, abi.encode(address(0), address(0), address(0), address(0), 0, ""))
        );
        assertFalse(valid);
    }

    /// @notice Tests that a finalizeBridgeERC20 call one byte shorter than the head is not decoded.
    function test_decodeTokenTransfer_shortHead_fails() external pure {
        (bool valid,,,) = BridgeHookDecoder.decodeTokenTransfer(
            BridgeHookTypes.Direction.Deposit,
            bytes.concat(
                IStandardBridge.finalizeBridgeERC20.selector,
                abi.encode(address(0), address(0), address(0), address(0), 0),
                bytes31(0)
            )
        );
        assertFalse(valid);
    }

    /// @notice Tests that the tokens and amount match abi.decode on every call it accepts.
    function testFuzz_decodeTokenTransfer_matchesAbiDecode_succeeds(
        BridgeHookTypes.Direction _direction,
        address _localToken,
        address _remoteToken,
        uint256 _amount,
        uint256 _gap,
        uint256 _length,
        bytes memory _tail
    )
        external
        view
    {
        // The offset skips the gap to the length word, so abi.decode reads the fuzzed length.
        _gap = bound(_gap, 0, 64);
        _length = bound(_length, 0, _tail.length + 64);
        bytes memory message = bytes.concat(
            IStandardBridge.finalizeBridgeERC20.selector,
            abi.encode(_localToken, _remoteToken, address(0), address(0), _amount, 0xc0 + _gap),
            new bytes(_gap),
            abi.encode(_length),
            _tail
        );
        bool isDeposit = _direction == BridgeHookTypes.Direction.Deposit;

        (bool valid, address l1Token, address l2Token, uint256 amount) =
            BridgeHookDecoder.decodeTokenTransfer(_direction, message);

        // The decoder does not read extraData, so it accepts calls that abi.decode rejects.
        try harness.decodeFinalizeBridgeERC20(message) returns (
            address localToken_, address remoteToken_, uint256 amount_
        ) {
            assertTrue(valid, "valid");
            assertEq(l1Token, isDeposit ? remoteToken_ : localToken_, "l1Token");
            assertEq(l2Token, isDeposit ? localToken_ : remoteToken_, "l2Token");
            assertEq(amount, amount_, "amount");
        } catch { }
    }
}

/// @title BridgeHookDecoder_ResolveParties_Test
/// @notice Tests the `resolveParties` function of the `BridgeHookDecoder` library.
contract BridgeHookDecoder_ResolveParties_Test is BridgeHookDecoder_TestInit {
    /// @notice Tests that an item off the messenger route resolves to its unaliased sender and its
    ///         target, whatever its data.
    function testFuzz_resolveParties_offMessengerRoute_succeeds(
        BridgeHookTypes.Direction _direction,
        address _from,
        bool _aliased,
        address _to,
        bytes memory _data
    )
        external
        pure
    {
        // An aliased withdrawal is left unspecified.
        bool aliased = _direction == BridgeHookTypes.Direction.Deposit && _aliased;
        address sender = aliased ? AddressAliasHelper.undoL1ToL2Alias(_from) : _from;
        (address source, address destination) = _messengers(_direction);
        vm.assume(sender != source || _to != destination);
        BridgeHookTypes.Item memory item;
        item.direction = _direction;
        item.from = _from;
        item.aliased = aliased;
        item.to = _to;
        item.data = _data;

        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, sender, "initiator");
        assertEq(recipient, _to, "recipient");
    }

    /// @notice Tests that a message not between the bridges resolves to its sender and target,
    ///         even when it is a finalizeBridgeERC20 call.
    function testFuzz_resolveParties_innerNotBridge_succeeds(
        BridgeHookTypes.Direction _direction,
        address _sender,
        address _target
    )
        external
        pure
    {
        (address source, address destination) = _bridges(_direction);
        vm.assume(_sender != source || _target != destination);
        BridgeHookTypes.Item memory item = _messengerItem(
            _direction,
            _envelope(
                _sender,
                _target,
                abi.encodeCall(
                    IStandardBridge.finalizeBridgeERC20, (address(0), address(0), address(1), address(2), 0, "")
                )
            )
        );

        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, _sender, "initiator");
        assertEq(recipient, _target, "recipient");
    }

    /// @notice Tests that a messenger item whose envelope does not decode resolves to the two
    ///         messengers.
    function testFuzz_resolveParties_envelopeUnreadable_succeeds(BridgeHookTypes.Direction _direction) external pure {
        (address source, address destination) = _messengers(_direction);

        (address initiator, address recipient) =
            BridgeHookDecoder.resolveParties(_messengerItem(_direction, ""), L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, source, "initiator");
        assertEq(recipient, destination, "recipient");
    }

    /// @notice Tests that a token transfer resolves to its sender and recipient.
    function testFuzz_resolveParties_finalizeBridgeERC20_succeeds(
        BridgeHookTypes.Direction _direction,
        address _from,
        address _to
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item = _bridgeItem(
            _direction, abi.encodeCall(IStandardBridge.finalizeBridgeERC20, (address(0), address(0), _from, _to, 0, ""))
        );

        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, _from, "initiator");
        assertEq(recipient, _to, "recipient");
    }

    /// @notice Tests that an ETH transfer between the bridges resolves to its sender and recipient.
    function testFuzz_resolveParties_finalizeBridgeETH_succeeds(
        BridgeHookTypes.Direction _direction,
        address _from,
        address _to,
        uint256 _amount
    )
        external
        pure
    {
        BridgeHookTypes.Item memory item =
            _bridgeItem(_direction, abi.encodeCall(IStandardBridge.finalizeBridgeETH, (_from, _to, _amount, "")));
        item.ethAmount = bound(_amount, 1, type(uint256).max);

        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, _from, "initiator");
        assertEq(recipient, _to, "recipient");
    }

    /// @notice Tests that a bridge call that does not decode resolves to the two bridges.
    function testFuzz_resolveParties_bridgeCallUnreadable_succeeds(
        BridgeHookTypes.Direction _direction,
        bytes4 _selector
    )
        external
        pure
    {
        vm.assume(
            _selector != IStandardBridge.finalizeBridgeERC20.selector
                && _selector != IStandardBridge.finalizeBridgeETH.selector
        );
        (address source, address destination) = _bridges(_direction);
        bytes[3] memory messages = [
            bytes.concat(_selector, abi.encode(address(1), address(2), 0, "")),
            bytes(""),
            // Three of the four head words.
            bytes.concat(IStandardBridge.finalizeBridgeETH.selector, abi.encode(address(1), address(2), 0))
        ];

        for (uint256 i; i < messages.length; i++) {
            (address initiator, address recipient) =
                BridgeHookDecoder.resolveParties(_bridgeItem(_direction, messages[i]), L1_MESSENGER, L1_BRIDGE);
            assertEq(initiator, source, vm.toString(i));
            assertEq(recipient, destination, vm.toString(i));
        }
    }

    /// @notice Tests that an envelope inside an envelope resolves one level only.
    function testFuzz_resolveParties_nestedEnvelope_succeeds(BridgeHookTypes.Direction _direction) external pure {
        (address messengerSource, address messengerDestination) = _messengers(_direction);
        (address bridgeSource, address bridgeDestination) = _bridges(_direction);
        BridgeHookTypes.Item memory item = _messengerItem(
            _direction,
            _envelope(
                messengerSource, messengerDestination, _envelope(bridgeSource, bridgeDestination, _tokenMessage())
            )
        );

        (address initiator, address recipient) = BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
        assertEq(initiator, messengerSource, "initiator");
        assertEq(recipient, messengerDestination, "recipient");
    }
}

/// @title BridgeHookDecoder_Uncategorized_Test
/// @notice Tests properties that hold across the `BridgeHookDecoder` library.
contract BridgeHookDecoder_Uncategorized_Test is BridgeHookDecoder_TestInit {
    /// @notice Tests that no function reverts on an arbitrary item.
    function testFuzz_decoder_arbitraryInput_works(
        BridgeHookTypes.Item memory _item,
        address _l1Messenger,
        address _l1Bridge,
        bytes memory _message
    )
        external
        pure
    {
        BridgeHookDecoder.recognizeTokenTransfer(_item, _l1Messenger, _l1Bridge);
        BridgeHookDecoder.extractTokenMessage(_item);
        BridgeHookDecoder.decodeTokenTransfer(_item.direction, _message);
        BridgeHookDecoder.resolveParties(_item, _l1Messenger, _l1Bridge);
    }

    /// @notice Tests that no function reverts on any offset and length word inside an envelope
    ///         between the bridges.
    function testFuzz_decoder_adversarialWords_works(
        BridgeHookTypes.Direction _direction,
        uint256 _offset,
        uint256 _length,
        bytes memory _tail
    )
        external
        pure
    {
        // Near the data's end, where the bounds checks decide. The extremes have their own tests.
        _offset = bound(_offset, 0, 0xe0 + _tail.length + 64);
        _length = bound(_length, 0, _tail.length + 64);
        (address source, address destination) = _bridges(_direction);
        BridgeHookTypes.Item memory item = _messengerItem(
            _direction,
            bytes.concat(
                ICrossDomainMessenger.relayMessage.selector,
                abi.encode(0, source, destination, 0, 0, _offset, _length),
                _tail
            )
        );
        item.asset = BridgeHookTypes.Asset.ERC20;

        BridgeHookDecoder.recognizeTokenTransfer(item, L1_MESSENGER, L1_BRIDGE);
        (, bytes memory message) = BridgeHookDecoder.extractTokenMessage(item);
        BridgeHookDecoder.decodeTokenTransfer(_direction, message);
        BridgeHookDecoder.resolveParties(item, L1_MESSENGER, L1_BRIDGE);
    }
}
