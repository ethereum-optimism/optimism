// SPDX-License-Identifier: MIT
pragma solidity >0.5.0;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L2TokenGateway
 */
interface iOVM_L2TokenGateway {

    /**********
     * Events *
     **********/

    event OutboundTransferInitiated(
        address indexed _from,
        address indexed _to,
        uint256 _amount
    );

    event InboundTransferFinalized(
        address indexed _from,
        address indexed _to,
        uint256 _amount
    );


    /********************
     * Public Functions *
     ********************/

    function outboundTransfer(
        uint _amount,
        bytes calldata _data
    )
        external;

    function outboundTransferTo(
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;


    /*************************
     * Cross-chain Functions *
     *************************/

    function finalizeInboundTransfer(
        address _from,
        address _to,
        uint _amount,
        bytes calldata _data
    )
        external;
}
