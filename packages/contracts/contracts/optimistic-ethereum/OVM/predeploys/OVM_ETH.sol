// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Library Imports */
import { Lib_AddressResolver } from "../../libraries/resolver/Lib_AddressResolver.sol";
import { Abs_L2TokenGateway } from "../../OVM/bridge/tokens/Abs_L2TokenGateway.sol";

/* Interface Imports */
import { iOVM_L1TokenGateway } from "../../iOVM/bridge/tokens/iOVM_L1TokenGateway.sol";

/* Contract Imports */
import { UniswapV2ERC20 } from "../../libraries/standards/UniswapV2ERC20.sol";
/**
 * @title OVM_ETH
 * @dev The ETH predeploy provides an ERC20 interface for ETH deposited to Layer 2. Note that
 * unlike on Layer 1, Layer 2 accounts do not have a balance field. The OVM_ETH token is also
 * its own gateway contract.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
contract OVM_ETH is Abs_L2TokenGateway, UniswapV2ERC20 {

    constructor(
        address _l2CrossDomainMessenger,
        address _l1ETHGateway
    )
        Abs_L2TokenGateway(_l2CrossDomainMessenger)
        UniswapV2ERC20(
            "Ether",
            "ETH"
        )
    {
        init(iOVM_L1TokenGateway(_l1ETHGateway));
    }


    /****************************
     * Gateway Accounting logic *
     ***************************/

    // When a withdrawal is initiated, we burn the withdrawer's funds to prevent subsequent L2 usage.
    function _handleInitiateOutboundTransfer(
        address, // _to,
        uint _amount
    )
        internal
        override
    {
        _burn(msg.sender, _amount);
    }

    // When a deposit is finalized, we credit the account on L2 with the same amount of tokens.
    function _handleFinalizeInboundTransfer(
        address _to,
        uint _amount
    )
        internal
        override
    {
        _mint(_to, _amount);
    }
}
