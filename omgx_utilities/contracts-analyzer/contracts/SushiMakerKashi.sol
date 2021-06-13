// SPDX-License-Identifier: MIT

pragma solidity 0.6.12;
import "./libraries/SafeMath.sol";
import "./libraries/SafeERC20.sol";

import "./uniswapv2/interfaces/IUniswapV2Pair.sol";
import "./uniswapv2/interfaces/IUniswapV2Factory.sol";

import "./Ownable.sol";

interface IBentoBoxWithdraw {
    function withdraw(
        IERC20 token_,
        address from,
        address to,
        uint256 amount,
        uint256 share
    ) external returns (uint256 amountOut, uint256 shareOut);
}

interface IKashiWithdrawFee {
    function asset() external view returns (address);
    function balanceOf(address account) external view returns (uint256);
    function withdrawFees() external;
    function removeAsset(address to, uint256 fraction) external returns (uint256 share);
}

// SushiMakerKashi is MasterChef's left hand and kinda a wizard. He can cook up Sushi from pretty much anything!
// This contract handles "serving up" rewards for xSushi holders by trading tokens collected from Kashi fees for Sushi.
contract SushiMakerKashi is Ownable {
    using SafeMath for uint256;
    using SafeERC20 for IERC20;

    IUniswapV2Factory private immutable factory;
    //0xC0AEe478e3658e2610c5F7A4A2E1777cE9e4f2Ac
    address private immutable bar;
    //0x8798249c2E607446EfB7Ad49eC89dD1865Ff4272
    IBentoBoxWithdraw private immutable bentoBox;
    //0xF5BCE5077908a1b7370B9ae04AdC565EBd643966 
    address private immutable sushi;
    //0x6B3595068778DD592e39A122f4f5a5cF09C90fE2
    address private immutable weth;
    //0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2
    bytes32 private immutable pairCodeHash;
    //0xe18a34eb0e04b04f7a0ac29a6e80748dca96319b42c54d679cb821dca90c6303

    mapping(address => address) private _bridges;

    event LogBridgeSet(address indexed token, address indexed bridge);
    event LogConvert(
        address indexed server,
        address indexed token0,
        uint256 amount0,
        uint256 amountBENTO,
        uint256 amountSUSHI
    );

    constructor(
        IUniswapV2Factory _factory,
        address _bar,
        IBentoBoxWithdraw _bentoBox,
        address _sushi,
        address _weth,
        bytes32 _pairCodeHash
    ) public {
        factory = _factory;
        bar = _bar;
        bentoBox = _bentoBox;
        sushi = _sushi;
        weth = _weth;
        pairCodeHash = _pairCodeHash;
    }

    function setBridge(address token, address bridge) external onlyOwner {
        // Checks
        require(
            token != sushi && token != weth && token != bridge,
            "Maker: Invalid bridge"
        );
        // Effects
        _bridges[token] = bridge;
        emit LogBridgeSet(token, bridge);
    }

    modifier onlyEOA() {
        // Try to make flash-loan exploit harder to do by only allowing externally-owned addresses.
        // CHANGE_OMGX
        //require(msg.sender == tx.origin, "Maker: Must use EOA");
        _;
    }

    function convert(IKashiWithdrawFee kashiPair) external onlyEOA {
        _convert(kashiPair);
    }

    function convertMultiple(IKashiWithdrawFee[] calldata kashiPair) external onlyEOA {
        for (uint256 i = 0; i < kashiPair.length; i++) {
            _convert(kashiPair[i]);
        }
    }

    function _convert(IKashiWithdrawFee kashiPair) private {
        // update Kashi fees for this Maker contract (`feeTo`)
        kashiPair.withdrawFees();

        // convert updated Kashi balance to Bento shares
        uint256 bentoShares = kashiPair.removeAsset(address(this), kashiPair.balanceOf(address(this)));

        // convert Bento shares to underlying Kashi asset (`token0`) balance (`amount0`) for Maker
        address token0 = kashiPair.asset();
        (uint256 amount0, ) = bentoBox.withdraw(IERC20(token0), address(this), address(this), 0, bentoShares);

        emit LogConvert(
            msg.sender,
            token0,
            amount0,
            bentoShares,
            _convertStep(token0, amount0)
        );
    }

    function _convertStep(address token0, uint256 amount0) private returns (uint256 sushiOut) {
        if (token0 == sushi) {
            IERC20(token0).safeTransfer(bar, amount0);
            sushiOut = amount0;
        } else if (token0 == weth) {
            sushiOut = _swap(token0, sushi, amount0, bar);
        } else {
            address bridge = _bridges[token0];
            if (bridge == address(0)) {
                bridge = weth;
            }
            uint256 amountOut = _swap(token0, bridge, amount0, address(this));
            sushiOut = _convertStep(bridge, amountOut);
        }
    }

    function _swap(
        address fromToken,
        address toToken,
        uint256 amountIn,
        address to
    ) private returns (uint256 amountOut) {
        (address token0, address token1) = fromToken < toToken ? (fromToken, toToken) : (toToken, fromToken);
        IUniswapV2Pair pair =
            IUniswapV2Pair(
                uint256(
                    keccak256(abi.encodePacked(hex"ff", factory, keccak256(abi.encodePacked(token0, token1)), pairCodeHash))
                )
            );
        
        (uint256 reserve0, uint256 reserve1, ) = pair.getReserves();
        uint256 amountInWithFee = amountIn.mul(997);
        
        if (toToken > fromToken) {
            amountOut =
                amountInWithFee.mul(reserve1) /
                reserve0.mul(1000).add(amountInWithFee);
            IERC20(fromToken).safeTransfer(address(pair), amountIn);
            pair.swap(0, amountOut, to, "");
        } else {
            amountOut =
                amountInWithFee.mul(reserve0) /
                reserve1.mul(1000).add(amountInWithFee);
            IERC20(fromToken).safeTransfer(address(pair), amountIn);
            pair.swap(amountOut, 0, to, "");
        }
    }
}
