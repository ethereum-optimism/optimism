// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L1Block } from "src/L2/L1Block.sol";

// Interfaces
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000015
/// @title L1BlockCGT
/// @notice The L1BlockCGT predeploy gives users access to information about the last known L1 block.
///         Values within this contract are updated once per epoch (every L1 block) and can only be
///         set by the "depositor" account, a special system address. Depositor account transactions
///         are created by the protocol whenever we move to a new epoch.
contract L1BlockCGT is L1Block {
    /// @notice Storage slot for the isCustomGasToken flag
    /// @dev bytes32(uint256(keccak256("l1block.isCustomGasToken")) - 1)
    bytes32 private constant IS_CUSTOM_GAS_TOKEN_SLOT =
        0xd2ff82c9b477ff6a09f530b1c627ffb4b0b81e2ae2ba427f824162e8dad020aa;

    /// @custom:semver +custom-gas-token
    function version() public pure override returns (string memory) {
        return string.concat(super.version(), "+custom-gas-token");
    }

    /// @notice Returns whether the gas paying token is custom.
    function isCustomGasToken() public view override returns (bool isCustom_) {
        bytes32 slot = IS_CUSTOM_GAS_TOKEN_SLOT;
        assembly {
            isCustom_ := sload(slot)
        }
    }

    /// @notice Returns the gas paying token, its decimals, name and symbol.
    function gasPayingToken() public pure override returns (address, uint8) {
        revert("L1BlockCGT: deprecated");
    }

    /// @notice Returns the gas paying token name.
    ///         If nothing is set in state, then it means ether is used.
    ///         This function cannot be removed because WETH depends on it.
    function gasPayingTokenName() public view override returns (string memory name_) {
        name_ =
            isCustomGasToken() ? ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).gasPayingTokenName() : "Ether";
    }

    /// @notice Returns the gas paying token symbol.
    ///         If nothing is set in state, then it means ether is used.
    ///         This function cannot be removed because WETH depends on it.
    function gasPayingTokenSymbol() public view override returns (string memory symbol_) {
        symbol_ =
            isCustomGasToken() ? ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).gasPayingTokenSymbol() : "ETH";
    }

    /// @notice Set chain to use custom gas token (callable by depositor account)
    function setCustomGasToken() external {
        require(
            msg.sender == Constants.DEPOSITOR_ACCOUNT,
            "L1Block: only the depositor account can set isCustomGasToken flag"
        );
        require(isCustomGasToken() == false, "L1Block: CustomGasToken already active");

        bytes32 slot = IS_CUSTOM_GAS_TOKEN_SLOT;
        assembly {
            sstore(slot, 1)
        }
    }
}
