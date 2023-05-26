pragma solidity 0.8.15;

import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";

contract EchidnaFuzzAddressAliasing {
    bool internal failedRoundtrip;

    /**
     * @notice Takes an address to be aliased with AddressAliasHelper and then unaliased
     *         and updates the test contract's state indicating if the round trip encoding
     *         failed.
     */
    function testRoundTrip(address addr) public {
        // Alias our address
        address aliasedAddr = AddressAliasHelper.applyL1ToL2Alias(addr);

        // Unalias our address
        address undoneAliasAddr = AddressAliasHelper.undoL1ToL2Alias(aliasedAddr);

        // If our round trip aliasing did not return the original result, set our state.
        if (addr != undoneAliasAddr) {
            failedRoundtrip = true;
        }
    }

    /**
     * @custom:invariant Address aliases are always able to be undone.
     *
     * Asserts that an address that has been aliased with `applyL1ToL2Alias` can always
     * be unaliased with `undoL1ToL2Alias`.
     */
    function echidna_round_trip_aliasing() public view returns (bool) {
        // ASSERTION: The round trip aliasing done in testRoundTrip(...) should never fail.
        return !failedRoundtrip;
    }
}
