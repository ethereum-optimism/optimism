// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Bytes32AddressLib } from "@rari-capital/solmate/src/utils/Bytes32AddressLib.sol";

// Target contract dependencies
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Target contract
import { CrossDomainOwnable3 } from "src/L2/CrossDomainOwnable3.sol";

/// @title XDomainSetter3
/// @notice A test contract that extends `CrossDomainOwnable3` to test ownership functionality.
contract XDomainSetter3 is CrossDomainOwnable3 {
    uint256 public value;

    function set(uint256 _value) external onlyOwner {
        value = _value;
    }
}

/// @title CrossDomainOwnable3_TestInit
/// @notice Reusable test initialization for `CrossDomainOwnable3` tests.
abstract contract CrossDomainOwnable3_TestInit is CommonTest {
    XDomainSetter3 setter;

    /// @notice CrossDomainOwnable3.sol transferOwnership event
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner, bool isLocal);

    function setUp() public virtual override {
        super.setUp();
        vm.prank(alice);
        setter = new XDomainSetter3();
    }
}

/// @title CrossDomainOwnable3_BasicFunctionality_Test
/// @notice Tests for the basic functionality of the `CrossDomainOwnable3` contract.
contract CrossDomainOwnable3_BasicFunctionality_Test is CrossDomainOwnable3_TestInit {
    /// @notice Tests that the constructor sets the correct variables.
    function test_constructor_succeeds() public view {
        assertEq(setter.owner(), alice);
        assertEq(setter.isLocal(), true);
    }

    /// @notice Tests that `set` reverts when the caller is not the owner.
    function test_localOnlyOwner_notOwner_reverts() public {
        vm.prank(bob);
        vm.expectRevert("CrossDomainOwnable3: caller is not the owner");
        setter.set(1);
    }

    /// @notice Tests that `onlyOwner` allows the owner to call a protected function locally.
    function test_localOnlyOwner_succeeds() public {
        assertEq(setter.isLocal(), true);
        vm.prank(setter.owner());
        setter.set(1);
        assertEq(setter.value(), 1);
    }
}

/// @title CrossDomainOwnable3_TransferOwnership_Test
/// @notice Tests for the ownership transfer functionality of the `CrossDomainOwnable3` contract.
contract CrossDomainOwnable3_TransferOwnership_Test is CrossDomainOwnable3_TestInit {
    /// @notice Tests that `transferOwnership` reverts when the caller is not the owner.
    function test_transferOwnership_notOwner_reverts() public {
        vm.prank(bob);
        vm.expectRevert("CrossDomainOwnable3: caller is not the owner");
        setter.transferOwnership({ _owner: bob, _isLocal: true });
    }

    /// @notice Tests that `transferOwnership` reverts for ownership transfers to the zero address
    ///         when set locally.
    function test_transferOwnership_zeroAddress_reverts() public {
        vm.prank(setter.owner());
        vm.expectRevert("CrossDomainOwnable3: new owner is the zero address");
        setter.transferOwnership({ _owner: address(0), _isLocal: true });
    }

    /// @notice Tests that `transferOwnership` reverts for ownership transfers to the zero address.
    function test_transferOwnership_noLocalZeroAddress_reverts() public {
        vm.prank(setter.owner());
        vm.expectRevert("Ownable: new owner is the zero address");
        setter.transferOwnership(address(0));
    }

    /// @notice Tests that `transferOwnership` succeeds when the caller is the owner and the
    ///         ownership is transferred locally.
    function test_localTransferOwnership_succeeds() public {
        vm.expectEmit(true, true, true, true, address(setter));
        emit OwnershipTransferred(alice, bob);
        emit OwnershipTransferred(alice, bob, true);

        vm.prank(setter.owner());
        setter.transferOwnership({ _owner: bob, _isLocal: true });

        assertEq(setter.isLocal(), true);

        vm.prank(bob);
        setter.set(2);
        assertEq(setter.value(), 2);
    }

    /// @notice The existing transferOwnership(address) method still exists on the contract.
    function test_transferOwnershipNoLocal_succeeds() public {
        bool isLocal = setter.isLocal();

        vm.expectEmit(true, true, true, true, address(setter));
        emit OwnershipTransferred(alice, bob);

        vm.prank(setter.owner());
        setter.transferOwnership(bob);

        // isLocal has not changed
        assertEq(setter.isLocal(), isLocal);

        vm.prank(bob);
        setter.set(2);
        assertEq(setter.value(), 2);
    }

    /// @notice Tests that `transferOwnership` succeeds when the caller is the owner and the
    ///         ownership is transferred non-locally.
    function test_crossDomainTransferOwnership_succeeds() public {
        vm.expectEmit(true, true, true, true, address(setter));
        emit OwnershipTransferred(alice, bob);
        emit OwnershipTransferred(alice, bob, false);

        vm.prank(setter.owner());
        setter.transferOwnership({ _owner: bob, _isLocal: false });

        assertEq(setter.isLocal(), false);

        // Simulate the L2 execution where the call is coming from the L1CrossDomainMessenger
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(1, 1), bob, address(setter), 0, 0, abi.encodeCall(XDomainSetter3.set, (2))
        );

        assertEq(setter.value(), 2);
    }
}

/// @title CrossDomainOwnable3_CrossDomainAccess_Test
/// @notice Tests for cross-domain access control in the `CrossDomainOwnable3` contract.
contract CrossDomainOwnable3_CrossDomainAccess_Test is CrossDomainOwnable3_TestInit {
    /// @notice Tests that the `XDomainSetter3` contract reverts after the ownership has been
    ///         transferred to a new owner.
    function test_crossDomainOnlyOwner_notOwner_reverts() public {
        vm.expectEmit(true, true, true, true);

        // OpenZeppelin Ownable.sol transferOwnership event
        emit OwnershipTransferred(alice, alice);

        // CrossDomainOwnable3.sol transferOwnership event
        emit OwnershipTransferred(alice, alice, false);

        vm.prank(setter.owner());
        setter.transferOwnership({ _owner: alice, _isLocal: false });

        // set the xDomainMsgSender storage slot
        bytes32 key = bytes32(uint256(204));
        bytes32 value = Bytes32AddressLib.fillLast12Bytes(bob);
        vm.store(address(l2CrossDomainMessenger), key, value);

        vm.prank(address(l2CrossDomainMessenger));
        vm.expectRevert("CrossDomainOwnable3: caller is not the owner");
        setter.set(1);
    }

    /// @notice Tests that a relayed message to the `XDomainSetter3` contract reverts after its
    ///         ownership has been transferred to a new owner.
    function test_crossDomainOnlyOwner_notOwner2_reverts() public {
        vm.expectEmit(true, true, true, true);

        // OpenZeppelin Ownable.sol transferOwnership event
        emit OwnershipTransferred(alice, alice);

        // CrossDomainOwnable3.sol transferOwnership event
        emit OwnershipTransferred(alice, alice, false);

        vm.prank(setter.owner());
        setter.transferOwnership({ _owner: alice, _isLocal: false });

        assertEq(setter.isLocal(), false);

        uint240 nonce = 0;
        address sender = bob;
        address target = address(setter);
        uint256 value = 0;
        uint256 minGasLimit = 0;
        bytes memory message = abi.encodeCall(XDomainSetter3.set, (1));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        // It should be a failed message. The revert is caught, so we cannot expectRevert here.
        vm.expectEmit(true, true, true, true, address(l2CrossDomainMessenger));
        emit FailedRelayedMessage(hash);

        vm.prank(AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(nonce, 1), sender, target, value, minGasLimit, message
        );

        assertEq(setter.value(), 0);
    }

    /// @notice Tests that the `XDomainSetter3` contract reverts for a non-messenger caller after
    ///         the ownership has been transferred to a new owner.
    function test_crossDomainOnlyOwner_notMessenger_reverts() public {
        vm.expectEmit(true, true, true, true);

        // OpenZeppelin Ownable.sol transferOwnership event
        emit OwnershipTransferred(alice, alice);

        // CrossDomainOwnable3.sol transferOwnership event
        emit OwnershipTransferred(alice, alice, false);

        vm.prank(setter.owner());
        setter.transferOwnership({ _owner: alice, _isLocal: false });

        vm.prank(bob);
        vm.expectRevert("CrossDomainOwnable3: caller is not the messenger");
        setter.set(1);
    }
}
