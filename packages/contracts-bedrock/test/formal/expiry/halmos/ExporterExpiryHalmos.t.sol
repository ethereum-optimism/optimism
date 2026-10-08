// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Halmos symbolic checks on the REAL UndeliveredMessageExporter (src/L2/UndeliveredMessageExporter.sol),
// etched at its predeploy address Predeploys.UNDELIVERED_MESSAGE_EXPORTER (never hardcoded). See README.md for exact
// statements.
// Group (3), retargeted: exportUndeliveredMessage
//   - succeeds iff !L2ToL2CrossDomainMessenger.successfulMessages(H), H = keccak256(abi.encode(block.chainid, source,
//     nonce, sender, target, message)); returns H;
//   - on success makes EXACTLY ONE call, to the L2CrossDomainMessenger, with calldata exactly
//     sendMessage(sourceMessenger, relayUndeliveredMessage(H, block.timestamp), minGas); none to 0x..16; on revert
// none; - for ARBITRARY calldata to the exporter (every function, symbolic arguments, via svm.createCalldata), every
// call the
//     exporter makes to 0x..07 is that export payload for the decoded arguments: the exporter never calls anything
// else.
// Mocks: 0x..23 is MockSuccessful (successfulMessages is an arbitrary symbolic mapping: the real getter is a plain
// mapping read); 0x..07 and 0x..16 are fallback-only call recorders.

import { Test } from "test/setup/Test.sol";
import { UndeliveredMessageExporter } from "src/L2/UndeliveredMessageExporter.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";

interface SVMExporter {
    function enableSymbolicStorage(address) external;
    function createCalldata(string memory) external returns (bytes memory);
}

contract MockSuccessful {
    mapping(bytes32 => bool) public successfulMessages; // symbolic (symbolic storage)
}

/// @notice Fallback-only recorder: slot 0 = calls from the exporter, slot 1 = all calls, slot 2 / 3 = keccak256 /
///         length of the exporter's last call.
contract CallRecorder {
    address internal immutable EXPORTER = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;

    uint256 internal callsFromExporter;
    uint256 internal totalCalls;
    bytes32 internal lastHash;
    uint256 internal lastLen;

    fallback() external payable {
        totalCalls++;
        if (msg.sender == EXPORTER) {
            callsFromExporter++;
            lastHash = keccak256(msg.data);
            lastLen = msg.data.length;
        }
    }
}

contract ExporterExpiryHalmos is Test {
    SVMExporter internal constant svm = SVMExporter(0xF3993A62377BCd56AE39D773740A5390411E8BC9);
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;
    address internal constant L2CDM = 0x4200000000000000000000000000000000000007;
    address internal constant PASSER = 0x4200000000000000000000000000000000000016;
    address internal immutable EXPORTER = Predeploys.UNDELIVERED_MESSAGE_EXPORTER;

    UndeliveredMessageExporter internal ex = UndeliveredMessageExporter(Predeploys.UNDELIVERED_MESSAGE_EXPORTER);
    MockSuccessful internal msgr = MockSuccessful(L2_TO_L2);

    struct ExportArgs {
        address sourceMessenger;
        uint256 source;
        uint256 nonce;
        address sender;
        address target;
        uint32 minGas;
    }

    function setUp() public {
        assert(L2_TO_L2 == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER && L2CDM == Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assert(PASSER == Predeploys.L2_TO_L1_MESSAGE_PASSER);
        vm.etch(EXPORTER, address(new UndeliveredMessageExporter()).code);
        vm.etch(L2_TO_L2, address(new MockSuccessful()).code);
        vm.etch(L2CDM, address(new CallRecorder()).code);
        vm.etch(PASSER, address(new CallRecorder()).code);
    }

    function _calls(address _r) internal view returns (uint256) {
        return uint256(vm.load(_r, bytes32(uint256(0))));
    }

    function _lastHash(address _r) internal view returns (bytes32) {
        return vm.load(_r, bytes32(uint256(2)));
    }

    function _h(uint256 _dest, ExportArgs memory _a, bytes memory _message) internal pure returns (bytes32) {
        return keccak256(abi.encode(_dest, _a.source, _a.nonce, _a.sender, _a.target, _message));
    }

    function _payloadHash(address _sm, bytes32 _hash, uint256 _ts, uint32 _minGas) internal pure returns (bytes32) {
        return keccak256(
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage,
                (_sm, abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (_hash, _ts)), _minGas)
            )
        );
    }

    function _export(
        address _caller,
        ExportArgs memory _a,
        bytes memory _message
    )
        internal
        returns (bool ok_, bytes memory ret_)
    {
        vm.prank(_caller);
        (ok_, ret_) = EXPORTER.call(
            abi.encodeCall(
                ex.exportUndeliveredMessage,
                (_a.sourceMessenger, _a.source, _a.nonce, _a.sender, _a.target, _message, _a.minGas)
            )
        );
    }

    /// @notice (3) iff + exact single payload + no other call + no write to the messenger's successfulMessages.
    function check_export_binding(
        address _caller,
        uint256 _chainId,
        uint256 _ts,
        ExportArgs memory _a,
        bytes calldata _message,
        bytes32 _k
    )
        public
    {
        vm.chainId(_chainId);
        vm.warp(_ts);
        svm.enableSymbolicStorage(L2_TO_L2);
        bytes32 h = _h(_chainId, _a, _message);
        bool relayed = msgr.successfulMessages(h);
        bool atK = msgr.successfulMessages(_k);

        (bool ok, bytes memory ret) = _export(_caller, _a, _message);

        assert(ok == !relayed);
        assert(msgr.successfulMessages(_k) == atK && msgr.successfulMessages(h) == relayed);
        if (ok) {
            assert(abi.decode(ret, (bytes32)) == h);
            assert(_calls(L2CDM) == 1);
            assert(_lastHash(L2CDM) == _payloadHash(_a.sourceMessenger, h, _ts, _a.minGas));
            assert(uint256(vm.load(L2CDM, bytes32(uint256(1)))) == 1); // total calls to 0x..07: just this one
        } else {
            assert(_calls(L2CDM) == 0);
        }
        assert(uint256(vm.load(PASSER, bytes32(uint256(1)))) == 0);
    }

    /// @notice Arbitrary calldata to the exporter (any of its functions, symbolic arguments; svm.createCalldata), from
    ///         any caller: every call it makes to 0x..07 (at most one) is the export payload for the decoded arguments,
    ///         and it never calls 0x..16. So the exporter never initiates any other withdrawal. Loop bound raised for
    ///         the word-by-word memory copies of the `bytes` argument (up to 1024 bytes = 32 words); run.sh still
    ///         rejects any bounded loop.
    /// @custom:halmos --loop 40
    function check_exporter_anyCalldata_onlyExportPayload(address _caller, uint256 _chainId, uint256 _ts) public {
        vm.chainId(_chainId);
        vm.warp(_ts);
        svm.enableSymbolicStorage(L2_TO_L2);
        bytes memory data = svm.createCalldata("UndeliveredMessageExporter");
        vm.prank(_caller);
        (bool ok,) = EXPORTER.call(data);
        ok;
        assert(_calls(L2CDM) <= 1);
        assert(uint256(vm.load(PASSER, bytes32(uint256(1)))) == 0);
        if (_calls(L2CDM) == 1) {
            assert(bytes4(data) == ex.exportUndeliveredMessage.selector);
            (address sm, uint256 src, uint256 nonce, address snd, address tgt, bytes memory msg_, uint32 g) =
                abi.decode(_tail(data), (address, uint256, uint256, address, address, bytes, uint32));
            bytes32 h = keccak256(abi.encode(_chainId, src, nonce, snd, tgt, msg_));
            assert(_lastHash(L2CDM) == _payloadHash(sm, h, _ts, g));
        }
    }

    /// @notice The arguments after the selector, as a bytes view into `_data` (no copy loop). Overwrites `_data`'s
    ///         length word and selector, so `_data` must not be used afterwards.
    function _tail(bytes memory _data) internal pure returns (bytes memory t_) {
        assembly {
            let len := mload(_data)
            t_ := add(_data, 4)
            mstore(t_, sub(len, 4))
        }
    }

    /// @notice NON-VACUITY (expected FAIL): from fresh storage, export never calls the L2CrossDomainMessenger.
    function check_FALSE_export_neverCallsL2CDM(ExportArgs memory _a, bytes calldata _message) public {
        _export(address(this), _a, _message);
        assert(_calls(L2CDM) == 0);
    }

    /// @notice NON-VACUITY (expected FAIL): createCalldata-generated calls never make the exporter call 0x..07.
    /// @custom:halmos --loop 40
    function check_FALSE_exporter_anyCalldata_neverCalls(uint256 _chainId) public {
        vm.chainId(_chainId);
        bytes memory data = svm.createCalldata("UndeliveredMessageExporter");
        (bool ok,) = EXPORTER.call(data);
        ok;
        assert(_calls(L2CDM) == 0);
    }

    /// @notice NON-VACUITY (expected FAIL): export hashes with the SOURCE as destination.
    function check_FALSE_export_hashUsesSourceAsDestination(
        uint256 _chainId,
        ExportArgs memory _a,
        bytes calldata _message
    )
        public
    {
        vm.chainId(_chainId);
        (bool ok, bytes memory ret) = _export(address(this), _a, _message);
        if (ok) assert(abi.decode(ret, (bytes32)) == _h(_a.source, _a, _message));
    }
}
