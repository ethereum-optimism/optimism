/**
 * @type import('hardhat/config').HardhatUserConfig
 */

require("@nomiclabs/hardhat-ethers");
require("hardhat-gas-reporter");
const fs = require("fs")

// attempt to read private key
let private = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
try {
  private = fs.readFileSync(process.env.HOME+"/.privatekey").toString()
} catch {
}


module.exports = {
  defaultNetwork: "hosthat",
  networks: {
    hosthat: {
      url: "http://localhost:8545/",
      accounts: ["0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"]
    },
    cheapeth: {
      url: "https://rpc.cheapeth.org/rpc",
      accounts: [private]
    },
  },
  solidity: {
    version: "0.7.3",
    settings: {
      optimizer: {
        enabled: true,
        runs: 200
      }
    }
  },
};
