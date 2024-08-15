// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Vm is a minimal interface to the forge cheatcode precompile
interface Vm {
    function getNonce(address account) external view returns (uint64 nonce);
}

// console is a minimal version of the console2 lib.
library console {
    address constant CONSOLE_ADDRESS = address(0x000000000000000000636F6e736F6c652e6c6f67);

    function _castLogPayloadViewToPure(
        function(bytes memory) internal view fnIn
    ) internal pure returns (function(bytes memory) internal pure fnOut) {
        assembly {
            fnOut := fnIn
        }
    }

    function _sendLogPayload(bytes memory payload) internal pure {
        _castLogPayloadViewToPure(_sendLogPayloadView)(payload);
    }

    function _sendLogPayloadView(bytes memory payload) private view {
        uint256 payloadLength = payload.length;
        address consoleAddress = CONSOLE_ADDRESS;
        /// @solidity memory-safe-assembly
        assembly {
            let payloadStart := add(payload, 32)
            let r := staticcall(gas(), consoleAddress, payloadStart, payloadLength, 0, 0)
        }
    }

    function log(string memory p0) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string)", p0));
    }

    function log(string memory p0, uint256 p1) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,uint256)", p0, p1));
    }

    function log(string memory p0, address p1) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,address)", p0, p1));
    }
}

/// @title ScriptExample
/// @notice ScriptExample is an example script. The Go forge script code tests that it can run this.
contract ScriptExample {

    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

    /// @notice example function, runs through basic cheat-codes and console logs.
    function run() public view {
        console.log("contract addr", address(this));
        console.log("contract nonce", vm.getNonce(address(this)));
        console.log("sender addr", address(msg.sender));
        console.log("sender nonce", vm.getNonce(address(msg.sender)));
        console.log("done!");
    }
}
