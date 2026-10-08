// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Mocks etched at L2 predeploy addresses for the Kontrol expiry proofs. Every mock is an explicit
// modelling assumption; see README-style comments on each contract.

// Interfaces
import { Vm } from "forge-std/Vm.sol";
import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @notice Cheat codes Kontrol 1.0.255 implements on top of forge's `Vm` (lib/kontrol-cheatcodes in
///         this repo predates freshAddress/freshBytes, so the interface is declared here).
interface KontrolExpiryCheats {
    function freshUInt(uint8) external returns (uint256);
    function freshBool() external returns (uint256);
    function freshAddress() external returns (address);
    function freshBytes(uint256) external returns (bytes memory);
    function symbolicStorage(address) external;
}

abstract contract ExpiryKontrolBaseL2 {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
    KontrolExpiryCheats internal constant kevm =
        KontrolExpiryCheats(address(uint160(uint256(keccak256("hevm cheat code")))));
    address internal constant CONSOLE = 0x000000000000000000636F6e736F6c652e6c6f67;
}

/// @notice Records the caller of EVERY non-static call it receives, whatever the calldata. Etched
///         at the L2CrossDomainMessenger (0x4200..0007) and the L2ToL1MessagePasser (0x4200..0016)
///         for OnlyExportReachesL1. It has no functions besides the fallback, so no selector can
///         bypass the recording. A STATICCALL into it reverts (SSTORE in a static context); a
///         static call cannot initiate a withdrawal, and the L2ToL2CrossDomainMessenger only
///         static-calls 0x..07 from `expireMessage`. Storage: slot 0 = mapping(address caller =>
///         uint256 count), slot 1 = keccak256 of the last calldata, slot 2 = last caller. Read with
///         vm.load.
contract RecordingMock {
    mapping(address => uint256) internal callsFrom;
    bytes32 internal lastCalldataHash;
    address internal lastCaller;

    fallback() external payable {
        callsFrom[msg.sender] += 1;
        lastCalldataHash = keccak256(msg.data);
        lastCaller = msg.sender;
    }
}

/// @notice CrossL2Inbox stand-in: `validateMessage` accepts EVERY (identifier, payload hash). This
///         over-approximates the real inbox (which only accepts identifiers of real initiating
///         messages), so a "relayMessage always reverts" result holds a fortiori for the real
///         inbox.
contract AcceptAllCrossL2Inbox {
    function validateMessage(Identifier calldata, bytes32) external { }
}

/// @notice L2CrossDomainMessenger stand-in for `expireMessage` auth: both getters return whatever
///         is stored (the proofs store fresh symbolic addresses). Slot 0 = xDomainMessageSender,
///         slot 1 = otherMessenger. Over-approximates the real L2CDM (whose getter reverts outside
///         a relay).
contract AuthL2CrossDomainMessenger {
    address internal xSender;
    address internal other;

    function xDomainMessageSender() external view returns (address) {
        return xSender;
    }

    function otherMessenger() external view returns (address) {
        return other;
    }
}
