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
	block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash),
	tx_hash VARCHAR NOT NULL
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

const createStateBatchesTable = `
CREATE TABLE IF NOT EXISTS state_batches (
	index INTEGER NOT NULL PRIMARY KEY,
	root VARCHAR NOT NULL,
	size INTEGER NOT NULL,
	prev_total INTEGER NOT NULL,
	extra_data BYTEA NOT NULL,
	block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash)
);
CREATE INDEX IF NOT EXISTS state_batches_block_hash ON state_batches(block_hash);
CREATE INDEX IF NOT EXISTS state_batches_size ON state_batches(size);
CREATE INDEX IF NOT EXISTS state_batches_prev_total ON state_batches(prev_total);
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
	block_hash VARCHAR NOT NULL REFERENCES l2_blocks(hash),
	tx_hash VARCHAR NOT NULL,
	state_batch INTEGER REFERENCES state_batches(index)
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

const updateWithdrawalsTable = `
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_hash VARCHAR NULL;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_proven_tx_hash VARCHAR NULL;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_proven_log_index INTEGER NULL;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_finalized_tx_hash VARCHAR NULL;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_finalized_log_index INTEGER NULL;
ALTER TABLE withdrawals ADD COLUMN IF NOT EXISTS br_withdrawal_finalized_success BOOLEAN NULL;
CREATE INDEX IF NOT EXISTS withdrawals_br_withdrawal_hash ON withdrawals(br_withdrawal_hash);
`

var schema = []string{
	createL1BlocksTable,
	createL2BlocksTable,
	createL1TokensTable,
	createL2TokensTable,
	createStateBatchesTable,
	insertETHL1Token,
	insertETHL2Token,
	createDepositsTable,
	createWithdrawalsTable,
	createL1L2NumberIndex,
	createAirdropsTable,
	updateWithdrawalsTable,
}
