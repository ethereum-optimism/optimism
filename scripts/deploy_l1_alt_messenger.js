const ethers = require('ethers')
require('dotenv').config()

const l1Provider = new ethers.providers.JsonRpcProvider(process.env.L1_NODE_WEB3_URL)
const bobl1Wallet = new ethers.Wallet(process.env.TEST_PRIVATE_KEY_1,l1Provider)

const factory = (name, ovm = false) => {
const artifact = require(`../artifacts${ovm ? '-ovm' : ''}/contracts/${name}.sol/${name}.json`)
return new ethers.ContractFactory(artifact.abi, artifact.bytecode)
}
  
const factory__L1_Messenger = factory('OVM_L1CrossDomainMessenger')

async function main() {
console.log('Deploying ...')
const L1_Messenger= await factory__L1_Messenger.connect(bobl1Wallet).deploy(
)
await L1_Messenger.deployTransaction.wait()

console.log('Deployed the L1_Alt_Messenger to ' + L1_Messenger.address)


console.log('Initializing ...')
// initialize with address_manager
const tx0 = await L1_Messenger.initialize(
    process.env.ETH1_ADDRESS_RESOLVER_ADDRESS
)
await tx0.wait()
}
main()
  