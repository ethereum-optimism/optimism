const { 
  Contract, 
  Wallet, 
  ContractFactory, 
  BigNumber, 
  providers 
} = require('ethers');
require('dotenv').config();

const Provider = new providers.JsonRpcProvider(process.env.L2_NODE_WEB3_URL || "http://localhost:8545");
const bob = new Wallet(process.env.TEST_PRIVATE_KEY_1, Provider);
const alice = new Wallet(process.env.TEST_PRIVATE_KEY_2, Provider);
const carol = new Wallet(process.env.TEST_PRIVATE_KEY_3, Provider);
const dev = new Wallet(process.env.TEST_PRIVATE_KEY_4, Provider);
const minter = new Wallet(process.env.TEST_PRIVATE_KEY_5, Provider);

module.exports = {
  bob,
  alice,
  carol,
  dev, 
  minter,
  Provider,
}