const { getContractFactory } = require('@eth-optimism/contracts')
require('dotenv').config()

async function main() {
console.log('Deploying ...')

const factory__L1_Messenger = await ethers.getContractFactory('OVM_L1CrossDomainMessengerFast')


const L1_Messenger= await factory__L1_Messenger.deploy()

console.log('Deployed the L1_CrossDomainMessenger_Fast to ' + L1_Messenger.address)

const L1_Messenger_Deployed = await factory__L1_Messenger.attach(L1_Messenger.address)

console.log('Initializing ...')
// initialize with address_manager
await L1_Messenger_Deployed.initialize(
    process.env.ADDRESS_MANAGER_ADDRESS
)

console.log('Fast L1 Messenger Initialized')

const [deployer] = await ethers.getSigners();

const myContract = getContractFactory(
  'Lib_AddressManager',
  deployer
)

const Lib_AddressManager = await myContract.attach(process.env.ADDRESS_MANAGER_ADDRESS)

// this will fail for non deployer account
console.log('Registering L1 Messenger...')
await Lib_AddressManager.setAddress(
  'OVM_L1CrossDomainMessengerFast',
  L1_Messenger.address
)

console.log('Fast L1 Messenger registered in AddressManager')
}
main()
.then(() => process.exit(0))
.catch(error => {
  console.error(error);
  process.exit(1);
});
  