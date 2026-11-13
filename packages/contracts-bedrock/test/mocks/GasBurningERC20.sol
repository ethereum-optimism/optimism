// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { Burn } from "src/libraries/Burn.sol";
import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";

/// @title GasBurningERC20
/// @notice ERC20 token that burns a specified amount of gas on every transfer.
contract GasBurningERC20 is ERC20 {
    /// @notice The amount of gas to burn on transfer.
    uint256 internal immutable GAS_TO_BURN;

    /// @notice Constructor.
    /// @param _gasToBurn The amount of gas to burn on transfer.
    constructor(uint256 _gasToBurn) ERC20("Gas Burning Token", "GBT", 18) {
        GAS_TO_BURN = _gasToBurn;
    }

    /// @notice Transfers tokens and then burns the specified amount of gas.
    function transfer(address _to, uint256 _amount) public override returns (bool) {
        bool success = super.transfer(_to, _amount);
        Burn.gas(GAS_TO_BURN);
        return success;
    }
}
