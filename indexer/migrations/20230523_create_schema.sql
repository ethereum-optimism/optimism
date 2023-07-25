
CREATE DOMAIN UINT256 AS NUMERIC NOT NULL
    CHECK (VALUE >= 0 AND VALUE < 2^256 and SCALE(VALUE) = 0);

/**
 * BLOCK DATA
 */

CREATE TABLE IF NOT EXISTS l1_block_headers (
	hash        VARCHAR NOT NULL PRIMARY KEY,
	parent_hash VARCHAR NOT NULL,
	number      UINT256,
	timestamp   INTEGER NOT NULL CHECK (timestamp > 0)
);

CREATE TABLE IF NOT EXISTS l2_block_headers (
    -- Block header
	hash                     VARCHAR NOT NULL PRIMARY KEY,
	parent_hash              VARCHAR NOT NULL,
	number                   UINT256,
	timestamp                INTEGER NOT NULL CHECK (timestamp > 0)
);

/** 
 * EVENT DATA
 */

CREATE TABLE IF NOT EXISTS l1_contract_events (
    guid             VARCHAR NOT NULL PRIMARY KEY,
	block_hash       VARCHAR NOT NULL REFERENCES l1_block_headers(hash),
    transaction_hash VARCHAR NOT NULL,
    event_signature  VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0)
);

CREATE TABLE IF NOT EXISTS l2_contract_events (
    guid             VARCHAR NOT NULL PRIMARY KEY,
	block_hash       VARCHAR NOT NULL REFERENCES l2_block_headers(hash),
    transaction_hash VARCHAR NOT NULL,
    event_signature  VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0)
);

-- Tables that index finalization markers for L2 blocks.

CREATE TABLE IF NOT EXISTS legacy_state_batches (
	index         INTEGER NOT NULL PRIMARY KEY,
	root          VARCHAR NOT NULL,
	size          INTEGER NOT NULL,
	prev_total    INTEGER NOT NULL,

    l1_contract_event_guid VARCHAR REFERENCES l1_contract_events(guid)
);

CREATE TABLE IF NOT EXISTS output_proposals (
    output_root     VARCHAR NOT NULL PRIMARY KEY,

    l2_output_index UINT256,
    l2_block_number UINT256,

    l1_contract_event_guid VARCHAR REFERENCES l1_contract_events(guid)
);

/**
 * BRIDGING DATA
 */

CREATE TABLE IF NOT EXISTS deposits (
	guid                 VARCHAR PRIMARY KEY NOT NULL,

    -- Event causing the deposit
    initiated_l1_event_guid VARCHAR NOT NULL REFERENCES l1_contract_events(guid),
    sent_message_nonce      UINT256 UNIQUE,

    -- Finalization marker for the deposit
    finalized_l2_event_guid VARCHAR REFERENCES l2_contract_events(guid),

    -- Deposit information (do we need indexes on from/to?)
	from_address     VARCHAR NOT NULL,

	to_address       VARCHAR NOT NULL,
	l1_token_address VARCHAR NOT NULL,
	l2_token_address VARCHAR NOT NULL,
	amount           UINT256,
	data             VARCHAR NOT NULL,
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0)
);

CREATE TABLE IF NOT EXISTS withdrawals (
	guid                VARCHAR PRIMARY KEY NOT NULL,

    -- Event causing this withdrawal
    initiated_l2_event_guid VARCHAR NOT NULL REFERENCES l2_contract_events(guid),
    sent_message_nonce      UINT256 UNIQUE,

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
	amount           UINT256,
	data             VARCHAR NOT NULL,
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0)
);
