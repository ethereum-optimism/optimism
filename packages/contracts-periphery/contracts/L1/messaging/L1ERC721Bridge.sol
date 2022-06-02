// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IL1ERC721Bridge } from "./IL1ERC721Bridge.sol";
import { IL2ERC721Bridge } from "../../L2/messaging/IL2ERC721Bridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

/* Library Imports */
import {
    CrossDomainEnabled
} from "@eth-optimism/contracts/libraries/bridge/CrossDomainEnabled.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";

/**
 * @title L1ERC721Bridge
 * @dev The L1 ERC721 Bridge is a contract which stores deposited L1 NFTs that are in use
 * on L2. It synchronizes a corresponding L2 Bridge, informing it of deposits and listening
 * to it for newly finalized withdrawals.
 */
contract L1ERC721Bridge is IL1ERC721Bridge, CrossDomainEnabled {
    /********************************
     * External Contract References *
     ********************************/

    address public l2ERC721Bridge;

    // Maps L1 token to L2 token to token ID to a boolean indicating if the token is deposited
    mapping(address => mapping(address => mapping(uint256 => bool))) public deposits;

    /***************
     * Constructor *
     ***************/

    // This contract lives behind a proxy, so the constructor parameters will go unused.
    constructor() CrossDomainEnabled(address(0)) {}

    /******************
     * Initialization *
     ******************/

    /**
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     * @param _l2ERC721Bridge L2 ERC721 bridge address.
     */
    // slither-disable-next-line external-function
    function initialize(address _l1messenger, address _l2ERC721Bridge) public {
        require(messenger == address(0), "Contract has already been initialized.");
        messenger = _l1messenger;
        l2ERC721Bridge = _l2ERC721Bridge;
    }

    /**************
     * Depositing *
     **************/

    /**
     * @inheritdoc IL1ERC721Bridge
     */
    function depositERC721(
        address _l1Token,
        address _l2Token,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    ) external virtual {
        // Modifier requiring sender to be EOA.  This check could be bypassed by a malicious
        // contract via initcode, but it takes care of the user error we want to avoid.
        require(!Address.isContract(msg.sender), "Account not EOA");

        _initiateERC721Deposit(_l1Token, _l2Token, msg.sender, msg.sender, _tokenId, _l2Gas, _data);
    }

    /**
     * @inheritdoc IL1ERC721Bridge
     */
    function depositERC721To(
        address _l1Token,
        address _l2Token,
        address _to,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    ) external virtual {
        _initiateERC721Deposit(_l1Token, _l2Token, msg.sender, _to, _tokenId, _l2Gas, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 NFT. (e.g. transferFrom)
     *
     * @param _l1Token Address of the L1 ERC721 we are depositing
     * @param _l2Token Address of the L1 respective L2 ERC721
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _tokenId Token ID of the ERC721 to deposit.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateERC721Deposit(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    ) internal {
        // When a deposit is initiated on L1, the L1 Bridge transfers the NFT to itself for future
        // withdrawals.
        // slither-disable-next-line reentrancy-events, reentrancy-benign
        IERC721(_l1Token).transferFrom(_from, address(this), _tokenId);

        // Construct calldata for _l2Token.finalizeDeposit(_to, _tokenId)
        bytes memory message = abi.encodeWithSelector(
            IL2ERC721Bridge.finalizeDeposit.selector,
            _l1Token,
            _l2Token,
            _from,
            _to,
            _tokenId,
            _data
        );

        // Send calldata into L2
        // slither-disable-next-line reentrancy-events, reentrancy-benign
        sendCrossDomainMessage(l2ERC721Bridge, _l2Gas, message);

        // slither-disable-next-line reentrancy-benign
        deposits[_l1Token][_l2Token][_tokenId] = true;

        // slither-disable-next-line reentrancy-events
        emit ERC721DepositInitiated(_l1Token, _l2Token, _from, _to, _tokenId, _data);
    }

    /*************************
     * Cross-chain Functions *
     *************************/

    /**
     * @inheritdoc IL1ERC721Bridge
     */
    function finalizeERC721Withdrawal(
        address _l1Token,
        address _l2Token,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    ) external onlyFromCrossDomainAccount(l2ERC721Bridge) {
        // Checks that the L1/L2 token pair has a token ID that is escrowed in the L1 Bridge
        require(
            deposits[_l1Token][_l2Token][_tokenId] == true,
            "Token ID is not escrowed in the L1 Bridge"
        );

        deposits[_l1Token][_l2Token][_tokenId] = false;

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the NFT to the withdrawer
        // slither-disable-next-line reentrancy-events
        IERC721(_l1Token).transferFrom(address(this), _to, _tokenId);

        // slither-disable-next-line reentrancy-events
        emit ERC721WithdrawalFinalized(_l1Token, _l2Token, _from, _to, _tokenId, _data);
    }
}
