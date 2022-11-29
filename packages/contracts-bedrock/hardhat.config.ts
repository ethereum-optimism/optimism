import { ethers } from 'ethers'
import { HardhatUserConfig } from 'hardhat/config'

// Hardhat plugins
import '@eth-optimism/hardhat-deploy-config'
import '@foundry-rs/hardhat-forge'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'

// Hardhat tasks
import './tasks'

let bytecodeHash = 'none'
if (process.env.FOUNDRY_PROFILE === 'echidna') {
  bytecodeHash = 'ipfs'
}

const config: HardhatUserConfig = {
  networks: {
    hardhat: {
      live: false,
    },
    devnetL1: {
      live: false,
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
    hivenet: {
      chainId: Number(process.env.CHAIN_ID),
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    goerli: {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    'alpha-1': {
      chainId: 5,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
    },
    deployer: {
      chainId: Number(process.env.CHAIN_ID),
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: process.env.VERIFY_CONTRACTS === 'true',
    },
    'mainnet-forked': {
      chainId: 1,
      url: process.env.L1_RPC || '',
      accounts: [process.env.PRIVATE_KEY_DEPLOYER || ethers.constants.HashZero],
      live: false,
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
  namedAccounts: {
    deployer: {
      default: 0,
    },
  },
  deployConfigSpec: {
    // Address of the L1 proxy admin owner.
    finalSystemOwner: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },

    // Address of the system controller.
    controller: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },

    // To anchor the rollup at for L1 genesis.
    // The L2 genesis script uses this to fill the storage of the L1Block info predeploy.
    // The rollup config script uses this to fill the L1 genesis info for the rollup.
    // The Output oracle deploy script may use it if the L2 starting timestamp is undefined,
    // assuming the L2 genesis is set up with this.
    l1StartingBlockTag: {
      type: 'string',
    },

    // Required to identify the L1 network and verify and create L1 signatures.
    // Part of L1 genesis config.
    // "l1_chain_id" in rollup config.
    l1ChainID: {
      type: 'number',
    },

    // Required to identify the L2 network and create p2p signatures unique for this chain.
    // Part of L2 genesis config.
    // "l2_chain_id" in rollup config.
    l2ChainID: {
      type: 'number',
    },

    // Seconds per L2 block.
    //
    // The Output oracle deploy script uses this.
    //
    // "block_time" in rollup config.
    l2BlockTime: {
      type: 'number',
    },

    // Rollup config parameters. These must ONLY be used by the rollup config script.
    // For scripts, use the optimism_rollupConfig RPC method to retrieve the rollup config dynamically.
    // -------------------------------------------------

    // Sequencer batches may not be more than MaxSequencerDrift seconds after
    // the L1 timestamp of the sequencing window end.
    //
    // Note: When L1 has many 1 second consecutive blocks, and L2 grows at fixed 2 seconds,
    // the L2 time may still grow beyond this difference.
    //
    // "max_sequencer_drift" in rollup config.
    maxSequencerDrift: {
      type: 'number',
    },
    // Number of epochs (L1 blocks) per sequencing window.
    // "seq_window_size" in rollup config.
    sequencerWindowSize: {
      type: 'number',
    },
    // Number of seconds (w.r.t. L1 time) that a frame can be valid when included in L1
    // "channel_timeout" in rollup config.
    channelTimeout: {
      type: 'number',
    },
    // Address of the key the sequencer uses to sign blocks on the P2P layer
    // "p2p_sequencer_address" in rollup config.
    p2pSequencerAddress: {
      type: 'address',
    },
    // L1 address that batches are sent to.
    // "batch_inbox_address" in rollup config.
    batchInboxAddress: {
      type: 'address',
    },
    // Acceptable batch-sender address, to filter transactions going into the batchInboxAddress on L1 for data.
    // Warning: this address is hardcoded now, but is intended to become governed via L1.
    // It may not be part of the rollup config in the near future, and instead be part of a L1 contract deployment.
    // "batch_sender_address" in rollup config.
    batchSenderAddress: {
      type: 'address',
    },
    // L1 Deposit Contract Address. Not part of the deploy config.
    // This is derived from the Portal contract deployment (warning: use proxy address).
    // "deposit_contract_address" in the rollup config.

    // L2 Output oracle deployment parameters.
    // -------------------------------------------------

    // uint256 - Interval in blocks at which checkpoints must be submitted.
    l2OutputOracleSubmissionInterval: {
      type: 'number',
    },
    // uint256 - The number of the first L2 block.
    l2OutputOracleStartingBlockNumber: {
      type: 'number',
      default: 0,
    },
    // Starting time stamp is optional, if it is configured with a negative
    // the deploy config user needs to fall back to the timestamp
    // of the L1 block that the rollup anchors at (genesis L1).
    //
    // Note that if you let it fall back to this L1 timestamp, then the L2
    // must have a matching timestamp in the block at height l2OutputOracleStartingBlockNumber.
    //
    // uint256 - The timestamp of the first L2 block.
    l2OutputOracleStartingTimestamp: {
      type: 'number',
    },

    // l2OutputOracleL2BlockTime:
    // Read from the global l2BlockTime
    //
    // uint256 - The time per L2 block, in seconds.

    // address - The address of the proposer.
    l2OutputOracleProposer: {
      type: 'address',
    },
    // address - The address of the owner.
    l2OutputOracleChallenger: {
      type: 'address',
    },

    // uint256 - Finalization period in seconds.
    finalizationPeriodSeconds: {
      type: 'number',
      default: 2,
    },

    systemConfigOwner: {
      type: 'address',
    },

    // Optional L1 genesis block values. These must ONLY be used by the L1 genesis config script.
    // Not all deployments may create a new L1 chain, but instead attach to an existing L1 chain, like Goerli.
    // -------------------------------------------------

    l1BlockTime: {
      type: 'number',
      default: 15,
    },
    l1GenesisBlockNonce: {
      type: 'string', // uint64
      default: '0x0',
    },
    // l1GenesisBlockTimestamp: not part of deploy config, configured with parameter to genesis task instead.
    // l1GenesisBlockExtraData: not configurable, used for clique singer data. See cliqueSignerAddress instead.
    cliqueSignerAddress: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    l1GenesisBlockGasLimit: {
      type: 'string',
      default: ethers.BigNumber.from(15_000_000).toHexString(),
    },
    l1GenesisBlockDifficulty: {
      type: 'string', // uint256
      default: '0x1',
    },
    l1GenesisBlockMixHash: {
      type: 'string', // bytes32
      default: ethers.constants.HashZero,
    },
    l1GenesisBlockCoinbase: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    // l1GenesisBlockAlloc: the storage tree is not configurable with deploy-config.
    l1GenesisBlockNumber: {
      type: 'string', // uint64
      default: '0x0',
    },
    l1GenesisBlockGasUsed: {
      type: 'string', // uint64
      default: '0x0',
    },
    l1GenesisBlockParentHash: {
      type: 'string', // bytes32
      default: ethers.constants.HashZero,
    },
    l1GenesisBlockBaseFeePerGas: {
      type: 'string', // uint256
      default: ethers.BigNumber.from(1000_000_000).toHexString(), // 1 gwei
    },

    // Optional L2 genesis block values. These must ONLY be used by the L2 genesis config script.
    // -------------------------------------------------
    l2GenesisBlockNonce: {
      type: 'string', // uint64
      default: '0x0',
    },
    // l2GenesisBlockTimestamp: configured dynamically, based on the timestamp of l1StartingBlockTag.
    l2GenesisBlockExtraData: {
      type: 'string', // important: in the case of L2, which uses post-Merge Ethereum rules, this must be <= 32 bytes.
      default: ethers.constants.HashZero,
    },
    l2GenesisBlockGasLimit: {
      type: 'string',
      default: ethers.BigNumber.from(15_000_000).toHexString(),
    },
    l2GenesisBlockDifficulty: {
      type: 'string', // uint256
      default: '0x1',
    },
    l2GenesisBlockMixHash: {
      type: 'string', // bytes32
      default: ethers.constants.HashZero,
    },
    l2GenesisBlockCoinbase: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    // l2GenesisBlockAlloc: the storage tree is not configurable with deploy-config.
    l2GenesisBlockNumber: {
      type: 'string', // uint64
      default: '0x0',
    },
    l2GenesisBlockGasUsed: {
      type: 'string', // uint64
      default: '0x0',
    },
    l2GenesisBlockParentHash: {
      type: 'string', // bytes32
      default: ethers.constants.HashZero,
    },
    l2GenesisBlockBaseFeePerGas: {
      type: 'string', // uint256
      default: ethers.BigNumber.from(1000_000_000).toHexString(), // 1 gwei
    },
    // L2 chain configuration values
    optimismBaseFeeRecipient: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    optimismL1FeeRecipient: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    // L2 predeploy variables
    l2CrossDomainMessengerOwner: {
      type: 'address',
      default: ethers.constants.AddressZero,
    },
    gasPriceOracleOverhead: {
      type: 'number',
      default: 2100,
    },
    gasPriceOracleScalar: {
      type: 'number',
      default: 1_000_000,
    },
    gasPriceOracleDecimals: {
      type: 'number',
      default: 6,
    },
    numDeployConfirmations: {
      type: 'number',
      default: 1,
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
      mainnet: [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
      'mainnet-forked': [
        '../contracts/deployments/mainnet',
        '../contracts-periphery/deployments/mainnet',
      ],
    },
  },
  solidity: {
    compilers: [
      {
        version: '0.8.15',
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
        bytecodeHash,
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
