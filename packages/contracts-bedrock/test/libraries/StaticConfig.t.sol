// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { Test } from "forge-std/Test.sol";
import { FFIInterface } from "test/setup/FFIInterface.sol";

// Target contract
import { StaticConfig } from "src/libraries/StaticConfig.sol";

/// @title StaticConfig_TestInit
/// @notice Reusable test initialization for `StaticConfig` tests.
abstract contract StaticConfig_TestInit is Test {
    FFIInterface constant ffi = FFIInterface(address(uint160(uint256(keccak256(abi.encode("optimism.ffi"))))));

    function setUp() public {
        vm.etch(address(ffi), vm.getDeployedCode("FFIInterface.sol:FFIInterface"));
        vm.label(address(ffi), "FFIInterface");
    }
}

/// @title StaticConfig_encodeSetGasPayingToken_Test
/// @notice Tests the `encodeSetGasPayingToken` function of the `StaticConfig` library.
contract StaticConfig_encodeSetGasPayingToken_Test is StaticConfig_TestInit {
    /// @notice Tests set gas paying token encoding.
    function testDiff_encodeSetGasPayingToken_succeeds(
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
}

/// @title StaticConfig_decodeSetGasPayingToken_Test
/// @notice Tests the `decodeSetGasPayingToken` function of the `StaticConfig` library.
contract StaticConfig_decodeSetGasPayingToken_Test is StaticConfig_TestInit {
    /// @notice Tests set gas paying token decoding.
    function testFuzz_decodeSetGasPayingToken_succeeds(
        address _token,
        uint8 _decimals,
        bytes32 _name,
        bytes32 _symbol
    )
        external
    {
        bytes memory encoding = ffi.encodeGasPayingToken(_token, _decimals, _name, _symbol);

        (address token, uint8 decimals, bytes32 name, bytes32 symbol) = StaticConfig.decodeSetGasPayingToken(encoding);

        assertEq(token, _token);
        assertEq(decimals, _decimals);
        assertEq(name, _name);
        assertEq(symbol, _symbol);
    }
}

/// @title StaticConfig_encodeAddDependency_Test
/// @notice Tests the `encodeAddDependency` function of the `StaticConfig` library.
contract StaticConfig_encodeAddDependency_Test is StaticConfig_TestInit {
    /// @notice Tests add dependency encoding.
    function testDiff_encodeAddDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = StaticConfig.encodeAddDependency(_chainId);

        bytes memory _encoding = ffi.encodeDependency(_chainId);

        assertEq(encoding, _encoding);
    }
}

/// @title StaticConfig_decodeAddDependency_Test
/// @notice Tests the `decodeAddDependency` function of the `StaticConfig` library.
contract StaticConfig_decodeAddDependency_Test is StaticConfig_TestInit {
    /// @notice Tests add dependency decoding.
    function testFuzz_decodeAddDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = ffi.encodeDependency(_chainId);

        uint256 chainId = StaticConfig.decodeAddDependency(encoding);

        assertEq(chainId, _chainId);
    }
}

/// @title StaticConfig_encodeRemoveDependency_Test
/// @notice Tests the `encodeRemoveDependency` function of the `StaticConfig` library.
contract StaticConfig_encodeRemoveDependency_Test is StaticConfig_TestInit {
    /// @notice Tests remove dependency encoding.
    function testDiff_encodeRemoveDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = StaticConfig.encodeRemoveDependency(_chainId);

        bytes memory _encoding = ffi.encodeDependency(_chainId);

        assertEq(encoding, _encoding);
    }
}

/// @title StaticConfig_decodeRemoveDependency_Test
/// @notice Tests the `decodeRemoveDependency` function of the `StaticConfig` library.
contract StaticConfig_decodeRemoveDependency_Test is StaticConfig_TestInit {
    /// @notice Tests remove dependency decoding.
    function testFuzz_decodeRemoveDependency_succeeds(uint256 _chainId) external {
        bytes memory encoding = ffi.encodeDependency(_chainId);

        uint256 chainId = StaticConfig.decodeRemoveDependency(encoding);

        assertEq(chainId, _chainId);
    }
}
