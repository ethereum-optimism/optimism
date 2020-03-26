/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { L2ToL1MessageReceiverContractDefinition } from '@eth-optimism/ovm'

import { JsonRpcProvider, Provider } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  startLocalL1Node,
  Environment,
} from '../app'
import { L1NodeContext, L2ToL1MessageSubmitter } from '../types'
import { Contract, ethers, Wallet } from 'ethers'
import { DefaultL2ToL1MessageSubmitter } from '../app/message-submitter'

const log: Logger = getLogger('rollup-fullnode')

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 * @returns The array of fullnode instance, L2ToL1MessageSubmitter
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<[ExpressHttpServer, L2ToL1MessageSubmitter]> => {
  let provider: JsonRpcProvider
  // TODO Get these from config
  const port: number = Environment.l2RpcServerPort()

  const messageSubmitter: L2ToL1MessageSubmitter = await runMessageSubmitter()

  log.info(`Starting L2 fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode`)

  if (Environment.l2NodeWeb3Url()) {
    log.info(`Connecting to L2 web3 URL: ${Environment.l2NodeWeb3Url()}`)
    provider = new JsonRpcProvider(Environment.l2NodeWeb3Url())
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(messageSubmitter, provider)
    : await DefaultWeb3Handler.create(messageSubmitter, provider)
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    Environment.l2RpcServerHost(),
    port
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return [fullnodeRpcServer, messageSubmitter]
}

const runMessageSubmitter = async (): Promise<L2ToL1MessageSubmitter> => {
  log.info(`Connecting to L1 fullnode.`)

  let l1NodeContext: L1NodeContext
  if (Environment.l1NodeWeb3Url()) {
    if (!Environment.sequencerMnemonic()) {
      const msg: string = `No L1 Sequencer Mnemonic Provided! Set the L1_SEQUENCER_MNEMONIC env var!.`
      log.error(msg)
      throw Error(msg)
    }
    if (!Environment.l2ToL1MessageReceiverAddress()) {
      const msg: string = `No L2 to L1 Sequencer Mnemonic Provided! Set the L2_TO_L1_MESSAGE_RECEIVER_ADDRESS env var!.`
      log.error(msg)
      throw Error(msg)
    }

    log.info(`Connecting to L1 web3 URL: ${Environment.l1NodeWeb3Url()}`)
    const provider: Provider = new JsonRpcProvider(Environment.l1NodeWeb3Url())

    l1NodeContext = {
      provider,
      sequencerWallet: Wallet.fromMnemonic(
        Environment.sequencerMnemonic()
      ).connect(provider),
      l2ToL1MessageReceiver: new Contract(
        Environment.l2ToL1MessageReceiverAddress(),
        L2ToL1MessageReceiverContractDefinition.interface,
        provider
      ),
    }
  } else {
    log.info(`Deploying local L1 node on port ${Environment.localL1NodePort()}`)
    const sequencerMnemonic = Environment.sequencerMnemonic(
      Wallet.createRandom().mnemonic
    )
    l1NodeContext = await startLocalL1Node(
      sequencerMnemonic,
      Environment.localL1NodePort()
    )
  }

  return DefaultL2ToL1MessageSubmitter.create(
    l1NodeContext.sequencerWallet,
    l1NodeContext.l2ToL1MessageReceiver
  )
}
