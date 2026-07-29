import {
    createWalletClient,
    http,
    publicActions,
    getContract,
} from 'viem'
import { privateKeyToAccount } from 'viem/accounts'
import { interopAlpha0, interopAlpha1, supersimL2A, supersimL2B } from '@eth-optimism/viem/chains'
import { walletActionsL2, publicActionsL2 } from '@eth-optimism/viem'

const tokenAddress = process.env.TOKEN_ADDRESS
const useSupersim = process.env.CHAIN_B_ID == "902"

const balanceOf = {
    "constant": true,
    "inputs": [{
        "name": "_owner",
        "type": "address"
    }],
    "name": "balanceOf",
    "outputs": [{
        "name": "balance",
        "type": "uint256"
    }],
    "payable": false,
    "stateMutability": "view",
    "type": "function"
    }

const account = privateKeyToAccount(process.env.PRIVATE_KEY as `0x${string}`)

const wallet0 = createWalletClient({
    chain: useSupersim ? supersimL2A : interopAlpha0,
    transport: http(),
    account
}).extend(publicActions)
//    .extend(publicActionsL2())
    .extend(walletActionsL2())
 
const wallet1 = createWalletClient({
    chain: useSupersim ? supersimL2B : interopAlpha1,
    transport: http(),
    account
}).extend(publicActions)
    .extend(publicActionsL2())
    .extend(walletActionsL2())

const token0 = getContract({
    address: tokenAddress,
    abi: [balanceOf],
    client: wallet0
})

const token1 = getContract({
    address: tokenAddress,
    abi: [balanceOf],
    client: wallet1
})


const reportBalances = async () => {
    const balance0 = await token0.read.balanceOf([account.address])
    const balance1 = await token1.read.balanceOf([account.address])

    console.log(`
Address: ${account.address}
    chain0: ${balance0.toString().padStart(20)}
    chain1: ${balance1.toString().padStart(20)}

`)
}

console.log("Initial balances")
await reportBalances()

const sendTxnHash = await wallet0.interop.sendSuperchainERC20({
    tokenAddress,
    to: account.address,
    amount: BigInt(1000000000),
    chainId: wallet1.chain.id
})

console.log(`Send transaction: ${sendTxnHash}`)
await wallet0.waitForTransactionReceipt({
    hash: sendTxnHash
})

console.log("Immediately after the transaction is processed")
await reportBalances()

await new Promise(resolve => setTimeout(resolve, 5000));

console.log("After waiting (hopefully, until the message is relayed)")
await reportBalances()
