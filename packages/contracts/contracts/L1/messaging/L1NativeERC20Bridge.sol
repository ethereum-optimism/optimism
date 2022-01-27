// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IL1NativeERC20Bridge } from "./IL1NativeERC20Bridge.sol";
import { IL2NativeERC20Bridge } from "../../L2/messaging/IL2NativeERC20Bridge.sol";

/* Library Imports */
import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { CrossDomainEnabled } from "../../libraries/bridge/CrossDomainEnabled.sol";

/* Contract Imports */
import { IL1StandardERC20 } from "../../standards/IL1StandardERC20.sol";

/**
 * @title L1NativeERC20Bridge
 * @dev
 */
contract L1NativeERC20Bridge is IL1NativeERC20Bridge, CrossDomainEnabled {
    /********************************
     * External Contract References *
     ********************************/

    address public l2TokenBridge;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l1CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _l2TokenBridge Address of the L2 bridge deployed on Optimism.
     */
    constructor(address _l1CrossDomainMessenger, address _l2TokenBridge)
        CrossDomainEnabled(_l1CrossDomainMessenger)
    {
        l2TokenBridge = _l2TokenBridge;
    }

    /***************
     * Withdrawing *
     ***************/

    /**
     * @inheritdoc IL1NativeERC20Bridge
     */
    function withdraw(
        address _l1Token,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    ) external virtual {
        _initiateWithdrawal(_l1Token, msg.sender, msg.sender, _amount, _l2Gas, _data);
    }

    /**
     * @inheritdoc IL1NativeERC20Bridge
     */
    function withdrawTo(
        address _l1Token,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    ) external virtual {
        _initiateWithdrawal(_l1Token, msg.sender, _to, _amount, _l2Gas, _data);
    }

    /**
     * @dev Performs the logic for the withdrawal by burning the token and informing the L2 token Gateway
     * of the withdrawal.
     * @param _l1Token Address of L2 token where withdrawal was initiated.
     * @param _from Account to burn funds from on L1.
     * @param _to Account to give the withdrawal to on L2.
     * @param _amount Amount of the token to withdraw.
     * param _l2Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateWithdrawal(
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _l2Gas,
        bytes calldata _data
    ) internal {
        // When a withdrawal is initiated, we burn the withdrawer's funds to prevent subsequent L1
        // usage
        // slither-disable-next-line reentrancy-events
        IL1StandardERC20(_l1Token).burn(msg.sender, _amount);

        // Construct calldata for l1TokenBridge.finalizeERC20Withdrawal(_to, _amount)
        // slither-disable-next-line reentrancy-events
        address l2Token = IL1StandardERC20(_l1Token).l2Token();
        bytes memory message;

        message = abi.encodeWithSelector(
            IL2NativeERC20Bridge.finalizeERC20Withdrawal.selector,
            l2Token,
            _l1Token,
            _from,
            _to,
            _amount,
            _data
        );

        // Send message up to L2 bridge
        // slither-disable-next-line reentrancy-events
        sendCrossDomainMessage(l2TokenBridge, _l2Gas, message);

        // slither-disable-next-line reentrancy-events
        emit NativeERC20WithdrawalInitiated(_l1Token, l2Token, msg.sender, _to, _amount, _data);
    }

    /************************************
     * Cross-chain Function: Depositing *
     ************************************/

    /**
     * @inheritdoc IL1NativeERC20Bridge
     */
    function finalizeDeposit(
        address _l2Token,
        address _l1Token,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) external virtual onlyFromCrossDomainAccount(l2TokenBridge) {
        // Check the target token is compliant and
        // verify the deposited token on L2 matches the L1 deposited token representation here
        if (
            // slither-disable-next-line reentrancy-events
            ERC165Checker.supportsInterface(_l1Token, 0x8bec62d2) &&
            _l2Token == IL1StandardERC20(_l1Token).l2Token()
        ) {
            // When a deposit is finalized, we credit the account on L1 with the same amount of
            // tokens.
            // slither-disable-next-line reentrancy-events
            IL1StandardERC20(_l1Token).mint(_to, _amount);
            // slither-disable-next-line reentrancy-events
            emit NativeERC20DepositFinalized(_l1Token, _l2Token, _from, _to, _amount, _data);
        } else {
            // Either the L1 token which is being deposited-into disagrees about the correct address
            // of its L2 token, or does not support the correct interface.
            // This should only happen if there is a malicious L1 token, or if a user somehow
            // specified the wrong L1 token address to deposit into.
            // In either case, we stop the process here and construct a withdrawal message so that
            // users can get their funds out in some cases.
            // There is no way to prevent malicious token contracts altogether, but this does limit
            // user error and mitigate some forms of malicious contract behavior.
            bytes memory message = abi.encodeWithSelector(
                IL2NativeERC20Bridge.finalizeERC20Withdrawal.selector,
                _l2Token,
                _l1Token,
                _to, // switched the _to and _from here to bounce back the deposit to the sender
                _from,
                _amount,
                _data
            );

            // Send message up to L2 bridge
            // slither-disable-next-line reentrancy-events
            sendCrossDomainMessage(l2TokenBridge, 0, message);
            // slither-disable-next-line reentrancy-events
            emit NativeERC20DepositFailed(_l1Token, _l2Token, _from, _to, _amount, _data);
        }
    }
}
