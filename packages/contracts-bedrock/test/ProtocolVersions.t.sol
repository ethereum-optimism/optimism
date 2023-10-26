// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/CommonTest.t.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";

// Target contract dependencies
import { Proxy } from "src/universal/Proxy.sol";

// Target contract
import { ProtocolVersions, ProtocolVersion } from "src/L1/ProtocolVersions.sol";

contract ProtocolVersions_Init is CommonTest {
    ProtocolVersions protocolVersions;
    ProtocolVersions protocolVersionsImpl;

    event ConfigUpdate(uint256 indexed version, ProtocolVersions.UpdateType indexed updateType, bytes data);

    // Dummy values used to test getters
    ProtocolVersion constant required = ProtocolVersion.wrap(0xabcd);
    ProtocolVersion constant recommended = ProtocolVersion.wrap(0x1234);

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
        assertEq(ProtocolVersion.unwrap(protocolVersions.required()), ProtocolVersion.unwrap(required));
        assertEq(ProtocolVersion.unwrap(protocolVersions.recommended()), ProtocolVersion.unwrap(recommended));
        assertEq(protocolVersions.owner(), alice);

        assertEq(ProtocolVersion.unwrap(protocolVersionsImpl.required()), 0);
        assertEq(ProtocolVersion.unwrap(protocolVersionsImpl.recommended()), 0);
        assertEq(protocolVersionsImpl.owner(), address(0xdEad));
    }

    /// @dev Ensures that the events are emitted during initialization.
    function test_initialize_events_succeeds() external {
        assertEq(protocolVersionsImpl.owner(), address(0xdEad));

        // Wipe out the initialized slot so the proxy can be initialized again
        vm.store(address(protocolVersions), bytes32(0), bytes32(0));

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
                    recommended // recommended
                )
            )
        );
    }
}

contract ProtocolVersions_Setters_TestFail is ProtocolVersions_Init {
    /// @dev Tests that `setRequired` reverts if the caller is not the owner.
    function test_setRequired_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        protocolVersions.setRequired(ProtocolVersion.wrap(0));
    }

    /// @dev Tests that `setRecommended` reverts if the caller is not the owner.
    function test_setRecommended_notOwner_reverts() external {
        vm.expectRevert("Ownable: caller is not the owner");
        protocolVersions.setRecommended(ProtocolVersion.wrap(0));
    }
}

contract ProtocolVersions_Setters_Test is ProtocolVersions_Init {
    /// @dev Tests that `setRequired` updates the required protocol version successfully.
    function testFuzz_setRequired_succeeds(uint256 _version) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.REQUIRED_PROTOCOL_VERSION, abi.encode(_version));

        vm.prank(protocolVersions.owner());
        protocolVersions.setRequired(ProtocolVersion.wrap(_version));
        assertEq(ProtocolVersion.unwrap(protocolVersions.required()), _version);
    }

    /// @dev Tests that `setRecommended` updates the recommended protocol version successfully.
    function testFuzz_setRecommended_succeeds(uint256 _version) external {
        vm.expectEmit(true, true, true, true);
        emit ConfigUpdate(0, ProtocolVersions.UpdateType.RECOMMENDED_PROTOCOL_VERSION, abi.encode(_version));

        vm.prank(protocolVersions.owner());
        protocolVersions.setRecommended(ProtocolVersion.wrap(_version));
        assertEq(ProtocolVersion.unwrap(protocolVersions.recommended()), _version);
    }
}
