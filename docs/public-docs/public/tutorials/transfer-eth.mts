import {
    createWalletClient,
    http,
    publicActions,
    getContract,
    Address,
    formatEther,
    parseEther,
} from 'viem'

import { privateKeyToAccount } from 'viem/accounts'

import { 
    supersimL2A, 
    supersimL2B, 
    interopAlpha0, 
    interopAlpha1 
} from '@eth-optimism/viem/chains'
 
import {
    walletActionsL2,
    publicActionsL2,
    contracts as optimismContracts
} from '@eth-optimism/viem'
 
import superchainEthBridgeAbi from './SuperchainETHBridge.abi.json'

const supersimAddress="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
const account = privateKeyToAccount(process.env.PRIVATE_KEY as `0x${string}`)
const sourceChain = account.address == supersimAddress ? supersimL2A : interopAlpha0
const destinationChain = account.address == supersimAddress ? supersimL2B : interopAlpha1

const sourceWallet = createWalletClient({
    chain: sourceChain,
    transport: http(),
    account
}).extend(publicActions)
    .extend(publicActionsL2())
    .extend(walletActionsL2())

const destinationWallet = createWalletClient({
    chain: destinationChain,
    transport: http(),
    account
}).extend(publicActions)
    .extend(publicActionsL2())
    .extend(walletActionsL2())

const ethBridgeOnSource = await getContract({
    address: optimismContracts.superchainETHBridge.address,
    abi: superchainEthBridgeAbi,
    client: sourceWallet
})

const reportBalance = async (address: string): Promise<void> => {
    const sourceBalance = await sourceWallet.getBalance({
        address: address
    });
    const destinationBalance = await destinationWallet.getBalance({
        address: address
    });

    console.log(`
        Address: ${address}
        Balance on source chain: ${formatEther(sourceBalance)}
        Balance on destination chain: ${formatEther(destinationBalance)}
    `);
}

console.log("Before transfer")
await reportBalance(account.address)

const sourceHash = await ethBridgeOnSource.write.sendETH({
    value: parseEther('0.001'),
    args: [account.address, destinationChain.id]
})
const sourceReceipt = await sourceWallet.waitForTransactionReceipt({
    hash: sourceHash
})


console.log("After transfer on source chain")
await reportBalance(account.address)


const sentMessages = await sourceWallet.interop.getCrossDomainMessages({
    logs: sourceReceipt.logs,
})
const sentMessage = sentMessages[0]
const relayMessageParams = await sourceWallet.interop.buildExecutingMessage({
    log: sentMessage.log,
})

const relayMsgTxnHash = await destinationWallet.interop.relayCrossDomainMessage(relayMessageParams)
const receiptRelay = await destinationWallet.waitForTransactionReceipt(
    {hash: relayMsgTxnHash})

console.log("After relaying message to destination chain")
await reportBalance(account.address)
