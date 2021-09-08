const {ethers} = require('ethers')

const Timelock = require('../build-ovm/Timelock.json')
const GovernorBravoDelegate = require('../build-ovm/GovernorBravoDelegate.json')
const GovernorBravoDelegator = require('../build-ovm/GovernorBravoDelegator.json')
const Comp = require('../build-ovm/Comp.json')

const addresses = require('../networks/addresses.json')

require('dotenv').config()

const env = process.env
const DECIMALS  = BigInt(10**18)

const compAddress = addresses.DAO_Comp;
const timelockAddress = addresses.DAO_Timelock;
const governorBravoDelegateAddress = addresses.DAO_GovernorBravoDelegate;
const governorBravoDelegatorAddress = addresses.DAO_GovernorBravoDelegator;

const sleep = async (timeout) => {
	return new Promise((resolve, reject) => {
		setTimeout(() => {
			resolve()
		}, timeout)
	})
}

async function getBlockNumber(web3url, chainID){
    const provider = new ethers.providers.JsonRpcProvider(web3url, {chainId: chainID});
    const blockNumber = await provider.getBlockNumber();
    return blockNumber;
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
    const delegate = await governorBravoDelegate.attach(governorBravoDelegator.address)
    
    console.log(delegate)

    const proposalCounts = await delegate.proposalCount()
    console.log('proposalCounts:',proposalCounts)

    const proposalID = (await delegate.proposalCount())._hex
    console.log(`Proposed. Proposal ID: ${proposalID}`)
    
    let proposal = await delegate.proposals(proposalID)
    console.log(`Proposal:`, proposal)

    console.log(`Proposal.startBlock:`, proposal.startBlock.toString())
    console.log(`Proposal.endBlock:`, proposal.endBlock.toString())

    console.log(`Block Number: ${await getBlockNumber(env.L2_NODE_WEB3_URL, 28)}`)

    //console.log(`Block Number: ${await getBlockNumber(env.L2_NODE_WEB3_URL, 28)}`)
    
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

    let state = await delegate.state(proposalID)
    console.log('State is: ', proposalStates[state])
    
      // const totalProposal = await proposalCounts.toNumber()
      // const filter = this.delegate.filters.ProposalCreated(
      //   null,
      //   null,
      //   null,
      //   null,
      //   null,
      //   null,
      //   null,
      //   null,
      //   null
      // )
      // const descriptionList = await this.delegate.queryFilter(filter);
      // for (let i = totalProposal; i > 1 && i > totalProposal - 3; i--) {
      //   let proposal = await this.delegate.getActions(i);
      //   let fullDescription = descriptionList[i - 2].args[8].toString();
      //   console.log('fullDescription:',fullDescription)
      //   let titleEnd = fullDescription.search(/\n/);
      //   let title = fullDescription.substring(0, titleEnd);
      //   let description = fullDescription.substring(titleEnd + 1);

      //   proposalList.push({
      //     proposal,
      //     title,
      //     description
      //   })
      // }

    // console.log(
    //     'Comp needed to propose: ',
    //     ethers.utils.parseEther('100000').toString()
    //     //original setting is 100,000
    // )

    // console.log(
    //     'Proposing to change to: ',
    //     ethers.utils.parseEther('65000').toString()
    //     //reduce to 65,000
    // )

    // const proposalStates = [
    //     'Pending',
    //     'Active',
    //     'Canceled',
    //     'Defeated',
    //     'Succeeded',
    //     'Queued',
    //     'Expired',
    //     'Executed',
    // ];

    // let addresses = [governorBravo.address] // the address of the contract where the function will be called
    // let values = [0] // the eth necessary to send to the contract above
    // let signatures = ['_setProposalThreshold(uint256)'] // the function that will carry out the proposal
    
    // let calldatas = [ethers.utils.defaultAbiCoder.encode( // the parameter for the above function
    //     ['uint256'],
    //     [ethers.utils.parseEther('65000')]
    // )]
    
    // let description = '#Changing Proposal Threshold to 65000 Comp' // the description of the proposal

    // console.log(
    //     'wallet1 current votes: ',
    //     (await comp.getCurrentVotes(wallet1.address)).toString()
    // )

    // console.log(`Proposing`)

    // // submitting the proposal
    // await governorBravo.connect(wallet1).propose(
    // 	addresses,
    // 	values,
    // 	signatures,
    // 	calldatas,
    // 	description,
    //     {
    //         gasPrice: 15000000,
    //         gasLimit: 8000000
    //     }
    // )

    // console.log()
    // sleep(15 * 1000)
    
    //const proposalID = (await delegate.proposalCount())._hex
    //console.log(`Proposed. Proposal ID: ${proposalID}`)
    


    // console.log(`Block Number: ${await getBlockNumber(env.L2_NODE_WEB3_URL, 28)}`)
    
    // let state = await governorBravo.state(proposalID)
    // console.log('State is: ', proposalStates[state])

    // console.log(`Waiting for voting delay.`)
    // await sleep(150 * 1000)
}

(async () =>{
    try {
        await main();
    } catch (error) {
        console.log(error)
    }
})();