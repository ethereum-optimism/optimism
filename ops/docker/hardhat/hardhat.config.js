const isForkModeEnabled = !!process.env.FORK_URL
const forkUrl = process.env.FORK_URL
const forkStartingBlock = parseInt(process.env.FORK_STARTING_BLOCK) || undefined
const gasPrice = parseInt(process.env.GAS_PRICE) || 0

const config = {
  networks: {
    hardhat: {
      gasPrice,
      initialBaseFeePerGas: 0
    },
  },
  analytics: { enabled: false },
}

if (isForkModeEnabled) {
  console.log(`Running hardhat in a fork mode! URL: ${forkUrl}`)
  if (forkStartingBlock) {
    console.log(`Starting block: ${forkStartingBlock}`)
  }
  config.networks.hardhat.forking = {
    url: forkUrl,
    blockNumber: forkStartingBlock,
  }
} else {
  console.log('Running with a fresh state...')
}

module.exports = config
