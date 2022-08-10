import fs from 'fs'

import { task } from 'hardhat/config'
import { OpNodeConfig, getChainId } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'
import 'hardhat-deploy'
import '@eth-optimism/hardhat-deploy-config'

task('rollup-config', 'create a genesis config')
  .addOptionalParam(
    'outfile',
    'The file to write the output JSON to',
    'rollup.json'
  )
  .addOptionalParam('l1RpcUrl', 'The L1 RPC URL', 'http://127.0.0.1:8545')
  .addOptionalParam('l2RpcUrl', 'The L2 RPC URL', 'http://127.0.0.1:9545')
  .setAction(async (args, hre) => {
    const { deployConfig } = hre

    const l1 = new ethers.providers.StaticJsonRpcProvider(args.l1RpcUrl)
    const l2 = new ethers.providers.StaticJsonRpcProvider(args.l2RpcUrl)

    // sanity check our RPC connections
    const l1ChainID = await getChainId(l1)
    if (l1ChainID !== deployConfig.l1ChainID) {
      throw new Error(
        `connected to L1 RPC ${args.l1RpcUrl} yielded chain ID ${l1ChainID} but expected ${deployConfig.l1ChainID}`
      )
    }
    const l2ChainID = await getChainId(l2)
    if (l2ChainID !== deployConfig.l2ChainID) {
      throw new Error(
        `connected to L2 RPC ${args.l2RpcUrl} yielded chain ID ${l2ChainID} but expected ${deployConfig.l2ChainID}`
      )
    }

    const l2GenesisBlock = await l2.getBlock('earliest')

    const portal = await hre.deployments.get('OptimismPortalProxy')
    const l1StartingBlock = await l1.getBlock(deployConfig.l1StartingBlockTag)
    if (l1StartingBlock === null) {
      throw new Error(
        `Cannot fetch block tag ${deployConfig.l1StartingBlockTag}`
      )
    }

    const config: OpNodeConfig = {
      genesis: {
        l1: {
          hash: l1StartingBlock.hash,
          number: l1StartingBlock.number,
        },
        l2: {
          hash: l2GenesisBlock.hash,
          number: l2GenesisBlock.number,
        },
        l2_time: l1StartingBlock.timestamp,
      },
      block_time: deployConfig.l2BlockTime,
      max_sequencer_drift: deployConfig.maxSequencerDrift,
      seq_window_size: deployConfig.sequencerWindowSize,
      channel_timeout: deployConfig.channelTimeout,

      l1_chain_id: deployConfig.l1ChainID,
      l2_chain_id: deployConfig.l2ChainID,

      p2p_sequencer_address: deployConfig.p2pSequencerAddress,
      fee_recipient_address: deployConfig.optimismL2FeeRecipient,
      batch_inbox_address: deployConfig.batchInboxAddress,
      batch_sender_address: deployConfig.batchSenderAddress,
      deposit_contract_address: portal.address,
    }

    fs.writeFileSync(args.outfile, JSON.stringify(config, null, 2))
  })
