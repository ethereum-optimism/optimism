const {ethers} = require('ethers');
const Timelock = require('../build-ovm/Timelock.json');
const GovernorBravoDelegate = require('../build-ovm/GovernorBravoDelegate.json');
const GovernorBravoDelegator = require('../build-ovm/GovernorBravoDelegator.json');
const Comp = require('../build-ovm/Comp.json');
const addresses = require('../networks/rinkeby-boba.json');
require('dotenv').config();
const env = process.env;
const compAddress = addresses.Comp;
const timelockAddress = addresses.Timelock;
const governorBravoDelegateAddress = addresses.GovernorBravoDelegate;
const governorBravoDelegatorAddress = addresses.GovernorBravoDelegator;


const sleep = (timeout) => {
    return new Promise((resolve, reject) => {
      setTimeout(() => {
        resolve()
      }, timeout)
    })
  }

async function getTimestamp(web3URL, chainID){
    let provider = new ethers.providers.JsonRpcProvider(web3URL, { chainId: chainID });
    let blockNumber = await provider.getBlockNumber();
    let block = await provider.getBlock(blockNumber);
    return block.timestamp;
}

async function main(){

    const l2_provider = new ethers.providers.JsonRpcProvider(env.L2_NODE_WEB3_URL, { chainId: 28 })
    const wallet1 = new ethers.Wallet(env.pk_0, l2_provider)

    // const comp = new ethers.Contract(compAddress , Comp.abi , wallet1);
    const governorBravoDelegate = new ethers.Contract(governorBravoDelegateAddress , GovernorBravoDelegate.abi , wallet1)
    const timelock = new ethers.Contract(timelockAddress, Timelock.abi, wallet1)
    const governorBravoDelegator = new ethers.Contract(governorBravoDelegatorAddress, GovernorBravoDelegator.abi, wallet1)

    const governorBravo = await governorBravoDelegate.attach(
        governorBravoDelegator.address
    );

    var blockNumber = await l2_provider.getBlockNumber();
    var block = await l2_provider.getBlock(blockNumber);
    var eta = block.timestamp + 300; // the time at which the transaction can be executed
    var setPendingAdminData = ethers.utils.defaultAbiCoder.encode( // the parameters for the setPendingAdmin function
    ['address'],
    [governorBravoDelegator.address]
    );
    console.log("-----------Initiating Compound-----------\n");
    console.log("Current Time: ", block.timestamp);
    console.log("Time at which transaction can be executed:", eta);

    console.log(
    '\n\n\n-----------queueing setPendingAdmin-----------\n'
    );

    // Queue the transaction that will set the admin of Timelock to the GovernorBravoDelegator contract
    await timelock.queueTransaction(
      timelock.address,
      0,
      'setPendingAdmin(address)', // the function to be called
      setPendingAdminData,
      eta
    );
    console.log('queued setPendingAdmin');
    console.log('execute setPendingAdmin');



    await sleep(300 * 1000);
    for(let i = 0; i < 30; i++){
      console.log(`Attempt: ${i + 1}`)
      console.log(`\tTimestamp: ${await getTimestamp(env.L2_NODE_WEB3_URL, 28)}`);
      try{
        // Execute the transaction that will set the admin of Timelock to the GovernorBravoDelegator contract
        await timelock.executeTransaction(
          timelock.address,
          0,
          'setPendingAdmin(address)', // the function to be called
          setPendingAdminData,
          eta
        );
        console.log('\texecuted setPendingAdmin')
        break;
      }catch(error){
        if(i === 29){
          console.log("\tFailed. Please try again.\n");
          return;
        }
        // console.log(error);
        console.log("\tTransaction hasn't surpassed time lock\n");
      }
      await sleep(15 * 1000);
    }

    console.log(
    '\n\n\n-----------queueing initiate-----------\n'
    );

    blockNumber = await l2_provider.getBlockNumber();
    block = await l2_provider.getBlock(blockNumber);
    eta = block.timestamp + 300
    var initiateData = ethers.utils.defaultAbiCoder.encode( // parameters to initate the GovernorBravoDelegate contract
    ['bytes'],
    [[]]
    );

    console.log("Current Time: ", block.timestamp);
    console.log("Time at which transaction can be executed:", eta);


    // Queuing the transaction that will initate the GovernorBravoDelegate contract

    await timelock.queueTransaction(
    governorBravo.address,
    0,
    '_initiate()',
    initiateData,
    eta
    )
    console.log('queued initiate');
    console.log('execute initiate');
    for(let i = 0; i < 15; i++ ){
        await sleep(120 * 1000);
        console.log(`Timestamp: ${await getTimestamp(env.L2_NODE_WEB3_URL, 28)}`);
        try{
            await timelock.executeTransaction(
                governorBravo.address,
                0,
                '_initiate()',
                initiateData,
                eta
            )
            console.log('Executed initiate');
            break;
        }catch(error){
            console.log("\n\n\n-----FAILED-----\n\n\n");
            console.log(JSON.stringify(error));
            console.log("\n\n\n-----RETRYING-----\n\n\n");
        }
    }
}

(async () =>{
    try{
        await main();
    }catch(error){
        console.log(error);
    }
})();