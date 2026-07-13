// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Minimal forge cheatcode interface (self-contained, no forge-std).
interface Vm {
    function store(address target, bytes32 slot, bytes32 value) external;
    function deal(address who, uint256 newBalance) external;
    function etch(address who, bytes calldata code) external;
    function label(address who, string calldata newLabel) external;
}

// Input precompile interface (Family B). The Go host installs a reflection precompile at the
// address passed to run(), answering these getters from the OPCMExampleInput struct fields.
interface IOPCMExampleInput {
    function owner() external view returns (address);
    function blob() external view returns (bytes memory);
}

// Output precompile interface (Family B). set(bytes4,address) is the WithFieldSetter setter; the
// result() getter reads back the last value that was set.
interface IOPCMExampleOutput {
    function set(bytes4 sel, address value) external;
    function result() external view returns (address);
}

/// @title OPCMExample
/// @notice A minimal script exercising the OPCM `RunScriptSingle` input/output-precompile
///         mechanism: it reads inputs via getters, mutates state at a FIXED address (so the state
///         dump is script-address-independent, keeping cross-engine byte parity), then writes and
///         reads back the output via the setter/getter.
contract OPCMExample {
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

    // A fixed, script-address-independent account we mutate, so the dump is deterministic.
    address internal constant TARGET = address(uint160(0x00C0FFEE));

    function run(IOPCMExampleInput _in, IOPCMExampleOutput _out) public {
        // Read inputs through the input precompile's getters.
        address owner = _in.owner();
        bytes memory data = _in.blob();

        // Mutate a fixed account deterministically via state-cheats.
        vm.store(TARGET, bytes32(uint256(1)), bytes32(data.length));
        vm.deal(TARGET, uint256(uint160(owner)));
        vm.etch(TARGET, hex"600160015500");
        vm.label(TARGET, "target");

        // Write the output through the setter, then read it back through the getter.
        _out.set(_out.result.selector, TARGET);
        require(_out.result() == TARGET, "OPCMExample: result mismatch");
    }
}

/// @title OPCMExampleVoid
/// @notice The input-only (RunScriptVoid) counterpart: reads inputs via getters and mutates a
///         fixed account, with no output precompile. Mirrors the real Void scripts
///         (UpgradeOPChain, SetDisputeGameImpl).
contract OPCMExampleVoid {
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    Vm internal constant vm = Vm(VM_ADDRESS);

    address internal constant TARGET = address(uint160(0x00C0FFEE));

    function run(IOPCMExampleInput _in) public {
        address owner = _in.owner();
        bytes memory data = _in.blob();
        vm.store(TARGET, bytes32(uint256(2)), bytes32(data.length + 1));
        vm.deal(TARGET, uint256(uint160(owner)) + 1);
        vm.etch(TARGET, hex"5b00");
        vm.label(TARGET, "target-void");
    }
}
