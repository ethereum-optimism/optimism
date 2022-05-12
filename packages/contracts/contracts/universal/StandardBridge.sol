// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

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
        bytes _data
    );

    event ETHBridgeFinalized(
        address indexed _from,
        address indexed _to,
        uint256 _amount,
        bytes _data
    );

    event ERC20BridgeInitiated(
        address indexed _localToken,
        address indexed _remoteToken,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

    event ERC20BridgeFinalized(
        address indexed _localToken,
        address indexed _remoteToken,
        address indexed _from,
        address _to,
        uint256 _amount,
        bytes _data
    );

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

    /********************
     * Public Functions *
     ********************/

    /**
     * @notice Send ETH to this contract. This is used during upgrades
     */
    function donateETH() external payable {}

    /**
     * @notice EOAs can simply send ETH to this contract to have it be deposited
     * to L2 through the standard bridge.
     */
    receive() external payable onlyEOA {
        _initiateBridgeETH(msg.sender, msg.sender, msg.value, 200_000, bytes(""));
    }

    /**
     * @notice Send ETH to the message sender on the remote domain
     */
    function bridgeETH(uint32 _minGasLimit, bytes calldata _data) public payable onlyEOA {
        _initiateBridgeETH(msg.sender, msg.sender, msg.value, _minGasLimit, _data);
    }

    /**
     * @notice Send ETH to a specified account on the remote domain
     */
    function bridgeETHTo(
        address _to,
        uint32 _minGasLimit,
        bytes calldata _data
    ) public payable {
        _initiateBridgeETH(msg.sender, _to, msg.value, _minGasLimit, _data);
    }

    /**
     * @notice Send an ERC20 to the message sender on the remote domain
     */
    function bridgeERC20(
        address _localToken,
        address _remoteToken,
        uint256 _amount,
        uint32 _minGasLimit,
        bytes calldata _data
    ) public virtual onlyEOA {
        _initiateBridgeERC20(
            _localToken,
            _remoteToken,
            msg.sender,
            msg.sender,
            _amount,
            _minGasLimit,
            _data
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
        bytes calldata _data
    ) public virtual {
        _initiateBridgeERC20(
            _localToken,
            _remoteToken,
            msg.sender,
            _to,
            _amount,
            _minGasLimit,
            _data
        );
    }

    /**
     * @notice Finalize an ETH sending transaction sent from a remote domain
     */
    function finalizeBridgeETH(
        address _from,
        address _to,
        uint256 _amount,
        bytes calldata _data
    ) public payable onlyOtherBridge {
        require(msg.value == _amount, "Amount sent does not match amount required.");

        emit ETHBridgeFinalized(_from, _to, _amount, _data);
        (bool success, ) = _to.call{ value: _amount }(new bytes(0));
        require(success, "TransferHelper::safeTransferETH: ETH transfer failed");
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
        bytes calldata _data
    ) public onlyOtherBridge {
        if (_isOptimismMintable(_localToken, _remoteToken)) {
            OptimismMintableERC20(_localToken).mint(_to, _amount);
        } else {
            deposits[_localToken][_remoteToken] = deposits[_localToken][_remoteToken] - _amount;
            IERC20(_localToken).safeTransfer(_to, _amount);
        }

        emit ERC20BridgeFinalized(_localToken, _remoteToken, _from, _to, _amount, _data);
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
        bytes memory _data
    ) internal {
        emit ETHBridgeInitiated(_from, _to, _amount, _data);

        messenger.sendMessage{ value: _amount }(
            address(otherBridge),
            abi.encodeWithSelector(this.finalizeBridgeETH.selector, _from, _to, _amount, _data),
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
        bytes calldata _data
    ) internal {
        if (_isOptimismMintable(_localToken, _remoteToken)) {
            OptimismMintableERC20(_localToken).burn(msg.sender, _amount);
        } else {
            IERC20(_localToken).safeTransferFrom(_from, address(this), _amount);
            deposits[_localToken][_remoteToken] = deposits[_localToken][_remoteToken] + _amount;
        }

        messenger.sendMessage(
            address(otherBridge),
            abi.encodeWithSelector(
                this.finalizeBridgeERC20.selector,
                _remoteToken,
                _localToken,
                _from,
                _to,
                _amount,
                _data
            ),
            _minGasLimit
        );

        emit ERC20BridgeInitiated(_localToken, _remoteToken, _from, _to, _amount, _data);
    }

    /**
     * @notice Check to make sure that the token pair is an OptimismMintable
     * token pair.
     * The selector 0x1d1d8b63 represents the ERC165 representation of
     * the methods l1Token(), mint(address,uint256), burn(address,uint256)
     * the selector 0x0bc32271 represents the ERC165 representation of
     * remoteToken(), mint(address,uint256), burn(address,uint256).
     * Both are required as l1Token() is a legacy function, prefer calling
     * remoteToken()
     */
    function _isOptimismMintable(address _localToken, address _remoteToken)
        internal
        view
        returns (bool)
    {
        return ((ERC165Checker.supportsInterface(_localToken, 0x1d1d8b63) &&
            _remoteToken == OptimismMintableERC20(_localToken).l1Token()) ||
            (ERC165Checker.supportsInterface(_localToken, 0x0bc32271) &&
                _remoteToken == OptimismMintableERC20(_localToken).remoteToken()));
    }
}
