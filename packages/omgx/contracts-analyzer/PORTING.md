# Porting to OMGX and Optimism - A case study.

- [Porting to OMGX and Optimism - A case study.](#porting-to-omgx-and-optimism---a-case-study)
  * [SUSHI](#sushi)
  * [0. Basics](#0-basics)
  * [1. No native ETH](#1-no-native-eth)
  * [2. Timing, `now`, and `block.timestamp`](#2-timing---now---and--blocktimestamp)
  * [3. Replace `chainid()` with `uint256 chainId = ___`](#3-replace--chainid----with--uint256-chainid)
  * [4. Update Depreciated Syntax](#5-update-depreciated-syntax)
  * [5. No tx.origin](#6-no-txorigin)
  * [6. TEST RESULTS: evm_increaseTime and evm_mine](#7-tests-results--all-good-except-evm-increasetime-and-evm-mine---workaround-pending)

## SUSHI

SUSHI is a DeFi exchange that supports token swapping and many other actions. We started by copying SUSHI's smart contracts into the `/contracts` folder. Then, we ran:

```bash
yarn install
yarn build
yarn analyze
yarn deploy
```

We then addressed the warnings and errors one by one. Let's now take that all, step by step.

## 0. Basics

**Your contracts.** As noted, run `yarn install` and drop your contracts into a folder called `contracts`. That's what solc and hardhat will look at, compile, and deploy. Then, run:

```bash
yarn build
```

The first time you do this, you will see dozens of errors.

**Missing libraries.** The provided `package.json` is generic and you will see HH411 errors such as 

```
Error HH411: The library foo, imported from contracts/something.sol, is not installed. Try installing it using npm.
```

This means that you have to install the foo library:

```bash
yarn add foo
```

**Solidity Versions.** Once all the missing libraries have been installed, you will see `HH606` errors:

```
Error HH606: The project cannot be compiled, see reasons below.

The Solidity version pragma statement in these files don't match any of the configured compilers in your config. Change the pragma or configure additional compiler versions in your hardhat config.

  * contracts/something.sol (^0.6.0)
```

The optimism solc compiler supports Solidity versions 0.5.16, 0.6.12, and 0.7.6. This value is set in `hardhat.config.js`. The first time you run `yarn build` you will typically see many errors relating to your pragmas. If most of your pragmas are around 0.6 you would chose 0.6.12, and so forth. In general, small modifications (such as replacing `^0.5.17` with `^0.5.16` or specifying a broader range such as `pragma solidity >= 0.5.16 < 0.6.5;`) will not affect your code and your unit and integration tests will pick up any exceptions. 

At this point, solc and hardhat have all the information they need to get started. Now, you will see actual code issues, such as: 

```
OVM Compiler Error (insert "// @unsupported: ovm" if you don't want this file to be compiled for the OVM):
 contracts/foo.sol:72:31: ParserError: OVM: ORIGIN is not implemented in the OVM.
        require(msg.sender == tx.origin, "not eoa");

OVM Compiler Error (insert "// @unsupported: ovm" if you don't want this file to be compiled for the OVM):
 contracts/WETH.sol:51:16: ParserError: OVM: SELFBALANCE is not implemented in the OVM. (We have no native ETH -- use deposited WETH instead!)
        return address(this).balance;
               ^-------------------^

Error HH600: Compilation failed
```

Let's now tackle those one by one.

## 1. No native ETH

In many smart contracts, ETH is handled slightly differently than ERC20 tokens, but on L2, there is no native ETH. Instead, L2s use an ERC20 representation of ETH such as wETH or oETH. This means that all ETH-specific functions can be deleted, since there are no longer needed. For example:

```diff

contracts/uniswapv2/interfaces/IUniswapV2Router01.sol 
@@ -16,14 +17,15 @@ interface IUniswapV2Router01 {
	...
-    function addLiquidityETH(
-        address token,
-        uint amountTokenDesired,
-        uint amountTokenMin,
-        uint amountETHMin,
-        address to,
-        uint deadline
-    ) external payable returns (uint amountToken, uint amountETH, uint liquidity);
+    // CHANGE_OMGX
+    // function addLiquidityETH(
+    //     address token,
+    //     uint amountTokenDesired,
+    //     uint amountTokenMin,
+    //     uint amountETHMin,
+    //     address to,
+    //     uint deadline
+    // ) external payable returns (uint amountToken, uint amountETH, uint liquidity);

```

From a UI/Frontend perspective, 'native' ETH functions are no longer needed and integration test code will also need any ETH-specific tests to be commented out. In the case of the SUSHI port, among other changes, `contracts/mocks/WETH9Mock.sol` can be deleted entirely and many functions in `contracts/uniswapv2/UniswapV2Router02.sol` can also be deleted, such as `removeLiquidityETH` and `swapExactETHForTokens` etc. Removing functions in the contracts also affects the interfaces, of course, e.g. `contracts/uniswapv2/interfaces/IUniswapV2Router01.sol`.

## 2. Timing, `now`, and `block.timestamp`

The L2 does not have traditional blocks. Control over time, and manipulation of apparent time, is critical for L2, since during a fraud proof, the L1 contacts will need to replay the L2 contracts at specific times _in the past_ to check their correctness. `block.timestamp` returns the last L1 block in which a rollup batch was posted. This means that the `block.timestamp` returned on L2 can lag as many as 10 minutes behind L1. Depending on how `block.timestamp` is being used, this 1-10 min lag could have **serious unexpected implications**. See [OVM-vs-EVM-Block-Timestamps](https://hackmd.io/@scopelift/Hy853dTsP#OVM-vs-EVM-Block-Timestamps) for a more extensive discussion. Briefly, consider:

1. The OVM timestamp lags behind the EVM, so it’s possible that e.g. OVM trades execute up 10 minutes after your specified deadline.  
2. `permit` method signatures contain a deadline, and the approval must be sent before that deadline. In certain cases, the approval could take place after the deadline.  
3. Bid and auction duration. If you are trying to run an auction with minute scale bid duration, then a 1-10 minute lag relative to L1 could throw that off completely.  

## 3. Replace `chainid()` with `uint256 chainId = ___`

```diff

contracts/SushiToken.sol
@@ -239,8 +241,8 @@ contract SushiToken is ERC20("SushiToken", "SUSHI"), Ownable {
	...
    function getChainId() internal pure returns (uint) {
-       uint256 chainId;
+       uint256 chainId = 28; //or whatever the L2 ChainID is...
+       //assembly { chainId := chainid() }

```

## 4. Update Depreciated Syntax

Not strictly L2 related, but updated it to help with future maintainability.

```diff

contracts/governance/Timelock.sol 
-   (bool success, bytes memory returnData) = target.call.value(value)(callData);
+   (bool success, bytes memory returnData) = target.call{value:value}(callData);

// The following syntax is deprecated: 
// f.gas(...)(), f.value(...)() and (new C).value(...)().
// Replace with:
// f{gas: ..., value: ...}() and (new C){value: ...}(). 

```

## 5. No tx.origin

L2 does not support `tx.origin`. This is typically a non-issue, since `tx.origin` is deprecated anyway and will be removed from L1 at some point. See [Vitalik's answer](https://ethereum.stackexchange.com/questions/196/how-do-i-make-my-dapp-serenity-proof). Secondly, only allowing txs from an EOA is considered an anti-pattern. It breaks composability, it prevents multisig wallets from using your product, and in general it's probably a hack to cover up some underlying security issues in the contracts. There is no easy/obvious one-line replacement for `tx.origin` - any attempt to try to detect the codesize or something of the calling contract would be spoofable. For Compound's use of `msg.sender == tx.origin`, as for Sushi, the best approach is to remove that restriction and make sure the contracts can safely handle calls from other contracts (which involves writing new code). For now, we just commented out the `require`.  

```diff

contracts/SushiMaker.sol
    // Try to make flash-loan exploit harder to do by only allowing externally owned addresses.
-   require(msg.sender == tx.origin, "SushiMaker: must use EOA");
+   //require(msg.sender == tx.origin, "SushiMaker: must use EOA");

```

## 6. TESTS RESULTS: All good EXCEPT evm_increaseTime and evm_mine

All tests clear EXCEPT things related to `evm_increaseTime` and `evm_mine`. Note that this does not affect the contracts _per se_ but affects testing. 