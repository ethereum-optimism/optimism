// SPDX-License-Identifier: MIT
pragma solidity ^0.8.13;

contract DummyTest {
    function prove_success() public pure {
        assert(true);
    }

    function prove_fail() public pure {
        assert(false);
    }
}
