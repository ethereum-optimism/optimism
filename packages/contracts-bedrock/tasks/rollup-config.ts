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

    const l2Genesis = await l2.getBlock('earliest')

    const portal = await hre.deployments.get('OptimismPortalProxy')
    const l1StartingBlock = await l1.getBlock(portal.receipt.blockHash)

    const config: OpNodeConfig = {
      genesis: {
        l1: {
          hash: portal.receipt.blockHash,
          number: portal.receipt.blockNumber,
        },
        l2: {
          hash: l2Genesis.hash,
          number: 0,
        },
        l2_time: l1StartingBlock.timestamp,
      },
      block_time: deployConfig.l2BlockTime,
      max_sequencer_drift: deployConfig.maxSequencerDrift,
      seq_window_size: deployConfig.sequencerWindowSize,
      channel_timeout: deployConfig.channelTimeout,

      l1_chain_id: await getChainId(l1),
      l2_chain_id: await getChainId(l2),

      p2p_sequencer_address: deployConfig.p2pSequencerAddress,
      fee_recipient_address: deployConfig.optimismL2FeeRecipient,
      batch_inbox_address: '0xff00000000000000000000000000000000000002',
      batch_sender_address: deployConfig.batchSenderAddress,
      deposit_contract_address: portal.address,
    }

    fs.writeFileSync(args.outfile, JSON.stringify(config, null, 2))
  })
