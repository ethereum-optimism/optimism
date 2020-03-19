const BN = require('bn.js');

const { toBN, toWei, fromWei, hexToAscii } = require('web3-utils');
const UNIT = toWei(new BN('1'), 'ether');

const Web3 = require('web3');
// web3 is injected to the global scope via truffle test, but
// we need this here for test/publish which bypasses truffle altogether.
// Note: providing the connection string 'http://127.0.0.1:8545' seems to break
// RewardEscrow Stress Tests - it is not clear why however.
if (typeof web3 === 'undefined') {
	global.web3 = new Web3(new Web3.providers.HttpProvider('http://127.0.0.1:8545'));
}

const ZERO_ADDRESS = '0x' + '0'.repeat(40);

/**
 * Sets default properties on the jsonrpc object and promisifies it so we don't have to copy/paste everywhere.
 */
const send = payload => {
	if (!payload.jsonrpc) payload.jsonrpc = '2.0';
	if (!payload.id) payload.id = new Date().getTime();

	return new Promise((resolve, reject) => {
		web3.currentProvider.send(payload, (error, result) => {
			if (error) return reject(error);

			return resolve(result);
		});
	});
};

/**
 *  Mines a single block in Ganache (evm_mine is non-standard)
 */
const mineBlock = () => send({ method: 'evm_mine' });

/**
 *  Gets the time of the last block.
 */
const currentTime = async () => {
	const { timestamp } = await web3.eth.getBlock('latest');
	return timestamp;
};

/**
 *  Increases the time in the EVM.
 *  @param seconds Number of seconds to increase the time by
 */
const fastForward = async seconds => {
	// It's handy to be able to be able to pass big numbers in as we can just
	// query them from the contract, then send them back. If not changed to
	// a number, this causes much larger fast forwards than expected without error.
	if (BN.isBN(seconds)) seconds = seconds.toNumber();

	// And same with strings.
	if (typeof seconds === 'string') seconds = parseFloat(seconds);

	await send({
		method: 'evm_increaseTime',
		params: [seconds],
	});

	await mineBlock();
};

/**
 *  Increases the time in the EVM to as close to a specific date as possible
 *  NOTE: Because this operation figures out the amount of seconds to jump then applies that to the EVM,
 *  sometimes the result can vary by a second or two depending on how fast or slow ganache is responding.
 *  @param time Date object representing the desired time at the end of the operation
 */
const fastForwardTo = async time => {
	if (typeof time === 'string') time = parseInt(time);

	const timestamp = await currentTime();
	const now = new Date(timestamp * 1000);
	if (time < now)
		throw new Error(
			`Time parameter (${time}) is less than now ${now}. You can only fast forward to times in the future.`
		);

	const secondsBetween = Math.floor((time.getTime() - now.getTime()) / 1000);

	await fastForward(secondsBetween);
};

/**
 *  Takes a snapshot and returns the ID of the snapshot for restoring later.
 */
const takeSnapshot = async () => {
	const { result } = await send({ method: 'evm_snapshot' });
	await mineBlock();

	return result;
};

/**
 *  Restores a snapshot that was previously taken with takeSnapshot
 *  @param id The ID that was returned when takeSnapshot was called.
 */
const restoreSnapshot = async id => {
	await send({
		method: 'evm_revert',
		params: [id],
	});
	await mineBlock();
};

/**
 *  Translates an amount to our cononical unit. We happen to use 10^18, which means we can
 *  use the built in web3 method for convenience, but if unit ever changes in our contracts
 *  we should be able to update the conversion factor here.
 *  @param amount The amount you want to re-base to UNIT
 */
const toUnit = amount => toBN(toWei(amount.toString(), 'ether'));
const fromUnit = amount => fromWei(amount, 'ether');

/**
 *  Translates an amount to our cononical precise unit. We happen to use 10^27, which means we can
 *  use the built in web3 method for convenience, but if precise unit ever changes in our contracts
 *  we should be able to update the conversion factor here.
 *  @param amount The amount you want to re-base to PRECISE_UNIT
 */
const PRECISE_UNIT_STRING = '1000000000000000000000000000';
const PRECISE_UNIT = toBN(PRECISE_UNIT_STRING);

const toPreciseUnit = amount => {
	// Code is largely lifted from the guts of web3 toWei here:
	// https://github.com/ethjs/ethjs-unit/blob/master/src/index.js
	const amountString = amount.toString();

	// Is it negative?
	var negative = amountString.substring(0, 1) === '-';
	if (negative) {
		amount = amount.substring(1);
	}

	if (amount === '.') {
		throw new Error(`Error converting number ${amount} to precise unit, invalid value`);
	}

	// Split it into a whole and fractional part
	// eslint-disable-next-line prefer-const
	let [whole, fraction, ...rest] = amount.split('.');
	if (rest.length > 0) {
		throw new Error(`Error converting number ${amount} to precise unit, too many decimal points`);
	}

	if (!whole) {
		whole = '0';
	}
	if (!fraction) {
		fraction = '0';
	}
	if (fraction.length > PRECISE_UNIT_STRING.length - 1) {
		throw new Error(`Error converting number ${amount} to precise unit, too many decimal places`);
	}

	while (fraction.length < PRECISE_UNIT_STRING.length - 1) {
		fraction += '0';
	}

	whole = new BN(whole);
	fraction = new BN(fraction);
	let result = whole.mul(PRECISE_UNIT).add(fraction);

	if (negative) {
		result = result.mul(new BN('-1'));
	}

	return result;
};

const fromPreciseUnit = amount => {
	// Code is largely lifted from the guts of web3 fromWei here:
	// https://github.com/ethjs/ethjs-unit/blob/master/src/index.js
	const negative = amount.lt(new BN('0'));

	if (negative) {
		amount = amount.mul(new BN('-1'));
	}

	let fraction = amount.mod(PRECISE_UNIT).toString();

	while (fraction.length < PRECISE_UNIT_STRING.length - 1) {
		fraction = `0${fraction}`;
	}

	// Chop zeros off the end if there are extras.
	fraction = fraction.replace(/0+$/, '');

	const whole = amount.div(PRECISE_UNIT).toString();
	let value = `${whole}${fraction === '' ? '' : `.${fraction}`}`;

	if (negative) {
		value = `-${value}`;
	}

	return value;
};

/*
 * Multiplies x and y interpreting them as fixed point decimal numbers.
 */
const multiplyDecimal = (x, y, unit = UNIT) => {
	const xBN = BN.isBN(x) ? x : new BN(x);
	const yBN = BN.isBN(y) ? y : new BN(y);
	return xBN.mul(yBN).div(unit);
};

/*
 * Multiplies x and y interpreting them as fixed point decimal numbers.
 */
const divideDecimal = (x, y, unit = UNIT) => {
	const xBN = BN.isBN(x) ? x : new BN(x);
	const yBN = BN.isBN(y) ? y : new BN(y);
	return xBN.mul(unit).div(yBN);
};

/*
 * Exponentiation by squares of x^n, interpreting them as fixed point decimal numbers.
 */
const powerToDecimal = (x, n, unit = UNIT) => {
	let xBN = BN.isBN(x) ? x : new BN(x);
	let temp = unit;
	while (n > 0) {
		if (n % 2 !== 0) {
			temp = temp.mul(xBN).div(unit);
		}
		xBN = xBN.mul(xBN).div(unit);
		n = parseInt(n / 2);
	}
	return temp;
};

/**
 *  Convenience method to assert that an event matches a shape
 *  @param actualEventOrTransaction The transaction receipt, or event as returned in the event logs from web3
 *  @param expectedEvent The event name you expect
 *  @param expectedArgs The args you expect in object notation, e.g. { newOracle: '0x...', updatedAt: '...' }
 */
const assertEventEqual = (actualEventOrTransaction, expectedEvent, expectedArgs) => {
	// If they pass in a whole transaction we need to extract the first log, otherwise we already have what we need
	const event = Array.isArray(actualEventOrTransaction.logs)
		? actualEventOrTransaction.logs[0]
		: actualEventOrTransaction;

	if (!event) {
		assert.fail(new Error('No event was generated from this transaction'));
	}

	// Assert the names are the same.
	assert.equal(event.event, expectedEvent);

	assertDeepEqual(event.args, expectedArgs);
	// Note: this means that if you don't assert args they'll pass regardless.
	// Ensure you pass in all the args you need to assert on.
};

/**
 * Converts a hex string of bytes into a UTF8 string with \0 characters (from padding) removed
 */
const bytesToString = bytes => {
	const result = hexToAscii(bytes);
	return result.replace(/\0/g, '');
};

const assertEventsEqual = (transaction, ...expectedEventsAndArgs) => {
	if (expectedEventsAndArgs.length % 2 > 0)
		throw new Error('Please call assert.eventsEqual with names and args as pairs.');
	if (expectedEventsAndArgs.length <= 2)
		throw new Error(
			"Expected events and args can be called with just assert.eventEqual as there's only one event."
		);

	for (let i = 0; i < expectedEventsAndArgs.length; i += 2) {
		const log = transaction.logs[Math.floor(i / 2)];

		assert.equal(log.event, expectedEventsAndArgs[i], 'Event name mismatch');
		assertDeepEqual(log.args, expectedEventsAndArgs[i + 1], 'Event args mismatch');
	}
};

/**
 *  Convenience method to assert that two BN.js instances are equal.
 *  @param actualBN The BN.js instance you received
 *  @param expectedBN The BN.js amount you expected to receive
 *  @param context The description to log if we fail the assertion
 */
const assertBNEqual = (actualBN, expectedBN, context) => {
	assert.equal(actualBN.toString(), expectedBN.toString(), context);
};

/**
 *  Convenience method to assert that two BN.js instances are NOT equal.
 *  @param actualBN The BN.js instance you received
 *  @param expectedBN The BN.js amount you expected NOT to receive
 *  @param context The description to log if we fail the assertion
 */
const assertBNNotEqual = (actualBN, expectedBN) => {
	assert.notEqual(actualBN.toString(), expectedBN.toString(), context);
};

/**
 *  Convenience method to assert that two BN.js instances are within 100 units of each other.
 *  @param actualBN The BN.js instance you received
 *  @param expectedBN The BN.js amount you expected to receive, allowing a varience of +/- 100 units
 */
const assertBNClose = (actualBN, expectedBN, varianceParam = '10') => {
	const actual = BN.isBN(actualBN) ? actualBN : new BN(actualBN);
	const expected = BN.isBN(expectedBN) ? expectedBN : new BN(expectedBN);
	const variance = BN.isBN(varianceParam) ? varianceParam : new BN(varianceParam);
	const actualDelta = expected.sub(actual).abs();

	assert.ok(
		actual.gte(expected.sub(variance)),
		`Number is too small to be close (Delta between actual and expected is ${actualDelta.toString()}, but variance was only ${variance.toString()}`
	);
	assert.ok(
		actual.lte(expected.add(variance)),
		`Number is too large to be close (Delta between actual and expected is ${actualDelta.toString()}, but variance was only ${variance.toString()})`
	);
};

/**
 *  Convenience method to assert that two objects or arrays which contain nested BN.js instances are equal.
 *  @param actual What you received
 *  @param expected The shape you expected
 */
const assertDeepEqual = (actual, expected, context) => {
	// Check if it's a value type we can assert on straight away.
	if (BN.isBN(actual) || BN.isBN(expected)) {
		assertBNEqual(actual, expected, context);
	} else if (
		typeof expected === 'string' ||
		typeof actual === 'string' ||
		typeof expected === 'number' ||
		typeof actual === 'number' ||
		typeof expected === 'boolean' ||
		typeof actual === 'boolean'
	) {
		assert.equal(actual, expected, context);
	}
	// Otherwise dig through the deeper object and recurse
	else if (Array.isArray(expected)) {
		for (let i = 0; i < expected.length; i++) {
			assertDeepEqual(actual[i], expected[i], `(array index: ${i}) `);
		}
	} else {
		for (const key of Object.keys(expected)) {
			assertDeepEqual(actual[key], expected[key], `(key: ${key}) `);
		}
	}
};

/**
 *  Convenience method to assert that an amount of ether (or other 10^18 number) was received from a contract.
 *  @param actualWei The value retrieved from a smart contract or wallet in wei
 *  @param expectedAmount The amount you expect e.g. '1'
 *  @param expectedUnit The unit you expect e.g. 'gwei'. Defaults to 'ether'
 */
const assertUnitEqual = (actualWei, expectedAmount, expectedUnit = 'ether') => {
	assertBNEqual(actualWei, toWei(expectedAmount, expectedUnit));
};

/**
 *  Convenience method to assert that an amount of ether (or other 10^18 number) was NOT received from a contract.
 *  @param actualWei The value retrieved from a smart contract or wallet in wei
 *  @param expectedAmount The amount you expect NOT to be equal to e.g. '1'
 *  @param expectedUnit The unit you expect e.g. 'gwei'. Defaults to 'ether'
 */
const assertUnitNotEqual = (actualWei, expectedAmount, expectedUnit = 'ether') => {
	assertBNNotEqual(actualWei, toWei(expectedAmount, expectedUnit));
};

/**
 * Convenience method to assert that the return of the given block when invoked or promise causes a
 * revert to occur, with an optional revert message.
 * @param blockOrPromise The JS block (i.e. function that when invoked returns a promise) or a promise itself
 * @param reason Optional reason string to search for in revert message
 */
const assertRevert = async (blockOrPromise, reason) => {
	let errorCaught = false;
	try {
		const result = typeof blockOrPromise === 'function' ? blockOrPromise() : blockOrPromise;
		await result;
	} catch (error) {
		assert.include(error.message, 'revert');
		if (reason) {
			assert.include(error.message, reason);
		}
		errorCaught = true;
	}

	assert.equal(errorCaught, true, 'Operation did not revert as expected');
};

const assertInvalidOpcode = async blockOrPromise => {
	let errorCaught = false;
	try {
		const result = typeof blockOrPromise === 'function' ? blockOrPromise() : blockOrPromise;
		await result;
	} catch (error) {
		assert.include(error.message, 'invalid opcode');
		errorCaught = true;
	}

	assert.equal(errorCaught, true, 'Operation did not cause an invalid opcode error as expected');
};

/**
 *  Gets the ETH balance for the account address
 * 	@param account Ethereum wallet address
 */
const getEthBalance = account => web3.eth.getBalance(account);

module.exports = {
	ZERO_ADDRESS,

	mineBlock,
	fastForward,
	fastForwardTo,
	takeSnapshot,
	restoreSnapshot,
	currentTime,
	multiplyDecimal,
	divideDecimal,
	powerToDecimal,

	toUnit,
	fromUnit,

	toPreciseUnit,
	fromPreciseUnit,

	assertEventEqual,
	assertEventsEqual,
	assertBNEqual,
	assertBNNotEqual,
	assertBNClose,
	assertDeepEqual,
	assertInvalidOpcode,
	assertUnitEqual,
	assertUnitNotEqual,
	assertRevert,

	getEthBalance,
	bytesToString,
};
