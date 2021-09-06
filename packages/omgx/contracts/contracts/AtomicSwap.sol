// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";

contract AtomicSwap {

    using SafeERC20 for IERC20;

    /**
     * @openValue The amount that seller wants to sell
     * @openTrader The seller address
     * @openContractAddress The token address
     * @closeValue The amount that buyer wants to pay
     * @closeTrader The buyer address
     * @closeContractAddress The token address
     */
    struct Swap {
        uint256 openValue;
        address openTrader;
        address openContractAddress;
        uint256 closeValue;
        address closeTrader;
        address closeContractAddress;
    }

    enum States {
        INVALID,
        OPEN,
        CLOSED,
        EXPIRED
    }

    mapping (bytes32 => Swap) private swaps;
    mapping (bytes32 => States) private swapStates;

    event Open(bytes32 _swapID, address _closeTrader);
    event Expire(bytes32 _swapID);
    event Close(bytes32 _swapID);

    modifier onlyInvalidSwaps(bytes32 _swapID) {
        require (swapStates[_swapID] == States.INVALID, "Swap Id is not fresh");
        _;
    }

    modifier onlyOpenSwaps(bytes32 _swapID) {
        require (swapStates[_swapID] == States.OPEN, "Swap Id is not open");
        _;
    }

    modifier onlyCloseTrader(bytes32 _swapID) {
        Swap memory swap = swaps[_swapID];
        require(msg.sender == swap.closeTrader, "Caller not authorized close Trader");
        _;
    }

    modifier onlyTraders(bytes32 _swapID) {
        Swap memory swap = swaps[_swapID];
        require(msg.sender == swap.openTrader || msg.sender == swap.closeTrader, "Caller not authorized for swap");
        _;
    }

    function open(
        bytes32 _swapID,
        uint256 _openValue,
        address _openContractAddress,
        uint256 _closeValue,
        address _closeTrader,
        address _closeContractAddress
    )
        public
        onlyInvalidSwaps(_swapID)
    {
        require(_openContractAddress != address(0) && _closeContractAddress != address(0), "Address should be non zero");
        require(_closeTrader != address(0) && _closeTrader != msg.sender, "Close trader incorrect");
        // Store the details of the swap.
        swaps[_swapID] = Swap({
            openValue: _openValue,
            openTrader: msg.sender,
            openContractAddress: _openContractAddress,
            closeValue: _closeValue,
            closeTrader: _closeTrader,
            closeContractAddress: _closeContractAddress
        });
        swapStates[_swapID] = States.OPEN;

        emit Open(_swapID, _closeTrader);
    }

    function close(
        bytes32 _swapID
    )
        public
        onlyOpenSwaps(_swapID)
        onlyCloseTrader(_swapID)
    {
        Swap memory swap = swaps[_swapID];

        swapStates[_swapID] = States.CLOSED;

        // Transfer the closing funds from the closing trader to the opening trader.
        IERC20(swap.closeContractAddress).safeTransferFrom(swap.closeTrader, swap.openTrader, swap.closeValue);

        // Transfer the opening funds from opening trader to the closing trader.
        IERC20(swap.openContractAddress).safeTransferFrom(swap.openTrader, swap.closeTrader, swap.openValue);

        emit Close(_swapID);
    }

    function expire(
        bytes32 _swapID
    )
        public
        onlyOpenSwaps(_swapID)
        onlyTraders(_swapID)
    {
        // Expire the swap.
        swapStates[_swapID] = States.EXPIRED;

        emit Expire(_swapID);
    }

    function check(
        bytes32 _swapID
    )
        public
        view
        returns (
            uint256 openValue,
            address openContractAddress,
            uint256 closeValue,
            address closeTrader,
            address closeContractAddress
        )
    {
        Swap memory swap = swaps[_swapID];
        return (
            swap.openValue,
            swap.openContractAddress,
            swap.closeValue,
            swap.closeTrader,
            swap.closeContractAddress
        );
    }
}