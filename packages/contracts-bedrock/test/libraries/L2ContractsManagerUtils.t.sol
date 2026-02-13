// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { L2ContractsManagerUtils } from "src/libraries/L2ContractsManagerUtils.sol";

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Contracts
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IStorageSetter } from "interfaces/universal/IStorageSetter.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title L2ContractsManagerUtils_ImplV1_Harness
/// @notice Implementation contract with version 1.0.0 for testing upgrades.
contract L2ContractsManagerUtils_ImplV1_Harness is ISemver {
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice It is a no-op for this test.
    function initialize() external { }
}

/// @title L2ContractsManagerUtils_ImplV2_Harness
/// @notice Implementation contract with version 2.0.0 for testing upgrades.
contract L2ContractsManagerUtils_ImplV2_Harness is ISemver {
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice It is a no-op for this test.
    function initialize() external { }
}

/// @title L2ContractsManagerUtils_UpgradeToAndCall_Test
/// @notice Tests the `L2ContractsManagerUtils.upgradeToAndCall` function.
contract L2ContractsManagerUtils_UpgradeToAndCall_Test is CommonTest {
    bytes32 internal constant INITIALIZABLE_SLOT_OZ_V4 = bytes32(0);

    bytes32 internal constant INITIALIZABLE_SLOT_OZ_V5 =
        0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00;

    address internal _storageSetterImpl;

    address internal implV1;
    address internal implV2;

    function setUp() public override {
        super.setUp();
        implV1 = address(new L2ContractsManagerUtils_ImplV1_Harness());
        implV2 = address(new L2ContractsManagerUtils_ImplV2_Harness());

        _storageSetterImpl = address(
            IStorageSetter(
                DeployUtils.create1({
                    _name: "StorageSetter",
                    _args: DeployUtils.encodeConstructor(abi.encodeCall(IStorageSetter.__constructor__, ()))
                })
            )
        );
    }

    /// @notice External wrapper so vm.expectRevert can catch reverts from the internal library call.
    function _callUpgradeToAndCall(
        address _proxy,
        address _implementation,
        address _storageSetter,
        bytes memory _data,
        bytes32 _slot,
        uint8 _offset
    )
        external
    {
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(_proxy, _implementation, _storageSetter, _data, _slot, _offset);
        vm.stopPrank();
    }

    /// @notice External wrapper so vm.expectRevert can catch reverts from the internal library call.
    function _callUpgradeTo(address _proxy, address _implementation) external {
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeTo(_proxy, _implementation);
        vm.stopPrank();
    }

    /// @notice Tests that v4 contracts are unaffected by the v5 slot clearing logic. For v4
    ///         contracts the ERC-7201 slot is all zeros, so the new code is a no-op.
    function test_upgrade_v4ContractStillWorks_succeeds() public {
        address proxy = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // Upgrade to v1.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Verify the ERC-7201 slot is zero.
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V5), bytes32(0));

        // Upgrade to v2 should succeed and the ERC-7201 slot should remain zero.
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );
        vm.stopPrank();

        vm.prank(Predeploys.PROXY_ADMIN);
        assertEq(IProxy(payable(proxy)).implementation(), address(implV2));
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V5), bytes32(0));
    }

    /// @notice Tests that a v5 contract with `_initialized = 1` at the ERC-7201 slot gets cleared.
    function test_upgrade_v5SlotCleared_succeeds() public {
        address proxy = Predeploys.SEQUENCER_FEE_WALLET;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Simulate a v5 contract with _initialized = 1 at the ERC-7201 slot.
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V5, bytes32(uint256(1)));

        // Upgrade to v2 should succeed.
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );
        vm.stopPrank();

        vm.prank(Predeploys.PROXY_ADMIN);
        assertEq(IProxy(payable(proxy)).implementation(), address(implV2));
        // The v5 _initialized field should have been cleared.
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V5), bytes32(0));
    }

    /// @notice Tests that a v5 contract with `_initialized = type(uint64).max` (from
    ///         `_disableInitializers()`) gets cleared.
    function test_upgrade_v5SlotMaxInitialized_succeeds() public {
        address proxy = Predeploys.SEQUENCER_FEE_WALLET;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Simulate a v5 contract with _initialized = type(uint64).max (disabled initializers).
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V5, bytes32(uint256(type(uint64).max)));

        // Upgrade to v2 should succeed.
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );
        vm.stopPrank();

        vm.prank(Predeploys.PROXY_ADMIN);
        assertEq(IProxy(payable(proxy)).implementation(), address(implV2));
        // The v5 _initialized field should have been cleared.
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V5), bytes32(0));
    }

    /// @notice Tests that upgrade reverts when v4 `_initializing` bool is set.
    function test_upgrade_v4InitializingDuringUpgrade_reverts() public {
        address proxy = Predeploys.L2_STANDARD_BRIDGE;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Simulate a v4 contract that is mid-initialization. The _initializing bool is at byte
        // offset 1 (right after _initialized uint8 at byte 0). Set _initialized = 1 and _initializing = true.
        uint256 v4Value = 1 | (uint256(1) << 8);
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V4, bytes32(v4Value));

        vm.expectRevert(L2ContractsManagerUtils.L2ContractsManager_InitializingDuringUpgrade.selector);
        this._callUpgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );
    }

    /// @notice Tests that v4 upgrade works with arbitrary `_offset` and slot contents, clearing
    ///         only the target byte while preserving all other bytes in the slot.
    function testFuzz_upgrade_v4NonZeroOffset_succeeds(uint256 _slotValue, uint8 _offset) public {
        _offset = uint8(bound(_offset, 0, 30));

        address proxy = Predeploys.L2_STANDARD_BRIDGE;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Clear the `_initializing` byte at offset+1 to avoid the revert, and set the
        // `_initialized` byte at offset to a nonzero value. All other bytes are fuzzed.
        uint256 mask = uint256(0xFFFF) << (_offset * 8);
        uint256 v4Value = (_slotValue & ~mask) | (uint256(1) << (_offset * 8));
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V4, bytes32(v4Value));

        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V4,
            _offset
        );
        vm.stopPrank();

        vm.prank(Predeploys.PROXY_ADMIN);
        assertEq(IProxy(payable(proxy)).implementation(), address(implV2));
        // Only the _initialized byte at offset should be cleared, all other bytes preserved.
        uint256 expected = v4Value & ~(uint256(0xFF) << (_offset * 8));
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V4), bytes32(expected));
    }

    /// @notice Tests that v4 upgrade reverts when `_initializing` is set at a non-zero offset.
    function test_upgrade_v4InitializingNonZeroOffset_reverts() public {
        address proxy = Predeploys.L2_STANDARD_BRIDGE;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Simulate a v4 contract where `_initialized` is at byte offset 2 and `_initializing`
        // is at byte offset 3. Set both to nonzero.
        uint8 offset = 2;
        uint256 v4Value = (uint256(1) << (offset * 8)) | (uint256(1) << ((offset + 1) * 8));
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V4, bytes32(v4Value));

        vm.expectRevert(L2ContractsManagerUtils.L2ContractsManager_InitializingDuringUpgrade.selector);
        this._callUpgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V4,
            offset
        );
    }

    /// @notice Tests that upgrade reverts when v5 slot is passed with a non-zero offset.
    function testFuzz_upgrade_v5InvalidOffset_reverts(uint8 _offset) public {
        vm.assume(_offset != 0);

        address proxy = Predeploys.SEQUENCER_FEE_WALLET;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        vm.expectRevert(L2ContractsManagerUtils.L2ContractsManager_InvalidV5Offset.selector);
        this._callUpgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V5,
            _offset
        );
    }

    /// @notice Tests that upgrade reverts when `_initializing` bool is set at the ERC-7201 slot.
    function test_upgrade_v5InitializingDuringUpgrade_reverts() public {
        address proxy = Predeploys.SEQUENCER_FEE_WALLET;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Simulate a v5 contract that is mid-initialization. The _initializing bool is at byte
        // offset 8 (bit 64). Set _initialized = 1 and _initializing = true.
        uint256 v5Value = 1 | (uint256(1) << 64);
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V5, bytes32(v5Value));

        vm.expectRevert(L2ContractsManagerUtils.L2ContractsManager_InitializingDuringUpgrade.selector);
        this._callUpgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );
    }

    /// @notice Tests that the upper bytes of the ERC-7201 slot beyond the Initializable struct
    ///         are preserved when clearing the `_initialized` field.
    function test_upgrade_v5SlotPreservesUpperBytes_succeeds() public {
        address proxy = Predeploys.SEQUENCER_FEE_WALLET;

        // Set v1 as current implementation.
        vm.prank(Predeploys.PROXY_ADMIN);
        IProxy(payable(proxy)).upgradeTo(implV1);

        // Set the v5 slot with _initialized = 1 in the low 8 bytes and some data in the upper
        // bytes (above the _initializing bool at byte offset 8). Bytes 9+ are unused by the
        // Initializable struct but should be preserved.
        uint256 upperData = uint256(0xDEADBEEF) << 128;
        uint256 v5Value = upperData | 1;
        vm.store(proxy, INITIALIZABLE_SLOT_OZ_V5, bytes32(v5Value));

        // Upgrade to v2 should succeed.
        vm.startPrank(Predeploys.PROXY_ADMIN);
        L2ContractsManagerUtils.upgradeToAndCall(
            proxy,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V5,
            0
        );
        vm.stopPrank();

        vm.prank(Predeploys.PROXY_ADMIN);
        assertEq(IProxy(payable(proxy)).implementation(), address(implV2));
        // The upper bytes should be preserved, only the low 8 bytes should be zeroed.
        assertEq(vm.load(proxy, INITIALIZABLE_SLOT_OZ_V5), bytes32(upperData));
    }

    /// @notice Tests that upgradeTo reverts for a non-upgradeable predeploy (WETH is not proxied).
    function test_upgradeTo_notUpgradeable_reverts() public {
        vm.expectRevert(
            abi.encodeWithSelector(L2ContractsManagerUtils.L2ContractsManager_NotUpgradeable.selector, Predeploys.WETH)
        );
        this._callUpgradeTo(Predeploys.WETH, implV1);
    }

    /// @notice Tests that upgradeToAndCall reverts for a non-upgradeable predeploy (WETH is not proxied).
    function test_upgradeToAndCall_notUpgradeable_reverts() public {
        vm.expectRevert(
            abi.encodeWithSelector(L2ContractsManagerUtils.L2ContractsManager_NotUpgradeable.selector, Predeploys.WETH)
        );
        this._callUpgradeToAndCall(
            Predeploys.WETH,
            implV2,
            _storageSetterImpl,
            abi.encodeCall(L2ContractsManagerUtils_ImplV2_Harness.initialize, ()),
            INITIALIZABLE_SLOT_OZ_V4,
            0
        );
    }
}
