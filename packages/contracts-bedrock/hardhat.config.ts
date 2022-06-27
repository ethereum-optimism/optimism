import { ethers } from 'ethers'
import { HardhatUserConfig, task, subtask } from 'hardhat/config'
import { TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS } from 'hardhat/builtin-tasks/task-names'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@typechain/hardhat'
import 'solidity-coverage'
import 'hardhat-deploy'
import '@foundry-rs/hardhat-forge'
import '@eth-optimism/hardhat-deploy-config'

// Hardhat tasks
import './tasks/genesis-l1'
import './tasks/genesis-l2'
import './tasks/deposits'
import './tasks/rollup-config'

subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS).setAction(
  async (_, __, runSuper) => {
    const paths = await runSuper()

    return paths.filter((p: string) => !p.endsWith('.t.sol'))
  }
)

task('accounts', 'Prints the list of accounts', async (_, hre) => {
  const accounts = await hre.ethers.getSigners()

  for (const account of accounts) {
    console.log(account.address)
  }
})

const config: HardhatUserConfig = {
  networks: {
    devnetL1: {
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    goerli: {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
  },
  foundry: {
    buildInfo: true,
  },
  paths: {
    deploy: './deploy',
    deployments: './deployments',
    deployConfig: './deploy-config',
  },
  typechain: {
    outDir: 'dist/types',
    target: 'ethers-v5',
  },
  namedAccounts: {
    deployer: {
      default: 0,
    },
  },
  deployConfigSpec: {
    submissionInterval: {
      type: 'number',
    },
    l2BlockTime: {
      type: 'number',
    },
    genesisOutput: {
      type: 'string',
      default: ethers.constants.HashZero,
    },
    historicalBlocks: {
      type: 'number',
    },
    startingBlockNumber: {
      type: 'number',
    },
    startingTimestamp: {
      type: 'number',
    },
    sequencerAddress: {
      type: 'address',
    },
    ownerAddress: {
      type: 'address',
    },
  },
  external: {
    contracts: [
      {
        artifacts: '../contracts/artifacts',
      },
      {
        artifacts: '../contracts-governance/artifacts',
      },
    ],
    deployments: {
      goerli: ['../contracts/deployments/goerli'],
    },
  },
  solidity: {
    compilers: [
      {
        version: '0.8.10',
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
      {
        version: '0.5.17', // Required for WETH9
        settings: {
          optimizer: { enabled: true, runs: 10_000 },
        },
      },
    ],
    settings: {
      metadata: {
        bytecodeHash: 'none',
      },
      outputSelection: {
        '*': {
          '*': ['metadata', 'storageLayout'],
        },
      },
    },
  },
}

export default config
