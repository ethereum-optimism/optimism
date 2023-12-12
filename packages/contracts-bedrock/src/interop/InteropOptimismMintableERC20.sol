// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { OptimismMintableERC20 } from "src/universal/OptimismMintableERC20.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title InteropOptimismMintableERC20
/// @notice A version of OptimismMintableERC20 that hardcodes
///         the separate native bridges (L2SB & InteropL2SB).
///         Needed since the base contract only allows a single
///         minter. Once the InteropL2SB replaces the L2SB, this
///         is no longer needed.
contract InteropOptimismMintableERC20 is OptimismMintableERC20 {

    constructor(
        address _remoteToken,
        string memory _name,
        string memory _symbol,
        uint8 _decimals
    )
        OptimismMintableERC20(Predeploys.INTEROP_L2_STANDARD_BRIDGE, _remoteToken, _name, _symbol, _decimals)
    { }

    /// @notice Adjusted modifier to allow cross-l2 native mint & burn
    modifier onlyBridge2 {
        require(
            msg.sender == Predeploys.L2_STANDARD_BRIDGE || msg.sender == Predeploys.INTEROP_L2_STANDARD_BRIDGE,
            "InteropOptimismMintableERC20: only bridge can mint and burn"
        );
        _;
    }

    /// @notice Adjust mint to allow cross-l2 native mint & burn
    function mint(address _to, uint256 _amount) external virtual override onlyBridge2 {
        super._mint(_to, _amount);
        emit Mint(_to, _amount);
    }

    /// @notice Adjusted modifier to allow cross-l2 native mint & burn
    function burn(address _from, uint256 _amount) external virtual override onlyBridge2 {
        super._burn(_from, _amount);
        emit Burn(_from, _amount);
    }
}
