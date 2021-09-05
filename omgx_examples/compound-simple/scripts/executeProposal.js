const {ethers} = require('ethers');
const Timelock = require('../build-ovm/Timelock.json');
const GovernorBravoDelegate = require('../build-ovm/GovernorBravoDelegate.json');
const GovernorBravoDelegator = require('../build-ovm/GovernorBravoDelegator.json');
const Comp = require('../build-ovm/Comp.json');
const addresses = require('../networks/rinkeby-l2.json');
const BigNumber = require('bignumber.js');
require('dotenv').config();

const env = process.env;
const DECIMALS  = BigInt(10**18);

const compAddress = addresses.Comp;
const timelockAddress = addresses.Timelock;
const governorBravoDelegateAddress = addresses.GovernorBravoDelegate;
const governorBravoDelegatorAddress = addresses.GovernorBravoDelegator;


const sleep = async (timeout) => {
	return new Promise((resolve, reject) => {
		setTimeout(() => {
			resolve()
		}, timeout)
	});
}

async function getBlockNumber(web3url, chainID){
    const provider = new ethers.providers.JsonRpcProvider(web3url, {chainId: chainID});
    const blockNumber = await provider.getBlockNumber();
    return blockNumber;
}

async function main(){

    const l2_provider = new ethers.providers.JsonRpcProvider(env.L2_NODE_WEB3_URL, { chainId: 28 });

    const wallet1 = new ethers.Wallet(env.pk_0, l2_provider);
    const wallet2 = new ethers.Wallet(env.pk_1, l2_provider);
    const wallet3 = new ethers.Wallet(env.pk_2, l2_provider);

    const governorBravoDelegate = new ethers.Contract(governorBravoDelegateAddress , GovernorBravoDelegate.abi , wallet1);
    const timelock = new ethers.Contract(timelockAddress, Timelock.abi, wallet1);

    const governorBravoDelegator = new ethers.Contract(governorBravoDelegatorAddress, GovernorBravoDelegator.abi, wallet1);

    const comp = new ethers.Contract(compAddress, Comp.abi, wallet1);

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
    ];


    const proposalID = (await governorBravo.proposalCount())._hex;
    console.log(`Proposed. Proposal ID: ${proposalID}`);

    console.log(`Executing Proposal`);
    // DO THIS FOURTH
    for(let i= 0; i < 30; i++){

        console.log(`Attempt: ${i + 1}`);
        try{
            await governorBravo.execute(proposalID);
            console.log('Success: Executed');
            break;
        }catch(error){
            if(i == 29){
                await governorBravo.cancel(proposalID);
                console.log(`Proposal failed and has been canceled, please try again`);
                console.log(error)
                return;
            }

            console.log(`\tproposal can only be executed if it is queued`);
        }
        await sleep(15 * 1000);
    }


    let proposalCount = await governorBravo.proposalCount();
    console.log(proposalCount.toString());
    // let proposal = await governorBravo.proposals(proposalID)
    // console.log(proposal);
    state = await governorBravo.state(proposalID);
    console.log('State is : ', proposalStates[state]);
    console.log(JSON.stringify(await governorBravo.getActions(proposalID)));
    console.log('BlockNum : ', await l2_provider.getBlockNumber());
    const proposalThreshold = BigInt(await governorBravo.proposalThreshold());
    console.log('Proposal Threshold : ', proposalThreshold.toString());
    console.log('proposalId : ', proposalID.toString());
}

(async () =>{
    try {
        await main();
    } catch (error) {
        console.log(error)
    }
})();