// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";

/* Contract Imports */
import { L2StandardERC20 } from "../../libraries/standards/L2StandardERC20.sol";
import { IWETH9 } from "../../libraries/standards/IWETH9.sol";

/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that
 * unlike on Layer 1, Layer 2 accounts do not have a balance field.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is L2StandardERC20, IWETH9 {

    /***************
     * Constructor *
     ***************/

    constructor()
        L2StandardERC20(
            Lib_PredeployAddresses.L2_STANDARD_BRIDGE,
            address(0),
            "Ether",
            "ETH"
        )
    {}


    /******************************
     * Custom WETH9 Functionality *
     ******************************/
    fallback() external payable {
        deposit();
    }

    /**
     * Implements the WETH9 deposit() function as a no-op.
     * WARNING: this function does NOT have to do with cross-chain asset bridging. The relevant
     * deposit and withdraw functions for that use case can be found at L2StandardBridge.sol.
     * This function allows developers to treat OVM_ETH as WETH without any modifications to their
     * code.
     */
    function deposit()
        public
        payable
        override
    {
        // Calling deposit() with nonzero value will send the ETH to this contract address.
        // Once received here, we transfer it back by sending to the msg.sender.
        _transfer(address(this), msg.sender, msg.value);

        emit Deposit(msg.sender, msg.value);
    }

    /**
     * Implements the WETH9 withdraw() function as a no-op.
     * WARNING: this function does NOT have to do with cross-chain asset bridging. The relevant
     * deposit and withdraw functions for that use case can be found at L2StandardBridge.sol.
     * This function allows developers to treat OVM_ETH as WETH without any modifications to their
     * code.
     * @param _wad Amount being withdrawn
     */
    function withdraw(
        uint256 _wad
    )
        external
        override
    {
        // Calling withdraw() with value exceeding the withdrawer's ovmBALANCE should revert,
        // as in WETH9.
        require(balanceOf(msg.sender) >= _wad);

        // Other than emitting an event, OVM_ETH already is native ETH, so we don't need to do
        // anything else.
        emit Withdrawal(msg.sender, _wad);
    }
}
