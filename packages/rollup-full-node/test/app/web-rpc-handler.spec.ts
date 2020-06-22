import '../setup'
/* External Imports */
import {
  BloomFilter,
  add0x,
  getLogger,
  keccak256,
  JSONRPC_ERRORS,
  hexStrToBuf,
  numberToHexString,
} from '@eth-optimism/core-utils'
import { CHAIN_ID, GAS_LIMIT } from '@eth-optimism/rollup-core'

import { ethers, ContractFactory, Wallet, Contract, utils } from 'ethers'
import { resolve } from 'path'
import * as rimraf from 'rimraf'
import * as fs from 'fs'
import assert from 'assert'

/* Internal Imports */
import { FullnodeRpcServer, DefaultWeb3Handler } from '../../src/app'
import * as SimpleStorage from '../contracts/build/untranspiled/SimpleStorage.json'
import * as EventEmitter from '../contracts/build/untranspiled/EventEmitter.json'
import * as SimpleReversion from '../contracts/build/transpiled/SimpleReversion.json'
import * as MasterEventEmitter from '../contracts/build/transpiled/MasterEventEmitter.json'
import * as SubEventEmitter from '../contracts/build/transpiled/SubEventEmitter.json'
import { Web3RpcMethods } from '../../src/types'

const log = getLogger('web3-handler', true)

const host = '0.0.0.0'
const port = 9999

// Create some constants we will use for storage
const storageKey = '0x' + '01'.repeat(32)
const storageValue = '0x' + '02'.repeat(32)

const EVM_REVERT_MSG = 'VM Exception while processing transaction: revert'

const tmpFilePath = resolve(__dirname, `./.test_db`)

const getWallet = (httpProvider) => {
  const privateKey = '0x' + '60'.repeat(32)
  const wallet = new ethers.Wallet(privateKey, httpProvider)
  log.debug('Wallet address:', wallet.address)
  return wallet
}

const deploySimpleStorage = async (wallet: Wallet): Promise<Contract> => {
  const factory = new ContractFactory(
    SimpleStorage.abi,
    SimpleStorage.bytecode,
    wallet
  )

  // Deploy tx normally
  const simpleStorage = await factory.deploy()
  // Get the deployment tx receipt
  const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
    simpleStorage.deployTransaction.hash
  )
  // Verify that the contract which was deployed is correct
  deploymentTxReceipt.contractAddress.should.equal(simpleStorage.address)

  return simpleStorage
}

const setAndGetStorage = async (
  simpleStorage: Contract,
  httpProvider,
  executionManagerAddress
): Promise<void> => {
  await setStorage(simpleStorage, httpProvider, executionManagerAddress)
  await getAndVerifyStorage(
    simpleStorage,
    httpProvider,
    executionManagerAddress
  )
}

const setStorage = async (
  simpleStorage: Contract,
  httpProvider,
  executionManagerAddress
): Promise<any> => {
  // Set storage with our new storage elements
  const tx = await simpleStorage.setStorage(
    executionManagerAddress,
    storageKey,
    storageValue
  )
  return httpProvider.getTransactionReceipt(tx.hash)
}

const getAndVerifyStorage = async (
  simpleStorage: Contract,
  httpProvider,
  executionManagerAddress
): Promise<void> => {
  // Get the storage
  const res = await simpleStorage.getStorage(
    executionManagerAddress,
    storageKey
  )
  // Verify we got the value!
  res.should.equal(storageValue)
}

/**
 * Creates an unsigned transaction.
 * @param {ethers.Contract} contract
 * @param {String} functionName
 * @param {Array} args
 */
export const getUnsignedTransactionCalldata = (
  contract,
  functionName,
  args
) => {
  return contract.interface.functions[functionName].encode(args)
}

const assertAsyncThrowsWithMessage = async (
  func: () => Promise<any>,
  message: string
): Promise<void> => {
  let succeeded = true
  try {
    await func()
    succeeded = false
  } catch (e) {
    if (e.message !== message) {
      succeeded = false
    }
  }
  succeeded.should.equal(
    true,
    "Function didn't throw as expected or threw with the wrong error message."
  )
}

/*********
 * TESTS *
 *********/

describe('Web3Handler', () => {
  let web3Handler: DefaultWeb3Handler
  let fullnodeRpcServer: FullnodeRpcServer
  let httpProvider

  beforeEach(async () => {
    web3Handler = await DefaultWeb3Handler.create()
    fullnodeRpcServer = new FullnodeRpcServer(web3Handler, host, port)

    fullnodeRpcServer.listen()

    httpProvider = new ethers.providers.JsonRpcProvider(
      `http://${host}:${port}`
    )
  })

  afterEach(() => {
    if (!!fullnodeRpcServer) {
      fullnodeRpcServer.close()
    }
  })

  describe('ephemeral node', () => {
    describe('the getBalance endpoint', () => {
      it('should return zero for all accounts', async () => {
        const wallet = getWallet(httpProvider)
        const balance = await httpProvider.getBalance(wallet.address)

        balance.toNumber().should.eq(0)
      })

      it('should return a parameter error if an invalid parameter is passed', async () => {
        const wallet = getWallet(httpProvider)

        await assertAsyncThrowsWithMessage(async () => {
          await httpProvider.send('eth_getBalance', [1])
        }, JSONRPC_ERRORS.INVALID_PARAMS.message)
      })
    })

    describe('EVM reversion handling', async () => {
      let wallet
      let simpleReversion
      const solidityRevertMessage = 'trolololo'
      beforeEach(async () => {
        wallet = getWallet(httpProvider)
        const factory = new ContractFactory(
          SimpleReversion.abi,
          SimpleReversion.bytecode,
          wallet
        )
        simpleReversion = await factory.deploy()
      })
      it('Should propogate generic internal EVM reverts upwards for eth_sendRawTransaction', async () => {
        await assertAsyncThrowsWithMessage(async () => {
          await simpleReversion.doRevert()
        }, EVM_REVERT_MSG)
      })
      it('Should propogate solidity require messages upwards for eth_sendRawTransaction', async () => {
        await assertAsyncThrowsWithMessage(async () => {
          await simpleReversion.doRevertWithMessage(solidityRevertMessage)
        }, EVM_REVERT_MSG + ' ' + solidityRevertMessage)
      })
      it('Should increment the nonce after a revert', async () => {
        const beforeNonce = await httpProvider.getTransactionCount(
          wallet.address
        )
        let didError = false
        try {
          await simpleReversion.doRevertWithMessage(solidityRevertMessage)
        } catch (e) {
          didError = true
        }
        didError.should.equal(
          true,
          'Expected doRevertWithMessage(...) to throw!'
        )
        const afterNonce = await httpProvider.getTransactionCount(
          wallet.address
        )

        afterNonce.should.equal(
          beforeNonce + 1,
          'Expected the nonce to be incremented by 1!'
        )
      })
      it('Should not serve receipts for reverting transactions', async () => {
        const revertingTx = {
          nonce: await wallet.getTransactionCount(),
          gasPrice: 0,
          gasLimit: 9999999,
          to: simpleReversion.address,
          chainId: CHAIN_ID,
          data: simpleReversion.interface.functions['doRevert'].encode([]),
        }
        const signedTx = await wallet.sign(revertingTx)
        try {
          await httpProvider.send('eth_sendRawTransaction', [signedTx])
          true.should.equal(false, 'above line should have thrown!')
        } catch (e) {
          e.message.should.equal(
            EVM_REVERT_MSG,
            'expected EVM revert but got some other error!'
          )
        }
      })
      it('Should propogate generic EVM reverts for eth_call', async () => {
        await assertAsyncThrowsWithMessage(async () => {
          await simpleReversion.doRevertPure()
        }, EVM_REVERT_MSG)
      })
      it('Should propogate custom message EVM reverts for eth_call', async () => {
        await assertAsyncThrowsWithMessage(async () => {
          await simpleReversion.doRevertWithMessagePure(solidityRevertMessage)
        }, EVM_REVERT_MSG + ' ' + solidityRevertMessage)
      })
    })

    describe('the getBlockByNumber endpoint', () => {
      it('should return a block with the correct timestamp', async () => {
        const block = await httpProvider.getBlock('latest')

        block.timestamp.should.be.gt(0)
      })

      it('should strip the execution manager deployment transaction from the transactions object', async () => {
        const block = await httpProvider.getBlock(1, true)

        block['transactions'].should.be.empty
      })

      it('should increase the timestamp when blocks are created', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const { timestamp } = await httpProvider.getBlock('latest')
        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)
        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )

        const block = await httpProvider.getBlock('latest', true)
        block.timestamp.should.be.gte(timestamp)
      })

      it('should return the latest block with transaction objects', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)
        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )

        const block = await httpProvider.getBlock('latest', true)
        block.transactions[0].from.should.eq(wallet.address)
        block.transactions[0].to.should.eq(simpleStorage.address)
      })

      it('should return the latest block with transaction hashes', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)
        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )

        const block = await httpProvider.getBlock('latest', false)
        hexStrToBuf(block.transactions[0]).length.should.eq(32)
      })

      it('should return a block with the correct logsBloom', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const wallet = getWallet(httpProvider)
        const balance = await httpProvider.getBalance(wallet.address)
        const factory = new ContractFactory(
          EventEmitter.abi,
          EventEmitter.bytecode,
          wallet
        )
        const eventEmitter = await factory.deploy()
        const deploymentTxReceipt = await wallet.provider.getTransactionReceipt(
          eventEmitter.deployTransaction.hash
        )
        const tx = await eventEmitter.emitEvent()
        await wallet.provider.getTransactionReceipt(tx.hash)
        const block = await httpProvider.send('eth_getBlockByNumber', [
          'latest',
          true,
        ])
        const bloomFilter = new BloomFilter(hexStrToBuf(block.logsBloom))
        bloomFilter.check(hexStrToBuf(eventEmitter.address)).should.be.true
      })
    })

    describe('the getBlockByHash endpoint', () => {
      it('should return the same block that is returned by eth_getBlockByNumber', async () => {
        const blockRetrievedByNumber = await httpProvider.getBlock('latest')
        const blockRetrievedByHash = await httpProvider.getBlock(
          blockRetrievedByNumber.hash
        )

        blockRetrievedByHash.should.deep.equal(blockRetrievedByNumber)
      })

      it('should return the same black as eth_getBlockByNumber even after another block is created', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)
        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )

        const blockRetrievedByNumber = await httpProvider.getBlock(
          'latest',
          true
        )
        const blockRetrievedByHash = await httpProvider.getBlock(
          blockRetrievedByNumber.hash,
          true
        )

        blockRetrievedByHash.should.deep.equal(blockRetrievedByNumber)
      })
    })

    describe('the eth_getTransactionByHash endpoint', () => {
      it('should return null if no tx exists', async () => {
        const garbageHash = add0x(
          keccak256(Buffer.from('garbage').toString('hex'))
        )
        const txByHash = await httpProvider.send(
          Web3RpcMethods.getTransactionByHash,
          [garbageHash]
        )

        assert(
          txByHash === null,
          'Should not have gotten a tx for garbage hash!'
        )
      })

      it('should return a tx by OVM hash', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )
        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)

        const calldata = simpleStorage.interface.functions[
          'setStorage'
        ].encode([executionManagerAddress, storageKey, storageValue])

        const tx = {
          nonce: await wallet.getTransactionCount(),
          gasPrice: 0,
          gasLimit: 9999999999,
          to: executionManagerAddress,
          data: calldata,
          chainId: CHAIN_ID,
        }

        const signedTransaction = await wallet.sign(tx)

        const hash = await httpProvider.send(
          Web3RpcMethods.sendRawTransaction,
          [signedTransaction]
        )

        await httpProvider.waitForTransaction(hash)

        const returnedSignedTx = await httpProvider.send(
          Web3RpcMethods.getTransactionByHash,
          [hash]
        )

        const parsedSignedTx = utils.parseTransaction(signedTransaction)

        JSON.stringify(parsedSignedTx).should.eq(
          JSON.stringify(returnedSignedTx),
          'Signed transactions do not match!'
        )
      })
    })

    describe('the getLogs endpoint', () => {
      let wallet
      beforeEach(async () => {
        wallet = getWallet(httpProvider)
      })
      describe('Non-subcall events', async () => {
        let eventEmitter
        let eventEmitterFactory
        beforeEach(async () => {
          eventEmitterFactory = new ContractFactory(
            EventEmitter.abi,
            EventEmitter.bytecode,
            wallet
          )
          eventEmitter = await eventEmitterFactory.deploy()
          await eventEmitter.emitEvent()
        })
        const DUMMY_EVENT_NAME = 'DummyEvent()'
        const verifyEventEmitterLogs = (logs: any) => {
          logs[0].address.should.eq(eventEmitter.address)
          logs[0].logIndex.should.eq(0)
          const parsedLogs = logs.map((x) =>
            eventEmitterFactory.interface.parseLog(x)
          )
          parsedLogs.length.should.eq(1)
          parsedLogs[0].signature.should.eq(DUMMY_EVENT_NAME)
        }
        it('should return correct logs with #nofilter', async () => {
          const logs = await httpProvider.getLogs({
            fromBlock: 'latest',
            toBlock: 'latest',
          })
          verifyEventEmitterLogs(logs)
        })
        it('should return correct logs with address filter', async () => {
          const logs = await httpProvider.getLogs({
            address: eventEmitter.address,
          })
          verifyEventEmitterLogs(logs)
        })
        it('should return correct logs with a topics filter', async () => {
          const dummyTopic =
            eventEmitterFactory.interface.events[DUMMY_EVENT_NAME].topic
          const logs = await httpProvider.getLogs({
            topics: [dummyTopic],
          })
          verifyEventEmitterLogs(logs)
        })
        it('Should throw throw with proper error for unsupported multi-topic filtering', async () => {
          const dummyTopic =
            eventEmitterFactory.interface.events[DUMMY_EVENT_NAME].topic
          assertAsyncThrowsWithMessage(async () => {
            await httpProvider.getLogs({
              topics: [dummyTopic, dummyTopic],
            })
          }, 'Unsupported filter parameters')
        })
      })
      describe('Nested contract call events', async () => {
        let sub
        let master
        let subFactory
        const SUB_EMITTER_EVENT_NAME = 'Burger'
        beforeEach(async () => {
          subFactory = new ContractFactory(
            SubEventEmitter.abi,
            SubEventEmitter.bytecode,
            wallet
          )
          sub = await subFactory.deploy()
          const masterFactory = new ContractFactory(
            MasterEventEmitter.abi,
            MasterEventEmitter.bytecode,
            wallet
          )
          master = await masterFactory.deploy(sub.address)
        })
        it('should return nested contract call events with the correct addresses for each log', async () => {
          await master.callSubEmitter()
          const logs = await httpProvider.send(Web3RpcMethods.getLogs, [
            {
              fromBlock: 'latest',
              toBlock: 'latest',
            },
          ])
          logs[0].address.should.eq(master.address)
          logs[1].address.should.eq(sub.address)
        })
        it('Should correctly filter by topic for the inner emission', async () => {
          await master.callSubEmitter()
          const subEventEmitterTopic =
            subFactory.interface.events[SUB_EMITTER_EVENT_NAME].topic
          const gotLogs = await httpProvider.send(Web3RpcMethods.getLogs, [
            {
              topics: [subEventEmitterTopic],
            },
          ])
          gotLogs.length.should.equal(1)
          gotLogs[0].topics.should.deep.equal([subEventEmitterTopic])
          gotLogs[0].address.should.equal(sub.address)
          gotLogs[0].logIndex.should.equal('0x1')
        })
        it("should return logs which are the same as a transaction receipt's logs", async () => {
          const tx = await master.callSubEmitter()
          const gotLogs = await httpProvider.send(Web3RpcMethods.getLogs, [
            {
              fromBlock: 'latest',
              toBlock: 'latest',
            },
          ])
          const receipt = await httpProvider.send(
            Web3RpcMethods.getTransactionReceipt,
            [tx.hash]
          )
          gotLogs.should.deep.equal(receipt.logs)
        })
      })
    })

    describe('SimpleStorage integration test', () => {
      it('should set storage & retrieve the value', async () => {
        const executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )

        const wallet = getWallet(httpProvider)
        const simpleStorage = await deploySimpleStorage(wallet)

        await setAndGetStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )
      })
    })

    describe('snapshot and revert', () => {
      it('should  fail (snapshot and revert should only be available in the TestHandler)', async () => {
        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_snapshot', []).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })

        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_snapshot', []).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })

        await new Promise((resolveFunc, reject) => {
          httpProvider.send('evm_revert', ['0x01']).catch((error) => {
            error.message.should.equal('Method not found')
            resolveFunc()
          })
        })
      })
    })

    describe('L1 to L2 Transaction Passing', () => {
      let executionManagerAddress
      let simpleStorage: Contract
      let wallet: Wallet
      beforeEach(async () => {
        executionManagerAddress = await httpProvider.send(
          'ovm_getExecutionManagerAddress',
          []
        )

        wallet = getWallet(httpProvider)
        simpleStorage = await deploySimpleStorage(wallet)
      })

      it('should process L1 to L2 Transaction', async () => {
        const callData = getUnsignedTransactionCalldata(
          simpleStorage,
          'setStorage',
          [executionManagerAddress, storageKey, storageValue]
        )
        await web3Handler.handleL1ToL2Transaction({
          nonce: 0,
          gasLimit: 0,
          calldata: callData,
          sender: wallet.address,
          target: simpleStorage.address,
        })

        await getAndVerifyStorage(
          simpleStorage,
          httpProvider,
          executionManagerAddress
        )
      })

      it('should not throw if L1 to L2 Transaction reverts', async () => {
        const callData = getUnsignedTransactionCalldata(
          simpleStorage,
          'justRevert',
          []
        )
        await web3Handler.handleL1ToL2Transaction({
          nonce: 0,
          gasLimit: 0,
          calldata: callData,
          sender: wallet.address,
          target: simpleStorage.address,
        })
      })
    })
  })

  describe('persisted node', () => {
    let emAddress: string
    let wallet: Wallet
    let simpleStorage: Contract

    before(() => {
      rimraf.sync(tmpFilePath)
      fs.mkdirSync(tmpFilePath)
      process.env.LOCAL_L2_NODE_PERSISTENT_DB_PATH = tmpFilePath
    })
    after(() => {
      rimraf.sync(tmpFilePath)
      delete process.env.LOCAL_L2_NODE_PERSISTENT_DB_PATH
    })

    it('1/2 deploys the contracts', async () => {
      emAddress = await httpProvider.send('ovm_getExecutionManagerAddress', [])
      wallet = getWallet(httpProvider)

      simpleStorage = await deploySimpleStorage(wallet)
    })

    it('2/2 uses previously deployed contract', async () => {
      await setAndGetStorage(simpleStorage, httpProvider, emAddress)
    })
  })
})
