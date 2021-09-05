const { ethers } = require('ethers')

const sleep = (timeout) => {
	return new Promise((resolve, reject) => {
		setTimeout(() => {
			resolve()
		}, timeout)
	})
}
async function main(callback) {
	try {
		// deployer.then(async () => {
		// 	await deployer.deploy(GovernorBravoDelegate)
		// 	await deployer.deploy(SafeMath)
		// 	await deployer.deploy(Comp, adminAddress)
		// 	await deployer.deploy(Timelock, adminAddress, 172800)
		// 	await deployer.deploy(
		// 		GovernorBravoDelegator,
		// 		Timelock.address,
		// 		Comp.address,
		// 		Timelock.address,
		// 		GovernorBravoDelegate.address,
		// 		17280,
		// 		1,
		// 		'100000000000000000000000'
		// 	)
		// })
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
		const GovernorBravoDelegate = artifacts.require('GovernorBravoDelegate')
		const governorBravoDelegate = await GovernorBravoDelegate.deployed()
		const GovernorBravoDelegator = artifacts.require('GovernorBravoDelegator')
		const governorBravoDelegator = await GovernorBravoDelegator.deployed()
		const Comp = artifacts.require('Comp')
		const comp = await Comp.deployed()
		const Timelock = artifacts.require('Timelock')
		const timelock = await Timelock.deployed()

		const GovernorBravo = await GovernorBravoDelegate.at(
			governorBravoDelegator.address
		)

		const accounts = await web3.eth.getAccounts()
		const value = await comp.balanceOf(accounts[0])
		// console.log('Comp power: ', value.toString())
		let addresses = [GovernorBravo.address]
		let values = [0]
		let signatures = ['_setProposalThreshold(uint256)']
		let calldatas = [
			'0x000000000000000000000000000000000000000000000dc3a8351f3d86a00000',
		]
		let description = '#Changing Proposal Threshold to 65000 Comp'

		await comp.delegate(accounts[0])

		console.log(
			'current votes: ',
			(await comp.getCurrentVotes(accounts[0])).toString()
		)

		await sleep(500)

		// THIS SECTION DOES ALL THE PROPOSING LOGIC YOU NEED TO
		// MAKE SURE THAT YOU'RE ONLY CALLING ONE OF THESE AT A TIME

		// DO THIS FIRST

		// await GovernorBravo.propose(
		// 	addresses,
		// 	values,
		// 	signatures,
		// 	calldatas,
		// 	description
		// )
		// console.log('proposed')

		// DO THIS SECOND

		// await GovernorBravo.castVote(1, 1)
		// console.log('vote cast')

		// DO THIS THIRD

		// await GovernorBravo.queue(1)
		// console.log('Queued')

		// DO THIS FOURTH

		// await GovernorBravo.execute(1)
		// console.log('Executed')

		await sleep(500)

		proposalCount = await GovernorBravo.proposalCount()
		console.log(proposalCount.toString())

		await GovernorBravo.proposals.call(1, function (err, res) {
			if (err) {
				console.log('PROPOSALS', err)
			}
			console.log('PROPOSALS', res)
		})

		state = await GovernorBravo.state(1)
		console.log('State is : ', proposalStates[state])
		// console.log(JSON.stringify(await GovernorBravo.getActions(1)))
		timeStamp = await timelock.exGetBlockTimestamp()
		console.log('Timestamp : ', timeStamp.toString())
		console.log('WEB3 : ', await web3.eth.getBlockNumber())
		const proposalThreshold = await GovernorBravo.proposalThreshold()
		console.log('Proposal Threshold : ', proposalThreshold.toString())
		const proposalId = await GovernorBravo.initialProposalId()
		console.log('proposalId : ', proposalId.toString())
	} catch (error) {
		console.log(error)
		callback(1)
	}
}


module.exports = main;