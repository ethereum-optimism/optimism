import { promises as fs } from 'fs'

import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { Contract, providers, utils, ethers } from 'ethers'
import { predeploys, sleep } from '@eth-optimism/core-utils'

import Artifact__L2ToL1MessagePasser from '../src/forge-artifacts/L2ToL1MessagePasser.json'
import Artifact__L2CrossDomainMessenger from '../src/forge-artifacts/L2CrossDomainMessenger.json'
import Artifact__L2StandardBridge from '../src/forge-artifacts/L2StandardBridge.json'
import Artifact__OptimismPortal from '../src/forge-artifacts/OptimismPortal.json'
import Artifact__L1CrossDomainMessenger from '../src/forge-artifacts/L1CrossDomainMessenger.json'
import Artifact__L1StandardBridge from '../src/forge-artifacts/L1StandardBridge.json'
import Artifact__L2OutputOracle from '../src/forge-artifacts/L2OutputOracle.json'
import Artifact_BOBA from '../src/forge-artifacts/BOBA.json'
import {
  CrossChainMessenger,
  MessageStatus,
  CONTRACT_ADDRESSES,
  OEContractsLike,
  DEFAULT_L2_CONTRACT_ADDRESSES,
} from '../src'

// Bridge a pre-deployed ERC20 from L1 to L2
task('deposit-boba', 'Deposits BOBA onto L2.')
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
  .setAction(async (args, hre) => {
    const signers = await hre.ethers.getSigners()
    if (signers.length === 0) {
      throw new Error('No configured signers')
    }
    // Use the first configured signer for simplicity
    const signer = signers[0]
    const address = await signer.getAddress()
    console.log(`Using signer ${address}`)

    // Ensure that the signer has a balance before trying to
    // do anything
    const balance = await signer.getBalance()
    if (balance.eq(0)) {
      throw new Error('Signer has no balance')
    }

    const l1Provider = new providers.StaticJsonRpcProvider(args.l1ProviderUrl)

    const l1Signer = new hre.ethers.Wallet(
      hre.network.config.accounts[0],
      l1Provider
    )

    const l2Provider = new providers.StaticJsonRpcProvider(args.l2ProviderUrl)

    const l2Signer = new hre.ethers.Wallet(
      hre.network.config.accounts[0],
      l2Provider
    )

    const l2ChainId = await l2Signer.getChainId()
    let contractAddrs = CONTRACT_ADDRESSES[l2ChainId]

    let l1BobaTokenAddress = ''
    const l2BobaTokenAddress = predeploys.L2BobaToken

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
          DisputeGameFactory: json?.DisputeGameFactoryProxy,
        },
        l2: DEFAULT_L2_CONTRACT_ADDRESSES,
      } as OEContractsLike

      l1BobaTokenAddress = json.BOBA
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

    const L1BobaToken = new Contract(
      l1BobaTokenAddress,
      Artifact_BOBA.abi,
      l1Signer
    )

    const L2BobaToken = new Contract(
      l2BobaTokenAddress,
      Artifact_BOBA.abi,
      l2Signer
    )

    console.log(`Approving BOBA for deposit`)
    const approvalTx = await messenger.approveERC20(
      L1BobaToken.address,
      L2BobaToken.address,
      hre.ethers.constants.MaxUint256
    )
    await approvalTx.wait()
    console.log('BOBA approved')

    console.log('Depositing BOBA to L2')
    const depositTx = await messenger.depositERC20(
      L1BobaToken.address,
      L2BobaToken.address,
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

    const l2Balance = await L2BobaToken.balanceOf(address)
    if (l2Balance.lt(utils.parseEther('1'))) {
      throw new Error(
        `bad deposit. recipient balance on L2: ${utils.formatEther(l2Balance)}`
      )
    }
    console.log(`Deposit success`)

    console.log('Starting withdrawal')
    const preBalance = await L1BobaToken.balanceOf(l1Signer.address)
    const withdraw = await messenger.withdrawERC20(
      L1BobaToken.address,
      L2BobaToken.address,
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
        case L1BobaToken.address: {
          const parsed = L1BobaToken.interface.parseLog(log)
          console.log(`Log ${parsed.name} from L1BobaToken (${log.address})`)
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

    const postBalance = await L1BobaToken.balanceOf(l1Signer.address)

    const expectedBalance = preBalance.add(utils.parseEther('1'))
    if (!expectedBalance.eq(postBalance)) {
      throw new Error(
        `Balance mismatch, expected: ${expectedBalance}, actual: ${postBalance}`
      )
    }
    console.log('Withdrawal success')
  })
