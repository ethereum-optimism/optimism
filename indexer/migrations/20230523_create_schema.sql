/**
 * BLOCK DATA
 */

CREATE TABLE IF NOT EXISTS l1_blocks (
	hash        VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number      NUMERIC NOT NULL,
	timestamp   INTEGER NOT NULL,

    CONSTRAINT
        l1_parent_hash_fkey FOREIGN KEY(parent_hash) REFERENCES l1_blocks(hash)
);

CREATE TABLE IF NOT EXISTS l2_blocks (
    -- Block header
	hash                     VARCHAR NOT NULL PRIMARY KEY,
	parent_hash              VARCHAR NOT NULL,
	number                   NUMERIC NOT NULL,
	timestamp                INTEGER NOT NULL,

    -- Finalization information
    l1_block_hash            VARCHAR NOT NULL REFERENCES l1_blocks(hash),
    legacy_state_batch_index INTEGER,

    CONSTRAINT
        l2_parent_hash_fkey FOREIGN KEY(parent_hash) REFERENCES l2_blocks(hash)
);

CREATE TABLE IF NOT EXISTS legacy_state_batches (
	index         INTEGER NOT NULL PRIMARY KEY,
	root          VARCHAR NOT NULL,
	size          INTEGER NOT NULL,
	prev_total    INTEGER NOT NULL,

    -- Finalization information
	l1_block_hash VARCHAR NOT NULL REFERENCES l1_blocks(hash)
);

/** 
 * EVENT DATA
 */

CREATE TABLE IF NOT EXISTS l1_contract_events (
    guid             VARCHAR NOT NULL PRIMARY KEY,
	block_hash       VARCHAR NOT NULL REFERENCES l1_blocks(hash),
    transaction_hash VARCHAR NOT NULL,
    event_signature  VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS l2_contract_events (
    guid             VARCHAR NOT NULL PRIMARY KEY,
	block_hash       VARCHAR NOT NULL REFERENCES l2_blocks(hash),
    transaction_hash VARCHAR NOT NULL,
    event_signature  VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL
);

/**
 * BRIDGING DATA
 */

CREATE TABLE IF NOT EXISTS deposits (
	guid                 VARCHAR PRIMARY KEY NOT NULL,

    -- Event causing the deposit
    initiated_l1_event_guid VARCHAR NOT NULL REFERENCES l1_contract_events(guid),

    -- Deposit information (do we need indexes on from/to?)
	from_address     VARCHAR NOT NULL,
	to_address       VARCHAR NOT NULL,
	l1_token_address VARCHAR NOT NULL,
	l2_token_address VARCHAR NOT NULL,
	amount           NUMERIC NOT NULL,
	data             BYTEA NOT NULL
);

CREATE TABLE IF NOT EXISTS withdrawals (
	guid                VARCHAR PRIMARY KEY NOT NULL,

    -- Event causing this withdrawal
    intiated_l2_event_guid VARCHAR NOT NULL REFERENCES l2_contract_events(guid),

    -- Multistep (bedrock) process of a withdrawal
    withdrawal_hash      VARCHAR NOT NULL,
    proven_l1_event_guid VARCHAR REFERENCES l1_contract_events(guid),

    -- Finalization marker (legacy & bedrock)
    finalized_l1_event_guid VARCHAR REFERENCES l1_contract_events(guid),

    -- Withdrawal information (do we need indexes on from/to?)
	from_address     VARCHAR NOT NULL,
	to_address       VARCHAR NOT NULL,
	l1_token_address VARCHAR NOT NULL,
	l2_token_address VARCHAR NOT NULL,
	amount           NUMERIC NOT NULL,
	data             BYTEA NOT NULL
);
