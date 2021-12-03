const isForkModeEnabled = !!process.env.FORK_URL
const forkUrl = process.env.FORK_URL
const forkStartingBlock = process.env.FORK_STARTING_BLOCK || undefined

if (isForkModeEnabled) {
  console.log(`Running hardhat in a fork mode! URL: ${forkUrl}`)
  if (forkStartingBlock) {
    console.log(`Starting block: ${forkStartingBlock}`)
  }
} else {
  console.log('Running with a fresh state...')
}

module.exports = {
  networks: {
    hardhat: {
      gasPrice: 0,
      initialBaseFeePerGas: 0,
      ...(isForkModeEnabled
        ? {
            forking: {
              url: forkUrl,
              blockNumber: forkStartingBlock,
            },
          }
        : {}),
    },
  },
  analytics: { enabled: false },
}
