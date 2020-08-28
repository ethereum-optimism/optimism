import path from 'path'
import { usePlugin, task } from '@nomiclabs/buidler/config'

usePlugin('@nomiclabs/buidler-ethers')
usePlugin('@nomiclabs/buidler-waffle')

import './src/buidler-plugins/buidler-ovm-compiler'
import './src/buidler-plugins/buidler-ovm-node'

task('test')
  .addFlag('ovm', 'Run tests on the OVM using a custom OVM provider')
  .setAction(async (taskArguments, bre: any, runSuper) => {
    if (taskArguments.ovm) {
      console.log('Compiling and running tests in the OVM...')
      bre.config.solc = {
        path: path.resolve(__dirname, '../../node_modules/@eth-optimism/solc'),
      }
      await bre.config.startOvmNode()
    }
    await runSuper(taskArguments)
  })

const config: any = {
  networks: {
    buidlerevm: {
      blockGasLimit: 100_000_000,
    },
  },
  paths: {
    sources: './test/common/contracts',
    tests: './test/test-buidler',
    cache: './test/temp/build/buidler/cache',
    artifacts: './test/temp/build/buidler/artifacts',
  },
  mocha: {
    timeout: 50000,
  },
  solc: {
    version: '0.5.16',
  },
}

export default config
