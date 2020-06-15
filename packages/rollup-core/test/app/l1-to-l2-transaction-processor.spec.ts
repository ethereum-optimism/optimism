/* External Imports */
import { newInMemoryDB } from '@eth-optimism/core-db'
import { keccak256, sleep } from '@eth-optimism/core-utils'
import * as BigNumber from 'bn.js'

/* Internal Imports */
import { L1ToL2Transaction, L1ToL2TransactionListener } from '../../src/types'
import {
  L1ToL2TransactionEventName,
  L1ToL2TransactionProcessor,
} from '../../src/app'
import { Wallet } from 'ethers'

class DummyListener implements L1ToL2TransactionListener {
  public readonly receivedTransactions: L1ToL2Transaction[] = []

  public async handleL1ToL2Transaction(
    transaction: L1ToL2Transaction
  ): Promise<void> {
    this.receivedTransactions.push(transaction)
  }
}

describe('L1 to L2 Transaction Processor', () => {
  let l1ToL2TransactionProcessor: L1ToL2TransactionProcessor
  let db
  let listener: DummyListener
  const eventID: string = 'test event id'

  const _nonce: BigNumber = new BigNumber(0)
  const _sender: string = Wallet.createRandom().address
  const _target: string = Wallet.createRandom().address
  const _callData: string = keccak256(Buffer.from('calldata').toString('hex'))

  const nonce2: BigNumber = new BigNumber(1)
  const sender2: string = Wallet.createRandom().address
  const target2: string = Wallet.createRandom().address
  const callData2: string = keccak256(Buffer.from('calldata 2').toString('hex'))

  beforeEach(async () => {
    db = newInMemoryDB()
    listener = new DummyListener()
    l1ToL2TransactionProcessor = await L1ToL2TransactionProcessor.create(
      db,
      eventID,
      [listener]
    )
  })

  it('should handle transaction properly', async () => {
    await l1ToL2TransactionProcessor.handle({
      eventID,
      name: L1ToL2TransactionEventName,
      signature: keccak256(Buffer.from('some random stuff').toString('hex')),
      values: {
        _nonce,
        _sender,
        _target,
        _callData,
      },
      blockNumber: 1,
      blockHash: keccak256(Buffer.from('block hash').toString('hex')),
      transactionHash: keccak256(Buffer.from('tx hash').toString('hex')),
    })

    await sleep(100)

    listener.receivedTransactions.length.should.equal(
      1,
      `Transaction not received!`
    )
    listener.receivedTransactions[0].nonce.should.equal(
      _nonce.toNumber(),
      `Incorrect nonce!`
    )
    listener.receivedTransactions[0].sender.should.equal(
      _sender,
      `Incorrect sender!`
    )
    listener.receivedTransactions[0].target.should.equal(
      _target,
      `Incorrect target!`
    )
    listener.receivedTransactions[0].calldata.should.equal(
      _callData,
      `Incorrect calldata!`
    )
  })

  it('should handle multiple transactions properly', async () => {
    await l1ToL2TransactionProcessor.handle({
      eventID,
      name: L1ToL2TransactionEventName,
      signature: keccak256(Buffer.from('some random stuff').toString('hex')),
      values: {
        _nonce,
        _sender,
        _target,
        _callData,
      },
      blockNumber: 1,
      blockHash: keccak256(Buffer.from('block hash').toString('hex')),
      transactionHash: keccak256(Buffer.from('tx hash').toString('hex')),
    })

    await l1ToL2TransactionProcessor.handle({
      eventID,
      name: L1ToL2TransactionEventName,
      signature: keccak256(Buffer.from('some random stuff').toString('hex')),
      values: {
        _nonce: nonce2,
        _sender: sender2,
        _target: target2,
        _callData: callData2,
      },
      blockNumber: 1,
      blockHash: keccak256(Buffer.from('block hash').toString('hex')),
      transactionHash: keccak256(Buffer.from('tx hash').toString('hex')),
    })

    await sleep(100)

    listener.receivedTransactions.length.should.equal(
      2,
      `Transactions not received!`
    )
    listener.receivedTransactions[0].nonce.should.equal(
      _nonce.toNumber(),
      `Incorrect nonce!`
    )
    listener.receivedTransactions[0].sender.should.equal(
      _sender,
      `Incorrect sender!`
    )
    listener.receivedTransactions[0].target.should.equal(
      _target,
      `Incorrect target!`
    )
    listener.receivedTransactions[0].calldata.should.equal(
      _callData,
      `Incorrect calldata!`
    )

    listener.receivedTransactions[1].nonce.should.equal(
      nonce2.toNumber(),
      `Incorrect nonce 2!`
    )
    listener.receivedTransactions[1].sender.should.equal(
      sender2,
      `Incorrect sender 2!`
    )
    listener.receivedTransactions[1].target.should.equal(
      target2,
      `Incorrect target 2!`
    )
    listener.receivedTransactions[1].calldata.should.equal(
      callData2,
      `Incorrect calldata 2!`
    )
  })

  it('should handle multiple out-of-order transactions properly', async () => {
    await l1ToL2TransactionProcessor.handle({
      eventID,
      name: L1ToL2TransactionEventName,
      signature: keccak256(Buffer.from('some random stuff').toString('hex')),
      values: {
        _nonce: nonce2,
        _sender: sender2,
        _target: target2,
        _callData: callData2,
      },
      blockNumber: 1,
      blockHash: keccak256(Buffer.from('block hash').toString('hex')),
      transactionHash: keccak256(Buffer.from('tx hash').toString('hex')),
    })

    await l1ToL2TransactionProcessor.handle({
      eventID,
      name: L1ToL2TransactionEventName,
      signature: keccak256(Buffer.from('some random stuff').toString('hex')),
      values: {
        _nonce,
        _sender,
        _target,
        _callData,
      },
      blockNumber: 1,
      blockHash: keccak256(Buffer.from('block hash').toString('hex')),
      transactionHash: keccak256(Buffer.from('tx hash').toString('hex')),
    })

    await sleep(100)

    listener.receivedTransactions.length.should.equal(
      2,
      `Transactions not received!`
    )
    listener.receivedTransactions[0].nonce.should.equal(
      _nonce.toNumber(),
      `Incorrect nonce!`
    )
    listener.receivedTransactions[0].sender.should.equal(
      _sender,
      `Incorrect sender!`
    )
    listener.receivedTransactions[0].target.should.equal(
      _target,
      `Incorrect target!`
    )
    listener.receivedTransactions[0].calldata.should.equal(
      _callData,
      `Incorrect calldata!`
    )

    listener.receivedTransactions[1].nonce.should.equal(
      nonce2.toNumber(),
      `Incorrect nonce 2!`
    )
    listener.receivedTransactions[1].sender.should.equal(
      sender2,
      `Incorrect sender 2!`
    )
    listener.receivedTransactions[1].target.should.equal(
      target2,
      `Incorrect target 2!`
    )
    listener.receivedTransactions[1].calldata.should.equal(
      callData2,
      `Incorrect calldata 2!`
    )
  })
})
