// SPDX-License-Identifier: MIT
// @unsupported: ovm
pragma solidity 0.7.6;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_L1NFTBridge } from "./interfaces/iOVM_L1NFTBridge.sol";
import { iOVM_L2NFTBridge } from "./interfaces/iOVM_L2NFTBridge.sol";
import { IERC721 } from "@openzeppelin/contracts/token/ERC721/IERC721.sol";

/* Library Imports */
import { OVM_CrossDomainEnabled } from "@eth-optimism/contracts/contracts/optimistic-ethereum/libraries/bridge/OVM_CrossDomainEnabled.sol";
import { Lib_PredeployAddresses } from "@eth-optimism/contracts/contracts/optimistic-ethereum/libraries/constants/Lib_PredeployAddresses.sol";
import { SafeMath } from "@openzeppelin/contracts/math/SafeMath.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { ERC721Holder } from "@openzeppelin/contracts/token/ERC721/ERC721Holder.sol";

/**
 * @title OVM_L1NFTBridge
 * @dev The L1 NFT Bridge is a contract which stores deposited L1 ERC721
 * tokens that are in use on L2. It synchronizes a corresponding L2 Bridge, informing it of deposits
 * and listening to it for newly finalized withdrawals.
 *
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L1NFTBridge is iOVM_L1NFTBridge, OVM_CrossDomainEnabled, ERC721Holder {
    using SafeMath for uint;

    /********************************
     * External Contract References *
     ********************************/

    address public l2NFTBridge;

    // Maps L1 token to tokenId to L2 token contract deposited
    mapping(address => mapping (uint256 => address)) public deposits;

    /***************
     * Constructor *
     ***************/

    // This contract lives behind a proxy, so the constructor parameters will go unused.
    constructor()
        OVM_CrossDomainEnabled(address(0))
    {}

    /******************
     * Initialization *
     ******************/

    /**
     * @param _l1messenger L1 Messenger address being used for cross-chain communications.
     * @param _l2NFTBridge L2 NFT bridge address.
     */
    function initialize(
        address _l1messenger,
        address _l2NFTBridge
    )
        public
    {
        require(messenger == address(0), "Contract has already been initialized.");
        messenger = _l1messenger;
        l2NFTBridge = _l2NFTBridge;
    }

    /**************
     * Depositing *
     **************/

    /** @dev Modifier requiring sender to be EOA.  This check could be bypassed by a malicious
     *  contract via initcode, but it takes care of the user error we want to avoid.
     */
    modifier onlyEOA() {
        // Used to stop deposits from contracts (avoid accidentally lost tokens)
        require(!Address.isContract(msg.sender), "Account not EOA");
        _;
    }

    // /**
    //  * @inheritdoc iOVM_L1NFTBridge
    //  */
    function depositNFT(
        address _l1Contract,
        address _l2Contract,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        virtual
        override
        onlyEOA()
    {
        _initiateNFTDeposit(_l1Contract, _l2Contract, msg.sender, msg.sender, _tokenId, _l2Gas, _data);
    }

    //  /**
    //  * @inheritdoc iOVM_L1NFTBridge
    //  */
    function depositNFTTo(
        address _l1Contract,
        address _l2Contract,
        address _to,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    )
        external
        override
        virtual
    {
        _initiateNFTDeposit(_l1Contract, _l2Contract, msg.sender, _to, _tokenId, _l2Gas, _data);
    }

    /**
     * @dev Performs the logic for deposits by informing the L2 Deposited Token
     * contract of the deposit and calling a handler to lock the L1 token. (e.g. transferFrom)
     *
     * @param _l1Contract Address of the L1 NFT contract we are depositing
     * @param _l2Contract Address of the respective L2 NFT contract
     * @param _from Account to pull the deposit from on L1
     * @param _to Account to give the deposit to on L2
     * @param _tokenId NFT token Id to deposit.
     * @param _l2Gas Gas limit required to complete the deposit on L2.
     * @param _data Optional data to forward to L2. This data is provided
     *        solely as a convenience for external contracts. Aside from enforcing a maximum
     *        length, these contracts provide no guarantees about its content.
     */
    function _initiateNFTDeposit(
        address _l1Contract,
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        uint32 _l2Gas,
        bytes calldata _data
    )
        internal
    {
        // When a deposit is initiated on L1, the L1 Bridge transfers the funds to itself for future
        // withdrawals. safeTransferFrom also checks if the contract has code, so this will fail if
        // _from is an EOA or address(0).
        IERC721(_l1Contract).safeTransferFrom(
            _from,
            address(this),
            _tokenId
        );

        // Construct calldata for _l2Contract.finalizeDeposit(_to, _amount)
        bytes memory message = abi.encodeWithSelector(
            iOVM_L2NFTBridge.finalizeDeposit.selector,
            _l1Contract,
            _l2Contract,
            _from,
            _to,
            _tokenId,
            _data
        );

        // Send calldata into L2
        sendCrossDomainMessage(
            l2NFTBridge,
            _l2Gas,
            message
        );

        deposits[_l1Contract][_tokenId] = _l2Contract;

        emit NFTDepositInitiated(_l1Contract, _l2Contract, _from, _to, _tokenId, _data);
    }

    // /**
    //  * @inheritdoc iOVM_L1NFTBridge
    //  */
    function finalizeNFTWithdrawal(
        address _l1Contract,
        address _l2Contract,
        address _from,
        address _to,
        uint256 _tokenId,
        bytes calldata _data
    )
        external
        override
        onlyFromCrossDomainAccount(l2NFTBridge)
    {
        // needs to verify comes from correct l2Contract
        require(deposits[_l1Contract][_tokenId] == _l2Contract, "Incorrect Burn");

        // When a withdrawal is finalized on L1, the L1 Bridge transfers the funds to the withdrawer
        IERC721(_l1Contract).safeTransferFrom(address(this), _to, _tokenId);

        emit NFTWithdrawalFinalized(_l1Contract, _l2Contract, _from, _to, _tokenId, _data);
    }
}
