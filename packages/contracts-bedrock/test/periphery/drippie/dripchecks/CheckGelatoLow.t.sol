// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { CheckGelatoLow, IGelatoTreasury } from "src/periphery/drippie/dripchecks/CheckGelatoLow.sol";

/// @title  MockGelatoTreasury
/// @notice Mocks the Gelato treasury for testing purposes. Allows arbitrary
///        setting of user balances.
contract MockGelatoTreasury is IGelatoTreasury {
    mapping(address => mapping(address => uint256)) private tokenBalances;

    function setTokenBalance(address _user, address _token, uint256 _balance) external {
        tokenBalances[_token][_user] = _balance;
    }

    function userTokenBalance(address _user, address _token) external view returns (uint256) {
        return tokenBalances[_token][_user];
    }
}

/// @title  CheckGelatoLowTest
/// @notice Tests the CheckGelatoLow contract via fuzzing both the success case
///         and the failure case.
contract CheckGelatoLowTest is Test {
    /// @notice An instance of the CheckGelatoLow contract.
    CheckGelatoLow c;

    /// @notice An instance of the MockGelatoTreasury contract.
    MockGelatoTreasury gelato;

    /// @notice The account Gelato uses to represent ether
    address internal constant eth = 0xEeeeeEeeeEeEeeEeEeEeeEEEeeeeEeeeeeeeEEeE;

    /// @notice Deploy the `CheckGelatoLow` and `MockGelatoTreasury` contracts.
    function setUp() external {
        c = new CheckGelatoLow();
        gelato = new MockGelatoTreasury();
    }

    /// @notice Fuzz the `check` function and assert that it always returns true
    ///         when the user's balance in the treasury is less than the threshold.
    function testFuzz_check_succeeds(uint256 _threshold, address _recipient) external {
        CheckGelatoLow.Params memory p =
            CheckGelatoLow.Params({ treasury: address(gelato), threshold: _threshold, recipient: _recipient });

        vm.assume(gelato.userTokenBalance(_recipient, eth) < _threshold);

        assertEq(c.check(abi.encode(p)), true);
    }

    /// @notice Fuzz the `check` function and assert that it always returns false
    ///         when the user's balance in the treasury is greater than or equal
    ///         to the threshold.
    function testFuzz_check_highBalance_fails(uint256 _threshold, address _recipient) external {
        CheckGelatoLow.Params memory p =
            CheckGelatoLow.Params({ treasury: address(gelato), threshold: _threshold, recipient: _recipient });

        gelato.setTokenBalance(_recipient, eth, _threshold);

        assertEq(c.check(abi.encode(p)), false);
    }
}
