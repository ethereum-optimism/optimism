// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";
import { iOVM_L2TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L2TokenGateway.sol";

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

    address l2TokenGateway;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l2TokenGateway iOVM_L2TokenGateway-compatible address on the chain being deposited into.
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     */
    constructor(
        address _l2TokenGateway,
        address _l1messenger
    )
        OVM_CrossDomainEnabled(_l1messenger)
    {
        l2TokenGateway = _l2TokenGateway;
    }

    /********************************
     * Overridable Accounting logic *
     ********************************/

    // Default gas value which can be overridden if more complex logic runs on L2.
    uint32 internal constant DEFAULT_FINALIZE_DEPOSIT_L2_GAS = 1200000;

    /**
     * @dev Core logic to be performed when a withdrawal is finalized on L1.
     * In most cases, this will simply send locked funds to the withdrawer.
     *
     * param _to Address being withdrawn to.
     * param _amount Amount being withdrawn.
     */
    function _handleFinalizeInboundTransfer(
        address, // _to,
        uint256 // _amount
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
     * param _from Address being deposited from on L1.
     * param _to Address being deposited into on L2.
     * param _amount Amount being deposited.
     */
    function _handleInitiateOutboundTransfer(
        address, // _from,
        address, // _to,
        uint256 // _amount
    )
        internal
        virtual
    {
        revert("Implement me in child contracts");
    }

    /********************
     * Public Functions *
     ********************/

    function counterpartGateway()
        external
        view
        override
        returns(
            address
        ) {
            return l2TokenGateway;
        }

    /**
     * @dev Overridable getter for the L2 gas limit, in the case it may be
     * dynamic, and the above public constant does not suffice.
     *
     */
    function getFinalizationGas()
        public
        pure
        virtual
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
    function outboundTransfer(
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateOutboundTransfer(msg.sender, msg.sender, _amount, _data);
    }

    /**
     * @dev deposit an amount of ERC20 to a recipients's balance on L2
     * @param _to L2 address to credit the withdrawal to
     * @param _amount Amount of the ERC20 to deposit
     */
    function outboundTransferTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateOutboundTransfer(msg.sender, _to, _amount, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 funds. (e.g. transferFrom)
     *
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _amount Amount of the ERC20 to deposit.
     */
    function _initiateOutboundTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        internal
    {
        // Call our deposit accounting handler implemented by child contracts.
        _handleInitiateOutboundTransfer(
            _from,
            _to,
            _amount
        );

        // Construct calldata for l2TokenGateway.finalizeInboundTransfer(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L2TokenGateway.finalizeInboundTransfer.selector,
            _from,
            _to,
            _amount,
            _data
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2TokenGateway,
            message,
            getFinalizationGas()
        );

        // We omit _data here because events only support bytes32 types.
        emit OutboundTransferInitiated(_from, _to, _amount, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the
     * L1 ERC20 token.
     * This call will fail if the initialized withdrawal from L2 has not been finalized.
     *
     * @param _from L2 address initiating the transfer
     * @param _to L1 address to credit the withdrawal to
     * @param _data Data provided by the sender on L2.
     */
    function finalizeInboundTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external
        override
        virtual
        onlyFromCrossDomainAccount(l2TokenGateway)
    {
        // todo: add verification check on _from and _data
        // Call our withdrawal accounting handler implemented by child contracts.
        _handleFinalizeInboundTransfer(
            _to,
            _amount
        );
        // We omit _data here because events only support bytes32 types.
        emit InboundTransferFinalized(_from, _to, _amount, _data);
    }
}
