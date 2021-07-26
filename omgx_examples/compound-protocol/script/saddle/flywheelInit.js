let https = require('https');
let fs = require('fs').promises;
let { promisify } = require('util');
let assert = require('assert');
let { getArray, getNumber, getBoolean } = require('./support/tokenConfig.js');

function printUsage() {
	console.log(`
usage:
	npx saddle script -n {network} flywheel:init {
		batch: number || null, // how many borrowrs to claim comp for at once. default 100 (~2.3M gas)
		stage: bool || null // use stage API? default false
		cTokens: string[] || null, // which borrowers to get. defaults to all cTokens
		readFixture: bool || null, // save api response? default false
		writeFixture: bool || null, // read from saved response? default false
	}

example:

	npx saddle -n rinkeby script flywheel:init '{batch: "200"}'

To test locally:
	1) ganache-cli --gasLimit 20000000 --gasPrice 20000 --defaultBalanceEther 1000000000 --allowUnlimitedContractSize true
		* use ^6.9
	2) PROVIDER="http://localhost:8545/" script/scen/scriptFlywheel.scen
	3) PROVIDER="http://localhost:8545/" npx saddle -n development script flywheel:init
  `);
}

let getConfig = (configArgs) => {
	let config;
	if (!configArgs) {
		config = {};
	} else {
		try {
			config = JSON.parse(configArgs);
		} catch (e) {
			printUsage();
			console.error(e);
			return null;
		}
	}
	let res = {
		cTokens: getArray(config, 'cTokens', false) || [],
		readFixture: getBoolean(config, 'readFixture', false) || false,
		writeFixture: getBoolean(config, 'writeFixture', false) || false,
		stage: getBoolean(config, 'stage', false) || false,
		batch: getNumber(config, 'batch', false) || 100
	};
	print('Running with actual args: ', res);
	return res;
};

let getCTokenAddresses = (cTokenArgs) => {
	let all = [
		'cUSDC',
		'cDAI',
		'cUSDT',
		'cBAT',
		'cETH',
		'cSAI',
		'cREP',
		'cZRX',
		'cWBTC',
	];
	let list = cTokenArgs.length == 0 ? all : cTokenArgs;
	let map = {};
	for(let val of list) {
		let addr = eval(`$${val}`).toLowerCase();
		map[val] = addr;
	}
	print('Using: ', map);
	return map;
};

let isKnownNetwork = (src) => {
	return ['kovan', 'ropsten', 'goerli', 'mainnet', 'rinkeby'].includes(src);
};


let fetch = async (url) => {
	console.log(`Requesting ${url}\n`);
	return new Promise((resolve, reject) => {
		https.get(url, (res) => {
			let data = '';
			res.on('data', (d) => {
				data += d;
			});
			res.on('end', () => {
				resolve(data);
			});
		});
	});
};

let print = (msg, obj) =>{
	console.log(msg, require('util').inspect(obj, false, null, true), '\n');
}

let writeFile = async (filename, text) => {
	return fs.writeFile(filename, text);
};

let readFile = async (filename) => {
	return fs.readFile(filename, { encoding: 'utf-8' });
};

let readFixture = async (filename) => {
	let url = `./script/saddle/fixture/${network}_borrowers.json`;
	console.log(`Reading ${url}`);
	let file = await readFile(url);
	return JSON.parse(file);
};

let writeFixture = async (data) => {
	await writeFile(`./script/saddle/fixture/${network}_borrowers.json`, JSON.stringify(data));
}

// chunk([1,2,3,4,5], 5) => [[1,2], [2,3], [5]]
let getChunks = (src, chunkSize) => {
	assert(chunkSize > 0, 'chunkSize cant be 0');
	let first = src.slice(0, chunkSize);
	let rest = src.slice(chunkSize);
	if (rest.length == 0) return [first];
	if (rest.length < chunkSize) return [first, rest];
	return [first, ...getChunks(rest, chunkSize)];
};

let getTestData = () => {
	let res = {};
	res[$cZRX] = accounts.slice(3, 7);
	return res;
};

let accountRequest = async (network, opts) => {
	let pageSize = 2000;
	let stageUrl = opts.stage ? 'stage.' : '';
	let url = `https://api.${stageUrl}compound.finance/api/v2/account?min_borrow_value_in_eth[value]=0.00000000000000001&network=${network}&page_size=${pageSize}&page_number=1`;
	let res = await fetch(url);
	return JSON.parse(res).accounts;
};

let filterInitialized = async (borrowersByCToken) => {
	let res = {}
	let batchSize = 75;
	console.log(`Calling compBorrowerIndex for borrowers in batches of ${batchSize}...\n`);
	for(let cTokenAddr of Object.keys(borrowersByCToken)) {
		let speed = await call(Comptroller, 'compSpeeds', [cTokenAddr]);
		if (Number(speed) != 0){
			for (let borrowerChunk of getChunks(borrowersByCToken[cTokenAddr], batchSize)) {
				try {
					let indices = await Promise.all(borrowerChunk.map(
						async(borrower) => {
							return await call(Comptroller, 'compBorrowerIndex',[cTokenAddr, borrower])
					}));
					let uninitialized = borrowerChunk.filter((borrower, i) => Number(indices[i]) == 0);
					res[cTokenAddr] = res[cTokenAddr] ? res[cTokenAddr].concat(uninitialized) : uninitialized;
				} catch(e) {
					console.error(`Web3 calls failed with ${e}`);
					throw `Web3 calls failed w ${e}`;
				}
			}
		}
	}
	return res;
};

// {[ctokenAddr] : borrowers}
let filterBorrowers = (apiAccounts, cTokenList) => {
	return apiAccounts.reduce((acc, account) => {
		let validBorrowers = account.tokens.filter(
			(accountCToken) =>
				cTokenList.includes(accountCToken.address) &&
				accountCToken.borrow_balance_underlying.value > 0
		);
		for (let borrower of validBorrowers) {
			let ctokenAddr = borrower.address;
			acc[ctokenAddr] = acc[ctokenAddr]
				? acc[ctokenAddr].concat(account.address)
				: [account.address];
		}
		return acc;
	}, {});
};

let claimCompBatch = async (borrowersByCToken, opts) => {
	for (let cTokenAddr of Object.keys(borrowersByCToken)) {
		let borrowers = borrowersByCToken[cTokenAddr];
		for (let chunk of getChunks(borrowers, opts.batch)) {
			if (chunk.length == 0) {
				console.log(`No borrowers to claim for ${cTokenAddr}`);
			} else {
				console.log(
					`Sending tx to claim ${cTokenAddr.toString()} borrows for ${JSON.stringify(
						chunk
					)}\n`
				);
				try {
					let tx = await send(Comptroller, 'claimComp', [
						chunk,
						[cTokenAddr],
						true,
						false,
					]);
					console.log(`TX SUCCEEDED: ${JSON.stringify(tx.transactionHash)}\n`);
				} catch (e) {
					console.error(`TX FAILED:  ${e}`);
					throw e;
				}
			}
		}
	}
	console.log('Finished claiming\n')
};

(async () => {
	let borrowersByCToken;
	let cTokenMap; // symbol => addrs
	let opts = getConfig(args[0]);
	if (network == 'development') {
		borrowersByCToken = getTestData();
	} else if (isKnownNetwork(network)) {
		let apiAccounts = opts.readFixture
			? await readFixture()
			: await accountRequest(network, opts);
		let cTokenAddresses = Object.values(getCTokenAddresses(opts.cTokens));
		borrowersByCToken = filterBorrowers(apiAccounts, cTokenAddresses);
		if (opts.writeFixture) await writeFixture(apiAccounts);
	} else {
		printUsage();
	}
	let unInit = await filterInitialized(borrowersByCToken);
	print('Uninitialized accounts before: ', unInit);

	await claimCompBatch(unInit, opts);
	unInit = await filterInitialized(borrowersByCToken);
	print('Uninitialized accounts after: ', unInit);
})();
