---
description: >-
  Learn how to use basic features of Boba (e.g. bridges, basic L2 ops) through
  examples
---

# Basic Operations

Below, we provide code snippets for several typical operations on the L2, such as checking the gas price and bridging funds. Overall, note that from the perspective of solidity code and rpc calls, Boba is identical to mainchain in most aspects, so your experience (and code) from mainchain should carry over directly. The main practical differences center on Gas and on cross-chain bridging operations.

To see examples of how to perform dozens of basic operations on Boba, you can also look at the react code for the [Boba Gateway](../../packages/boba/gateway/src/services/networkService.js).



<figure><img src="../../.gitbook/assets/check the current gas price.png" alt=""><figcaption></figcaption></figure>

The Gas Price on L2 changes every **30 seconds**, with some smoothing to reduce sharp discontinuities in the price from one moment to the next. The maximum percentage change of the L2 gas price is 5% in the gas price oracle. Like on mainchain, the current gas price can be obtained via `.getGasPrice()`:

```javascript
  this.L2Provider = new ethers.providers.StaticJsonRpcProvider('mainnet.boba.network')

  const gasPrice = await this.L2Provider.getGasPrice()
   
  console.log("Current gas price:", gasPrice )
  //prints: Current gas price: BigNumber {_hex: '0x02540be400', _isBigNumber: true}

  console.log("Current gas price", gasPrice.toString())
  //prints: Current gas price: 10000000000
```

Typical values are 3 to 10 Gwei.



<figure><img src="../../.gitbook/assets/estimate the cost of a contract call.png" alt=""><figcaption></figcaption></figure>

Like on mainchain, the cost of a L2 transaction is the product of the current gas price and the 'complexity' of the contract call, with some calls being much more expensive than others. The contract call complexity is quantified via the `gas`. For example, the cost of an approval on L2 is about 0.0004 ETH, or about $1.70 (Oct. 2021):

```javascript
  const L2ERC20Contract = new ethers.Contract(
    currencyAddress,
    L2ERC20Json.abi,
    this.provider.getSigner()
  )

  //this is the key call - this results in a TX body that can be used
  //by estimateGas(TX) to estimate the gas
  const tx = await L2ERC20Contract.populateTransaction.approve(
    allAddresses.L2LPAddress,
    utils.parseEther('1.0')
  )

  const approvalGas_BN = await this.L2Provider.estimateGas(tx)

  approvalCost_BN = approvalGas_BN.mul(gasPrice)

  console.log("Current gas price", gasPrice.toString())
  console.log("Approval gas:", approvalGas_BN.toString())
  console.log("Approval cost in ETH:", utils.formatEther(approvalCost_BN))

  //Current gas price: 10000000000
  //Approval gas: 44138
  //Approval cost in ETH: 0.00044138
```

NOTE: The gas for a particular transaction depends both on the nature of the call (e.g. `approve`) and the call parameters, such as the amount (in this case, 1.0 ETH). A common source of reverted transactions is to mis-estimate the gas, such as by calling `.estimateGas()` with a TX generated for a different value.

```bash
  Typical L2 gas values:

  Approve 0 ETH:  24866
  Approve 1 ETH:  44138
  Fast Exit:     141698
```

NOTE: Unlike on L1, on _L2 there is no benefit to paying more_ - you just waste ETH. The sequencer operates in first in first serve, and transaction order is determined by the arrival time of your transaction, not by how much you are willing to pay.

NOTE: To protect users, _overpaying by more than a 10% percent will also revert your transactions_. The core gas price logic is as follows:

```javascript
  //Core l2 gas price code logic

	fee := new(big.Int).Set(opts.ExpectedGasPrice)
	
	// Allow for a downward buffer to protect against L1 gas price volatility
	if opts.ThresholdDown != nil {
		fee = mulByFloat(fee, opts.ThresholdDown)
	}
	
	// Protect the sequencer from being underpaid
	// if user fee < expected fee, return error
	if opts.UserGasPrice.Cmp(fee) == -1 {
		return ErrGasPriceTooLow
	}
	
	// Protect users from overpaying by too much
	if opts.ThresholdUp != nil {
		// overpaying = user fee - expected fee
		overpaying := new(big.Int).Sub(opts.UserGasPrice, opts.ExpectedGasPrice)
		threshold := mulByFloat(opts.ExpectedGasPrice, opts.ThresholdUp)
		// if overpaying > threshold, return error
		if overpaying.Cmp(threshold) == 1 {
			return ErrGasPriceTooHigh
		}
	}
```

**Gas Price tolerance band** : The `gasPrice` you use should be **within 10% of the value** reported by `.getGasPrice()`. Let’s say the gasPrice is 100 Gwei. Then, the l2geth will accept any `gasPrice` between 90 Gwei to 110 Gwei.



<figure><img src="../../.gitbook/assets/l2-l2 transfer.png" alt=""><figcaption></figcaption></figure>

```javascript
//Transfer funds from one account to another, on the L2
async transfer(address, value_Wei_String, currency) {

	let tx = null

	try {
	  
	  if(currency === allAddresses.L2_ETH_Address) {
	    
	    //we are transferring ETH - special call
	    let wei = BigNumber.from(value_Wei_String)

	    tx = await this.provider.send('eth_sendTransaction', 
	      [
	        { 
	          from: this.account,
	          to: address,
	          value: ethers.utils.hexlify(wei)
	        }
	      ]
	    )

	  } else {
	    // we are transferring an ERC20...
	    tx = await this.STANDARD_ERC20_Contract
	    	.attach(currency)
	    	.transfer(
		      address,
		      value_Wei_String
		    )
	    await tx.wait()
	  }
	  
	  return tx
	} catch (error) {
	  console.log("Transfer error:", error)
	  return error
	}
}
```



<figure><img src="../../.gitbook/assets/l1-l2 classic bridge operation.png" alt=""><figcaption></figcaption></figure>

```javascript
  //Move ERC20 Tokens from L1 to L2
  async depositErc20(value_Wei_String, currency, currencyL2) {

    const L1_TEST_Contract = this.L1_TEST_Contract.attach(currency)

    let allowance_BN = await L1_TEST_Contract.allowance(
      this.account,
      allAddresses.L1StandardBridgeAddress
    )

	const allowed = allowance_BN.gte(BigNumber.from(value_Wei_String))

	if(!allowed) {
		const approveStatus = await L1_TEST_Contract.approve(
		  allAddresses.L1StandardBridgeAddress,
		  value_Wei_String
		)
		await approveStatus.wait()
		console.log("ERC 20 L1 ops approved:",approveStatus)
	}

	const depositTxStatus = await this.L1StandardBridgeContract.depositERC20(
		currency,
		currencyL2,
		value_Wei_String,
		this.L2GasLimit,
		utils.formatBytes32String(new Date().getTime().toString())
	)

	//at this point the tx has been submitted, and we are waiting...
	await depositTxStatus.wait()

	const [l1ToL2msgHash] = await this.watcher.getMessageHashesFromL1Tx(
		depositTxStatus.hash
	)
	console.log(' got L1->L2 message hash', l1ToL2msgHash)

	const l2Receipt = await this.watcher.getL2TransactionReceipt(
		l1ToL2msgHash
	)
	console.log(' completed Deposit! L2 tx hash:', l2Receipt.transactionHash)

	return l2Receipt
    
  }
```



<figure><img src="../../.gitbook/assets/accessing latest L1 block number.png" alt=""><figcaption></figcaption></figure>

The hex value that corresponds to the `L1BLOCKNUMBER` opcode (`0x4B`) may be changed in the future. **We strongly discourage direct use of this opcode within your contracts.** Instead, if you want to access the latest L1 block number, please use the `OVM_L1BlockNumber` contract as described below.

The block number of the latest L1 block seen by the L2 system can be accessed via the `L1BLOCKNUMBER` opcode. Solidity doesn't make it easy to use non-standard opcodes, so there is a simple contract located at \[`0x4200000000000000000000000000000000000013` that will allow you to trigger this opcode. You can use this contract as follows:

```javascript
import { iOVM_L1BlockNumber } from "@eth-optimism/contracts/L2/predeploys/iOVM_L1BlockNumber.sol";
import { Lib_PredeployAddresses } from "@eth-optimism/contracts/libraries/constants/Lib_PredeployAddresses.sol";

contract MyContract {
   function myFunction() public {
      // ... your code here ...

      uint256 l1BlockNumber = iOVM_L1BlockNumber(
         Lib_PredeployAddresses.L1_BLOCK_NUMBER // located at 0x4200000000000000000000000000000000000013
      ).getL1BlockNumber();

      // ... your code here ...
   }
}
```



<figure><img src="../../.gitbook/assets/block numbers and timestamps.png" alt=""><figcaption></figcaption></figure>

### Block production is not constant

On Ethereum, the `NUMBER` opcode (`block.number` in Solidity) corresponds to the current Ethereum block number. Similarly, in Boba Network, `block.number` corresponds to the current L2 block number. However, **each transaction on L2 is placed in a separate block and blocks are NOT produced at a constant rate.**

This is important because it means that `block.number` is currently NOT a reliable source of timing information. If you want access to the current time, you should use `block.timestamp` (the `TIMESTAMP` opcode) instead.

### Timestamp lags by up to 15 minutes

Note that `block.timestamp` is pulled automatically from the latest L1 block seen by the L2. L2 currently waits for about 15 minutes (\~50 confirmations) before the L1 block is accepted. As a result, the timestamp may lag behind the current time by up to 15 minutes.
