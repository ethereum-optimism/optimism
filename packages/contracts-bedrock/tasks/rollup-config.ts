import fs from 'fs'

import { task } from 'hardhat/config'
import { OpNodeConfig, getChainId } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'

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

    const l1Genesis = await l1.getBlock('earliest')
    const l2Genesis = await l2.getBlock('earliest')

    const portal = await hre.deployments.get('OptimismPortalProxy')

    const config: OpNodeConfig = {
      genesis: {
        l1: {
          hash: l1Genesis.hash,
          number: 0,
        },
        l2: {
          hash: l2Genesis.hash,
          number: 0,
        },
        l2_time: deployConfig.startingTimestamp,
      },
      block_time: deployConfig.l2BlockTime,
      max_sequencer_drift: deployConfig.maxSequencerDrift,
      seq_window_size: deployConfig.sequencerWindowSize,

      l1_chain_id: await getChainId(l1),
      l2_chain_id: await getChainId(l2),

      p2p_sequencer_address: '0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc',
      fee_recipient_address: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
      batch_inbox_address: '0xff00000000000000000000000000000000000002',
      batch_sender_address: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC',
      deposit_contract_address: portal.address,
    }

    fs.writeFileSync(args.outfile, JSON.stringify(config, null, 2))
  })
