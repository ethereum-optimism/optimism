import _ from 'lodash'
import abiDecoder from 'abi-decoder'
import web3 from 'web3'
import { TransactionReceipt, TransactionResponse, WebSocketProvider } from '@ethersproject/providers'
import dotenv from 'dotenv'
dotenv.config()
import logger from './logger'

const l1WsUrl = process.env.L1_NODE_WEB3_WS
const l2WsUrl = process.env.L2_NODE_WEB3_WS
const l1PoolAddress = process.env.L1_LIQUIDITY_POOL_ADDRESS
const l2PoolAddress = process.env.L2_LIQUIDITY_POOL_ADDRESS
const relayerAddress = process.env.RELAYER_ADDRESS
const sequencerAddress = process.env.SEQUENCER_ADDRESS
const reconnectTime = parseInt(process.env.RECONNECT_TIME, 10) || 10000
let l1PoolBalance
let l1RelayerBalance
let l1SequencerBalance
let l1BlockNumber
let l1GasPrice
let l2PoolBalance
let l2BlockNumber
let l2GasPrice

// logger.info({ key: 'config', data: 'config.parsed' })
// console.log(parseInt('0x51cff8d9', 16))
// const LitContractABI = [{ 'anonymous': false, 'inputs': [{ 'indexed': false, 'internalType': 'uint256', 'name': 'recipientMinBalance', 'type': 'uint256' }, { 'indexed': false, 'internalType': 'uint256', 'name': 'recipientDesiredBalance', 'type': 'uint256' }], 'name': 'Configure', 'type': 'event' },{ 'anonymous': false, 'inputs': [{ 'indexed': true, 'internalType': 'address', 'name': 'caller', 'type': 'address' },{ 'indexed': false, 'internalType': 'uint256', 'name': 'amount', 'type': 'uint256' }], 'name': 'Deposit', 'type': 'event' },{ 'anonymous': false, 'inputs': [{ 'indexed': true, 'internalType': 'address', 'name': 'previousOwner', 'type': 'address' },{ 'indexed': true, 'internalType': 'address', 'name': 'newOwner', 'type': 'address' }], 'name': 'OwnershipTransferred', 'type': 'event' },{ 'anonymous': false, 'inputs': [{ 'indexed': true, 'internalType': 'address', 'name': 'account', 'type': 'address' },{ 'indexed': false, 'internalType': 'bool', 'name': 'whitelisted', 'type': 'bool' }], 'name': 'Whitelist', 'type': 'event' },{ 'anonymous': false, 'inputs': [{ 'indexed': true, 'internalType': 'address', 'name': 'caller', 'type': 'address' },{ 'indexed': true, 'internalType': 'address', 'name': 'to', 'type': 'address' },{ 'indexed': false, 'internalType': 'uint256', 'name': 'amount', 'type': 'uint256' }], 'name': 'Withdraw', 'type': 'event' },{ 'inputs': [{ 'internalType': 'uint256', 'name': 'minBalance', 'type': 'uint256' },{ 'internalType': 'uint256', 'name': 'desiredBalance', 'type': 'uint256' }], 'name': 'configure', 'outputs': [], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'inputs': [], 'name': 'deposit', 'outputs': [], 'stateMutability': 'payable', 'type': 'function' },{ 'inputs': [], 'name': 'owner', 'outputs': [{ 'internalType': 'address', 'name': '', 'type': 'address' }], 'stateMutability': 'view', 'type': 'function' },{ 'inputs': [], 'name': 'recipientDesiredBalance', 'outputs': [{ 'internalType': 'uint256', 'name': '', 'type': 'uint256' }], 'stateMutability': 'view', 'type': 'function' },{ 'inputs': [], 'name': 'recipientMinBalance', 'outputs': [{ 'internalType': 'uint256', 'name': '', 'type': 'uint256' }], 'stateMutability': 'view', 'type': 'function' },{ 'inputs': [], 'name': 'renounceOwnership', 'outputs': [], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'inputs': [{ 'internalType': 'address', 'name': 'account', 'type': 'address' },{ 'internalType': 'bool', 'name': 'whitelisted', 'type': 'bool' }], 'name': 'setWhitelist', 'outputs': [], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'inputs': [{ 'internalType': 'address', 'name': 'newOwner', 'type': 'address' }], 'name': 'transferOwnership', 'outputs': [], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'inputs': [{ 'internalType': 'address', 'name': '', 'type': 'address' }], 'name': 'whitelist', 'outputs': [{ 'internalType': 'bool', 'name': '', 'type': 'bool' }], 'stateMutability': 'view', 'type': 'function' },{ 'inputs': [{ 'internalType': 'address payable', 'name': 'to', 'type': 'address' }], 'name': 'withdraw', 'outputs': [{ 'internalType': 'uint256', 'name': '', 'type': 'uint256' }], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'inputs': [{ 'internalType': 'address payable', 'name': 'to', 'type': 'address' },{ 'internalType': 'uint256', 'name': 'amount', 'type': 'uint256' }], 'name': 'withdrawAll', 'outputs': [], 'stateMutability': 'nonpayable', 'type': 'function' },{ 'stateMutability': 'payable', 'type': 'receive' }]
// abiDecoder.addABI(LitContractABI)
// const testData = '0x095ea7b3000000000000000000000000d9e1ce17f2641f24ae83637ab66a2cca9c378b9fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
// const decodedData = abiDecoder.decodeMethod(testData)
// console.log(decodedData)

enum OMGXNetwork {
  L1 = 'L1',
  L2 = 'L2',
}

const convertWeiToEther = (wei: any) => {
  return parseFloat(web3.utils.fromWei(wei.toString(), 'ether'))
}

const logError = (message: string, key: string, extra = {}) => {
  return (err: Error) => {
    logger.error(message, {
      key,
      ...extra,
      error: err.message,
    })
  }
}

const checkNetwork = (l1Provider: WebSocketProvider, l2Provider: WebSocketProvider) => {
  let result = true
  if (!l1Provider || l1Provider._websocket.readyState !== 1) {
    logger.error('Connect to L1', {
      key: 'network',
      error: 'L1 network is not connected',
    })
    result = false
  }
  if (!l2Provider || l2Provider._websocket.readyState !== 1) {
    logger.error('Connect to L2', {
      key: 'network',
      error: 'L2 network is not connected',
    })
    result = false
  }
  return result
}

const logBalance = (provider: WebSocketProvider, blockNumber: number, networkName: OMGXNetwork) => {
  const promiseData = networkName === OMGXNetwork.L1 ? [
    provider.getBalance(l1PoolAddress),
    provider.getBalance(relayerAddress),
    provider.getBalance(sequencerAddress),
    provider.getGasPrice(),
  ] : [
    provider.getBalance(l2PoolAddress),
    provider.getGasPrice(),
  ]

  Promise.all(promiseData)
  .then((values) => {
    if (values.length === 4) {
      l1PoolBalance = convertWeiToEther(values[0])
      l1RelayerBalance = convertWeiToEther(values[1])
      l1SequencerBalance = convertWeiToEther(values[2])
      l1GasPrice = parseFloat(values[3].toString())
      l1BlockNumber = blockNumber
    } else {
      l2PoolBalance = convertWeiToEther(values[0])
      l2GasPrice = parseFloat(values[1].toString())
      l2BlockNumber = blockNumber
    }
  })
  .then(() => {
    if (l1PoolBalance !== undefined) {
      logger.info(`${OMGXNetwork.L1} balance`, {
        networkName: OMGXNetwork.L1,
        key: 'balance',
        data: {
          poolAddress: l1PoolAddress,
          poolBalance: l1PoolBalance,
          relayerBalance: l1RelayerBalance,
          sequencerBalance: l1SequencerBalance,
          gasPrice: l1GasPrice,
          blockNumber: l1BlockNumber,
        }
      })
    }
    if (l2PoolBalance !== undefined) {
      logger.info(`${OMGXNetwork.L2} balance`, {
        networkName: OMGXNetwork.L2,
        key: 'balance',
        data: {
          poolAddress: l2PoolAddress,
          poolBalance: l2PoolBalance,
          gasPrice: l2GasPrice,
          blockNumber: l2BlockNumber,
        }
      })
    }
  }).catch((err) => {
    logger.error(`Get ${networkName} balance error`, {
      networkName,
      key: 'balance',
      error: err.message,
    })
  })
}

const logTransaction = (socket: WebSocketProvider, trans: TransactionResponse, networkName: OMGXNetwork, metadata = {}) => {
  // check from/to address is pool address
  const poolAddress = (networkName === OMGXNetwork.L1) ? l1PoolAddress : l2PoolAddress
  logger.debug({
    from: trans.from,
    to: trans.to,
    poolAddress
  })
  if (trans.from !== poolAddress && trans.to !== poolAddress) {
    return
  }

  socket.getTransactionReceipt(trans.hash)
  .then((receipt: TransactionReceipt) => {
    try {
      logger.info('transaction ' + trans.hash, {
        key: 'transaction',
        poolAddress,
        networkName,
        data: _.omit(receipt, ['logs']),
      })
      receipt.logs.forEach((log) => {
        logger.info('event ' + log.address, {
          poolAddress,
          networkName,
          key: 'event',
          data: log
        })
      })
    } catch (err) {
      logError('Error while logging transaction receipt', 'receipt', { ...metadata, receipt })
    }
  })
  .catch(logError('Error while getting transaction receipt', 'transaction', { ...metadata }))
}

const logData = (socket: WebSocketProvider, blockNumber: string, networkName: OMGXNetwork) => {
  const poolAddress = (networkName === OMGXNetwork.L1) ? l1PoolAddress : l2PoolAddress
  const metadata = {
    blockNumber,
    networkName,
    poolAddress,
  }

  socket.getBlockWithTransactions(blockNumber)
  .then((block) => {
    block.transactions.forEach((trans) => {
      logTransaction(socket, trans, networkName, metadata)
    })
  })
  .catch(logError('Error while getting block', 'block', { ...metadata }))
}

const onConnected = (networkName: OMGXNetwork) => {
  return (event: { target: { _url: any } }) => {
    logger.info(`${networkName} network connected`, {
      url: event.target._url,
      key: 'network'
    })
  }
}

const onError = (networkName: OMGXNetwork, provider: WebSocketProvider) => {
  return async (event: { message: any; target: { _url: string } }) => {
    logger.error(`${networkName} network failed to connect`, {
      networkName,
      error: event.message,
      url: event.target._url,
      key: 'network'
    })
    await provider.destroy()
    setTimeout(() => {
      logger.info(`${networkName} reconnecting ...`, {
        networkName,
        url: event.target._url,
        key: 'network'
      })
      setupProvider(networkName, event.target._url)
    }, reconnectTime)
  }
}

const setupProvider = (networkName: OMGXNetwork, url: string) => {
  const provider = new WebSocketProvider(url)
  provider._websocket.addEventListener('open', onConnected(networkName))
  provider._websocket.addEventListener('error', onError(networkName, provider))
  provider._subscribe('block', ['newHeads'], (result: any) => {
    const blockNumber = parseInt(result.number, 16)

    // log transactions and events
    logData(provider, result.number, networkName)

    // log balances
    logBalance(provider, blockNumber, networkName)
  }).catch()
}

setupProvider(OMGXNetwork.L1, l1WsUrl)
setupProvider(OMGXNetwork.L2, l2WsUrl)

// l1Socket.getTransactionReceipt('0x4ba7c3ea403461b159df4ca0c6e38ece3b7c82fe35aa8d4669c2c00572a278e6')
// l1Socket.getTransactionReceipt('0x041e9d9c19c3a11ef4a37f95a3661195f95b67619f5a04faa9fe534e8a9f937a')
// .then((receipt) => {
  // logger.debug(receipt.logs)
  // logger.debug(abiDecoder.decodeLogs(receipt.logs))
// })
// .catch()
