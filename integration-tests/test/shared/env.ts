/* Imports: External */
import { Contract, utils, Wallet } from 'ethers'
import { TransactionResponse } from '@ethersproject/providers'
import { getContractFactory, predeploys } from '@eth-optimism/contracts'
import { Watcher } from '@eth-optimism/core-utils'
import { getMessagesAndProofsForL2Transaction } from '@eth-optimism/message-relayer'

/* Imports: Internal */
import {
  getAddressManager,
  l1Provider,
  l2Provider,
  l1Wallet,
  l2Wallet,
  fundUser,
  getOvmEth,
  getL1Bridge,
  getL2Bridge,
  IS_LIVE_NETWORK,
  sleep,
} from './utils'
import {
  initWatcher,
  CrossDomainMessagePair,
  Direction,
  waitForXDomainTransaction,
} from './watcher-utils'

/// Helper class for instantiating a test environment with a funded account
export class OptimismEnv {
  // L1 Contracts
  addressManager: Contract
  l1Bridge: Contract
  l1Messenger: Contract
  ctc: Contract
  scc: Contract

  // L2 Contracts
  ovmEth: Contract
  l2Bridge: Contract
  l2Messenger: Contract
  gasPriceOracle: Contract

  // The L1 <> L2 State watcher
  watcher: Watcher

  // The wallets
  l1Wallet: Wallet
  l2Wallet: Wallet

  constructor(args: any) {
    this.addressManager = args.addressManager
    this.l1Bridge = args.l1Bridge
    this.l1Messenger = args.l1Messenger
    this.ovmEth = args.ovmEth
    this.l2Bridge = args.l2Bridge
    this.l2Messenger = args.l2Messenger
    this.gasPriceOracle = args.gasPriceOracle
    this.watcher = args.watcher
    this.l1Wallet = args.l1Wallet
    this.l2Wallet = args.l2Wallet
    this.ctc = args.ctc
    this.scc = args.scc
  }

  static async new(): Promise<OptimismEnv> {
    const addressManager = getAddressManager(l1Wallet)
    const watcher = await initWatcher(l1Provider, l2Provider, addressManager)
    const l1Bridge = await getL1Bridge(l1Wallet, addressManager)

    // fund the user if needed
    const balance = await l2Wallet.getBalance()
    if (balance.isZero()) {
      await fundUser(watcher, l1Bridge, utils.parseEther('20'))
    }
    const l1Messenger = getContractFactory('iOVM_L1CrossDomainMessenger')
      .connect(l1Wallet)
      .attach(watcher.l1.messengerAddress)
    const ovmEth = getOvmEth(l2Wallet)
    const l2Bridge = await getL2Bridge(l2Wallet)
    const l2Messenger = getContractFactory('iOVM_L2CrossDomainMessenger')
      .connect(l2Wallet)
      .attach(watcher.l2.messengerAddress)

    const ctcAddress = await addressManager.getAddress(
      'OVM_CanonicalTransactionChain'
    )
    const ctc = getContractFactory('OVM_CanonicalTransactionChain')
      .connect(l1Wallet)
      .attach(ctcAddress)

    const gasPriceOracle = getContractFactory('OVM_GasPriceOracle')
      .connect(l2Wallet)
      .attach(predeploys.OVM_GasPriceOracle)

    const sccAddress = await addressManager.getAddress(
      'OVM_StateCommitmentChain'
    )
    const scc = getContractFactory('OVM_StateCommitmentChain')
      .connect(l1Wallet)
      .attach(sccAddress)

    return new OptimismEnv({
      addressManager,
      l1Bridge,
      ctc,
      scc,
      l1Messenger,
      ovmEth,
      gasPriceOracle,
      l2Bridge,
      l2Messenger,
      watcher,
      l1Wallet,
      l2Wallet,
    })
  }

  async waitForXDomainTransaction(
    tx: Promise<TransactionResponse> | TransactionResponse,
    direction: Direction
  ): Promise<CrossDomainMessagePair> {
    return waitForXDomainTransaction(this.watcher, tx, direction)
  }

  /**
   * Relays all L2 => L1 messages found in a given L2 transaction.
   *
   * @param tx Transaction to find messages in.
   */
  async relayXDomainMessages(
    tx: Promise<TransactionResponse> | TransactionResponse
  ): Promise<void> {
    tx = await tx

    let messagePairs = []
    while (true) {
      try {
        messagePairs = await getMessagesAndProofsForL2Transaction(
          l1Provider,
          l2Provider,
          this.scc.address,
          predeploys.OVM_L2CrossDomainMessenger,
          tx.hash
        )
        break
      } catch (err) {
        if (err.message.includes('unable to find state root batch for tx')) {
          await sleep(5000)
        } else {
          throw err
        }
      }
    }

    for (const { message, proof } of messagePairs) {
      while (true) {
        try {
          const result = await this.l1Messenger
            .connect(this.l1Wallet)
            .relayMessage(
              message.target,
              message.sender,
              message.message,
              message.messageNonce,
              proof
            )
          await result.wait()
          break
        } catch (err) {
          if (err.message.includes('execution failed due to an exception')) {
            await sleep(5000)
          } else if (
            err.message.includes('message has already been received')
          ) {
            break
          } else {
            throw err
          }
        }
      }
    }
  }
}

/**
 * Sets the timeout of a test based on the challenge period of the current network. If the
 * challenge period is greater than 60s (e.g., on Mainnet) then we skip this test entirely.
 *
 * @param testctx Function context of the test to modify (i.e. `this` when inside a test).
 * @param env Optimism environment used to resolve the StateCommitmentChain.
 */
export const useDynamicTimeoutForWithdrawals = async (
  testctx: any,
  env: OptimismEnv
) => {
  if (!IS_LIVE_NETWORK) {
    return
  }

  const challengePeriod = await env.scc.FRAUD_PROOF_WINDOW()
  if (challengePeriod.gt(60)) {
    console.log(
      `WARNING: challenge period is greater than 60s (${challengePeriod.toString()}s), skipping test`
    )
    testctx.skip()
  }

  // 60s for state root batch to be published + (challenge period x 4)
  const timeoutMs = 60000 + challengePeriod.toNumber() * 1000 * 4
  console.log(
    `NOTICE: inside a withdrawal test on a prod network, dynamically setting timeout to ${timeoutMs}ms`
  )
  testctx.timeout(timeoutMs)
}
