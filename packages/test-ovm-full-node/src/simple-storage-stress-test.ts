/* External Imports */
import {add0x, keccak256} from '@eth-optimism/core-utils'
import {CHAIN_ID, GAS_LIMIT} from '@eth-optimism/ovm'

import {Contract, Wallet} from 'ethers'
import {JsonRpcProvider} from 'ethers/providers'
import {deployContract} from 'ethereum-waffle'

/* Internal Imports */
import {getUnsignedTransactionCalldata} from './helpers'
import {FullNodeStressTest} from './stress-test'

/* Contract Imports */
import * as SimpleStorage from '../build/SimpleStorage.json'


class SimpleStorageStressTest extends FullNodeStressTest {
  private contract: Contract
  constructor(numberOfRequests: number, nodeUrl: string) {
    super(numberOfRequests, nodeUrl);
  }

  /**
   * @inheritDoc
   */
  protected async deployContract(): Promise<void> {
    const provider = new JsonRpcProvider(this.nodeUrl)
    const wallet: Wallet = Wallet.createRandom().connect(provider)

    this.contract = await deployContract(wallet, SimpleStorage, [])
  }

  /**
   * @inheritDoc
   */
  protected getSignedTransaction(): Promise<string> {

    const key = keccak256(Math.floor(Math.random()* 100_000_000_000).toString(16))
    const value = keccak256(Math.floor(Math.random()* 100_000_000_000).toString(16))

    const calldata = getUnsignedTransactionCalldata(
      this.contract,
      'setStorage',
      [add0x(key), add0x(value)]
    )

    const wallet: Wallet = Wallet.createRandom()

    return wallet.sign({
      nonce: 0,
      gasLimit: GAS_LIMIT,
      to: this.contract.address,
      value: 0,
      data: calldata,
      chainId: CHAIN_ID
    })
  }
}

new SimpleStorageStressTest(100, 'http://3.14.246.203:8545').runBatches(100)
