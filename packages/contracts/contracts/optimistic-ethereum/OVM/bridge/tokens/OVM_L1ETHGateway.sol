// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1ETHGateway } from "../../../iOVM/bridge/tokens/iOVM_L1ETHGateway.sol";
import { iOVM_L2TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L2TokenGateway.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";
import { Lib_AddressResolver } from "../../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_AddressManager } from "../../../libraries/resolver/Lib_AddressManager.sol";

/**
 * @title OVM_L1ETHGateway
 * @dev The L1 ETH Gateway is a contract which stores deposited ETH that is in use on L2.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L1ETHGateway is iOVM_L1ETHGateway, OVM_CrossDomainEnabled, Lib_AddressResolver {

    /********************
     * Public Constants *
     ********************/

    uint32 public constant override FINALIZATION_GAS = 1200000;

    /********************************
     * External Contract References *
     ********************************/

    address public ovmEth;

    /***************
     * Constructor *
     ***************/

    // This contract lives behind a proxy, so the constructor parameters will go unused.
    constructor()
        OVM_CrossDomainEnabled(address(0))
        Lib_AddressResolver(address(0))
        public
    {}

    /******************
     * Initialization *
     ******************/

    /**
     * @param _libAddressManager Address manager for this OE deployment
     * @param _ovmEth L2 OVM_ETH implementation of iOVM_DepositedToken
     */
    function initialize(
        address _libAddressManager,
        address _ovmEth
    )
        public
    {
        require(libAddressManager == Lib_AddressManager(0), "Contract has already been initialized.");
        libAddressManager = Lib_AddressManager(_libAddressManager);
        ovmEth = _ovmEth;
        messenger = resolve("Proxy__OVM_L1CrossDomainMessenger");
    }

    /**************
     * Depositing *
     **************/

    /**
     * @dev This function can be called with no data
     * to deposit an amount of ETH to the caller's balance on L2
     */
    receive()
        external
        payable
    {
        _initiateOutboundTransfer(msg.sender, msg.sender, bytes(""));
    }

    /**
     * @dev deposit an amount of the ETH to the caller's balance on L2
     * @param _data Data to forward to L2.
     */
    function outboundTransfer(
        // @flag: How does adding this data affect the cost of finalizing on the other side?
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateOutboundTransfer(msg.sender, msg.sender, _data);
    }

    /**
     * @dev deposit an amount of ETH to a recipients's balance on L2
     * @param _to L2 address to credit the withdrawal to
     * @param _data Data to forward to L2.
     */
    function outboundTransferTo(
        address _to,
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateOutboundTransfer(msg.sender, _to, _data);
    }

    /**
     * @dev Performs the logic for deposits by storing the ETH and informing the L2 ETH Gateway of the deposit.
     *
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     */
    function _initiateOutboundTransfer(
        address _from,
        address _to,
        bytes memory _data
    )
        internal
    {
        // Construct calldata for l2ETHGateway.finalizeInboundTransfer(_to, _amount)
        bytes memory message =
            abi.encodeWithSelector(
                iOVM_L2TokenGateway.finalizeInboundTransfer.selector,
                _from,
                _to,
                msg.value,
                _data
            );

        // Send calldata into L2
        sendCrossDomainMessage(
            ovmEth,
            message,
            FINALIZATION_GAS
        );

        emit OutboundTransferInitiated(_from, _to, msg.value);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the
     * L1 ETH token.
     * Since only the xDomainMessenger can call this function, it will never be called before the withdrawal is finalized.
     *
     * @param _to L1 address to credit the withdrawal to
     * @param _amount Amount of the ETH to withdraw
     */
    function finalizeInboundTransfer(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    )
        external
        override
        onlyFromCrossDomainAccount(ovmEth)
    {
        _safeTransferETH(_to, _amount);

        emit InboundTransferFinalized(_to, _amount);
    }

    /**********************************
     * Internal Functions: Accounting *
     **********************************/

    /**
     * @dev Internal accounting function for moving around L1 ETH.
     *
     * @param _to L1 address to transfer ETH to
     * @param _value Amount of ETH to send to
     */
    function _safeTransferETH(
        address _to,
        uint256 _value
    )
        internal
    {
        (bool success, ) = _to.call{value: _value}(new bytes(0));
        require(success, 'TransferHelper::safeTransferETH: ETH transfer failed');
    }
}
