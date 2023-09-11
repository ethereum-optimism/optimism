// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "./CommonTest.t.sol";

// Libraries
import { Constants } from "../src/libraries/Constants.sol";

// Target contract dependencies
import { Proxy } from "../src/universal/Proxy.sol";

// Target contract
import { ProtocolVersions, ProtocolVersion } from "../src/L1/ProtocolVersions.sol";

contract ProtocolVersions_Init is CommonTest {
    ProtocolVersions protocolVersions;
    ProtocolVersions protocolVersionsImpl;

    event ConfigUpdate(uint256 indexed version, ProtocolVersions.UpdateType indexed updateType, bytes data);

    // Dummy values used to test getters
    ProtocolVersion constant required = ProtocolVersion.wrap(uint256(0xabcd));
    ProtocolVersion constant recommended = ProtocolVersion.wrap(uint256(0x1234));

    function setUp() public virtual override {
        super.setUp();

        Proxy proxy = new Proxy(multisig);
        protocolVersionsImpl = new ProtocolVersions();

        vm.prank(multisig);
        proxy.upgradeToAndCall(
            address(protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (
                    alice, // _owner,
                    required,
                    recommended
                )
            )
        );

        protocolVersions = ProtocolVersions(address(proxy));
    }
}

contract ProtocolVersions_Initialize_Test is ProtocolVersions_Init {
    /// @dev Tests that initialization sets the correct values.
    function test_initialize_values_succeeds() external {
        assertEq(protocolVersions.required(), required);
        assertEq(protocolVersions.recommended(), recommended);
    }

    /// @dev Ensures that the events are emitted during initialization.
    function test_initialize_events_succeeds() external {
        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(protocolVersions), bytes32(0), bytes32(0));
        vm.store(address(protocolVersions), bytes32(uint256(106)), bytes32(0));
        assertEq(protocolVersions.owner(), address());

        // The order depends here
        vm.expectEmit(true, true, true, true, address(protocolVersions));
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.REQUIRED_PROTOCOL_VERSION, abi.encode(required));
        vm.expectEmit(true, true, true, true, address(protocolVersions));
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.RECOMMENDED_PROTOCOL_VERSION, abi.encode(recommended));

        vm.prank(multisig);
        Proxy(payable(address(protocolVersions))).upgradeToAndCall(
            address(protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (
                    alice, // _owner
                    required, // _required
                    recommended, // recommended
                )
            )
        );
    }
}

contract ProtocolVersions_Initialize_TestFail is ProtocolVersions_Init {
    /// @dev Tests that initialization reverts if the gas limit is too low.
    function test_initialize_lowGasLimit_reverts() external {
        uint64 minimumGasLimit = protocolVersions.minimumGasLimit();

        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(protocolVersions), bytes32(0), bytes32(0));
        vm.prank(multisig);
        // The call to initialize reverts due to: "ProtocolVersions: gas limit too low"
        // but the proxy revert message bubbles up.
        vm.expectRevert("Proxy: delegatecall to new implementation contract failed");
        Proxy(payable(address(protocolVersions))).upgradeToAndCall(
            address(protocolVersionsImpl),
            abi.encodeCall(
                ProtocolVersions.initialize,
                (
                    alice, // _owner
                    required, // _required
                    recommended, // recommended
                )
            )
        );
    }
}

contract ProtocolVersions_Setters_TestFail is ProtocolVersions_Init {
    /// @dev Tests that `setRequired` reverts if the caller is not the owner.
    function test_setRequired_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        protocolVersions.setRequired(bytes32(hex""));
    }

    /// @dev Tests that `setRecommended` reverts if the caller is not the owner.
    function test_setRecommended_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        protocolVersions.setRecommended(bytes32(hex""));
    }
}

contract ProtocolVersions_Setters_Test is ProtocolVersions_Init {
    /// @dev Tests that `setRequired` updates the required protocol version successfully.
    function testFuzz_setRequired_succeeds(bytes32 newProtocolVersion) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.REQUIRED_PROTOCOL_VERSION, abi.encode(newProtocolVersion));

        vm.prank(protocolVersions.owner());
        protocolVersions.setRequired(newProtocolVersion);
        assertEq(protocolVersions.required(), newProtocolVersion);
    }

    /// @dev Tests that `setRecommended` updates the recommended protocol version successfully.
    function testFuzz_setRecommended_succeeds(bytes32 newProtocolVersion) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.RECOMMENDED_PROTOCOL_VERSION, abi.encode(newProtocolVersion));

        vm.prank(protocolVersions.owner());
        protocolVersions.setRecommended(newProtocolVersion);
        assertEq(protocolVersions.recommended(), newProtocolVersion);
    }
}
