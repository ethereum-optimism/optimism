// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1ETHGateway } from "../../../iOVM/bridge/tokens/iOVM_L1ETHGateway.sol";
import { iOVM_L2DepositedToken } from "../../../iOVM/bridge/tokens/iOVM_L2DepositedToken.sol";

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

    /*************
     * Constants *
     ************/

    uint32 constant ETH_FINALIZE_L2_GAS = 1_200_000;

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
     * to deposit an amount of ETH to the caller's balance on L2.
     */
    receive()
        external
        payable
    {
        _initiateDeposit(msg.sender, msg.sender, 0, bytes(""));
    }

    /**
     * @dev Deposit an amount of the ETH to the caller's balance on L2.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function deposit(
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateDeposit(
            msg.sender,
            msg.sender,
            _l2Gas,
            _data
        );
    }

    /**
     * @dev Deposit an amount of ETH to a recipient's balance on L2.
     * @param _to L2 address to credit the withdrawal to.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function depositTo(
        address _to,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        payable
    {
        _initiateDeposit(
            msg.sender,
            _to,
            _l2Gas,
            _data
        );
    }

    /**
     * @dev Performs the logic for deposits by storing the ETH and informing the L2 ETH Gateway of the deposit.
     * @param _from Account to pull the deposit from on L1.
     * @param _to Account to give the deposit to on L2.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateDeposit(
        address _from,
        address _to,
        uint32 _l2Gas,
        bytes memory _data
    )
        internal
    {
        // Construct calldata for l2ETHGateway.finalizeDeposit(_to, _amount)
        bytes memory message =
            abi.encodeWithSelector(
                iOVM_L2DepositedToken.finalizeDeposit.selector,
                _from,
                _to,
                msg.value,
                _data
            );

        // Send calldata into L2
        sendCrossDomainMessage(
            ovmEth,
            message,
            _l2Gas
        );

        emit DepositInitiated(_from, _to, msg.value, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @dev Complete a withdrawal from L2 to L1, and credit funds to the recipient's balance of the
     * L1 ETH token.
     * Since only the xDomainMessenger can call this function, it will never be called before the withdrawal is finalized.
     * @param _from L2 address initiating the transfer.
     * @param _to L1 address to credit the withdrawal to.
     * @param _amount Amount of the ERC20 to deposit.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function finalizeWithdrawal(
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

        emit WithdrawalFinalized(_from, _to, _amount, _data);
    }

    /**********************************
     * Internal Functions: Accounting *
     **********************************/

    /**
     * @dev Internal accounting function for moving around L1 ETH.
     *
     * @param _to L1 address to transfer ETH to.
     * @param _value Amount of ETH to transfer.
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

    /*****************************
     * Temporary - Migrating ETH *
     *****************************/

    /**
     * @dev Migrates entire ETH balance to another gateway.
     * @param _to Gateway Proxy address to migrate ETH to.
     */
    function migrateEth(address payable _to) external {
        address owner = Lib_AddressManager(libAddressManager).owner();
        require(msg.sender == owner, "Only the owner can migrate ETH");
        uint256 balance = address(this).balance;
        OVM_L1ETHGateway(_to).donateETH{value:balance}();
    }

    /**
     * @dev Adds ETH balance to the account. This is meant to allow for ETH
     * to be migrated from an old gateway to a new gateway.
     */
    function donateETH() external payable {}
}
