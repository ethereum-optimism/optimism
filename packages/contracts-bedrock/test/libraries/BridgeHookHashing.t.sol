// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Libraries
import { BridgeHookHashing } from "src/libraries/BridgeHookHashing.sol";
import { BridgeHookTypes } from "src/libraries/BridgeHookTypes.sol";

/// @title BridgeHookHashing_hashItem_Test
/// @notice Tests the `hashItem` function of the `BridgeHookHashing` library.
contract BridgeHookHashing_hashItem_Test is Test {
    /// @notice Tests that a fixed item hashes to an ID computed offchain, pinning the encoding.
    function test_hashItem_knownItem_succeeds() external pure {
        BridgeHookTypes.Item memory item = BridgeHookTypes.Item({
            direction: BridgeHookTypes.Direction.Withdrawal,
            asset: BridgeHookTypes.Asset.ERC20,
            from: 0x1111111111111111111111111111111111111111,
            aliased: true,
            to: 0x2222222222222222222222222222222222222222,
            ethAmount: 3,
            l2Value: 4,
            gasLimit: 5,
            isCreation: true,
            data: hex"deadbeef",
            nonce: 6
        });

        // Computed offchain with cast, as the keccak hash of the ABI encoded item above.
        assertEq(BridgeHookHashing.hashItem(item), 0x3a78d36b9abbdf147902f081af3e3c02f07c7a6599e30e32bb78d906bb878356);
    }
}
