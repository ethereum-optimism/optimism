import { BigNumber, providers, Wallet } from 'ethers'
import { Contract, ContractFactory, utils } from 'ethers'

import { Direction } from './libs/watcher-utils'
import { OptimismEnv } from './libs/env'

import L1MessageJson from './artifacts/contracts/test-helpers/Message/L1Message.sol/L1Message.json'
import L2MessageJson from './artifacts-ovm/contracts/test-helpers/Message/L2Message.sol/L2Message.json'

import logger from './logger'


const L1MessageAddress = process.env.L1_MESSAGE
const L2MessageAddress = process.env.L2_MESSAGE
const walletPKey = process.env.WALLET_PRIVATE_KEY
const walletAddess = process.env.WALLET_ADDRESS
const l1Web3Url = process.env.L1_NODE_WEB3_URL
const l1GasUsed = process.env.L1_GAS_USED

const l1Provider = new providers.JsonRpcProvider(l1Web3Url)
const l1Wallet = new Wallet(walletPKey).connect(l1Provider)

let L1Message: Contract
let L2Message: Contract




const transferL1toL2 = async () => {
    logger.debug(`Init OptimismEnv`)
    const env = await OptimismEnv.new()

    L1Message = new Contract(
        L1MessageAddress,
        L1MessageJson.abi,
        env.bobl1Wallet
    )

    logger.debug(`SendMessage L1 to L2`)
    const trans = await env.waitForXDomainTransaction(
        L1Message.sendMessageL1ToL2(),
        Direction.L1ToL2
    )

    logger.debug(
        `Transaction is sent successfully with hash:${trans.tx.hash}`,
        { trans }
    )
}


const transferL2toL1 = async () => {
    logger.debug(`Init OptimismEnv`)
    const env = await OptimismEnv.new()

    L2Message = new Contract(
        L2MessageAddress,
        L2MessageJson.abi,
        env.bobl2Wallet
    )


    logger.debug(`SendMessage L2 to L1`)
    const trans = await env.waitForXFastDomainTransaction(
        L2Message.sendMessageL2ToL1({ gasLimit: 3000000, gasPrice: 0 }),
        Direction.L2ToL1
    )

    logger.debug(
        `Transaction is sent successfully with hash:${trans.tx.hash}`,
        { trans }
    )
}

transferL2toL1().catch((err) => {
    logger.error(err.message)
})
