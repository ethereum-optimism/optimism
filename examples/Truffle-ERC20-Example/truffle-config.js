module.exports = {
  contracts_build_directory: './build',
  networks: {
    ethereum: {
      network_id: 31337,
      host: '127.0.0.1',
      port: 9545,
      gasPrice: 0,
    },
  },
  // Configure your compilers
  compilers: {
    solc: {
      version: "0.7.6",    // Fetch exact version from solc-bin (default: truffle's version)
    }
  }
}
