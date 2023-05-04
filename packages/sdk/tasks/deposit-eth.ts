import { promises as fs } from 'fs'

import { task, types } from 'hardhat/config'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import { Deployment } from 'hardhat-deploy/types'
import {
  predeploys,
  getContractDefinition,
} from '@eth-optimism/contracts-bedrock'
import { providers, utils, ethers } from 'ethers'

import {
  CrossChainMessenger,
  MessageStatus,
  CONTRACT_ADDRESSES,
  OEContractsLike,
  DEFAULT_L2_CONTRACT_ADDRESSES,
} from '../src'

const { formatEther } = utils

task('deposit-eth', 'Deposits ether to L2.')
  .addParam(
    'l2ProviderUrl',
    'L2 provider URL.',
    'http://localhost:9545',
    types.string
  )
  .addOptionalParam('to', 'Recipient of the ether', '', types.string)
  .addOptionalParam(
    'amount',
    'Amount of ether to send (in ETH)',
    '',
    types.string
  )
  .addOptionalParam(
    'withdraw',
    'Follow up with a withdrawal',
    true,
    types.boolean
  )
  .addOptionalParam(
    'l1ContractsJsonPath',
    'Path to a JSON with L1 contract addresses in it',
    '',
    types.string
  )
  .addOptionalParam('withdrawAmount', 'Amount to withdraw', '', types.string)
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
    console.log(`Signer balance: ${formatEther(balance.toString())}`)

    const l2Provider = new providers.StaticJsonRpcProvider(args.l2ProviderUrl)

    // send to self if not specified
    const to = args.to ? args.to : address
    const amount = args.amount
      ? utils.parseEther(args.amount)
      : utils.parseEther('1')
    const withdrawAmount = args.withdrawAmount
      ? utils.parseEther(args.withdrawAmount)
      : amount.div(2)

    const l2Signer = new hre.ethers.Wallet(
      hre.network.config.accounts[0],
      l2Provider
    )

    const l2ChainId = await l2Signer.getChainId()
    let contractAddrs = CONTRACT_ADDRESSES[l2ChainId]
    if (args.l1ContractsJsonPath) {
      const data = await fs.readFile(args.l1ContractsJsonPath)
      contractAddrs = {
        l1: JSON.parse(data.toString()),
        l2: DEFAULT_L2_CONTRACT_ADDRESSES,
      } as OEContractsLike
    } else if (!contractAddrs) {
      // If the contract addresses have not been hardcoded,
      // attempt to read them from deployment artifacts
      let Deployment__AddressManager: Deployment
      try {
        Deployment__AddressManager = await hre.deployments.get('AddressManager')
      } catch (e) {
        Deployment__AddressManager = await hre.deployments.get(
          'Lib_AddressManager'
        )
      }
      let Deployment__L1CrossDomainMessenger: Deployment
      try {
        Deployment__L1CrossDomainMessenger = await hre.deployments.get(
          'L1CrossDomainMessengerProxy'
        )
      } catch (e) {
        Deployment__L1CrossDomainMessenger = await hre.deployments.get(
          'Proxy__OVM_L1CrossDomainMessenger'
        )
      }
      let Deployment__L1StandardBridge: Deployment
      try {
        Deployment__L1StandardBridge = await hre.deployments.get(
          'L1StandardBridgeProxy'
        )
      } catch (e) {
        Deployment__L1StandardBridge = await hre.deployments.get(
          'Proxy__OVM_L1StandardBridge'
        )
      }

      const Deployment__OptimismPortal = await hre.deployments.get(
        'OptimismPortalProxy'
      )
      const Deployment__L2OutputOracle = await hre.deployments.get(
        'L2OutputOracleProxy'
      )
      contractAddrs = {
        l1: {
          AddressManager: Deployment__AddressManager.address,
          L1CrossDomainMessenger: Deployment__L1CrossDomainMessenger.address,
          L1StandardBridge: Deployment__L1StandardBridge.address,
          StateCommitmentChain: ethers.constants.AddressZero,
          CanonicalTransactionChain: ethers.constants.AddressZero,
          BondManager: ethers.constants.AddressZero,
          OptimismPortal: Deployment__OptimismPortal.address,
          L2OutputOracle: Deployment__L2OutputOracle.address,
        },
        l2: DEFAULT_L2_CONTRACT_ADDRESSES,
      }
    }

    const Artifact__L2ToL1MessagePasser = await getContractDefinition(
      'L2ToL1MessagePasser'
    )

    const Artifact__L2CrossDomainMessenger = await getContractDefinition(
      'L2CrossDomainMessenger'
    )

    const Artifact__L2StandardBridge = await getContractDefinition(
      'L2StandardBridge'
    )

    const Artifact__OptimismPortal = await getContractDefinition(
      'OptimismPortal'
    )

    const Artifact__L1CrossDomainMessenger = await getContractDefinition(
      'L1CrossDomainMessenger'
    )

    const Artifact__L1StandardBridge = await getContractDefinition(
      'L1StandardBridge'
    )

    const OptimismPortal = new hre.ethers.Contract(
      contractAddrs.l1.OptimismPortal,
      Artifact__OptimismPortal.abi,
      signer
    )

    const L1CrossDomainMessenger = new hre.ethers.Contract(
      contractAddrs.l1.L1CrossDomainMessenger,
      Artifact__L1CrossDomainMessenger.abi,
      signer
    )

    const L1StandardBridge = new hre.ethers.Contract(
      contractAddrs.l1.L1StandardBridge,
      Artifact__L1StandardBridge.abi,
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

    const opBalanceBefore = await signer.provider.getBalance(
      OptimismPortal.address
    )

    const l1BridgeBalanceBefore = await signer.provider.getBalance(
      L1StandardBridge.address
    )

    // Deposit ETH
    console.log('Depositing ETH through StandardBridge')
    console.log(`Sending ${formatEther(amount)} ether`)
    const ethDeposit = await messenger.depositETH(amount, { recipient: to })
    console.log(`Transaction hash: ${ethDeposit.hash}`)
    const depositMessageReceipt = await messenger.waitForMessageReceipt(
      ethDeposit
    )
    if (depositMessageReceipt.receiptStatus !== 1) {
      throw new Error('deposit failed')
    }
    console.log(
      `Deposit complete - included in block ${depositMessageReceipt.transactionReceipt.blockNumber}`
    )

    const opBalanceAfter = await signer.provider.getBalance(
      OptimismPortal.address
    )

    const l1BridgeBalanceAfter = await signer.provider.getBalance(
      L1StandardBridge.address
    )

    console.log(
      `L1StandardBridge balance before: ${formatEther(l1BridgeBalanceBefore)}`
    )

    console.log(
      `L1StandardBridge balance after: ${formatEther(l1BridgeBalanceAfter)}`
    )

    console.log(
      `OptimismPortal balance before: ${formatEther(opBalanceBefore)}`
    )
    console.log(`OptimismPortal balance after: ${formatEther(opBalanceAfter)}`)

    if (!opBalanceBefore.add(amount).eq(opBalanceAfter)) {
      throw new Error(`OptimismPortal balance mismatch`)
    }

    const l2Balance = await l2Provider.getBalance(to)
    console.log(
      `L2 balance of deposit recipient: ${utils.formatEther(
        l2Balance.toString()
      )}`
    )

    if (!args.withdraw) {
      return
    }

    console.log('Withdrawing ETH')
    const ethWithdraw = await messenger.withdrawETH(withdrawAmount)
    console.log(`Transaction hash: ${ethWithdraw.hash}`)
    const ethWithdrawReceipt = await ethWithdraw.wait()
    console.log(
      `ETH withdrawn on L2 - included in block ${ethWithdrawReceipt.blockNumber}`
    )

    {
      // check the logs
      for (const log of ethWithdrawReceipt.logs) {
        switch (log.address) {
          case L2ToL1MessagePasser.address: {
            const parsed = L2ToL1MessagePasser.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            break
          }
          case L2StandardBridge.address: {
            const parsed = L2StandardBridge.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            break
          }
          case L2CrossDomainMessenger.address: {
            const parsed = L2CrossDomainMessenger.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            break
          }
          default: {
            console.log(`Unknown log from ${log.address} - ${log.topics[0]}`)
          }
        }
      }
    }

    console.log('Waiting to be able to prove withdrawal')

    const proveInterval = setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(ethWithdrawReceipt)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
    }, 3000)

    try {
      await messenger.waitForMessageStatus(
        ethWithdrawReceipt,
        MessageStatus.READY_TO_PROVE
      )
    } finally {
      clearInterval(proveInterval)
    }

    console.log('Proving eth withdrawal...')
    const ethProve = await messenger.proveMessage(ethWithdrawReceipt)
    console.log(`Transaction hash: ${ethProve.hash}`)
    const ethProveReceipt = await ethProve.wait()
    if (ethProveReceipt.status !== 1) {
      throw new Error('Prove withdrawal transaction reverted')
    }
    console.log('Successfully proved withdrawal')

    console.log('Waiting to be able to finalize withdrawal')

    const finalizeInterval = setInterval(async () => {
      const currentStatus = await messenger.getMessageStatus(ethWithdrawReceipt)
      console.log(`Message status: ${MessageStatus[currentStatus]}`)
    }, 3000)

    try {
      await messenger.waitForMessageStatus(
        ethWithdrawReceipt,
        MessageStatus.READY_FOR_RELAY
      )
    } finally {
      clearInterval(finalizeInterval)
    }

    console.log('Finalizing eth withdrawal...')
    const ethFinalize = await messenger.finalizeMessage(ethWithdrawReceipt)
    console.log(`Transaction hash: ${ethFinalize.hash}`)
    const ethFinalizeReceipt = await ethFinalize.wait()
    if (ethFinalizeReceipt.status !== 1) {
      throw new Error('Finalize withdrawal reverted')
    }

    console.log(
      `ETH withdrawal complete - included in block ${ethFinalizeReceipt.blockNumber}`
    )
    {
      // Check that the logs are correct
      for (const log of ethFinalizeReceipt.logs) {
        switch (log.address) {
          case L1StandardBridge.address: {
            const parsed = L1StandardBridge.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            if (
              parsed.name !== 'ETHBridgeFinalized' &&
              parsed.name !== 'ETHWithdrawalFinalized'
            ) {
              throw new Error('Wrong event name from L1StandardBridge')
            }
            if (!parsed.args.amount.eq(withdrawAmount)) {
              throw new Error('Wrong amount in event')
            }
            if (parsed.args.from !== address) {
              throw new Error('Wrong to in event')
            }
            if (parsed.args.to !== address) {
              throw new Error('Wrong from in event')
            }
            break
          }
          case L1CrossDomainMessenger.address: {
            const parsed = L1CrossDomainMessenger.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            if (parsed.name !== 'RelayedMessage') {
              throw new Error('Wrong event from L1CrossDomainMessenger')
            }
            break
          }
          case OptimismPortal.address: {
            const parsed = OptimismPortal.interface.parseLog(log)
            console.log(parsed.name)
            console.log(parsed.args)
            console.log()
            // TODO: remove this if check
            if (parsed.name === 'WithdrawalFinalized') {
              if (parsed.args.success !== true) {
                throw new Error('Unsuccessful withdrawal call')
              }
            }
            break
          }
          default: {
            console.log(`Unknown log from ${log.address} - ${log.topics[0]}`)
          }
        }
      }
    }

    const opBalanceFinally = await signer.provider.getBalance(
      OptimismPortal.address
    )

    if (!opBalanceFinally.add(withdrawAmount).eq(opBalanceAfter)) {
      throw new Error('OptimismPortal balance mismatch')
    }
    console.log('Withdraw success')
  })
