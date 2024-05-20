// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";

// Target contract
import { StaticConfig } from "src/libraries/StaticConfig.sol";

contract StaticConfig_Test is CommonTest {
    /// @dev Tests gas paying token encoding.
    function testDiff_encodeGasPayingToken_succeeds(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        external
    {
        bytes memory encoding = StaticConfig.encodeSetGasPayingToken(_token, _decimals, _name, _symbol);

        bytes memory _encoding = ffi.encodeGasPayingToken(_token, _decimals, _name, _symbol);

        assertEq(encoding, _encoding);
    }

    /// @dev Tests add dependency encoding.
    function testDiff_addDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = StaticConfig.encodeAddDependency(_chainId);

        bytes memory _encoding = ffi.encodeDependency(_chainId);

        assertEq(encoding, _encoding);
    }

    /// @dev Tests remove dependency encoding.
    function testDiff_removeDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = StaticConfig.encodeRemoveDependency(_chainId);

        bytes memory _encoding = ffi.encodeDependency(_chainId);

        assertEq(encoding, _encoding);
    }
}
