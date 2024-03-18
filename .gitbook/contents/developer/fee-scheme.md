---
description: Fee scheme in Boba Network
---

# Fees

Fees on Boba are, for the most part, significantly lower than L1s. The cost of every transaction is the sum of two values:

1. Your L2 (execution) fee, and
2. Your L1 (security) fee.

At a high level, the L2 fee is the cost to execute your transaction in L2 and the L1 fee is the estimated cost to submit your transaction to L1 (in a rollup batch).

1. **L2 execution fee** is charged as `tx.gasPrice * l2GasUsed` (up to `tx.gasLimit`). The L2 gas price will vary depending on network congestion.
2. **L1 security fee** is charged as `l1GasPrice * l1GasUsed` . This is the cost of storing the transaction's data on L1.
   * `l1GasPrice` is the same as the normal gas price in L1 Ethereum
   * `l1GasUsed` is calculated as `1.2*(overhead + calldataGas)`. Thus, more calldata your transaction includes, the more expensive your L1 fee will be. For example, an ETH transfer has no calldata, so it will have the cheapest L1 fee, whereas large contract deployments can have over 25kb of calldata and will result in a high L1 fee. We currently just the overhead value to the L1 fee to ensure the fee paid covers the actual L1 costs.
3.  **Total cost**

    We accept BOBA and ETH (where applicable) as the fee token. The total cost for **ETH** is calculated as `tx.gasPrice * (l2GasUsed + l1GasPrice * l1GasUsed / tx.gasPrice)`. The total cost for **BOBA** is calculated as `tx.gasPrice * (l2GasUsed + l1GasPrice * l1GasUsed / tx.gasPrice) * priceRatio` where the price ratio is `ETH price / BOBA price * discount percentage`. The L2 gas Used is higher than l1, because we add the **L1 security fee**.

    * The gas uage of transferring ETH is 26730 on Boba Network. It includes 21000 (l2GasUsed) and 5370 (l1SecurityFee).

To obtain ETH and BOBA on Boba Network you can deposit ETH via[ https://gateway.boba.network](https://gateway.boba.network) on both Goerli or Mainnet. Soon you will be able to also deposit ETH for slightly cheaper via Teleportation.



<figure><img src="../../.gitbook/assets/for backend developers.png" alt=""><figcaption></figcaption></figure>

* You must send your transaction with a tx.gasPrice that is greater than or equal to the sequencer's l2 gas price. You can read this value from the Sequencer by querying the `OVM_GasPriceOracle` contract (`OVM_GasPriceOracle.gasPrice`) or by simply making an RPC query to `eth_gasPrice`. If you don't specify your `gasPrice` as an override when sending a transaction, `ethers` by default queries `eth_gasPrice` which will return the lowest acceptable L2 gas price.
* You can set your `tx.gasLimit` however you might normally set it (e.g. via `eth_estimateGas`). The gas usage for transactions on Boba Network will be larger than the gas usage on Ethereum, because the `l1SecurityFee` is included in the gas usage.
* We recommend building error handling around the `Fee too Low` error detailed below, to allow users to re-calculate their `tx.gasLimit` and resend their transaction if L1 gas price spikes.



<figure><img src="../../.gitbook/assets/for frontend and wallet developers.png" alt=""><figcaption></figcaption></figure>

* We recommend displaying an estimated fee to users using `eth_estimateGas`

```solidity
import { ethers } from 'ethers'
const WETH = new Contract(...) //Contract with no signer
const fee = WETH.estimateGas.transfer(to, amount)
```

* You should _not_ allow users to change their `tx.gasLimit` to any number that is smaller than the value from `eth_estimateGas`.
  * If they lower it, their transaction will revert and tx fee will be charged
* You should _not_ allow users to change their `tx.gasPrice`
  * If they lower it, their transaction will revert
  * If they increase it, they will still have their tx immediately included, but will have overpaid.
*   Users are welcome to change their `tx.gasLimit` as it functions exactly like on L1. You can show the math :

    ```
    ETH Fee: .00098 ETH ($3.94)
    Boba Fee: 1.96 BOBA ($3.94)
    ```

    We recommend displaying the right fee token and transaction fee to users by calling `Boba_GasPriceOracle` to calculate total fee and get the fee token

    ```solidity
    import { ethers } from 'ethers'
    const BobaGasPriceOracleInterface = new utils.Interface([
      'function useBobaAsFeeToken()',
      'function useETHAsFeeToken()',
      'function bobaFeeTokenUsers(address) view returns (bool)',
      'function priceRatio() view returns (uint256)'
    ])
    const Proxy__Boba_GasPriceOracle = new Contract(
    	Proxy__Boba_GasPriceOracleAddress,
    	BobaGasPriceOracleInterface
    	l2Provider
    ) //Contract with no signer
    const isBobaAsFeeToken = await Boba_GasPriceOracle.bobaFeeTokenUsers(walletAddress)
    const priceRatio = await Boba_GasPriceOracle.priceRatio()
    ```

    If `isBobaAsFeeToken` is `true`, then the user picks BOBA as the fee token and the total Boba transaction fee is `tx.gasLimit * tx.GasPrice * priceRatio`.
* Might need to regularly refresh the L2 Fee estimate to ensure it is accurate at the time the user sends it (e.g. they get the fee quote and leave for 12 hours then come back)
* Ideas: If the L2 fee quoted is > X minutes old, could display a warning next to it



<figure><img src="../../.gitbook/assets/common RPC errors.png" alt=""><figcaption></figcaption></figure>

There are three common errors that would cause your transaction to be rejected

1.  **Insufficient funds**

    > ETH as the fee token

    * If you are trying to send a transaction and you do not have enough ETH to pay for that L2 fee charged, your transaction will be rejected.
    * Error code: `-32000`
    * Error message: `invalid transaction: insufficient funds for l1Fee + l2Fee + value`

    > BOBA as the fee token

    * If you are trying to send a transaction and you do not have enough BOBA to pay for that L2 fee charged, your transaction will be rejected.
    * Error code: `-32000`
    * Error message: `invalid transaction: insufficient Boba funds for l1Fee + l2Fee + value`
2. **Gas Price to low**
   * Error code: `-32000`
   * Error message: `gas price too low: 1000 wei, use at least tx.gasPrice = X wei` where `x` is l2GasPrice.
     * Note: values in this error message vary based on the tx sent and current L2 gas prices
   * It is recommended to build in error handling for this. If a user's transaction is rejected at this level, just set a new `tx.gasPrice` via RPC query at `eth_gasPrice` or by calling `OVM_GasPriceOracle.gasPrice`
3. **Fee too large**
   * Error code: `-32000`
   * Error message: `gas price too high: 1000000000000000 wei, use at most tx.gasPrice = Y wei` where `x` is 3\*l2GasPrice.
   * When the `tx.gasPrice` provided is ≥3x the expected `tx.gasPrice`, you will get this error^, note this is a runtime config option and is subject to change
