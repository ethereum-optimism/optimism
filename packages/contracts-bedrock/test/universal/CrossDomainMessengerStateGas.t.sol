// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";

/// @title CrossDomainMessenger_CalldataCost_Harness
/// @notice Call target that accepts arbitrary calldata and does nothing, so the gas charged for a
///         transaction to it is dominated by the cost of delivering its calldata.
contract CrossDomainMessenger_CalldataCost_Harness {
    /// @notice Accepts arbitrary calldata and does nothing.
    fallback() external { }
}

/// @title CrossDomainMessenger_PostCall_Harness
/// @notice Mirrors the bookkeeping `relayMessage` performs after its call to the target returns.
///         `RELAY_RESERVED_GAS` exists to guarantee this sequence can always complete; this harness
///         measures what it actually costs so the guarantee can be checked rather than assumed.
contract CrossDomainMessenger_PostCall_Harness {
    /// @notice Mirrors `CrossDomainMessenger.failedMessages`.
    mapping(bytes32 => bool) public failedMessages;

    /// @notice Mirrors `CrossDomainMessenger.xDomainMsgSender`, pre-set to the default sender.
    address public xDomainMsgSender = 0x000000000000000000000000000000000000dEaD;

    /// @notice Mirrors `CrossDomainMessenger.FailedRelayedMessage`.
    event FailedRelayedMessage(bytes32 indexed msgHash);

    /// @notice Performs the post-call sequence and reports the gas it consumed.
    /// @param _hash    Message hash to record, unused elsewhere so the slot is always fresh.
    /// @param _restore Address to restore the sender slot to.
    /// @return used_ Gas consumed by the sequence.
    function measure(bytes32 _hash, address _restore) external returns (uint256 used_) {
        uint256 start = gasleft();
        failedMessages[_hash] = true;
        emit FailedRelayedMessage(_hash);
        xDomainMsgSender = _restore;
        used_ = start - gasleft();
    }
}

/// @title CrossDomainMessenger_StateGas_Test
/// @notice Sizes two independent ways EIP-8037 / Amsterdam gas repricing invalidates
///         `CrossDomainMessenger`'s gas accounting, by varying one axis at a time.
///
///         `testFuzz_relayMessage_baseGasSufficient_succeeds` fails under Amsterdam, but it moves
///         both axes at once: raising `RELAY_RESERVED_GAS` relocates its counterexample from small
///         messages to large ones rather than fixing it. These tests hold one axis fixed so each
///         defect can be sized on its own.
///
///         Every test here is run under both `--evm-version cancun` and `--evm-version amsterdam`.
///         The cancun run is the control: these constants are correct under EIP-7623, so a test
///         that measures them honestly must pass there. A test that fails its cancun control is
///         measuring its own harness and its Amsterdam number means nothing.
///
///         NOT MEASURED HERE: the per-token calldata prices themselves, i.e. whether
///         `MIN_GAS_CALLDATA_OVERHEAD` (16) and `FLOOR_CALLDATA_OVERHEAD` (40) are stale. Three
///         attempts failed their cancun control — varying payload length (reported 8), burning gas
///         in the target to escape the floor regime (12), and comparing equal-length zero-filled
///         against non-zero-filled payloads (16, where the floor is 40). The last is the telling
///         one: it cancels execution cost correctly and still reports the standard price rather
///         than the floor, which indicates `vm.lastCallGas().gasTotalUsed` reports execution-path
///         gas and does not apply the EIP-7623 floor, that being a transaction-settlement rule.
///         Decomposing calldata price needs a different instrument than this cheatcode.
///         `testFuzz_baseGas_coversCalldataIntrinsic_succeeds` below sidesteps the problem by never
///         decomposing a price: it only asks whether `baseGas()` covers what is actually charged.
contract CrossDomainMessenger_StateGas_Test is CommonTest {
    /// @notice Builds a payload of `_length` bytes, every byte set to `_fill`.
    /// @param _length Length of the payload in bytes.
    /// @param _fill   Byte to fill with.
    /// @return payload_ The payload.
    function _payload(uint256 _length, bytes1 _fill) internal pure returns (bytes memory payload_) {
        payload_ = new bytes(_length);
        for (uint256 i = 0; i < _length; i++) {
            payload_[i] = _fill;
        }
    }

    /// @notice Sends `_data` to `_to` as its own transaction and reports total gas charged,
    ///         intrinsic cost included. Only meaningful under `isolate = true`.
    /// @param _to   Recipient.
    /// @param _data Calldata to send.
    /// @return used_ Total gas charged for the transaction.
    function _chargeFor(address _to, bytes memory _data) internal returns (uint256 used_) {
        (bool success,) = _to.call(_data);
        assertTrue(success, "CrossDomainMessenger_StateGas_Test: probe call reverted");
        used_ = uint256(vm.lastCallGas().gasTotalUsed);
    }

    /// @notice Axis 1, calldata. `baseGas()` is the entire gas budget the relay transaction gets,
    ///         and the cost of delivering its calldata comes out before execution begins. If
    ///         `baseGas()` is below that, the relay cannot start, let alone record an outcome — so
    ///         no value of `RELAY_RESERVED_GAS` can compensate. Message length is the only variable.
    ///
    ///         This asserts a relation between two measured quantities rather than a constant, so
    ///         it stays valid regardless of how calldata is priced.
    /// forge-config: default.isolate = true
    function testFuzz_baseGas_coversCalldataIntrinsic_succeeds(uint24 _messageLength) external {
        skipIfForkTest("CrossDomainMessenger_StateGas_Test: measures local EVM pricing");

        _messageLength = uint24(bound(_messageLength, 0, 34_000));

        // All non-zero bytes: the worst case for calldata pricing.
        bytes memory message = _payload(_messageLength, 0xff);
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), alice, address(0xBEEF), 0, 0, message
        );

        // A minGasLimit of zero isolates the calldata term: no target budget is included.
        uint256 baseGas = uint256(l1CrossDomainMessenger.baseGas(message, 0));

        CrossDomainMessenger_CalldataCost_Harness probe = new CrossDomainMessenger_CalldataCost_Harness();
        uint256 charged = _chargeFor(address(probe), encoded);

        assertGe(baseGas, charged, "baseGas does not cover the cost of delivering its own calldata");
    }

    /// @notice Axis 2, the post-call reserve. `RELAY_RESERVED_GAS` is withheld from the target call
    ///         so `relayMessage` can always record an outcome afterwards. Under EIP-8037 the status
    ///         write becomes a priced state creation, so the reserve has to cover far more than it
    ///         was sized for. Calldata length does not enter here, so this measures the reserve
    ///         shortfall alone, and the reported figure is the value the constant should hold.
    ///
    ///         The measurement is a lower bound: it omits the cold-slot access `relayMessage` pays
    ///         on a mapping it has not touched in the same frame. It is not guarded by
    ///         `skipIfUnoptimized` because the state-creation charge dominates the surrounding
    ///         Solidity overhead by two orders of magnitude, so optimizer settings cannot move the
    ///         result across the threshold being asserted.
    function test_relayReservedGas_coversPostCallWork_succeeds() external {
        skipIfForkTest("CrossDomainMessenger_StateGas_Test: measures local EVM pricing");

        CrossDomainMessenger_PostCall_Harness probe = new CrossDomainMessenger_PostCall_Harness();
        uint256 required = probe.measure(keccak256("CrossDomainMessenger_StateGas_Test"), address(this));

        assertLe(
            required,
            uint256(l1CrossDomainMessenger.RELAY_RESERVED_GAS()),
            "RELAY_RESERVED_GAS does not cover relayMessage's post-call bookkeeping"
        );
    }
}
