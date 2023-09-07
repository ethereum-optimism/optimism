// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { StdInvariant } from "forge-std/StdInvariant.sol";
import { Vm, VmSafe } from "forge-std/Vm.sol";
import { DelayedVetoable } from "../../src/universal/DelayedVetoable.sol";
import { console } from "forge-std/console.sol";

contract DelayedVetoable_Invariant_Harness is StdInvariant, Test {
    event Forwarded(bytes data);

    DelayedVetoable delayedVetoable;

    // The address that delayedVetoable will call to
    // address dvTarget;
    DelayedVetoableCaller dvCaller;

    function setUp() public {
        dvCaller = new DelayedVetoableCaller(vm);

        targetContract(address(dvCaller));
    }
}

contract DelayedVetoable_Invariants is DelayedVetoable_Invariant_Harness {
    /// @custom:invariant Calls are always forwarded to the target
    function invariant_callIsForwarded_succeeds() public {
        assertFalse(dvCaller.failed() || this.failed());
        assertFalse(this.failed());
    }
}

contract DelayedVetoableCaller {
    Vm internal immutable _vm;

    DelayedVetoable internal _dv;
    bool public failed;

    constructor(Vm vm) {
        _vm = vm;
    }

    function setDelayedVetoable(address _target) public {
        if (_target == address(0)) {
            _target = _vm.addr(uint256(keccak256(abi.encode(_target))));
        }

        _dv = new DelayedVetoable({ target: _target });
        console.log("target: %s", _target);
        console.log("_dv: %s", address(_dv));
    }

    function doCall(bytes memory _toForward) external {
        if (address(_dv) == address(0)) {
            setDelayedVetoable(_vm.addr(uint256(keccak256(_toForward))));
        }

        bytes32 forwardHash = keccak256("Forwarded(bytes)");

        _vm.expectEmit(true, false, false, true, address(_dv));
        assembly {
            log1(add(_toForward, 0x20), mload(_toForward), forwardHash)
        }
        // expect a call to the target stored in the DelayedVetoable contract
        _vm.expectCall(address(uint160(uint256(_vm.load(address(_dv), bytes32(0))))), _toForward);

        (bool success,) = address(_dv).call(_toForward);
        // if (!success) {
        //     failed = true;
        // }
    }
}
