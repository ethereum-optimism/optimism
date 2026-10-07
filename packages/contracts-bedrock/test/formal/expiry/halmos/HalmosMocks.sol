// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Mocks for the Halmos expiry checks. Every mock is an ORACLE: it returns whatever the test (or a
// symbolic storage slot) says, so the checks hold for every possible answer of the real contract
// it stands in for. Recording mocks count calls and hash the exact calldata they receive.

/// @notice halmos-cheatcodes SVM interface (inlined; lib/halmos-cheatcodes is not a submodule here).
interface SVM {
    function enableSymbolicStorage(address) external;
    function createUint256(string memory) external returns (uint256);
    function createBytes32(string memory) external returns (bytes32);
    function createAddress(string memory) external returns (address);
    function createBool(string memory) external returns (bool);
}

address constant SVM_ADDRESS = 0xF3993A62377BCd56AE39D773740A5390411E8BC9;

/// @notice CrossL2Inbox stand-in (etched at 0x4200..0022): validateMessage always succeeds.
///         Assumption: the identifier/payload pair is valid (the protocol's job, not the messenger's).
contract MockCrossL2Inbox {
    fallback() external payable { }
}

/// @notice Fallback-only call recorder (etched at 0x4200..0007 and at 0x4200..0016). It has NO external
///         functions, so every call, whatever its selector, reaches the fallback and is counted. Read its
///         state with vm.load: slot 0 = calls whose msg.sender is the L2ToL2CrossDomainMessenger (0x4200..0023),
///         slot 1 = all calls, slot 2 = keccak256(msg.data) of the last call from 0x..23, slot 3 = its length.
contract Recorder {
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;

    uint256 internal callsFrom23;
    uint256 internal totalCalls;
    bytes32 internal lastHashFrom23;
    uint256 internal lastLenFrom23;

    fallback() external payable {
        totalCalls++;
        if (msg.sender == L2_TO_L2) {
            callsFrom23++;
            lastHashFrom23 = keccak256(msg.data);
            lastLenFrom23 = msg.data.length;
        }
    }
}

/// @notice L2CrossDomainMessenger stand-in for expireMessage (etched at 0x4200..0007): answers the two getters
///         expireMessage reads with values the test sets (symbolic). Oracle over-approximation: the real
///         xDomainMessageSender() reverts when no message is being relayed; returning an arbitrary value instead
///         only adds behaviours.
contract MockL2CDMGetters {
    address internal xSender;
    address internal other;

    function setGetters(address _xSender, address _other) external {
        xSender = _xSender;
        other = _other;
    }

    function xDomainMessageSender() external view returns (address) {
        return xSender;
    }

    function otherMessenger() external view returns (address) {
        return other;
    }
}

interface IExportAndSend {
    function exportUndeliveredMessage(
        address _sourceMessenger,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes calldata _message,
        uint32 _minGasLimit
    )
        external
        returns (bytes32);
    function sendMessage(uint256 _destination, address _target, bytes calldata _message) external returns (bytes32);
}

/// @notice A relay target with code that RE-ENTERS the L2ToL2CrossDomainMessenger while it is being relayed to:
///         armed at construction with symbolic arguments, its fallback calls exportUndeliveredMessage (mode 1) or
///         sendMessage (mode 2) on 0x..23 and ignores the outcome, so the relay itself succeeds either way.
contract ReentrantTarget {
    address internal constant L2_TO_L2 = 0x4200000000000000000000000000000000000023;

    struct Args {
        uint256 mode; // 1 = exportUndeliveredMessage, 2 = sendMessage, else nothing
        address sourceMessenger;
        uint256 source;
        uint256 nonce;
        address sender;
        address target;
        uint32 minGas;
        uint256 destination;
    }

    Args internal a;
    bytes internal message;

    /// @dev Armed at construction, so the deployed code has ONLY the fallback: whatever calldata a relay sends, it
    ///      reaches the fallback (no selector can hit a setter and re-arm it).
    constructor(Args memory _a, bytes memory _message) {
        a = _a;
        message = _message;
    }

    fallback() external {
        bool ok;
        if (a.mode == 1) {
            (ok,) = L2_TO_L2.call(
                abi.encodeCall(
                    IExportAndSend.exportUndeliveredMessage,
                    (a.sourceMessenger, a.source, a.nonce, a.sender, a.target, message, a.minGas)
                )
            );
        } else if (a.mode == 2) {
            (ok,) = L2_TO_L2.call(abi.encodeCall(IExportAndSend.sendMessage, (a.destination, a.target, message)));
        }
        ok;
    }
}
