// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

library Transient {
    uint256 internal constant CONTEXT_SLOT = 0;

    function getTransientContext() internal view returns (uint256 ctx) {
        assembly {
            mstore(0x00, tload(CONTEXT_SLOT))
            ctx := keccak256(0x00, 0x20)
        }
    }

    function setTransientValue(uint256 value, address target, bytes memory payload) public {
        assembly {
            tstore(CONTEXT_SLOT, add(tload(CONTEXT_SLOT), 1))
        }

        uint256 ctx = getTransientContext();

        assembly {
            tstore(ctx, value)
        }

        if (target == address(0)) return;

        (bool success,) = target.call(payload);

        require(success, "setTransientValue::call");
    }

    function getTransientValue() public view returns (uint256 value) {
        uint256 ctx = getTransientContext();

        assembly {
            value := tload(ctx)
        }
    }
}
