pragma solidity ^0.8.9;

import { MockMessenger } from "./MockMessenger.sol";

contract MockBridge {
    event ETHDepositInitiated(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );

    event ERC20DepositInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event ERC20WithdrawalFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event WithdrawalInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event DepositFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event DepositFailed(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    struct TokenEventStruct {
        address l1Token;
        address l2Token;
        address from;
        address to;
        uint256 amount;
        bytes data;
    }

    MockMessenger public messenger;

    constructor(MockMessenger _messenger) {
        messenger = _messenger;
    }

    function emitERC20DepositInitiated(
        TokenEventStruct memory _params
    ) public {
        emit ERC20DepositInitiated(_params.l1Token, _params.l2Token, _params.from, _params.to, _params.amount, _params.data);
        messenger.triggerSentMessageEvent(
            MockMessenger.SentMessageEventParams(
                address(0),
                address(0),
                hex"1234",
                1234,
                12345678,
                0
            )
        );
    }

    function emitERC20WithdrawalFinalized(
        TokenEventStruct memory _params
    ) public {
        emit ERC20WithdrawalFinalized(_params.l1Token, _params.l2Token, _params.from, _params.to, _params.amount, _params.data);
    }

    function emitWithdrawalInitiated(
        TokenEventStruct memory _params
    ) public {
        emit WithdrawalInitiated(_params.l1Token, _params.l2Token, _params.from, _params.to, _params.amount, _params.data);
        messenger.triggerSentMessageEvent(
            MockMessenger.SentMessageEventParams(
                address(0),
                address(0),
                hex"1234",
                1234,
                12345678,
                0
            )
        );
    }

    function emitDepositFinalized(
        TokenEventStruct memory _params
    ) public {
        emit DepositFinalized(_params.l1Token, _params.l2Token, _params.from, _params.to, _params.amount, _params.data);
    }

    function emitDepositFailed(
        TokenEventStruct memory _params
    ) public {
        emit DepositFailed(_params.l1Token, _params.l2Token, _params.from, _params.to, _params.amount, _params.data);
    }

    function depositETH(
        uint32 _l2GasLimit,
        bytes memory _data
    )
        public
        payable
    {
        emit ETHDepositInitiated(
            msg.sender,
            msg.sender,
            msg.value,
            _data
        );
    }

    function withdraw(
        address _l2Token,
        uint256 _amount,
        uint32 _l1Gas,
        bytes calldata _data
    )
        public
        payable
    {
        emit WithdrawalInitiated(
            address(0),
            _l2Token,
            msg.sender,
            msg.sender,
            _amount,
            _data
        );
    }
}
