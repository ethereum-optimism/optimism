package db

const createL1BlocksTable = `
CREATE TABLE IF NOT EXISTS l1_blocks (
	hash VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

const createL2BlocksTable = `
CREATE TABLE IF NOT EXISTS l2_blocks (
	hash VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number INTEGER NOT NULL,
	timestamp INTEGER NOT NULL
)
`

const createDepositsTable = `
CREATE TABLE IF NOT EXISTS deposits (
	guid VARCHAR PRIMARY KEY NOT NULL,
	from_address VARCHAR NOT NULL,
	to_address VARCHAR NOT NULL,
	l1_token VARCHAR NOT NULL REFERENCES l1_tokens(address),
	l2_token VARCHAR NOT NULL,
	amount VARCHAR NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	l1_block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash),
	l2_block_hash VARCHAR REFERENCES l2_blocks(hash),
	tx_hash VARCHAR NOT NULL,
	failed BOOLEAN NOT NULL DEFAULT false
)
`

const createL1TokensTable = `
CREATE TABLE IF NOT EXISTS l1_tokens (
	address VARCHAR NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	symbol VARCHAR NOT NULL,
	decimals INTEGER NOT NULL
)
`

const createL2TokensTable = `
CREATE TABLE IF NOT EXISTS l2_tokens (
	address TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL,
	symbol TEXT NOT NULL,
	decimals INTEGER NOT NULL
)
`

const createWithdrawalsTable = `
CREATE TABLE IF NOT EXISTS withdrawals (
	guid VARCHAR PRIMARY KEY NOT NULL,
	from_address VARCHAR NOT NULL,
	to_address VARCHAR NOT NULL,
	l1_token VARCHAR NOT NULL,
	l2_token VARCHAR NOT NULL REFERENCES l2_tokens(address),
	amount VARCHAR NOT NULL,
	data BYTEA NOT NULL,
	log_index INTEGER NOT NULL,
	l1_block_hash VARCHAR REFERENCES l1_blocks(hash),
	l2_block_hash VARCHAR NOT NULL REFERENCES l2_blocks(hash),
	tx_hash VARCHAR NOT NULL
)
`

const insertETHL1Token = `
INSERT INTO l1_tokens
	(address, name, symbol, decimals)
VALUES ('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
`

// earlier transactions used 0x0000000000000000000000000000000000000000 as
// address of ETH so insert both that and
// 0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000
const insertETHL2Token = `
INSERT INTO l2_tokens
	(address, name, symbol, decimals)
VALUES ('0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
INSERT INTO l2_tokens
	(address, name, symbol, decimals)
VALUES ('0x0000000000000000000000000000000000000000', 'Ethereum', 'ETH', 18)
ON CONFLICT (address) DO NOTHING;
`

const createL1L2NumberIndex = `
CREATE UNIQUE INDEX IF NOT EXISTS l1_blocks_number ON l1_blocks(number);
CREATE UNIQUE INDEX IF NOT EXISTS l2_blocks_number ON l2_blocks(number);
`

const createAirdropsTable = `
CREATE TABLE IF NOT EXISTS airdrops (
	address VARCHAR(42) PRIMARY KEY,
	voter_amount VARCHAR NOT NULL DEFAULT '0' CHECK(voter_amount ~ '^\d+$') ,
	multisig_signer_amount VARCHAR NOT NULL DEFAULT '0' CHECK(multisig_signer_amount ~ '^\d+$'),
	gitcoin_amount VARCHAR NOT NULL DEFAULT '0' CHECK(gitcoin_amount ~ '^\d+$'),
	active_bridged_amount VARCHAR NOT NULL DEFAULT '0' CHECK(active_bridged_amount ~ '^\d+$'),
	op_user_amount VARCHAR NOT NULL DEFAULT '0' CHECK(op_user_amount ~ '^\d+$'),
	op_repeat_user_amount VARCHAR NOT NULL DEFAULT '0' CHECK(op_user_amount ~ '^\d+$'),
	op_og_amount VARCHAR NOT NULL DEFAULT '0' CHECK(op_og_amount ~ '^\d+$'),
	bonus_amount VARCHAR NOT NULL DEFAULT '0' CHECK(bonus_amount ~ '^\d+$'),
	total_amount VARCHAR NOT NULL CHECK(voter_amount ~ '^\d+$')
)
`

var schema = []string{
	createL1BlocksTable,
	createL2BlocksTable,
	createL1TokensTable,
	createL2TokensTable,
	insertETHL1Token,
	insertETHL2Token,
	createDepositsTable,
	createWithdrawalsTable,
	createL1L2NumberIndex,
	createAirdropsTable,
}
