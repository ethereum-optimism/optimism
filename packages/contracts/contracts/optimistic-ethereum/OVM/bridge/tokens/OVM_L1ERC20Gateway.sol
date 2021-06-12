// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";
import { Abs_L1TokenGateway } from "./Abs_L1TokenGateway.sol";
import { iOVM_ERC20 } from "../../../iOVM/predeploys/iOVM_ERC20.sol";
import { Lib_AddressResolver } from "../../../libraries/resolver/Lib_AddressResolver.sol";
import { Lib_AddressManager } from "../../../libraries/resolver/Lib_AddressManager.sol";

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
contract OVM_L1ERC20Gateway is Abs_L1TokenGateway, Lib_AddressResolver {

    /********************************
     * External Contract References *
     ********************************/

    iOVM_ERC20 public l1ERC20;

    /***************
     * Constructor *
     ***************/

    constructor()
        Abs_L1TokenGateway(address(0), address(0))
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
        address _l2DepositedERC20,
        address _l1ERC20
    )
        public
    {
        require(libAddressManager == Lib_AddressManager(0), "Contract has already been initialized.");
        libAddressManager = Lib_AddressManager(_libAddressManager);
        l2DepositedToken = _l2DepositedERC20;
        l1ERC20 = iOVM_ERC20(_l1ERC20);
        messenger = resolve("Proxy__OVM_L1CrossDomainMessenger");
    }


    /**************
     * Accounting *
     **************/
    function depositByChainId(uint256 _chainId, uint256 _amount) 
        external
        override
    {
        _initiateDepositByChainId(_chainId, msg.sender, msg.sender, _amount);
    }
    
    function depositToByChainId(
        uint256 _chainId,
        address _to,
        uint256 _amount
    )
        external
        override
    {
        _initiateDepositByChainId(_chainId, msg.sender, _to, _amount);
    }
    
    /**
     * @dev When a deposit is initiated on L1, the L1 Gateway
     * transfers the funds to itself for future withdrawals
     *
     * @param _from L1 address ETH is being deposited from
     * param _to L2 address that the ETH is being deposited to
     * @param _amount Amount of ERC20 to send
     */
    function _handleInitiateDeposit(
        address _from,
        address, // _to,
        uint256 _amount
    )
        internal
        override
    {
         // Hold on to the newly deposited funds
        l1ERC20.transferFrom(
            _from,
            address(this),
            _amount
        );
    }

    /**
     * @dev When a withdrawal is finalized on L1, the L1 Gateway
     * transfers the funds to the withdrawer
     *
     * @param _to L1 address that the ERC20 is being withdrawn to
     * @param _amount Amount of ERC20 to send
     */
    function _handleFinalizeWithdrawal(
        address _to,
        uint _amount
    )
        internal
        override
    {
        // Transfer withdrawn funds out to withdrawer
        l1ERC20.transfer(_to, _amount);
    }
}
