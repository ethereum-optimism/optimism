// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";

import { iOVM_ERC20 } from "../../../iOVM/predeploys/iOVM_ERC20.sol";

/**
 * @title OVM_L1ERC20Gateway
 * @dev The L1 ERC20 Gateway is a contract which stores deposited L1 funds that are in use on L2.
 * It synchronizes a corresponding L2 ERC20 Gateway, informing it of deposits, and listening to it
 * for newly finalized withdrawals.
 *
 * NOTE: This contract extends Abs_L1TokenGateway, which is where we
 * takes care of most of the initialization and the cross-chain logic.
 * If you are looking to implement your own deposit/withdrawal contracts, you
 * may also want to extend the abstract contract in a similar manner.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L1ERC20Gateway is iOVM_L1TokenGateway, OVM_CrossDomainEnabled {

    /********************************
     * External Contract References *
     ********************************/

    // Maps L1 token to L2 token to balance of the L1 token escrowed
    mapping(address => mapping (address => uint256)) l2TokenState;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        address _l1messenger
    )
        OVM_CrossDomainEnabled(_l1messenger)
    {
    }

    /**************
     * Depositing *
     **************/

    /**
     * @dev deposit an amount of the ERC20 to the caller's balance on L2.
     * @param _l1Token Address of the L1 ERC20 we are depositing
     * @param _l2Token Address of the L1 respective L2 ERC20
     * @param _amount Amount of the ERC20 to deposit
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function deposit(
        address _l1Token,
        address _l2Token,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateDeposit(_l1Token, _l2Token, msg.sender, msg.sender, _amount, _l2Gas, _data);
    }

    /**
     * @dev deposit an amount of ERC20 to a recipient's balance on L2.
     * @param _l1Token Address of the L1 ERC20 we are depositing
     * @param _l2Token Address of the L1 respective L2 ERC20
     * @param _to L2 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function depositTo(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateDeposit(_l1Token, _l2Token, msg.sender, _to, _amount, _l2Gas, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 funds. (e.g. transferFrom)
     *
     * @param _l1Token Address of the L1 ERC20 we are depositing
     * @param _l2Token Address of the L1 respective L2 ERC20
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _amount Amount of the ERC20 to deposit.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateDeposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    )
        internal
    {
        // When a deposit is initiated on L1, the L1 Gateway transfers the funds to itself for future withdrawals.
        iOVM_ERC20(_l1Token).transferFrom(
            _from,
            address(this),
            _amount
        );

        // Construct calldata for _l2Token.finalizeDeposit(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L2DepositedToken(_l2Token).finalizeDeposit.selector,
            _l1Token,
            _from,
            _to,
            _amount,
            _data
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            _l2Token,
            _l2Gas,
            message
        );

        l2TokenState[_l1Token][_l2Token] += _amount;

        // We omit _data here because events only support bytes32 types.
        emit DepositInitiated(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the
     * L1 ERC20 token.
     * This call will fail if the initialized withdrawal from L2 has not been finalized.
     *
     * @param _l1Token Address of L1 token to finalizeWithdrawal for.
     * @param _l2Token Address of L2 token where withdrawal was initiated.
     * @param _from L2 address initiating the transfer.
     * @param _to L1 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _data Data provided by the sender on L2. This data is provided
     *   solely as a convenience for external contracts. Aside from enforcing a maximum
     *   length, these contracts provide no guarantees about its content.
     */
    function finalizeWithdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyFromCrossDomainAccount(_l2Token)
    {
        // todo secure against underflow
        l2TokenState[_l1Token][_l2Token] -= _amount;

        // When a withdrawal is finalized on L1, the L1 Gateway transfers the funds to the withdrawer.
        iOVM_ERC20(_l1Token).transfer(_to, _amount);

        emit WithdrawalFinalized(_l1Token, _l2Token, _from, _to, _amount, _data);
    }
}
