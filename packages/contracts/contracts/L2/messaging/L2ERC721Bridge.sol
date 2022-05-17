// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IL1ERC721Bridge } from "../../L1/messaging/IL1ERC721Bridge.sol";
import { IL1ERC721Bridge } from "../../L1/messaging/IL1ERC721Bridge.sol";
import { IL2ERC721Bridge } from "./IL2ERC721Bridge.sol";

/* Library Imports */
import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { CrossDomainEnabled } from "../../libraries/bridge/CrossDomainEnabled.sol";
import { Lib_PredeployAddresses } from "../../libraries/constants/Lib_PredeployAddresses.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/* Contract Imports */
import { IL2StandardERC721 } from "../../standards/IL2StandardERC721.sol";

/**
 * @title L2ERC721Bridge
 * @dev The L2 ERC721 bridge is a contract which works together with the L1 ERC721 bridge to
 * enable ERC721 transitions between L1 and L2.
 * This contract acts as a minter for new tokens when it hears about deposits into the L1 ERC721
 * bridge.
 * This contract also acts as a burner of the token intended for withdrawal, informing the L1
 * bridge to release the L1 NFT.
 */
contract L2ERC721Bridge is IL2ERC721Bridge, CrossDomainEnabled {
    /********************************
     * External Contract References *
     ********************************/

    address public l1ERC721Bridge;

    /***************
     * Constructor *
     ***************/

    /**
     * @param _l2CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _l1ERC721Bridge Address of the L1 bridge deployed to the main chain.
     */
    constructor(address _l2CrossDomainMessenger, address _l1ERC721Bridge)
        CrossDomainEnabled(_l2CrossDomainMessenger)
    {
        l1ERC721Bridge = _l1ERC721Bridge;
    }

    /***************
     * Withdrawing *
     ***************/

    /**
     * @inheritdoc IL2ERC721Bridge
     */
    function withdraw(
        address _l2Token,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    ) external virtual {
        // Modifier requiring sender to be EOA.  This check could be bypassed by a malicious
        // contract via initcode, but it takes care of the user error we want to avoid.
        require(!Address.isContract(msg.sender), "Account not EOA");

        _initiateWithdrawal(_l2Token, msg.sender, msg.sender, _tokenId, _l1Gas, _data);
    }

    /**
     * @inheritdoc IL2ERC721Bridge
     */
    function withdrawTo(
        address _l2Token,
        address _to,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    ) external virtual {
        _initiateWithdrawal(_l2Token, msg.sender, _to, _tokenId, _l1Gas, _data);
    }

    /**
     * @dev Performs the logic for withdrawals by burning the token and informing
     *      the L1 token Gateway of the withdrawal.
     * @param _l2Token Address of L2 token where withdrawal is initiated.
     * @param _from Account to pull the withdrawal from on L2.
     * @param _to Account to give the withdrawal to on L1.
     * @param _tokenId Token ID of the token to withdraw.
     * @param _l1Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateWithdrawal(
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    ) internal {
        // Check that the withdrawal is being initiated by the NFT owner
        require(
            _from == IL2StandardERC721(_l2Token).ownerOf(_tokenId),
            "Withdrawal is not being initiated by NFT owner"
        );

        // When a withdrawal is initiated, we burn the withdrawer's NFT to prevent subsequent L2
        // usage
        // slither-disable-next-line reentrancy-events
        IL2StandardERC721(_l2Token).burn(msg.sender, _tokenId);

        // Construct calldata for l1ERC721Bridge.finalizeERC721Withdrawal(_to, _tokenId)
        // slither-disable-next-line reentrancy-events
        address l1Token = IL2StandardERC721(_l2Token).l1Token();
        bytes memory message = abi.encodeWithSelector(
            IL1ERC721Bridge.finalizeERC721Withdrawal.selector,
            l1Token,
            _l2Token,
            _from,
            _to,
            _tokenId,
            _data
        );

        // Send message to L1 bridge
        // slither-disable-next-line reentrancy-events
        sendCrossDomainMessage(l1ERC721Bridge, _l1Gas, message);

        // slither-disable-next-line reentrancy-events
        emit WithdrawalInitiated(l1Token, _l2Token, msg.sender, _to, _tokenId, _data);
    }

    /************************************
     * Cross-chain Function: Depositing *
     ************************************/

    /**
     * @inheritdoc IL2ERC721Bridge
     */
    function finalizeDeposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    ) external virtual onlyFromCrossDomainAccount(l1ERC721Bridge) {
        // Check the target token is compliant and
        // verify the deposited token on L1 matches the L2 deposited token representation here
        if (
            // slither-disable-next-line reentrancy-events
            ERC165Checker.supportsInterface(_l2Token, 0x1d1d8b63) &&
            _l1Token == IL2StandardERC721(_l2Token).l1Token()
        ) {
            // When a deposit is finalized, we give the NFT with the same tokenId to the account
            // on L2.
            // slither-disable-next-line reentrancy-events
            IL2StandardERC721(_l2Token).mint(_to, _tokenId);
            // slither-disable-next-line reentrancy-events
            emit DepositFinalized(_l1Token, _l2Token, _from, _to, _tokenId, _data);
        } else {
            // Either the L2 token which is being deposited-into disagrees about the correct address
            // of its L1 token, or does not support the correct interface.
            // This should only happen if there is a  malicious L2 token, or if a user somehow
            // specified the wrong L2 token address to deposit into.
            // In either case, we stop the process here and construct a withdrawal
            // message so that users can get their NFT out in some cases.
            // There is no way to prevent malicious token contracts altogether, but this does limit
            // user error and mitigate some forms of malicious contract behavior.
            bytes memory message = abi.encodeWithSelector(
                IL1ERC721Bridge.finalizeERC721Withdrawal.selector,
                _l1Token,
                _l2Token,
                _to, // switched the _to and _from here to bounce back the deposit to the sender
                _from,
                _tokenId,
                _data
            );

            // Send message up to L1 bridge
            // slither-disable-next-line reentrancy-events
            sendCrossDomainMessage(l1ERC721Bridge, 0, message);
            // slither-disable-next-line reentrancy-events
            emit DepositFailed(_l1Token, _l2Token, _from, _to, _tokenId, _data);
        }
    }
}
