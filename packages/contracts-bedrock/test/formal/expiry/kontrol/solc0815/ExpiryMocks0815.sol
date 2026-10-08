// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Mocks for the Kontrol expiry proofs of L1CrossDomainMessenger.relayUndeliveredMessage and
// SuperchainETHBridge.refundETH. Every mock is an explicit modelling assumption. Mock getters
// return whatever is in storage, and the proofs make that storage symbolic, so each mock stands for
// ANY contract that answers the getter (each getter is called at most once per call path, so a
// constant answer is without loss of generality).

// Interfaces
import { Vm } from "forge-std/Vm.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";

/// @notice Cheat codes Kontrol 1.0.255 implements on top of forge's `Vm`.
interface KontrolExpiryCheatsL1 {
    function freshUInt(uint8) external returns (uint256);
    function freshBool() external returns (uint256);
    function freshAddress() external returns (address);
    function freshBytes(uint256) external returns (bytes memory);
    function symbolicStorage(address) external;
    function allowCallsToAddress(address) external;
}

abstract contract ExpiryKontrolBaseL1 {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
    KontrolExpiryCheatsL1 internal constant kevm =
        KontrolExpiryCheatsL1(address(uint160(uint256(keccak256("hevm cheat code")))));
}

/// @notice Stands for the CALLER of relayUndeliveredMessage (in the honest case: chain B's
///         L1CrossDomainMessenger relaying B's withdrawal). portal() and xDomainMessageSender()
///         return slots 0 and 1.
contract MockCallerMessenger {
    address internal portalRet;
    address internal xSenderRet;

    function portal() external view returns (address) {
        return portalRet;
    }

    function xDomainMessageSender() external view returns (address) {
        return xSenderRet;
    }

    function callRelay(
        address _target,
        bytes32 _messageHash,
        uint256 _undeliveredAt
    )
        external
        returns (bool, bytes memory)
    {
        return
            _target.call(
                abi.encodeCall(IL1CrossDomainMessenger.relayUndeliveredMessage, (_messageHash, _undeliveredAt))
            );
    }
}

/// @notice Stands for the caller's OptimismPortal: systemConfig() returns slot 0.
contract MockCallerPortal {
    address internal systemConfigRet;

    function systemConfig() external view returns (address) {
        return systemConfigRet;
    }
}

/// @notice Stands for the caller portal's SystemConfig: l1CrossDomainMessenger() returns slot 0.
contract MockSystemConfig {
    address internal l1CrossDomainMessengerRet;

    function l1CrossDomainMessenger() external view returns (address) {
        return l1CrossDomainMessengerRet;
    }
}

/// @notice Stands for chain A's ETHLockbox: authorizedPortals(p) = mapping at slot 0 (symbolic in
///         the proofs).
contract MockLockbox {
    mapping(address => bool) internal authorized;

    function authorizedPortals(address _portal) external view returns (bool) {
        return authorized[_portal];
    }
}

/// @notice Stands for chain A's OptimismPortal: ethLockbox() returns slot 0; depositTransaction
///         records its arguments (read back with `lastDeposit()`).
contract MockPortalA {
    address internal lockbox;
    uint256 internal deposits;
    address internal lastTo;
    uint256 internal lastValue;
    uint64 internal lastGasLimit;
    bool internal lastIsCreation;
    bytes32 internal lastDataHash;
    address internal lastSender;
    uint256 internal lastMsgValue;

    function ethLockbox() external view returns (address) {
        return lockbox;
    }

    function lastDeposit() external view returns (uint256, address, uint256, uint64, bool, bytes32, address, uint256) {
        return (deposits, lastTo, lastValue, lastGasLimit, lastIsCreation, lastDataHash, lastSender, lastMsgValue);
    }

    function depositTransaction(
        address _to,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes calldata _data
    )
        external
        payable
    {
        deposits += 1;
        lastTo = _to;
        lastValue = _value;
        lastGasLimit = _gasLimit;
        lastIsCreation = _isCreation;
        lastDataHash = keccak256(_data);
        lastSender = msg.sender;
        lastMsgValue = msg.value;
    }
}

/// @notice Stands for chain A's L2ToL2CrossDomainMessenger in the refundETH proofs:
///         expiredMessages(h) = mapping at slot 0 (symbolic in the proofs, i.e. ANY set of expired
///         hashes).
contract MockExpiredMessages {
    mapping(bytes32 => bool) internal expired;
    bytes32 internal sendMessageRet;
    address internal contextSender;
    uint256 internal contextSource;

    function expiredMessages(bytes32 _messageHash) external view returns (bool) {
        return expired[_messageHash];
    }

    /// @notice Used by sendETH in the any-selector proof; returns slot 1.
    function sendMessage(uint256, address, bytes calldata) external view returns (bytes32) {
        return sendMessageRet;
    }

    /// @notice Used by relayETH in the any-selector proof; returns slots 2 and 3.
    function crossDomainMessageContext() external view returns (address, uint256) {
        return (contextSender, contextSource);
    }
}

/// @notice Stands for chain A's SystemConfig (relayUndeliveredMessage's interop gate):
///         isFeatureEnabled(f) = mapping at slot 0 (symbolic in the proofs).
contract MockSystemConfigA {
    mapping(bytes32 => bool) internal features;

    function isFeatureEnabled(bytes32 _feature) external view returns (bool) {
        return features[_feature];
    }
}

/// @notice Stands for the L2ToL2CrossDomainMessenger in the exporter proofs: successfulMessages(h)
///         = mapping at slot 0 (symbolic in the proofs, i.e. ANY set of relayed hashes).
contract MockSuccessfulMessages {
    mapping(bytes32 => bool) internal relayed;

    function successfulMessages(bytes32 _messageHash) external view returns (bool) {
        return relayed[_messageHash];
    }
}

/// @notice Records the caller of EVERY non-static call it receives, whatever the calldata (no
///         functions besides the fallback). Storage: slot 0 = mapping(address caller => uint256
///         count), slot 1 = keccak256 of the last calldata, slot 2 = last caller. Read with
///         vm.load.
contract RecordingMock0815 {
    mapping(address => uint256) internal callsFrom;
    bytes32 internal lastCalldataHash;
    address internal lastCaller;

    fallback() external payable {
        callsFrom[msg.sender] += 1;
        lastCalldataHash = keccak256(msg.data);
        lastCaller = msg.sender;
    }
}
