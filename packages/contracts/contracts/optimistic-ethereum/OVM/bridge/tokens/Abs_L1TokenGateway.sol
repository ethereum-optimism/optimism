// SPDX-License-Identifier: MIT
// @unsupported: ovm 
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @title Abs_L1TokenGateway
 * @dev An L1 Token Gateway is a contract which stores deposited L1 funds that are in use on L2.
 * It synchronizes a corresponding L2 representation of the "deposited token", informing it
 * of new deposits and releasing L1 funds when there are newly finalized withdrawals.
 *
 * NOTE: This abstract contract gives all the core functionality of an L1 token gateway, 
 * but provides easy hooks in case developers need extensions in child contracts.
 * In many cases, the default OVM_L1ERC20Gateway will suffice.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
abstract contract Abs_L1TokenGateway is iOVM_L1TokenGateway, OVM_CrossDomainEnabled {

    /********************************
     * External Contract References *
     ********************************/

    address public l2DepositedToken;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l2DepositedToken iOVM_L2DepositedToken-compatible address on the chain being deposited into.
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        address _l2DepositedToken,
        address _l1messenger 
    )
        OVM_CrossDomainEnabled(_l1messenger)
    {
        l2DepositedToken = _l2DepositedToken;
    }

    /********************************
     * Overridable Accounting logic *
     ********************************/

    // Default gas value which can be overridden if more complex logic runs on L2.
    uint32 public DEFAULT_FINALIZE_DEPOSIT_L2_GAS = 1200000;

    /**
     * @dev Core logic to be performed when a withdrawal is finalized on L1.
     * In most cases, this will simply send locked funds to the withdrawer.
     *
     * @param _to Address being withdrawn to.
     * @param _amount Amount being withdrawn.
     */
    function _handleFinalizeWithdrawal(
        address _to,
        uint256 _amount
    )
        internal
        virtual
    {
        revert("Implement me in child contracts");
    }

    /**
     * @dev Core logic to be performed when a deposit is initiated on L1.
     * In most cases, this will simply send locked funds to the withdrawer.
     *
     * @param _from Address being deposited from on L1.
     * @param _to Address being deposited into on L2.
     * @param _amount Amount being deposited.
     */
    function _handleInitiateDeposit(
        address _from,
        address _to,
        uint256 _amount
    )
        internal
        virtual
    {
        revert("Implement me in child contracts");
    }

    /**
     * @dev Overridable getter for the L2 gas limit, in the case it may be
     * dynamic, and the above public constant does not suffice.
     *
     */

    function getFinalizeDepositL2Gas()
        public
        view
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_DEPOSIT_L2_GAS;
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
        public
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
        public
        override
    {
        _initiateDeposit(msg.sender, _to, _amount);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 funds. (e.g. transferFrom)
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
        // Call our deposit accounting handler implemented by child contracts.
        _handleInitiateDeposit(
            _from,
            _to,
            _amount
        );

        // Construct calldata for l2DepositedToken.finalizeDeposit(_to, _amount)
        bytes memory data = abi.encodeWithSelector(
            iOVM_L2DepositedToken.finalizeDeposit.selector,
            _to,
            _amount
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2DepositedToken,
            data,
            getFinalizeDepositL2Gas()
        );

        emit DepositInitiated(_from, _to, _amount);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

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
        onlyFromCrossDomainAccount(l2DepositedToken)
    {
        // Call our withdrawal accounting handler implemented by child contracts.
        _handleFinalizeWithdrawal(
            _to,
            _amount
        );

        emit WithdrawalFinalized(_to, _amount);
    }
}
