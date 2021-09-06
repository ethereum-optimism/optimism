// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;
pragma experimental ABIEncoderV2;

/**
 * @title iOVM_L2NFTBridge
 */
interface iOVM_L2NFTBridge {

    // add events
    event WithdrawalInitiated (
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        bytes _data
    );

    event DepositFinalized (
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        bytes _data
    );

    event DepositFailed (
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _tokenId,
        bytes _data
    );

    function withdraw(
        address _l2Contract,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external;

    function withdrawTo(
        address _l2Contract,
        address _to,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external;

    function finalizeDeposit(
        address _l1Contract,
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    )
        external;

}
