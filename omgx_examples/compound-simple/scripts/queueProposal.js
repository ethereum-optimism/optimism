const {ethers} = require('ethers');

const Timelock = require('../build-ovm/Timelock.json');
const GovernorBravoDelegate = require('../build-ovm/GovernorBravoDelegate.json');
const GovernorBravoDelegator = require('../build-ovm/GovernorBravoDelegator.json');
const Comp = require('../build-ovm/Comp.json');

const addresses = require('../networks/addresses.json');
const BigNumber = require('bignumber.js');

require('dotenv').config();

const env = process.env

const compAddress = addresses.DAO_Comp
const timelockAddress = addresses.DAO_Timelock
const governorBravoDelegateAddress = addresses.DAO_GovernorBravoDelegate
const governorBravoDelegatorAddress = addresses.DAO_GovernorBravoDelegator

const gasSet = {
  gasPrice: 15000000,
  gasLimit: 8000000
}

const sleep = async (timeout) => {
	return new Promise((resolve, reject) => {
		setTimeout(() => {
			resolve()
		}, timeout)
	});
}

async function main(){

    const l2_provider = new ethers.providers.JsonRpcProvider(env.L2_NODE_WEB3_URL, { chainId: 28 })

    const wallet1 = new ethers.Wallet(env.pk_0, l2_provider)
    const wallet2 = new ethers.Wallet(env.pk_1, l2_provider)
    const wallet3 = new ethers.Wallet(env.pk_2, l2_provider)

    const governorBravoDelegate = new ethers.Contract(governorBravoDelegateAddress , GovernorBravoDelegate.abi , wallet1)
    const timelock = new ethers.Contract(timelockAddress, Timelock.abi, wallet1)

    const governorBravoDelegator = new ethers.Contract(governorBravoDelegatorAddress, GovernorBravoDelegator.abi, wallet1)

    const comp = new ethers.Contract(compAddress, Comp.abi, wallet1)

    const governorBravo = await governorBravoDelegate.attach(
        governorBravoDelegator.address
    );

    const proposalStates = [
        'Pending',
        'Active',
        'Canceled',
        'Defeated',
        'Succeeded',
        'Queued',
        'Expired',
        'Executed',
    ]

    const proposalID = (await governorBravo.proposalCount())._hex
    console.log(`Proposed. Proposal ID: ${proposalID}`)

    console.log(`Queuing Proposal`);
    for(let i= 0; i < 30; i++){
        console.log(`Attempt: ${i + 1}`)
        try{
            await governorBravo.queue(proposalID, gasSet)
            console.log('Success: Queued')
            break;
        }catch(error){
            if(i == 29){
                await governorBravo.cancel(proposalID, gasSet)
                console.log(`Proposal failed and has been canceled, please try again`);
                console.log(error);
                return;
            }
            console.log(`\tproposal can only be queued if it is succeeded`);
        }
        await sleep(15 * 1000)
    }

    let state = await governorBravo.state(proposalID)
    console.log('State is: ', proposalStates[state])
}

(async () =>{
    try {
        main();
    } catch (error) {
        console.log(error);
    }
})();