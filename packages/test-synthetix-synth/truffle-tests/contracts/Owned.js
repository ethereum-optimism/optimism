require('.'); // import common test scaffolding

const Owned = artifacts.require('Owned');
const { ZERO_ADDRESS } = require('../utils/testUtils');

contract.skip('Owned - Test contract deployment', accounts => {
	const [deployerAccount, account1] = accounts;

	it.skip('should revert when owner parameter is passed the zero address', async () => {
		await assert.revert(Owned.new(ZERO_ADDRESS, { from: deployerAccount }));
	});

	// TODO check events on contract creation
	it('should set owner address on deployment', async () => {
		const ownedContractInstance = await Owned.new(account1, { from: deployerAccount });
		const owner = await ownedContractInstance.owner();
		assert.equal(owner, account1);
	});
});

contract.skip('Owned - Pre deployed contract', async accounts => {
	const [account1, account2, account3, account4] = accounts.slice(1); // The first account is the deployerAccount above

	it.skip('should not nominate new owner when not invoked by current contract owner', async () => {
		const ownedContractInstance = await Owned.deployed();
		const nominatedOwner = account3;

		await assert.revert(ownedContractInstance.nominateNewOwner(nominatedOwner, { from: account2 }));

		const nominatedOwnerFrmContract = await ownedContractInstance.nominatedOwner();
		assert.equal(nominatedOwnerFrmContract, ZERO_ADDRESS);
	});

	it('should nominate new owner when invoked by current contract owner', async () => {
		const ownedContractInstance = await Owned.deployed();
		const nominatedOwner = account2;

		const txn = await ownedContractInstance.nominateNewOwner(nominatedOwner, { from: account1 });
		assert.eventEqual(txn, 'OwnerNominated', { newOwner: nominatedOwner });

		const nominatedOwnerFromContract = await ownedContractInstance.nominatedOwner();
		assert.equal(nominatedOwnerFromContract, nominatedOwner);
	});

	it('should not accept new owner nomination when not invoked by nominated owner', async () => {
		const ownedContractInstance = await Owned.deployed();
		const nominatedOwner = account3;

		await assert.revert(ownedContractInstance.acceptOwnership({ from: account4 }));

		const owner = await ownedContractInstance.owner();
		assert.notEqual(owner, nominatedOwner);
	});

	// TODO: Figure out nonce issues
	it.skip('should accept new owner nomination when invoked by nominated owner', async () => {
		const ownedContractInstance = await Owned.deployed();
		const nominatedOwner = account2;

		let txn = await ownedContractInstance.nominateNewOwner(nominatedOwner, { from: account1 });
		assert.eventEqual(txn, 'OwnerNominated', { newOwner: nominatedOwner });

		const nominatedOwnerFromContract = await ownedContractInstance.nominatedOwner();
		assert.equal(nominatedOwnerFromContract, nominatedOwner);

		txn = await ownedContractInstance.acceptOwnership({ from: account2 });

		assert.eventEqual(txn, 'OwnerChanged', { oldOwner: account1, newOwner: account2 });

		const owner = await ownedContractInstance.owner();
		const nominatedOwnerFromContact = await ownedContractInstance.nominatedOwner();

		assert.equal(owner, nominatedOwner);
		assert.equal(nominatedOwnerFromContact, ZERO_ADDRESS);
	});
});
