// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import {
    Lib_PredeployAddresses
} from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";

/**
 * @title OVM_ETH
 * @dev Deprecated contract that used to hold user ETH balances
 */
contract OVM_ETH is OptimismMintableERC20 {
    /***************
     * Constructor *
     ***************/

    constructor()
        OptimismMintableERC20(Lib_PredeployAddresses.L2_STANDARD_BRIDGE, address(0), "Ether", "ETH")
    {}

    function mint(address _to, uint256 _amount) public virtual override {
        revert("OVM_ETH: mint is disabled");
    }

    function burn(address _from, uint256 _amount) public virtual override {
        revert("OVM_ETH: burn is disabled");
    }

    function transfer(address _recipient, uint256 _amount) public virtual override returns (bool) {
        revert("OVM_ETH: transfer is disabled");
    }

    function approve(address _spender, uint256 _amount) public virtual override returns (bool) {
        revert("OVM_ETH: approve is disabled");
    }

    function transferFrom(
        address _sender,
        address _recipient,
        uint256 _amount
    ) public virtual override returns (bool) {
        revert("OVM_ETH: transferFrom is disabled");
    }

    function increaseAllowance(address _spender, uint256 _addedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("OVM_ETH: increaseAllowance is disabled");
    }

    function decreaseAllowance(address _spender, uint256 _subtractedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("OVM_ETH: decreaseAllowance is disabled");
    }
}
