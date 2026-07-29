// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Target contract dependencies
import { IAddressManager } from "interfaces/legacy/IAddressManager.sol";

// Target contract
import { IResolvedDelegateProxy } from "interfaces/legacy/IResolvedDelegateProxy.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";

contract ResolvedDelegateProxy_SimpleImplementation_Harness {
    function foo(uint256 _x) public pure returns (uint256) {
        return _x;
    }

    function bar() public pure {
        revert("SimpleImplementation: revert");
    }
}

/// @title ResolvedDelegateProxy_TestInit
/// @notice Reusable test initialization for `ResolvedDelegateProxy` tests.
abstract contract ResolvedDelegateProxy_TestInit is Test {
    IAddressManager internal addressManager;
    ResolvedDelegateProxy_SimpleImplementation_Harness internal impl;
    ResolvedDelegateProxy_SimpleImplementation_Harness internal proxy;

    /// @notice Sets up the test suite.
    function setUp() public {
        // Set up the address manager.
        addressManager = IAddressManager(
            DeployUtils.create1({
                _name: "AddressManager",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IAddressManager.__constructor__, ()))
            })
        );
        impl = new ResolvedDelegateProxy_SimpleImplementation_Harness();
        addressManager.setAddress("SimpleImplementation", address(impl));

        // Set up the proxy.
        proxy = ResolvedDelegateProxy_SimpleImplementation_Harness(
            address(
                DeployUtils.create1({
                    _name: "ResolvedDelegateProxy",
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(IResolvedDelegateProxy.__constructor__, (addressManager, "SimpleImplementation"))
                    )
                })
            )
        );
    }
}

/// @title ResolvedDelegateProxy_Fallback_Test
/// @notice Tests the `fallback` function of the `ResolvedDelegateProxy` contract.
contract ResolvedDelegateProxy_Fallback_Test is ResolvedDelegateProxy_TestInit {
    /// @notice Tests that the proxy properly bubbles up returndata when the delegatecall succeeds.
    function testFuzz_fallback_delegateCallFoo_succeeds(uint256 x) public {
        vm.expectCall(address(impl), abi.encodeCall(impl.foo, (x)));
        assertEq(proxy.foo(x), x);
    }

    /// @notice Tests that the proxy properly bubbles up returndata when the delegatecall reverts.
    function test_fallback_delegateCallBar_reverts() public {
        vm.expectRevert("SimpleImplementation: revert");
        vm.expectCall(address(impl), abi.encodeCall(impl.bar, ()));
        proxy.bar();
    }

    /// @notice Tests that the proxy fallback reverts as expected if the implementation within the
    ///         address manager is not set.
    function test_fallback_addressManagerNotSet_reverts() public {
        IAddressManager am = IAddressManager(
            DeployUtils.create1({
                _name: "AddressManager",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IAddressManager.__constructor__, ()))
            })
        );
        ResolvedDelegateProxy_SimpleImplementation_Harness p = ResolvedDelegateProxy_SimpleImplementation_Harness(
            address(
                DeployUtils.create1({
                    _name: "ResolvedDelegateProxy",
                    _args: DeployUtils.encodeConstructor(
                        abi.encodeCall(IResolvedDelegateProxy.__constructor__, (am, "SimpleImplementation"))
                    )
                })
            )
        );

        vm.expectRevert("ResolvedDelegateProxy: target address must be initialized");
        p.foo(0);
    }
}
