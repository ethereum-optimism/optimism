'use strict';

const fs = require('fs');
const path = require('path');

const { loadCompiledFiles, getLatestSolTimestamp } = require('../../publish/src/solidity');

const { CONTRACTS_FOLDER } = require('../../publish/src/constants');
const deployCmd = require('../../publish/src/commands/deploy');
const { buildPath } = deployCmd.DEFAULTS;

module.exports = {
	loadLocalUsers() {
		return Object.entries(
			JSON.parse(fs.readFileSync(path.join(__dirname, '..', '..', 'keys.json'))).private_keys
		).map(([pub, pri]) => ({
			public: pub,
			private: `0x${pri}`,
		}));
	},
	isCompileRequired() {
		// get last modified sol file
		const latestSolTimestamp = getLatestSolTimestamp(CONTRACTS_FOLDER);

		// get last build
		const { earliestCompiledTimestamp } = loadCompiledFiles({ buildPath });

		return latestSolTimestamp > earliestCompiledTimestamp;
	},
};
