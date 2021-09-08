const {ethers} = require('ethers');

const Timelock = require('../build-ovm/Timelock.json')
const GovernorBravoDelegate = require('../build-ovm/GovernorBravoDelegate.json')
const GovernorBravoDelegator = require('../build-ovm/GovernorBravoDelegator.json')
const Comp = require('../build-ovm/Comp.json')

const addresses = require('../networks/addresses.json')
const BigNumber = require('bignumber.js')

require('dotenv').config()

const env = process.env

const gasSet = {
  gasPrice: 15000000,
  gasLimit: 8000000
}

const compAddress = addresses.DAO_Comp
const timelockAddress = addresses.DAO_Timelock
const governorBravoDelegateAddress = addresses.DAO_GovernorBravoDelegate
const governorBravoDelegatorAddress = addresses.DAO_GovernorBravoDelegator

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

    let value = await comp.balanceOf(wallet1.address)
    console.log('Wallet1: Comp power: ', value.toString())

    comp.transfer(wallet2.address, ethers.utils.parseEther("1000000"));
    await sleep(5 * 1000);
    value = await comp.balanceOf(wallet2.address);
    console.log('Wallet2: Comp power: ', value.toString());

    comp.transfer(wallet3.address, ethers.utils.parseEther("900000"));
    await sleep(5 * 1000);
    value = await comp.balanceOf(wallet3.address);
    console.log('Wallet3: Comp power: ', value.toString());

    await comp.delegate(wallet1.address); // delegate votes to yourself
    await sleep(5 * 1000);
    await comp.connect(wallet2).delegate(wallet2.address);
    await sleep(5 * 1000);
    await comp.connect(wallet3).delegate(wallet3.address);
    await sleep(5 * 1000);

    console.log(
        'wallet1 current votes: ',
        (await comp.getCurrentVotes(wallet1.address)).toString()
    )
    
    await sleep(5 * 1000)
    
    console.log(
        'wallet2 current votes: ',
        (await comp.getCurrentVotes(wallet2.address)).toString()
    )

    await sleep(5 * 1000)
    
    console.log(
        'wallet3 current votes: ',
        (await comp.getCurrentVotes(wallet3.address)).toString()
    )

    console.log(`Wait 5 minutes to make sure votes are processed.`);
    await sleep(360 * 1000)
}

(async () => {
    try {
        main();
    } catch (error) {
        console.log(error);
    }
})();