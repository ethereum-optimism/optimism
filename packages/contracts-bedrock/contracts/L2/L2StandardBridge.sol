// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";

/**
 * @custom:proxied
 * @custom:predeploy 0x4200000000000000000000000000000000000010
 * @title L2StandardBridge
 * @notice The L2StandardBridge is responsible for transfering ETH and ERC20 tokens between L1 and
 *         L2. ERC20 tokens sent to L1 are escrowed within this contract.
 */
contract L2StandardBridge is StandardBridge {
    /**
     * @custom:legacy
     * @notice Emitted whenever a withdrawal from L2 to L1 is initiated.
     *
     * @param _l1Token Address of the token on L1.
     * @param _l2Token Address of the corresponding token on L2.
     * @param _from    Address of the withdrawer.
     * @param _to      Address of the recipient on L1.
     * @param _amount  Amount of the ERC20 withdrawn.
     * @param _data    Extra data attached to the withdrawal.
     */
    event WithdrawalInitiated(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    /**
     * @custom:legacy
     * @notice Emitted whenever an ERC20 deposit is finalized.
     *
     * @param _l1Token Address of the token on L1.
     * @param _l2Token Address of the corresponding token on L2.
     * @param _from    Address of the depositor.
     * @param _to      Address of the recipient on L2.
     * @param _amount  Amount of the ERC20 deposited.
     * @param _data    Extra data attached to the deposit.
     */
    event DepositFinalized(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    /**
     * @custom:legacy
     * @notice Emitted whenever a deposit fails.
     *
     * @param _l1Token Address of the token on L1.
     * @param _l2Token Address of the corresponding token on L2.
     * @param _from    Address of the depositor.
     * @param _to      Address of the recipient on L2.
     * @param _amount  Amount of the ERC20 deposited.
     * @param _data    Extra data attached to the deposit.
     */
    event DepositFailed(
        address indexed _l1Token,
        address indexed _l2Token,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    /**
     * @notice Initializes the L2StandardBridge.
     *
     * @param _otherBridge Address of the L1StandardBridge.
     */
    function initialize(address payable _otherBridge) public {
        _initialize(payable(Lib_PredeployAddresses.L2_CROSS_DOMAIN_MESSENGER), _otherBridge);
    }

    /**
     * @custom:legacy
     * @notice Initiates a withdrawal from L2 to L1.
     *
     * @param _l2Token     Address of the L2 token to withdraw.
     * @param _amount      Amount of the L2 token to withdraw.
     * @param _minGasLimit Minimum gas limit to use for the transaction.
     * @param _data        Extra data attached to the withdrawal.
     */
    function withdraw(
        address _l2Token,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _data
    ) external payable virtual {
        _initiateWithdrawal(_l2Token, msg.sender, msg.sender, _amount, _minGasLimit, _data);
    }

    /**
     * @custom:legacy
     * @notice Initiates a withdrawal from L2 to L1 to a target account on L1.
     *
     * @param _l2Token     Address of the L2 token to withdraw.
     * @param _to          Recipient account on L1.
     * @param _amount      Amount of the L2 token to withdraw.
     * @param _minGasLimit Minimum gas limit to use for the transaction.
     * @param _data        Extra data attached to the withdrawal.
     */
    function withdrawTo(
        address _l2Token,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _data
    ) external payable virtual {
        _initiateWithdrawal(_l2Token, msg.sender, _to, _amount, _minGasLimit, _data);
    }

    /**
     * @custom:legacy
     * @notice Finalizes a deposit from L1 to L2.
     *
     * @param _l1Token Address of the L1 token to deposit.
     * @param _l2Token Address of the corresponding L2 token.
     * @param _from    Address of the depositor.
     * @param _to      Address of the recipient.
     * @param _amount  Amount of the tokens being deposited.
     * @param _data    Extra data attached to the deposit.
     */
    function finalizeDeposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external payable virtual {
        if (_l1Token == address(0) && _l2Token == Lib_PredeployAddresses.OVM_ETH) {
            finalizeBridgeETH(_from, _to, _amount, _data);
        } else {
            finalizeBridgeERC20(_l2Token, _l1Token, _from, _to, _amount, _data);
        }
        emit DepositFinalized(_l1Token, _l2Token, _from, _to, _amount, _data);
    }

    /**
     * @custom:legacy
     * @notice Internal function to a withdrawal from L2 to L1 to a target account on L1.
     *
     * @param _l2Token     Address of the L2 token to withdraw.
     * @param _from        Address of the withdrawer.
     * @param _to          Recipient account on L1.
     * @param _amount      Amount of the L2 token to withdraw.
     * @param _minGasLimit Minimum gas limit to use for the transaction.
     * @param _data        Extra data attached to the withdrawal.
     */
    function _initiateWithdrawal(
        address _l2Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _data
    ) internal {
        address l1Token = OptimismMintableERC20(_l2Token).l1Token();
        if (_l2Token == Lib_PredeployAddresses.OVM_ETH) {
            _initiateBridgeETH(_from, _to, _amount, _minGasLimit, _data);
        } else {
            _initiateBridgeERC20(_l2Token, l1Token, _from, _to, _amount, _minGasLimit, _data);
        }
        emit WithdrawalInitiated(l1Token, _l2Token, _from, _to, _amount, _data);
    }
}
