---
description: >-
  Learn how to use basic features of Boba (e.g. bridges, basic L2 ops) through
  examples
---

# Basic Operations

Below, we provide code snippets for several typical operations on the L2, such as checking the gas price and bridging funds. Overall, note that from the perspective of solidity code and rpc calls, Boba is identical to mainchain in most aspects, so your experience (and code) from mainchain should carry over directly. The main practical differences center on Gas and on cross-chain bridging operations.

To see examples of how to perform dozens of basic operations on Boba, you can also look at the react code for the [Boba Gateway](https://github.com/bobanetwork/gateway/blob/main/src/services/networkService.ts).

## Check the Current Gas Price

The Gas Price on L2 changes as part of the chain derivation protocol. The exact derivation is described in the [protocol documentation](https://github.com/ethereum-optimism/specs/blob/298e745a7aa2e0aa78b6f18eb88bf1c484c00c4a/specs/protocol/predeploys.md#gaspriceoracle). Like on Mainnet, the current gas price can be obtained via `.getGasPrice()`:

```javascript
  this.L2Provider = new ethers.providers.StaticJsonRpcProvider('mainnet.boba.network')

  const gasPrice = await this.L2Provider.getGasPrice()

  console.log("Current gas price:", gasPrice )
  //prints: Current gas price: BigNumber {_hex: '0x02540be400', _isBigNumber: true}

  console.log("Current gas price", gasPrice.toString())
  //prints: Current gas price: 10000000000
```

## Estimate the Cost of a Contract Call

Like on Mainnet, the cost of a L2 transaction is the product of the current gas price and the 'complexity' of the contract call, with some calls being much more expensive than others. The contract call complexity is quantified via the `gas`.

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

NOTE: Unlike on L1, there is no transaction pool.  Although transactions can be ordered based on gas priority, because only the sequencer sees the pending transactions there are generally no opportunities for MEV with respect to transaction ordering.  Thus there is generally no benefit to paying more than the estimated gas fee.

## L2-L2 Transfer

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

## L1-L2 Classic Bridge Operation

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

## Accessing Latest L1 Block Number

Information about the L1 is available via the [L1 Block Attributes Predeploy
Contract](https://github.com/ethereum-optimism/specs/blob/298e745a7aa2e0aa78b6f18eb88bf1c484c00c4a/specs/protocol/deposits.md#l1-attributes-predeployed-contract).  This contract is always updated by the first implicit deposit transaction in every block.

```javascript
import { L1Block } from "@eth-optimism/contracts-bedrock/L2/L1Block.sol";
import { Predeploys } from "@eth-optimism/contracts-bedrock/libraries/Predeploys.sol";

contract MyContract {
   function myFunction() public {
      // ... your code here ...

      uint256 l1BlockNumber = L1Block(
         Predeploys.L1_BLOCK_ATTRIBUTES
      ).number();

      // ... your code here ...
   }
}
```
