// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1NFTBridge } from "./interfaces/iOVM_L1NFTBridge.sol";
import { iOVM_L2NFTBridge } from "./interfaces/iOVM_L2NFTBridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

/* Library Imports */
import { ERC165Checker } from "@openzeppelin/contracts/introspection/ERC165Checker.sol";
import { OVM_CrossDomainEnabled } from "@eth-optimism/contracts/contracts/optimistic-ethereum/libraries/bridge/OVM_CrossDomainEnabled.sol";

/* Contract Imports */
import { IL2StandardERC721 } from "../standards/IL2StandardERC721.sol";

/**
 * @title OVM_L2NFTBridge
 * @dev The L2 NFT bridge is a contract which works together with the L1 Standard bridge to
 * enable ERC721 transitions between L1 and L2.
 * This contract acts as a minter for new tokens when it hears about deposits into the L1 Standard
 * bridge.
 * This contract also acts as a burner of the tokens intended for withdrawal, informing the L1
 * bridge to release L1 funds.
 *
 * Compiler used: optimistic-solc
 * Runtime target: OVM
 */
 // add is interface
contract OVM_L2NFTBridge is iOVM_L2NFTBridge, OVM_CrossDomainEnabled {

    /********************************
     * External Contract References *
     ********************************/

    address public l1NFTBridge;

    /***************
     * Constructor *
     ***************/

    constructor()
        OVM_CrossDomainEnabled(address(0))
    {}

    /**
     * @param _l2CrossDomainMessenger Cross-domain messenger used by this contract.
     * @param _l1NFTBridge Address of the L1 bridge deployed to the main chain.
     */
    function initialize(
        address _l2CrossDomainMessenger,
        address _l1NFTBridge
    )
        public
    {
        require(messenger == address(0), "Contract has already been initialized.");
        messenger = _l2CrossDomainMessenger;
        l1NFTBridge = _l1NFTBridge;
    }

    /***************
     * Withdrawing *
     ***************/

    // /**
    //  * @inheritdoc iOVM_L2NFTBridge
    //  */
    function withdraw(
        address _l2Contract,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external
        virtual
        override
    {
        _initiateWithdrawal(
            _l2Contract,
            msg.sender,
            msg.sender,
            _tokenId,
            _l1Gas,
            _data
        );
    }

    // /**
    //  * @inheritdoc iOVM_L2NFTBridge
    //  */
    function withdrawTo(
        address _l2Contract,
        address _to,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    )
        external
        virtual
        override
    {
        _initiateWithdrawal(
            _l2Contract,
            msg.sender,
            _to,
            _tokenId,
            _l1Gas,
            _data
        );
    }

    /**
     * @dev Performs the logic for withdrawals by burning the token and informing the L1 ERC721 Gateway
     * of the withdrawal.
     * @param _l2Contract Address of L2 ERC721 where withdrawal was initiated.
     * @param _from Account to pull the deposit from on L2.
     * @param _to Account to give the withdrawal to on L1.
     * @param _tokenId Amount of the token to withdraw.
     * param _l1Gas Unused, but included for potential forward compatibility considerations.
     * @param _data Optional data to forward to L1. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateWithdrawal(
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _l1Gas,
        bytes calldata _data
    )
        internal
    {
        // When a withdrawal is initiated, we burn the withdrawer's funds to prevent subsequent L2
        // usage

        address owner = IL2StandardERC721(_l2Contract).ownerOf(_tokenId);
        require(msg.sender == owner || IL2StandardERC721(_l2Contract).getApproved(_tokenId) == msg.sender || IL2StandardERC721(_l2Contract).isApprovedForAll(owner, msg.sender));

        IL2StandardERC721(_l2Contract).burn(_tokenId);

        // Construct calldata for l1NFTBridge.finalizeNFTWithdrawal(_to, _amount)
        address l1Contract = IL2StandardERC721(_l2Contract).l1Contract();
        bytes memory message;

        message = abi.encodeWithSelector(
                    iOVM_L1NFTBridge.finalizeNFTWithdrawal.selector,
                    l1Contract,
                    _l2Contract,
                    _from,
                    _to,
                    _tokenId,
                    _data
                );

        // Send message up to L1 bridge
        sendCrossDomainMessage(
            l1NFTBridge,
            _l1Gas,
            message
        );

        emit WithdrawalInitiated(l1Contract, _l2Contract, msg.sender, _to, _tokenId, _data);
    }

    /************************************
     * Cross-chain Function: Depositing *
     ************************************/

    // /**
    //  * @inheritdoc iOVM_L2ERC20Bridge
    //  */
    function finalizeDeposit(
        address _l1Contract,
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    )
        external
        virtual
        override
        onlyFromCrossDomainAccount(l1NFTBridge)
    {
        // Check the target token is compliant and
        // verify the deposited token on L1 matches the L2 deposited token representation here
        if (
            // check with interface of IL2StandardERC721
            ERC165Checker.supportsInterface(_l2Contract, 0x646dd6ec) &&
            _l1Contract == IL2StandardERC721(_l2Contract).l1Contract()
        ) {
            // When a deposit is finalized, we credit the account on L2 with the same amount of
            // tokens.
            IL2StandardERC721(_l2Contract).mint(_to, _tokenId);
            emit DepositFinalized(_l1Contract, _l2Contract, _from, _to, _tokenId, _data);
        } else {
            // Either the L2 token which is being deposited-into disagrees about the correct address
            // of its L1 token, or does not support the correct interface.
            // This should only happen if there is a  malicious L2 token, or if a user somehow
            // specified the wrong L2 token address to deposit into.
            // In either case, we stop the process here and construct a withdrawal
            // message so that users can get their funds out in some cases.
            // There is no way to prevent malicious token contracts altogether, but this does limit
            // user error and mitigate some forms of malicious contract behavior.
            bytes memory message = abi.encodeWithSelector(
                iOVM_L1NFTBridge.finalizeNFTWithdrawal.selector,
                _l1Contract,
                _l2Contract,
                _to,   // switched the _to and _from here to bounce back the deposit to the sender
                _from,
                _tokenId,
                _data
            );

            // Send message up to L1 bridge
            sendCrossDomainMessage(
                l1NFTBridge,
                0,
                message
            );
            emit DepositFailed(_l1Contract, _l2Contract, _from, _to, _tokenId, _data);
        }
    }
}
