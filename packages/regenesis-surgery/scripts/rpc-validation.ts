import { providers, ethers, Contract, BigNumber, utils } from 'ethers'
import { getContractFactory } from '@eth-optimism/contracts'
import { bytecode as poolBytecode } from '@uniswap/v3-core/artifacts/contracts/UniswapV3Pool.sol/UniswapV3Pool.json'
import { createReadStream } from 'fs'
import { parseChunked } from '@discoveryjs/json-ext'
import dotenv from 'dotenv'
import { AllEventsOutput } from './event-indexer'

dotenv.config()

const env = process.env
const SEQUENCER_URL = env.SEQUENCER_URL || 'http://localhost:8545'
const NEW_SEQUENCER_URL = env.NEW_SEQUENCER_URL || 'http://localhost:8546'
const ETH_ADDR = env.ETH_ADDR || '0x4200000000000000000000000000000000000006'
const OUTPUT_FILE_PATH = env.OUTPUT_FILE_PATH || './all-events.json'

const checkAllTokenBalances = async (
  transferEvents: ethers.Event[],
  oldErc20Contract: Contract,
  newErc20Contract: Contract
) => {
  const addresses = new Set()

  // find all addresses
  for (const e of transferEvents) {
    addresses.add(e.args[0])
  }

  console.log('checking all known addresses for their eth balance')
  console.log('number of addresses', addresses.size)
  for (const address of addresses) {
    console.log('getting balances', address)
    const [newBalance, oldBalance]: BigNumber[] = await Promise.all([
      oldErc20Contract.balanceOf(address),
      newErc20Contract.balanceOf(address),
    ])

    console.log('checking balances')
    if (!newBalance.eq(oldBalance)) {
      console.error(
        'found mismatched balance',
        address,
        oldBalance.toString(),
        newBalance.toString()
      )
      throw new Error('Found mismatched balance!')
    }
  }
}

// from @uniswap/v3-core
// https://github.com/Uniswap/uniswap-v3-core/blob/b2c5555d696428c40c4b236069b3528b2317f3c1/test/shared/utilities.ts#L37
const getCreate2Address = (
  factoryAddress: string,
  [tokenA, tokenB]: [string, string],
  fee: number,
  bytecode: string
): string => {
  const [token0, token1] =
    tokenA.toLowerCase() < tokenB.toLowerCase()
      ? [tokenA, tokenB]
      : [tokenB, tokenA]
  const constructorArgumentsEncoded = utils.defaultAbiCoder.encode(
    ['address', 'address', 'uint24'],
    [token0, token1, fee]
  )
  const create2Inputs = [
    '0xff',
    factoryAddress,
    // salt
    utils.keccak256(constructorArgumentsEncoded),
    // init code. bytecode + constructor arguments
    utils.keccak256(bytecode),
  ]
  const sanitizedInputs = `0x${create2Inputs.map((i) => i.slice(2)).join('')}`
  return utils.getAddress(`0x${utils.keccak256(sanitizedInputs).slice(-40)}`)
}

const checkPoolAddresses = async (
  poolEvents: ethers.Event[],
  oldSigner: ethers.Signer,
  newSigner: ethers.Signer
) => {
  // get balances of both contracts
  const poolsToTokens = {}
  for (const e of poolEvents) {
    // token0, token1, fee
    poolsToTokens[e.args[4]] = [e.args[0], e.args[1], e.args[2]]
  }

  for (const pool in poolsToTokens) {
    if (poolsToTokens.hasOwnProperty(pool)) {
      console.log('checking pool with old address', pool)
      const token0Addr = poolsToTokens[pool][0]
      const token1Addr = poolsToTokens[pool][1]
      const fee = poolsToTokens[pool][2]
      console.log('got token0, token1, fee', token0Addr, token1Addr, fee)
      const oldToken0Contract = getContractFactory('L2StandardERC20')
        .connect(oldSigner)
        .attach(token0Addr)
      const oldToken1Contract = getContractFactory('L2StandardERC20')
        .connect(oldSigner)
        .attach(token1Addr)

      const newToken0Contract = getContractFactory('L2StandardERC20')
        .connect(newSigner)
        .attach(token0Addr)
      const newToken1Contract = getContractFactory('L2StandardERC20')
        .connect(newSigner)
        .attach(token1Addr)

      // last pool address should have balance 0 in new sequencer
      const [lastPoolToken0Balance, lastPoolToken1Balance]: BigNumber[] =
        await Promise.all([
          newToken0Contract.balanceOf(pool),
          newToken1Contract.balanceOf(pool),
        ])
      if (lastPoolToken0Balance.gt(0) || lastPoolToken1Balance.gt(0)) {
        console.error(
          'new sequencer still has balance in old pool address',
          pool,
          lastPoolToken0Balance.toString(),
          lastPoolToken1Balance.toString()
        )
        throw new Error('Token balance not 0 for old pool on new sequencer!')
      }

      // check pool address balance on old sequencer
      const [oldSeqToken0Balance, oldSeqToken1Balance]: BigNumber[] =
        await Promise.all([
          oldToken0Contract.balanceOf(pool),
          oldToken1Contract.balanceOf(pool),
        ])
      const newPoolAddr = getCreate2Address(
        token0Addr,
        token1Addr,
        fee,
        poolBytecode
      )
      // check that new pool address has the same balance as old pool addreses
      const [newPoolToken0Balance, newPoolToken1Balance]: BigNumber[] =
        await Promise.all([
          newToken0Contract.balanceOf(newPoolAddr),
          newToken1Contract.balanceOf(newPoolAddr),
        ])
      if (
        !newPoolToken0Balance.eq(oldSeqToken0Balance) ||
        !newPoolToken1Balance.eq(oldSeqToken1Balance)
      ) {
        console.error(
          'balance at new pool is not equal to old balance',
          newPoolAddr
        )
      }
    }
  }
}

;(async () => {
  const oldSequencer = new ethers.providers.StaticJsonRpcProvider(SEQUENCER_URL)
  const newSequencer = new ethers.providers.StaticJsonRpcProvider(
    NEW_SEQUENCER_URL
  )
  const oldSigner = ethers.Wallet.createRandom().connect(oldSequencer)
  const newSigner = ethers.Wallet.createRandom().connect(newSequencer)

  const eventsOutput: AllEventsOutput = await parseChunked(
    createReadStream(OUTPUT_FILE_PATH)
  )

  const oldEthContract = getContractFactory('OVM_ETH')
    .connect(oldSigner)
    .attach(ETH_ADDR)
  const newEthContract = getContractFactory('OVM_ETH')
    .connect(newSigner)
    .attach(ETH_ADDR)
  await checkAllTokenBalances(
    eventsOutput.ethTransfers,
    oldEthContract,
    newEthContract
  )

  await checkPoolAddresses(eventsOutput.uniV3PoolCreated, oldSigner, newSigner)
})().catch((err) => {
  console.log(err)
  process.exit(1)
})
