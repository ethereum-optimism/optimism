// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Library Imports */
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { L2StandardERC20 } from "../../standards/L2StandardERC20.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 */
contract OVM_ETH is L2StandardERC20 {
    /***************
     * Constructor *
     ***************/

    constructor()
        L2StandardERC20(Lib_PredeployAddresses.L2_STANDARD_BRIDGE, address(0), "Ether", "ETH")
    {}

    // ETH ERC20 features are disabled until further notice.
    // Discussion here: https://github.com/ethereum-optimism/optimism/discussions/1444

    function transfer(address recipient, uint256 amount) public virtual override returns (bool) {
        revert("OVM_ETH: transfer is disabled pending further community discussion.");
    }

    function approve(address spender, uint256 amount) public virtual override returns (bool) {
        revert("OVM_ETH: approve is disabled pending further community discussion.");
    }

    function transferFrom(
        address sender,
        address recipient,
        uint256 amount
    ) public virtual override returns (bool) {
        revert("OVM_ETH: transferFrom is disabled pending further community discussion.");
    }

    function increaseAllowance(address spender, uint256 addedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("OVM_ETH: increaseAllowance is disabled pending further community discussion.");
    }

    function decreaseAllowance(address spender, uint256 subtractedValue)
        public
        virtual
        override
        returns (bool)
    {
        revert("OVM_ETH: decreaseAllowance is disabled pending further community discussion.");
    }
}
