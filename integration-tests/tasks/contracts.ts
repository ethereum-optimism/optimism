import dotenv from 'dotenv'

dotenv.config()

import { Wallet, ContractFactory, Contract, constants, BigNumber } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { l2Provider } from '../test/shared/utils'
import ERC20 from '../artifacts/contracts/ERC20.sol/ERC20.json'
import { writeStderr } from './util'
import { UniswapV3Deployer } from 'uniswap-v3-deploy-plugin/dist/deployer/UniswapV3Deployer'
import { abi as NFTABI } from '@uniswap/v3-periphery/artifacts/contracts/NonfungiblePositionManager.sol/NonfungiblePositionManager.json'
import { FeeAmount, TICK_SPACINGS } from '@uniswap/v3-sdk'
import ERC721 from '../artifacts/contracts/NFT.sol/NFT.json'

// Below methods taken from the Uniswap test suite, see
// https://github.com/Uniswap/v3-periphery/blob/main/test/shared/ticks.ts
export const getMinTick = (tickSpacing: number) =>
  Math.ceil(-887272 / tickSpacing) * tickSpacing
export const getMaxTick = (tickSpacing: number) =>
  Math.floor(887272 / tickSpacing) * tickSpacing

task('deploy-erc20')
  .addOptionalParam('name', 'Name of the ERC20.', 'OVM Test', types.string)
  .addOptionalParam('symbol', 'Symbol of the ERC20.', 'OVM', types.string)
  .addOptionalParam('decimals', 'Decimals of the ERC20.', 18, types.int)
  .addOptionalParam(
    'initialSupply',
    'Token initial supply.',
    constants.MaxUint256.toString(),
    types.string
  )
  .setAction(async (args) => {
    writeStderr(`Deploying ERC20 ${args.name}...`)
    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const factory = new ContractFactory(ERC20.abi, ERC20.bytecode, wallet)
    const token = await factory.deploy(
      args.initialSupply,
      args.name,
      args.decimals,
      args.symbol
    )
    await token.deployed()
    writeStderr(`Successfully deployed ERC20 ${args.name}.`)

    console.log(
      JSON.stringify(
        {
          address: token.address,
          name: args.name,
          symbol: args.symbol,
          decimals: args.decimals,
          initialSupply: args.initialSupply,
        },
        null,
        2
      )
    )
  })

task('deploy-erc721').setAction(async () => {
  const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
  const factory = new ContractFactory(ERC721.abi, ERC721.bytecode, wallet)
  writeStderr('Deploying ERC-721...')
  const contract = await factory.deploy()
  await contract.deployed()
  writeStderr('Done')
  console.log(
    JSON.stringify({
      address: contract.address,
    })
  )
})

task('approve-erc20')
  .addParam('tokenAddress', 'Address of the token.', '', types.string)
  .addParam('approvingAddress', 'Address to approve.', '', types.string)
  .addParam(
    'amount',
    'Amount to approve',
    constants.MaxUint256.toString(),
    types.string
  )
  .setAction(async (args) => {
    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    writeStderr(
      `Approving ${args.approvingAddress} to spend ${args.amount} from ${args.tokenAddress}...`
    )
    const contract = new Contract(args.tokenAddress, ERC20.abi).connect(wallet)
    const tx = await contract.approve(args.approvingAddress, args.amount)
    await tx.wait()
    writeStderr('Done.')
  })

task('deploy-uniswap').setAction(async () => {
  const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
  writeStderr('Deploying Uniswap ecosystem...')
  const contracts = await UniswapV3Deployer.deploy(wallet)
  writeStderr('Done.')
  console.log(
    JSON.stringify(
      Object.entries(contracts).reduce((acc, [k, v]) => {
        acc[k] = v.address
        return acc
      }, {}),
      null,
      2
    )
  )
})

task('bootstrap-uniswap-pool')
  .addParam(
    'positionManagerAddress',
    'Address of the position manager.',
    '',
    types.string
  )
  .addParam('token0Address', 'Address of the first token', '', types.string)
  .addParam('token1Address', 'Address of the second token', '', types.string)
  .addParam(
    'initialRatio',
    'Initial price ratio.',
    BigNumber.from('79228162514264337593543950336').toString(),
    types.string
  )
  .addParam(
    'amount0',
    'Amount of the first token to put in the position',
    '1000000000',
    types.string
  )
  .addParam(
    'amount1',
    'Amount of the second token to put in the position',
    '1000000000',
    types.string
  )
  .setAction(async (args) => {
    let tokensAmounts = [
      {
        address: args.token0Address,
        amount: args.amount0,
      },
      {
        address: args.token1Address,
        amount: args.amount1,
      },
    ]

    if (tokensAmounts[0].address > tokensAmounts[1].address) {
      tokensAmounts = [tokensAmounts[1], tokensAmounts[0]]
    }

    const wallet = new Wallet(process.env.PRIVATE_KEY).connect(l2Provider)
    const positionManager = new Contract(
      args.positionManagerAddress,
      NFTABI
    ).connect(wallet)
    writeStderr('Creating pool...')
    let tx = await positionManager.createAndInitializePoolIfNecessary(
      tokensAmounts[0].address,
      tokensAmounts[1].address,
      FeeAmount.MEDIUM,
      BigNumber.from(args.initialRatio)
    )
    await tx.wait()

    writeStderr('Minting position...')
    tx = await positionManager.mint(
      {
        token0: tokensAmounts[0].address,
        token1: tokensAmounts[1].address,
        tickLower: getMinTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        tickUpper: getMaxTick(TICK_SPACINGS[FeeAmount.MEDIUM]),
        fee: FeeAmount.MEDIUM,
        recipient: wallet.address,
        amount0Desired: tokensAmounts[0].amount,
        amount1Desired: tokensAmounts[1].amount,
        amount0Min: 0,
        amount1Min: 0,
        deadline: Date.now() * 2,
      },
      {
        gasLimit: 10000000,
      }
    )
    await tx.wait()
  })
