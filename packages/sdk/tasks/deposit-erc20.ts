import { promises as fs } from 'fs'

import { task, types } from 'hardhat/config'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { Event, Contract, Wallet, providers, utils, ethers } from 'ethers'
import { predeploys, sleep } from '@eth-optimism/core-utils'

import Artifact__WETH9 from '../src/forge-artifacts/WETH9.json'
import Artifact__OptimismMintableERC20TokenFactory from '../src/forge-artifacts/OptimismMintableERC20Factory.json'
import Artifact__OptimismMintableERC20Token from '../src/forge-artifacts/OptimismMintableERC20.json'
import Artifact__L2ToL1MessagePasser from '../src/forge-artifacts/L2ToL1MessagePasser.json'
import Artifact__L2CrossDomainMessenger from '../src/forge-artifacts/L2CrossDomainMessenger.json'
import Artifact__L2StandardBridge from '../src/forge-artifacts/L2StandardBridge.json'
import Artifact__OptimismPortal from '../src/forge-artifacts/OptimismPortal.json'
import Artifact__L1CrossDomainMessenger from '../src/forge-artifacts/L1CrossDomainMessenger.json'
import Artifact__L1StandardBridge from '../src/forge-artifacts/L1StandardBridge.json'
import Artifact__L2OutputOracle from '../src/forge-artifacts/L2OutputOracle.json'
import {
  CrossChainMessenger,
  MessageStatus,
  CONTRACT_ADDRESSES,
  OEContractsLike,
  DEFAULT_L2_CONTRACT_ADDRESSES,
} from '../src'

const deployWETH9 = async (
  hre: HardhatRuntimeEnvironment,
  signer: SignerWithAddress,
  wrap: boolean
): Promise<Contract> => {
  const Factory__WETH9 = new hre.ethers.ContractFactory(
    Artifact__WETH9.abi,
    Artifact__WETH9.bytecode.object,
    signer
  )

  console.log('Sending deployment transaction')
  const WETH9 = await Factory__WETH9.deploy()
  const receipt = await WETH9.deployTransaction.wait()
  console.log(`WETH9 deployed: ${receipt.transactionHash}`)

  if (wrap) {
    const deposit = await signer.sendTransaction({
      value: utils.parseEther('1'),
      to: WETH9.address,
    })
    await deposit.wait()
  }

  return WETH9
}

const createOptimismMintableERC20 = async (
  hre: HardhatRuntimeEnvironment,
  L1ERC20: Contract,
  l2Signer: Wallet
): Promise<Contract> => {
  const OptimismMintableERC20TokenFactory = new Contract(
    predeploys.OptimismMintableERC20Factory,
    Artifact__OptimismMintableERC20TokenFactory.abi,
    l2Signer
  )

  const name = await L1ERC20.name()
  const symbol = await L1ERC20.symbol()

  const tx =
    await OptimismMintableERC20TokenFactory.createOptimismMintableERC20(
      L1ERC20.address,
      `L2 ${name}`,
      `L2-${symbol}`
    )

  const receipt = await tx.wait()
  const event = receipt.events.find(
    (e: Event) => e.event === 'OptimismMintableERC20Created'
  )

  if (!event) {
    throw new Error('Unable to find OptimismMintableERC20Created event')
  }

  const l2WethAddress = event.args.localToken
  console.log(`Deployed to ${l2WethAddress}`)

  return new Contract(
    l2WethAddress,
    Artifact__OptimismMintableERC20Token.abi,
    l2Signer
  )
}

// TODO(tynes): this task could be modularized in the future
// so that it can deposit an arbitrary token. Right now it
// deploys a WETH9 contract, mints some WETH9 and then
// deposits that into L2 through the StandardBridge.
task('deposit-erc20', 'Deposits WETH9 onto L2.')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .addParam(
    'opNodeProviderUrl',
    'op-node provider URL',
    'http://localhost:7545',
    types.string
  )
  .addOptionalParam(
    'l1ContractsJsonPath',
    'Path to a JSON with L1 contract addresses in it',
    '',
    types.string
  )
  .addOptionalParam('signerIndex', 'Index of signer to use', 0, types.int)
  .setAction(async (args, hre) => {
    const signers = await hre.ethers.getSigners()
    if (signers.length === 0) {
      throw new Error('No configured signers')
    }
    if (args.signerIndex < 0 || signers.length <= args.signerIndex) {
      throw new Error('Invalid signer index')
    }
    const signer = signers[args.signerIndex]
    const address = await signer.getAddress()
    console.log(`Using signer ${address}`)

    // Ensure that the signer has a balance before trying to
    // do anything
    const balance = await signer.getBalance()
    if (balance.eq(0)) {
      throw new Error('Signer has no balance')
    }

    const l2Provider = new providers.StaticJsonRpcProvider(args.l2ProviderUrl)

    const l2Signer = new hre.ethers.Wallet(
      hre.network.config.accounts[args.signerIndex],
      l2Provider
    )

    const l2ChainId = await l2Signer.getChainId()
    let contractAddrs = CONTRACT_ADDRESSES[l2ChainId]
    if (args.l1ContractsJsonPath) {
      const data = await fs.readFile(args.l1ContractsJsonPath)
      const json = JSON.parse(data.toString())
      contractAddrs = {
        l1: {
          AddressManager: json.AddressManager,
          L1CrossDomainMessenger: json.L1CrossDomainMessengerProxy,
          L1StandardBridge: json.L1StandardBridgeProxy,
          StateCommitmentChain: ethers.constants.AddressZero,
          CanonicalTransactionChain: ethers.constants.AddressZero,
          BondManager: ethers.constants.AddressZero,
          OptimismPortal: json.OptimismPortalProxy,
          L2OutputOracle: json.L2OutputOracleProxy,
          OptimismPortal2: json.OptimismPortalProxy,
          DisputeGameFactory: json.DisputeGameFactoryProxy,
        },
        l2: DEFAULT_L2_CONTRACT_ADDRESSES,
      } as OEContractsLike
    }

    console.log(`OptimismPortal: ${contractAddrs.l1.OptimismPortal}`)
    const OptimismPortal = new hre.ethers.Contract(
      contractAddrs.l1.OptimismPortal,
      Artifact__OptimismPortal.abi,
      signer
    )

    console.log(
      `L1CrossDomainMessenger: ${contractAddrs.l1.L1CrossDomainMessenger}`
    )
    const L1CrossDomainMessenger = new hre.ethers.Contract(
      contractAddrs.l1.L1CrossDomainMessenger,
      Artifact__L1CrossDomainMessenger.abi,
      signer
    )

    console.log(`L1StandardBridge: ${contractAddrs.l1.L1StandardBridge}`)
    const L1StandardBridge = new hre.ethers.Contract(
      contractAddrs.l1.L1StandardBridge,
      Artifact__L1StandardBridge.abi,
      signer
    )

    const L2OutputOracle = new hre.ethers.Contract(
      contractAddrs.l1.L2OutputOracle,
      Artifact__L2OutputOracle.abi,
      signer
    )

    const L2ToL1MessagePasser = new hre.ethers.Contract(
      predeploys.L2ToL1MessagePasser,
      Artifact__L2ToL1MessagePasser.abi
    )

    const L2CrossDomainMessenger = new hre.ethers.Contract(
      predeploys.L2CrossDomainMessenger,
      Artifact__L2CrossDomainMessenger.abi
    )

    const L2StandardBridge = new hre.ethers.Contract(
      predeploys.L2StandardBridge,
      Artifact__L2StandardBridge.abi
    )

    const messenger = new CrossChainMessenger({
      l1SignerOrProvider: signer,
      l2SignerOrProvider: l2Signer,
      l1ChainId: await signer.getChainId(),
      l2ChainId,
      bedrock: true,
      contracts: contractAddrs,
    })

    const params = await OptimismPortal.params()
    console.log('Intial OptimismPortal.params:')
    console.log(params)

    console.log('Deploying WETH9 to L1')
    const WETH9 = await deployWETH9(hre, signer, true)
    console.log(`Deployed to ${WETH9.address}`)

    console.log('Creating L2 WETH9')
    const OptimismMintableERC20 = await createOptimismMintableERC20(
      hre,
      WETH9,
      l2Signer
    )

    console.log(`Approving WETH9 for deposit`)
    const approvalTx = await messenger.approveERC20(
      WETH9.address,
      OptimismMintableERC20.address,
      hre.ethers.constants.MaxUint256
    )
    await approvalTx.wait()
    console.log('WETH9 approved')

    console.log('Depositing WETH9 to L2')
    const depositTx = await messenger.depositERC20(
      WETH9.address,
      OptimismMintableERC20.address,
      utils.parseEther('1')
    )
    await depositTx.wait()
    console.log(`ERC20 deposited - ${depositTx.hash}`)

    console.log('Checking to make sure deposit was successful')
    // Deposit might get reorged, wait and also log for reorgs.
    let prevBlockHash: string = ''
    for (let i = 0; i < 12; i++) {
      const messageReceipt = await signer.provider!.getTransactionReceipt(
        depositTx.hash
      )
      if (messageReceipt.status !== 1) {
        console.log(`Deposit failed, retrying...`)
      }

      // Wait for stability, we want some amount of time after any reorg
      if (prevBlockHash !== '' && messageReceipt.blockHash !== prevBlockHash) {
        console.log(
          `Block hash changed from ${prevBlockHash} to ${messageReceipt.blockHash}`
        )
        i = 0
      } else if (prevBlockHash !== '') {
        console.log(`No reorg detected: ${i}`)
      }

      prevBlockHash = messageReceipt.blockHash
      await sleep(1000)
    }
    console.log(`Deposit confirmed`)

    const l2Balance = await OptimismMintableERC20.balanceOf(address)
    if (l2Balance.lt(utils.parseEther('1'))) {
      throw new Error(
        `bad deposit. recipient balance on L2: ${utils.formatEther(l2Balance)}`
      )
    }
    console.log(`Deposit success`)

    console.log('Starting withdrawal')
    const preBalance = await WETH9.balanceOf(signer.address)
    const withdraw = await messenger.withdrawERC20(
      WETH9.address,
      OptimismMintableERC20.address,
      utils.parseEther('1')
    )
    const withdrawalReceipt = await withdraw.wait()
    for (const log of withdrawalReceipt.logs) {
      switch (log.address) {
        case L2ToL1MessagePasser.address: {
          const parsed = L2ToL1MessagePasser.interface.parseLog(log)
          console.log(`Log ${parsed.name} from ${log.address}`)
          console.log(parsed.args)
          console.log()
          break
        }
        case L2StandardBridge.address: {
          const parsed = L2StandardBridge.interface.parseLog(log)
          console.log(`Log ${parsed.name} from ${log.address}`)
          console.log(parsed.args)
          console.log()
          break
        }
        case L2CrossDomainMessenger.address: {
          const parsed = L2CrossDomainMessenger.interface.parseLog(log)
          console.log(`Log ${parsed.name} from ${log.address}`)
          console.log(parsed.args)
          console.log()
          break
        }
        default: {
          console.log(`Unknown log from ${log.address} - ${log.topics[0]}`)
        }
      }
    }

    setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(withdraw)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
      const latest = await L2OutputOracle.latestBlockNumber()
      console.log(
        `Latest L2OutputOracle commitment number: ${latest.toString()}`
      )
      const tip = await signer.provider!.getBlockNumber()
      console.log(`L1 chain tip: ${tip.toString()}`)
    }, 3000)

    const now = Math.floor(Date.now() / 1000)

    console.log('Waiting for message to be able to be proved')
    await messenger.waitForMessageStatus(withdraw, MessageStatus.READY_TO_PROVE)

    console.log('Proving withdrawal...')
    const prove = await messenger.proveMessage(withdraw)
    const proveReceipt = await prove.wait()
    console.log(proveReceipt)
    if (proveReceipt.status !== 1) {
      throw new Error('Prove withdrawal transaction reverted')
    }

    console.log('Waiting for message to be able to be relayed')
    await messenger.waitForMessageStatus(
      withdraw,
      MessageStatus.READY_FOR_RELAY
    )

    console.log('Finalizing withdrawal...')
    // TODO: Update SDK to properly estimate gas
    const finalize = await messenger.finalizeMessage(withdraw, {
      overrides: { gasLimit: 500_000 },
    })
    const finalizeReceipt = await finalize.wait()
    console.log('finalizeReceipt:', finalizeReceipt)
    console.log(`Took ${Math.floor(Date.now() / 1000) - now} seconds`)

    for (const log of finalizeReceipt.logs) {
      switch (log.address) {
        case OptimismPortal.address: {
          const parsed = OptimismPortal.interface.parseLog(log)
          console.log(`Log ${parsed.name} from OptimismPortal (${log.address})`)
          console.log(parsed.args)
          console.log()
          break
        }
        case L1CrossDomainMessenger.address: {
          const parsed = L1CrossDomainMessenger.interface.parseLog(log)
          console.log(
            `Log ${parsed.name} from L1CrossDomainMessenger (${log.address})`
          )
          console.log(parsed.args)
          console.log()
          break
        }
        case L1StandardBridge.address: {
          const parsed = L1StandardBridge.interface.parseLog(log)
          console.log(
            `Log ${parsed.name} from L1StandardBridge (${log.address})`
          )
          console.log(parsed.args)
          console.log()
          break
        }
        case WETH9.address: {
          const parsed = WETH9.interface.parseLog(log)
          console.log(`Log ${parsed.name} from WETH9 (${log.address})`)
          console.log(parsed.args)
          console.log()
          break
        }
        default:
          console.log(
            `Unknown log emitted from ${log.address} - ${log.topics[0]}`
          )
      }
    }

    const postBalance = await WETH9.balanceOf(signer.address)

    const expectedBalance = preBalance.add(utils.parseEther('1'))
    if (!expectedBalance.eq(postBalance)) {
      throw new Error(
        `Balance mismatch, expected: ${expectedBalance}, actual: ${postBalance}`
      )
    }
    console.log('Withdrawal success')
  })
