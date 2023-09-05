import { task, types } from 'hardhat/config'
import { OpNodeProvider } from '@eth-optimism/core-utils'

// TODO(tynes): add in config validation
task('check-op-node', 'Validate the config of the op-node')
  .addParam(
    'opNodeUrl',
    'URL of the OP Node.',
    'http://localhost:7545',
    types.string
  )
  .setAction(async (args) => {
    const provider = new OpNodeProvider(args.opNodeUrl)

    const syncStatus = await provider.syncStatus()
    console.log(JSON.stringify(syncStatus, null, 2))

    const config = await provider.rollupConfig()
    console.log(JSON.stringify(config, null, 2))
  })
