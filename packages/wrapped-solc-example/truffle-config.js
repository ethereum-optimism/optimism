// Note, will need EXECUTION_MANAGER_ADDRESS environment variable set.
module.exports = {
  compilers: {
    solc: {
      // Add path to the solc-transpiler
      version: "../../node_modules/@eth-optimism/solc-transpiler",
    }
  }
}
