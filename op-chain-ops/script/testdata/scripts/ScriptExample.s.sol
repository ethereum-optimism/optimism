// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Vm is a minimal interface to the forge cheatcode precompile
interface Vm {
    function envOr(string calldata name, bool defaultValue) external view returns (bool value);
    function getNonce(address account) external view returns (uint64 nonce);
    function parseJsonKeys(string calldata json, string calldata key) external pure returns (string[] memory keys);
    function startPrank(address msgSender) external;
    function stopPrank() external;
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

    function log(string memory p0, bool p1) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,bool)", p0, p1));
    }

    function log(string memory p0, uint256 p1) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,uint256)", p0, p1));
    }

    function log(string memory p0, address p1) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,address)", p0, p1));
    }

    function log(string memory p0, string memory p1, string memory p2) internal pure {
        _sendLogPayload(abi.encodeWithSignature("log(string,string,string)", p0, p1, p2));
    }
}

/// @title ScriptExample
/// @notice ScriptExample is an example script. The Go forge script code tests that it can run this.
contract ScriptExample {

    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

    /// @notice example function, runs through basic cheat-codes and console logs.
    function run() public {
        bool x = vm.envOr("EXAMPLE_BOOL", false);
        console.log("bool value from env", x);

        console.log("contract addr", address(this));
        console.log("contract nonce", vm.getNonce(address(this)));
        console.log("sender addr", address(msg.sender));
        console.log("sender nonce", vm.getNonce(address(msg.sender)));

        string memory json = '{"root_key": [{"a": 1, "b": 2}]}';
        string[] memory keys = vm.parseJsonKeys(json, ".root_key[0]");
        console.log("keys", keys[0], keys[1]);

        this.hello("from original");
        vm.startPrank(address(uint160(0x42)));
        this.hello("from prank 1");
        console.log("parent scope msg.sender", address(msg.sender));
        console.log("parent scope contract.addr", address(this));
        this.hello("from prank 2");
        vm.stopPrank();
        this.hello("from original again");

        console.log("done!");
    }

    /// @notice example external function, to force a CALL, and test vm.startPrank with.
    function hello(string calldata _v) external view {
        console.log(_v);
        console.log("hello msg.sender", address(msg.sender));
    }
}
