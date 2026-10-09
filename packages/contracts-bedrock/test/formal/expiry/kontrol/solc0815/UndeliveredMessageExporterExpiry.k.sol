// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Kontrol (KEVM) proofs for UndeliveredMessageExporter (OnlyExportReachesL1, exporter side).
//
// Model: the REAL UndeliveredMessageExporter is etched at Predeploys.UNDELIVERED_MESSAGE_EXPORTER
// (the address is read from the library, never hardcoded); a mock with a fully symbolic
// successfulMessages mapping stands for the L2ToL2CrossDomainMessenger at 0x..23; a recording mock
// stands for the L2CrossDomainMessenger at 0x..07. block.chainid and block.timestamp are symbolic;
// the message has a fixed length of 600 bytes. Not runnable with plain forge (Kontrol-only cheat
// codes).

// Contracts
import { UndeliveredMessageExporter } from "src/L2/UndeliveredMessageExporter.sol";
import {
    ExpiryKontrolBaseL1,
    MockSuccessfulMessages,
    RecordingMock0815
} from "test/formal/expiry/kontrol/solc0815/ExpiryMocks0815.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";

contract UndeliveredMessageExporterExpiryKontrol is ExpiryKontrolBaseL1 {
    function setUp() public {
        _etch(Predeploys.UNDELIVERED_MESSAGE_EXPORTER, address(new UndeliveredMessageExporter()));
        _etch(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER, address(new MockSuccessfulMessages()));
        _etch(Predeploys.L2_CROSS_DOMAIN_MESSENGER, address(new RecordingMock0815()));
    }

    function _etch(address _at, address _impl) internal {
        vm.etch(_at, _impl.code);
        vm.etch(_impl, hex"");
    }

    function _callsFrom(address _mock, address _caller) internal view returns (uint256) {
        return uint256(vm.load(_mock, keccak256(abi.encode(_caller, uint256(0)))));
    }

    function _exportCall(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodeCall(
            UndeliveredMessageExporter.exportUndeliveredMessage,
            (_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit)
        );
    }

    /// @notice sendMessage(sourceMessenger, relayUndeliveredMessage(H, timestamp), minGasLimit).
    function _expectedL2CDMCall(
        address _sourceMessenger,
        bytes32 _messageHash,
        uint256 _timestamp,
        uint32 _minGasLimit
    )
        internal
        pure
        returns (bytes memory)
    {
        return abi.encodeCall(
            ICrossDomainMessenger.sendMessage,
            (
                _sourceMessenger,
                abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (_messageHash, _timestamp)),
                _minGasLimit
            )
        );
    }

    /// @notice For ALL sourceMessenger, source, nonce, sender, target, 600-byte message,
    ///         minGasLimit, chainid, timestamp and set of relayed hashes, with H =
    ///         keccak256(abi.encode(chainid, source, nonce, sender, target, message)):
    ///         - exportUndeliveredMessage succeeds IFF !successfulMessages[H], and then returns H;
    ///         - its ONLY non-static external call is to 0x..07 with calldata exactly
    ///           sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp),
    ///           minGasLimit), made once.
    ///         "Only" combines two checks. (1) Kontrol's call whitelist allows CALLs only to the
    ///         exporter (the test's own call), to 0x..07 and to the cheat-code address (harness
    ///         only). Any other CALL is cut off with KONTROL_WHITELISTCALL, which reverts the
    ///         export, so on every successful export (all !relayed paths, by the first assertion)
    ///         no other CALL happened. (2) The recording mock at 0x..07 sees exactly one call from
    ///         the exporter, with calldata hash equal to keccak256 of the expected calldata.
    ///         STATICCALLs (the successfulMessages read) are not whitelisted; they cannot change
    ///         state.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_exporter_onlyCallsL2CDMWithFixedPayload(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        uint256 chainId = kevm.freshUInt(32);
        vm.chainId(chainId);
        uint256 timestamp = kevm.freshUInt(8);
        vm.warp(timestamp);
        kevm.symbolicStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        bytes32 h = keccak256(abi.encode(chainId, _source, _nonce, _sender, _target, _message));
        bool relayed = MockSuccessfulMessages(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).successfulMessages(h);
        bytes memory exportCall =
            _exportCall(_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit);
        bytes memory expected = _expectedL2CDMCall(_sourceMessenger, h, timestamp, _minGasLimit);

        // Cheat codes (vm.load below) go through the cheat-code address, which no real chain has.
        kevm.allowCallsToAddress(address(vm));
        kevm.allowCallsToAddress(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        kevm.allowCallsToAddress(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        (bool ok, bytes memory ret) = Predeploys.UNDELIVERED_MESSAGE_EXPORTER.call(exportCall);

        assert(ok == !relayed);
        uint256 calls = _callsFrom(Predeploys.L2_CROSS_DOMAIN_MESSENGER, Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        if (ok) {
            assert(abi.decode(ret, (bytes32)) == h);
            assert(calls == 1);
            assert(vm.load(Predeploys.L2_CROSS_DOMAIN_MESSENGER, bytes32(uint256(1))) == keccak256(expected));
        } else {
            assert(calls == 0);
        }
    }

    /// @notice WITNESS (expected to FAIL): under the same assumptions and call whitelist as the
    ///         proof above, the export can succeed.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_exporter_canSucceed_WITNESS(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        vm.chainId(kevm.freshUInt(32));
        vm.warp(kevm.freshUInt(8));
        kevm.symbolicStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bytes memory exportCall =
            _exportCall(_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit);
        kevm.allowCallsToAddress(address(vm));
        kevm.allowCallsToAddress(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        kevm.allowCallsToAddress(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        (bool ok,) = Predeploys.UNDELIVERED_MESSAGE_EXPORTER.call(exportCall);
        assert(!ok);
    }

    /// @notice WITNESS (expected to FAIL): the call whitelist is live. Here 0x..07 is left out of
    ///         the whitelist, so the export's call to it is cut off (Kontrol ends that call frame
    ///         with KONTROL_WHITELISTCALL, which reverts the export) and `ok == !relayed` must
    ///         fail.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_exporter_whitelistIsLive_WITNESS(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        uint256 chainId = kevm.freshUInt(32);
        vm.chainId(chainId);
        vm.warp(kevm.freshUInt(8));
        kevm.symbolicStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bytes32 h = keccak256(abi.encode(chainId, _source, _nonce, _sender, _target, _message));
        bool relayed = MockSuccessfulMessages(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).successfulMessages(h);
        bytes memory exportCall =
            _exportCall(_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit);
        kevm.allowCallsToAddress(address(vm));
        kevm.allowCallsToAddress(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        (bool ok,) = Predeploys.UNDELIVERED_MESSAGE_EXPORTER.call(exportCall);
        assert(ok == !relayed);
    }

    // ---------------------------------------------------------------------------------------------
    // Phase 2: whole-contract reachability (any selector)
    // ---------------------------------------------------------------------------------------------

    /// @notice For ANY 4-byte selector and any caller, with the argument region laid out as an
    ///         export call (600-byte message): every non-static call the exporter makes is to
    ///         0x..07 (call whitelist, as above), there is at most one, and if there is one then
    ///         the selector is exportUndeliveredMessage's and the calldata is exactly the export
    ///         payload for the decoded arguments. Selectors other than the exporter's two functions
    ///         hit no function (no fallback) and revert.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_exporter_anySelector_onlyExportPayload(
        bytes4 _selector,
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        uint256 chainId = kevm.freshUInt(32);
        vm.chainId(chainId);
        uint256 timestamp = kevm.freshUInt(8);
        vm.warp(timestamp);
        kevm.symbolicStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        bool isExport = _selector == UndeliveredMessageExporter.exportUndeliveredMessage.selector;
        bytes32 expectedHash = keccak256(
            _expectedL2CDMCall(
                _sourceMessenger,
                keccak256(abi.encode(chainId, _source, _nonce, _sender, _target, _message)),
                timestamp,
                _minGasLimit
            )
        );
        // The argument region of an export call, behind an arbitrary selector.
        _callAndCheck(
            abi.encodePacked(
                _selector, abi.encode(_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit)
            ),
            isExport,
            expectedHash
        );
    }

    /// @notice Calls the exporter with `_data` under the call whitelist and checks the recorder.
    function _callAndCheck(bytes memory _data, bool _isExport, bytes32 _expectedHash) internal {
        kevm.allowCallsToAddress(address(vm));
        kevm.allowCallsToAddress(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        kevm.allowCallsToAddress(Predeploys.L2_CROSS_DOMAIN_MESSENGER);

        vm.prank(kevm.freshAddress());
        (bool ok,) = Predeploys.UNDELIVERED_MESSAGE_EXPORTER.call(_data);
        ok; // success or failure, the property below must hold

        uint256 calls = _callsFrom(Predeploys.L2_CROSS_DOMAIN_MESSENGER, Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        assert(calls <= 1);
        if (calls == 1) {
            assert(_isExport);
            assert(vm.load(Predeploys.L2_CROSS_DOMAIN_MESSENGER, bytes32(uint256(1))) == _expectedHash);
        }
    }

    /// @notice WITNESS (expected to FAIL): in the any-selector setting, the exporter can reach
    ///         0x..07.
    /// @custom:kontrol-bytes-length-equals _message: 600,
    function prove_exporter_anySelectorReachesL2CDM_WITNESS(
        bytes4 _selector,
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
    {
        vm.chainId(kevm.freshUInt(32));
        vm.warp(kevm.freshUInt(8));
        kevm.symbolicStorage(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        kevm.allowCallsToAddress(address(vm));
        kevm.allowCallsToAddress(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
        kevm.allowCallsToAddress(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        vm.prank(kevm.freshAddress());
        (bool ok,) = Predeploys.UNDELIVERED_MESSAGE_EXPORTER
            .call(
                abi.encodePacked(
                    _selector, abi.encode(_sourceMessenger, _source, _nonce, _sender, _target, _message, _minGasLimit)
                )
            );
        ok;
        assert(_callsFrom(Predeploys.L2_CROSS_DOMAIN_MESSENGER, Predeploys.UNDELIVERED_MESSAGE_EXPORTER) == 0);
    }
}
