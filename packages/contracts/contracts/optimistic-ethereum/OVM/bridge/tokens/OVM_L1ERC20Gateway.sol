// SPDX-License-Identifier: MIT
// @unsupported: ovm 
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1ERC20Gateway } from "../../../iOVM/bridge/tokens/iOVM_L1ERC20Gateway.sol";
import { iOVM_L2DepositedERC20 } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedERC20.sol";
import { iOVM_ERC20 } from "../../../iOVM/precompiles/iOVM_ERC20.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @title OVM_L1ERC20Gateway
 * @dev The L1 ERC20 Gateway is a contract which stores deposited L1 funds that are in use on L2.
 * It synchronizes a corresponding L2 ERC20 Gateway, informing it of deposits, and listening to it 
 * for newly finalized withdrawals.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L1ERC20Gateway is iOVM_L1ERC20Gateway, OVM_CrossDomainEnabled {
    
    /********************************
     * External Contract References *
     ********************************/
    
    iOVM_ERC20 public l1ERC20;
    address public l2ERC20Gateway;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l1ERC20 L1 ERC20 address this contract stores deposits for
     * @param _l2ERC20Gateway L2 Gateway address on the chain being deposited into
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        iOVM_ERC20 _l1ERC20,
        address _l2ERC20Gateway,
        address _l1messenger 
    )
        OVM_CrossDomainEnabled(_l1messenger)
    {
        l1ERC20 = _l1ERC20;
        l2ERC20Gateway = _l2ERC20Gateway;
    }

    /**************
     * Depositing *
     **************/

    /**
     * @dev deposit an amount of the ERC20 to the caller's balance on L2
     * @param _amount Amount of the ERC20 to deposit
     */
    function deposit(
        uint _amount
    )
        external
        override
    {
        _initiateDeposit(msg.sender, msg.sender, _amount);
    }

    /**
     * @dev deposit an amount of ERC20 to a recipients's balance on L2
     * @param _to L2 address to credit the withdrawal to
     * @param _amount Amount of the ERC20 to deposit
     */
    function depositTo(
        address _to,
        uint _amount
    )
        external
        override
    {
        _initiateDeposit(msg.sender, _to, _amount);
    }

    /**
     * @dev Performs the logic for deposits by storing the ERC20 and informing the L2 ERC20 Gateway of the deposit.
     *
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _amount Amount of the ERC20 to deposit.
     */
    function _initiateDeposit(
        address _from,
        address _to,
        uint _amount
    )
        internal
    {
        // Hold on to the newly deposited funds
        l1ERC20.transferFrom(
            _from,
            address(this),
            _amount
        );

        // Construct calldata for l2ERC20Gateway.finalizeDeposit(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            iOVM_L2DepositedERC20.finalizeDeposit.selector,
            _to,
            _amount
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2ERC20Gateway,
            data,
            DEFAULT_FINALIZE_DEPOSIT_L2_GAS
        );

        emit DepositInitiated(_from, _to, _amount);
    }

    /*************************************
     * Cross-chain Function: Withdrawing *
     *************************************/

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the 
     * L1 ERC20 token. 
     * This call will fail if the initialized withdrawal from L2 has not been finalized. 
     *
     * @param _to L1 address to credit the withdrawal to
     * @param _amount Amount of the ERC20 to withdraw
     */
    function finalizeWithdrawal(
        address _to,
        uint _amount
    )
        external
        override 
        onlyFromCrossDomainAccount(l2ERC20Gateway) 
    {
        l1ERC20.transfer(_to, _amount);

        emit WithdrawalFinalized(_to, _amount);
    }
}
