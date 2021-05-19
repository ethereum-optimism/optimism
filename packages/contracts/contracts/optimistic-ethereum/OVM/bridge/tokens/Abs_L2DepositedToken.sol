// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";

/**
 * @title Abs_L2DepositedToken
 * @dev An L2 Deposited Token is an L2 representation of funds which were deposited from L1.
 * Usually contract mints new tokens when it hears about deposits into the L1 ERC20 gateway.
 * This contract also burns the tokens intended for withdrawal, informing the L1 gateway to release L1 funds.
 *
 * NOTE: This abstract contract gives all the core functionality of a deposited token implementation except for the
 * token's internal accounting itself.  This gives developers an easy way to implement children with their own token code.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
abstract contract Abs_L2DepositedToken is iOVM_L2DepositedToken, OVM_CrossDomainEnabled {

    /*******************
     * Contract Events *
     *******************/

    event Initialized(iOVM_L1TokenGateway _l1TokenGateway);

    /********************************
     * External Contract References *
     ********************************/

    iOVM_L1TokenGateway public l1TokenGateway;

    /********************************
     * Constructor & Initialization *
     ********************************/

    /**
     * @param _l2CrossDomainMessenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        address _l2CrossDomainMessenger
    )
        OVM_CrossDomainEnabled(_l2CrossDomainMessenger)
    {}

    /**
     * @dev Initialize this contract with the L1 token gateway address.
     * The flow: 1) this contract gets deployed on L2, 2) the L1
     * gateway is deployed with addr from (1), 3) L1 gateway address passed here.
     *
     * @param _l1TokenGateway Address of the corresponding L1 gateway deployed to the main chain
     */
    function init(
        iOVM_L1TokenGateway _l1TokenGateway
    )
        public
    {
        require(address(l1TokenGateway) == address(0), "Contract has already been initialized");

        l1TokenGateway = _l1TokenGateway;

        emit Initialized(l1TokenGateway);
    }

    /**********************
     * Function Modifiers *
     **********************/

    modifier onlyInitialized() {
        require(address(l1TokenGateway) != address(0), "Contract has not yet been initialized");
        _;
    }

    /********************************
     * Overridable Accounting logic *
     ********************************/

    // Default gas value which can be overridden if more complex logic runs on L1.
    uint32 internal constant DEFAULT_FINALIZE_WITHDRAWAL_L1_GAS = 100_000;

    /**
     * @dev Core logic to be performed when a withdrawal from L2 is initialized.
     * In most cases, this will simply burn the withdrawn L2 funds.
     *
     * param _to Address being withdrawn to
     * param _amount Amount being withdrawn
     */
    function _handleInitiateWithdrawal(
        address, // _to,
        uint256 // _amount
    )
        internal
        virtual
    {
        revert("Accounting must be implemented by child contract.");
    }

    /**
     * @dev Core logic to be performed when a deposit from L2 is finalized on L2.
     * In most cases, this will simply _mint() to credit L2 funds to the recipient.
     *
     * param _to Address being deposited to on L2
     * param _amount Amount which was deposited on L1
     */
    function _handleFinalizeDeposit(
        address, // _to
        uint256 // _amount
    )
        internal
        virtual
    {
        revert("Accounting must be implemented by child contract.");
    }

    /**
     * @dev Overridable getter for the *L1* gas limit of settling the withdrawal, in the case it may be
     * dynamic, and the above public constant does not suffice.
     */
    function getFinalizeWithdrawalL1Gas()
        public
        pure
        override
        virtual
        returns(
            uint32
        )
    {
        return DEFAULT_FINALIZE_WITHDRAWAL_L1_GAS;
    }


    /***************
     * Withdrawing *
     ***************/

    /**
     * @dev initiate a withdraw of some tokens to the caller's account on L1
     * @param _amount Amount of the token to withdraw.
     * @param _data Data to forward to L1.
     * @param _l1Gas Gas limit for the provided message.
     */
    function withdraw(
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateWithdrawal(
            msg.sender,
            msg.sender,
            _amount,
            _l1Gas,
            _data
        );
    }

    /**
     * @dev initiate a withdraw of some token to a recipient's account on L1.
     * @param _to L1 adress to credit the withdrawal to.
     * @param _amount Amount of the token to withdraw.
     * @param _data Data to forward to L1.
     * @param _l1Gas Gas limit for the provided message.
     */
    function withdrawTo(
        address _to,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
    {
        _initiateWithdrawal(
            msg.sender,
            _to,
            _amount,
            _l1Gas,
            _data
        );
    }

    /**
     * @dev Performs the logic for deposits by storing the token and informing the L2 token Gateway of the deposit.
     *
     * @param _from Account to pull the deposit from on L2.
     * @param _to Account to give the withdrawal to on L1.
     * @param _amount Amount of the token to withdraw.
     * @param _l1Gas Optional gas limit to complete the deposit on l2.
     *  If not provided, the default amount is passed.
     * @param _data Optional data to forward to L2. This data is provided
     *   solely as a convenience for external contracts. Aside from enforcing a maximum
     *   length, these contracts provide no guarantees about it's content.
     */
    function _initiateWithdrawal(
        address _from,
        address _to,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        internal
    {
        // Call our withdrawal accounting handler implemented by child contracts (usually a _burn)
        _handleInitiateWithdrawal(_to, _amount);

        // Construct calldata for l1TokenGateway.finalizeWithdrawal(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L1TokenGateway.finalizeWithdrawal.selector,
            _from,
            _to,
            _amount,
            _data
        );

        // Prevent tokens stranded on other side by taking
        // the max of the user provided gas and DEFAULT_FINALIZE_WITHDRAWAL_L1_GAS
        uint32 defaultGas = getFinalizeWithdrawalL1Gas();
        uint32 l1Gas = _l1Gas > defaultGas ? _l1Gas : defaultGas;
        // Send message up to L1 gateway
        sendCrossDomainMessage(
            address(l1TokenGateway),
            message,
            l1Gas
        );

        emit WithdrawalInitiated(msg.sender, _to, _amount, _data);
    }

    /************************************
     * Cross-chain Function: Depositing *
     ************************************/

    /**
     * @dev Complete a deposit from L1 to L2, and credits funds to the recipient's balance of this
     * L2 token.
     * This call will fail if it did not originate from a corresponding deposit in OVM_l1TokenGateway.
     *
     * @param _from Account to pull the deposit from on L2.
     * @param _to Address to receive the withdrawal at
     * @param _amount Amount of the token to withdraw
     * @param _data Data provider by the sender on L1. This data is provided
     *   solely as a convenience for external contracts. Aside from enforcing a maximum
     *   length, these contracts provide no guarantees about it's content.
     */
    function finalizeDeposit(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyInitialized()
        onlyFromCrossDomainAccount(address(l1TokenGateway))
    {
        _handleFinalizeDeposit(_to, _amount);
        emit DepositFinalized(_from, _to, _amount, _data);
    }
}
