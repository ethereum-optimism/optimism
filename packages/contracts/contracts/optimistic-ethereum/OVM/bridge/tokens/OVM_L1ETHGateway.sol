// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1ETHGateway } from "../../../iOVM/bridge/tokens/iOVM_L1ETHGateway.sol";
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";
import { iOVM_ERC20 } from "../../../iOVM/precompiles/iOVM_ERC20.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "../../../libraries/bridge/OVM_CrossDomainEnabled.sol";
import { Lib_AddressResolver } from "../../../libraries/resolver/Lib_AddressResolver.sol";

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

    uint32 public constant override getFinalizeDepositL2Gas = 1200000;

    /********************************
     * External Contract References *
     ********************************/

    address public ovmEth;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _libAddressManager Address manager for this OE deployment
     * @param _ovmEth L2 OVM_ETH implementation of iOVM_DepositedToken
     */
    constructor(
        address _libAddressManager,
        address _ovmEth
    )
        OVM_CrossDomainEnabled(address(0)) // overridden in constructor code
        Lib_AddressResolver(_libAddressManager)
    {
        ovmEth = _ovmEth;
        messenger = resolve("Proxy__OVM_L1CrossDomainMessenger"); // overrides OVM_CrossDomainEnabled constructor setting because resolve() is not yet accessible
    }

    /**************
     * Depositing *
     **************/

    /**
     * @dev deposit an amount of the ERC20 to the caller's balance on L2
     */
    function deposit() 
        external
        override
        payable
    {
        _initiateDeposit(msg.sender, msg.sender);
    }

    /**
     * @dev deposit an amount of ERC20 to a recipients's balance on L2
     * @param _to L2 address to credit the withdrawal to
     */
    function depositTo(
        address _to
    )
        external
        override
        payable
    {
        _initiateDeposit(msg.sender, _to);
    }

    /**
     * @dev Performs the logic for deposits by storing the ERC20 and informing the L2 ERC20 Gateway of the deposit.
     *
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     */
    function _initiateDeposit(
        address _from,
        address _to
    )
        internal
    {
        // Construct calldata for l2ERC20Gateway.finalizeDeposit(_to, _amount)
        bytes memory data =
            abi.encodeWithSelector(
                iOVM_L2DepositedToken.finalizeDeposit.selector,
                _to,
                msg.value
            );

        // Send calldata into L2
        sendCrossDomainMessage(
            ovmEth,
            data,
            getFinalizeDepositL2Gas
        );

        emit DepositInitiated(_from, _to, msg.value);
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
        uint256 _amount
    )
        external
        override
        onlyFromCrossDomainAccount(ovmEth)
    {
        _safeTransferETH(_to, _amount);

        emit WithdrawalFinalized(_to, _amount);
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

    /**
     * @dev Prevent users from sending ETH directly to this contract without calling deposit
     */
    receive()
        external
        payable
    {
        revert("Deposits must be initiated via deposit() or depositTo()");
    }
}
