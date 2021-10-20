let forkUrl = process.env.FORK_URL;
if (typeof forkUrl === 'undefined') {
  console.log("FORK_URL env missing and required")
  process.exit(1)
}

// TODO check that FORK_BLOCK_NUMBER is a Number
let forkBlockumber = process.env.FORK_BLOCK_NUMBER;
if (typeof forkBlockumber === 'undefined') {
  console.log("FORK_BLOCK_NUMBER env missing and required")
  process.exit(1)
} // 27837134

module.exports = {
  networks: {
    hardhat: {
      forking: {
        url: forkUrl,
        blockNumber: Number(forkBlockumber),
      }
    },
  },
  analytics: { enabled: false },
}
