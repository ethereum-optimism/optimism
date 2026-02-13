// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title TokenVault - realistic contract for eth_getProof testing
/// @notice Demonstrates mappings, nested mappings, and dynamic arrays
contract TokenVault {
    struct Allowance {
        uint256 amount;
        bool active;
    }

    // Mapping: user => balance
    mapping(address => uint256) public balances;

    // Nested Mapping: owner => spender => allowance info
    mapping(address => mapping(address => Allowance)) public allowances;

    // Dynamic array: list of all depositors
    address[] public depositors;

    constructor() {
        // initialize contract with a few entries
        address alice = address(0xA11CE);
        address bob = address(0xB0B);

        balances[alice] = 1000;
        balances[bob] = 2000;

        allowances[alice][bob] = Allowance({amount: 300, active: true});
        allowances[bob][alice] = Allowance({amount: 150, active: false});

        depositors.push(alice);
        depositors.push(bob);
    }

    function deposit() external payable {
        balances[msg.sender] += msg.value;
        depositors.push(msg.sender);
    }

    function approve(address spender, uint256 amount) external {
        allowances[msg.sender][spender] = Allowance({
            amount: amount,
            active: true
        });
    }

    function deactivateAllowance(address spender) external {
        allowances[msg.sender][spender].active = false;
    }

    function getDepositors() external view returns (address[] memory) {
        return depositors;
    }
}