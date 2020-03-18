const Migrations = artifacts.require('./Migrations.sol');

module.exports = function(deployer, network, accounts) {
	const deployerAcct = accounts[0];
	deployer.deploy(Migrations, { from: deployerAcct });
};
