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

    /**
     * Implements the WETH9 fallback functionality.
     */
    fallback() external payable {
        deposit();
    }

    /**
     * Implements the WETH9 deposit() function as a no-op.
     * WARNING: this function does NOT have to do with cross-chain asset bridging. The
     * relevant deposit and withdraw functions for that use case can be found at L2StandardBridge.sol.
     * This function allows developers to treat OVM_ETH as WETH without any modifications to their code.
     */
    function deposit()
        public
        payable
        override
    {
        // Calling deposit() with nonzero value will send the ETH to this contract address.  Once recieved here,
        // We transfer it back by sending to the msg.sender.
        _transfer(address(this), msg.sender, msg.value);

        emit Deposit(msg.sender, msg.value);
    }

    /**
     * Implements the WETH9 withdraw() function as a no-op.
     * WARNING: this function does NOT have to do with cross-chain asset bridging. The
     * relevant deposit and withdraw functions for that use case can be found at L2StandardBridge.sol.
     * This function allows developers to treat OVM_ETH as WETH without any modifications to their code.
     * @param _wad Amount being withdrawn
     */
    function withdraw(
        uint256 _wad
    )
        external
        override
    {
        // Calling withdraw() with value exceeding the withdrawer's ovmBALANCE should revert, as in WETH9.
        require(balanceOf(msg.sender) >= _wad);

        // Other than emitting an event, OVM_ETH already is native ETH, so we don't need to do anything else.
        emit Withdrawal(msg.sender, _wad);
    }


    /**********************************
     * Overridden ERC20 Functionality *
     **********************************/

    /**
     * This override allows for OVM_ETH to be sent to address(0), which is not normally permitted by the
     * OZ ERC20 implementation, but IS possible for EVM calls to the zero address with nonzero value.
     * @param _sender Address sending the OVM_ETH.
     * @param _recipient Address recieving the OVM_ETH.
     * @param _amount The amount of OVM_ETH being sent.
     */
    function _transfer(
        address _sender,
        address _recipient,
        uint256 _amount
    )
        internal
        virtual
        override
    {
        // super._transfer disallows this condition, but we can just interpret it as a burn.
        if (_recipient == address(0)) {
            _burn(_sender, _amount);
        } else {
            super._transfer(_sender, _recipient, _amount);
        }
    }
}
