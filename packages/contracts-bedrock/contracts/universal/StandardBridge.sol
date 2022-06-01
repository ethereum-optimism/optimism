// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "./SupportedInterfaces.sol";

/* Library Imports */
import { ERC165Checker } from "@openzeppelin/contracts/utils/introspection/ERC165Checker.sol";
import { Address } from "@openzeppelin/contracts/utils/Address.sol";
import { SafeERC20 } from "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

import { CrossDomainMessenger } from "./CrossDomainMessenger.sol";
import { OptimismMintableERC20 } from "./OptimismMintableERC20.sol";

/**
 * @title StandardBridge
 * This contract can manage a 1:1 bridge between two domains for both
 * ETH (native asset) and ERC20s.
 * This contract should be deployed behind a proxy.
 * TODO: do we want a donateERC20 function as well?
 */
abstract contract StandardBridge {
    using SafeERC20 for IERC20;

    /**********
     * Events *
     **********/

    event ETHBridgeInitiated(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _extraData
    );

    event ETHBridgeFinalized(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _extraData
    );

    event ERC20BridgeInitiated(
        address indexed _localToken,
        address indexed _remoteToken,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _extraData
    );

    event ERC20BridgeFinalized(
        address indexed _localToken,
        address indexed _remoteToken,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _extraData
    );

    event ERC20BridgeFailed(
        address indexed _localToken,
        address indexed _remoteToken,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _extraData
    );

    /*************
     * Constants *
     *************/

    /**
     * @notice The L2 gas limit set when eth is depoisited using the receive() function.
     */
    uint32 internal constant RECEIVE_DEFAULT_GAS_LIMIT = 200_000;

    /*************
     * Variables *
     *************/

    /**
     * @notice The messenger contract on the same domain
     */
    CrossDomainMessenger public messenger;

    /**
     * @notice The corresponding bridge on the other domain
     */
    StandardBridge public otherBridge;

    mapping(address => mapping(address => uint256)) public deposits;

    /*************
     * Modifiers *
     *************/

    /**
     * @notice Only allow EOAs to call the functions. Note that this
     * is not safe against contracts calling code during their constructor
     */
    modifier onlyEOA() {
        require(!Address.isContract(msg.sender), "Account not EOA");
        _;
    }

    /**
     * @notice Ensures that the caller is the messenger, and that
     * it has the l2Sender value set to the address of the remote Token Bridge.
     */
    modifier onlyOtherBridge() {
        require(
            msg.sender == address(messenger) &&
                messenger.xDomainMessageSender() == address(otherBridge),
            "Could not authenticate bridge message."
        );
        _;
    }

    modifier onlySelf() {
        require(msg.sender == address(this), "Function can only be called by self.");
        _;
    }

    /********************
     * Public Functions *
     ********************/

    /**
     * @notice EOAs can simply send ETH to this contract to have it be deposited
     * to L2 through the standard bridge.
     */
    receive() external payable onlyEOA {
        _initiateBridgeETH(msg.sender, msg.sender, msg.value, RECEIVE_DEFAULT_GAS_LIMIT, bytes(""));
    }

    /**
     * @notice Send ETH to the message sender on the remote domain
     */
    function bridgeETH(uint32 _minGasLimit, bytes calldata _extraData) public payable onlyEOA {
        _initiateBridgeETH(msg.sender, msg.sender, msg.value, _minGasLimit, _extraData);
    }

    /**
     * @notice Send ETH to a specified account on the remote domain. Note that if ETH is sent to a
     *         contract and the call fails, then that ETH will be locked in the other bridge.
     */
    function bridgeETHTo(
        address _to,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public payable {
        _initiateBridgeETH(msg.sender, _to, msg.value, _minGasLimit, _extraData);
    }

    /**
     * @notice Send an ERC20 to the message sender on the remote domain
     */
    function bridgeERC20(
        address _localToken,
        address _remoteToken,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public virtual onlyEOA {
        _initiateBridgeERC20(
            _localToken,
            _remoteToken,
            msg.sender,
            msg.sender,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /**
     * @notice Send an ERC20 to a specified account on the remote domain
     */
    function bridgeERC20To(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) public virtual {
        _initiateBridgeERC20(
            _localToken,
            _remoteToken,
            msg.sender,
            _to,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /**
     * @notice Finalize an ETH sending transaction sent from a remote domain
     */
    function finalizeBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    ) public payable onlyOtherBridge {
        require(msg.value == _amount, "Amount sent does not match amount required.");
        require(_to != address(this), "Cannot send to self.");

        emit ETHBridgeFinalized(_from, _to, _amount, _extraData);
        (bool success, ) = _to.call{ value: _amount }(new bytes(0));
        require(success, "ETH transfer failed.");
    }

    /**
     * @notice Finalize an ERC20 sending transaction sent from a remote domain
     */
    function finalizeBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _extraData
    ) public onlyOtherBridge {
        try this.completeOutboundTransfer(_localToken, _remoteToken, _to, _amount) {
            emit ERC20BridgeFinalized(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        } catch {
            // Something went wrong during the bridging process, return to sender.
            // Can happen if a bridge UI specifies the wrong L2 token.
            // We reverse both the local and remote token addresses, as well as the to and from
            // addresses. This will preserve the accuracy of accounting based on emitted events.
            _initiateBridgeERC20Unchecked(
                _localToken,
                _remoteToken,
                _to,
                _from,
                _amount,
                0, // _minGasLimit, 0 is fine here
                _extraData
            );
            emit ERC20BridgeFailed(_localToken, _remoteToken, _from, _to, _amount, _extraData);
        }
    }

    function completeOutboundTransfer(
        address _localToken,
        address _remoteToken,
        address _to,
        uint256 _amount
    ) public onlySelf {
        // Make sure external function calls can't be used to trigger calls to
        // completeOutboundTransfer. We only make external (write) calls to _localToken.
        require(_localToken != address(this), "Local token cannot be self");

        if (_isOptimismMintableERC20(_localToken)) {
            require(
                _isCorrectTokenPair(_localToken, _remoteToken),
                "Wrong remote token for Optimism Mintable ERC20 local token"
            );

            OptimismMintableERC20(_localToken).mint(_to, _amount);
        } else {
            deposits[_localToken][_remoteToken] = deposits[_localToken][_remoteToken] - _amount;
            IERC20(_localToken).safeTransfer(_to, _amount);
        }
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * @notice Initialize the StandardBridge contract with the address of
     * the messenger on the same domain as well as the address of the bridge
     * on the remote domain
     */
    function _initialize(address payable _messenger, address payable _otherBridge) internal {
        require(address(messenger) == address(0), "Contract has already been initialized.");

        messenger = CrossDomainMessenger(_messenger);
        otherBridge = StandardBridge(_otherBridge);
    }

    /**
     * @notice Bridge ETH to the remote chain through the messenger
     */
    function _initiateBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes memory _extraData
    ) internal {
        emit ETHBridgeInitiated(_from, _to, _amount, _extraData);

        messenger.sendMessage{ value: _amount }(
            address(otherBridge),
            abi.encodeWithSelector(
                this.finalizeBridgeETH.selector,
                _from,
                _to,
                _amount,
                _extraData
            ),
            _minGasLimit
        );
    }

    /**
     * @notice Bridge an ERC20 to the remote chain through the messengers
     */
    function _initiateBridgeERC20(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal {
        // Make sure external function calls can't be used to trigger calls to
        // completeOutboundTransfer. We only make external (write) calls to _localToken.
        require(_localToken != address(this), "Local token cannot be self");

        if (_isOptimismMintableERC20(_localToken)) {
            require(
                _isCorrectTokenPair(_localToken, _remoteToken),
                "Wrong remote token for Optimism Mintable ERC20 local token"
            );

            OptimismMintableERC20(_localToken).burn(_from, _amount);
        } else {
            // TODO: Do we need to confirm that the transfer was successful?
            IERC20(_localToken).safeTransferFrom(_from, address(this), _amount);
            deposits[_localToken][_remoteToken] = deposits[_localToken][_remoteToken] + _amount;
        }

        _initiateBridgeERC20Unchecked(
            _localToken,
            _remoteToken,
            _from,
            _to,
            _amount,
            _minGasLimit,
            _extraData
        );
    }

    /**
     * @notice Bridge an ERC20 to the remote chain through the messengers
     */
    function _initiateBridgeERC20Unchecked(
        address _localToken,
        address _remoteToken,
        address _from,
        address _to,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _extraData
    ) internal {
        messenger.sendMessage(
            address(otherBridge),
            abi.encodeWithSelector(
                this.finalizeBridgeERC20.selector,
                // Because this call will be executed on the remote chain, we reverse the order of
                // the remote and local token addresses relative to their order in the
                // finalizeBridgeERC20 function.
                _remoteToken,
                _localToken,
                _from,
                _to,
                _amount,
                _extraData
            ),
            _minGasLimit
        );

        emit ERC20BridgeInitiated(_localToken, _remoteToken, _from, _to, _amount, _extraData);
    }

    /**
     * Checks if a given address is an OptimismMintableERC20. Not perfect, but good enough.
     * Just the way we like it.
     *
     * @param _token Address of the token to check.
     * @return True if the token is an OptimismMintableERC20.
     */
    function _isOptimismMintableERC20(address _token) internal view returns (bool) {
        return ERC165Checker.supportsInterface(_token, type(IL1Token).interfaceId);
    }

    /**
     * Checks if the "other token" is the correct pair token for the OptimismMintableERC20.
     *
     * @param _mintableToken OptimismMintableERC20 to check against.
     * @param _otherToken Pair token to check.
     * @return True if the other token is the correct pair token for the OptimismMintableERC20.
     */
    function _isCorrectTokenPair(address _mintableToken, address _otherToken)
        internal
        view
        returns (bool)
    {
        return _otherToken == OptimismMintableERC20(_mintableToken).l1Token();
    }
}
