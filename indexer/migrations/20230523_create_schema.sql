DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'uint256') THEN
        CREATE DOMAIN UINT256 AS NUMERIC
            CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
    ELSE
        -- To remain backwards compatible, drop the old constraint and re-add.
        ALTER DOMAIN UINT256 DROP CONSTRAINT uint256_check;
        ALTER DOMAIN UINT256 ADD
            CHECK (VALUE >= 0 AND VALUE < POWER(CAST(2 AS NUMERIC), CAST(256 AS NUMERIC)) AND SCALE(VALUE) = 0);
    END IF;
END $$;

/**
 * BLOCK DATA
 */

CREATE TABLE IF NOT EXISTS l1_block_headers (
    -- Searchable fields
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE,
    timestamp   INTEGER NOT NULL UNIQUE CHECK (timestamp > 0),

    -- Raw Data
    rlp_bytes VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS l1_block_headers_timestamp ON l1_block_headers(timestamp);
CREATE INDEX IF NOT EXISTS l1_block_headers_number ON l1_block_headers(number);

CREATE TABLE IF NOT EXISTS l2_block_headers (
    -- Searchable fields
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE,
    timestamp   INTEGER NOT NULL,

    -- Raw Data
    rlp_bytes VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS l2_block_headers_timestamp ON l2_block_headers(timestamp);
CREATE INDEX IF NOT EXISTS l2_block_headers_number ON l2_block_headers(number);

/**
 * EVENT DATA
 */

CREATE TABLE IF NOT EXISTS l1_contract_events (
    -- Searchable fields
    guid             VARCHAR PRIMARY KEY,
    block_hash       VARCHAR NOT NULL REFERENCES l1_block_headers(hash) ON DELETE CASCADE,
    contract_address VARCHAR NOT NULL,
    transaction_hash VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    event_signature  VARCHAR NOT NULL, -- bytes32(0x0) when topics are missing
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0),

    -- Raw Data
    rlp_bytes VARCHAR NOT NULL,

    UNIQUE(block_hash, log_index)
);
CREATE INDEX IF NOT EXISTS l1_contract_events_timestamp ON l1_contract_events(timestamp);
CREATE INDEX IF NOT EXISTS l1_contract_events_block_hash ON l1_contract_events(block_hash);
CREATE INDEX IF NOT EXISTS l1_contract_events_event_signature ON l1_contract_events(event_signature);
CREATE INDEX IF NOT EXISTS l1_contract_events_contract_address ON l1_contract_events(contract_address);

CREATE TABLE IF NOT EXISTS l2_contract_events (
    -- Searchable fields
    guid             VARCHAR PRIMARY KEY,
    block_hash       VARCHAR NOT NULL REFERENCES l2_block_headers(hash) ON DELETE CASCADE,
    contract_address VARCHAR NOT NULL,
    transaction_hash VARCHAR NOT NULL,
    log_index        INTEGER NOT NULL,
    event_signature  VARCHAR NOT NULL, -- bytes32(0x0) when topics are missing
    timestamp        INTEGER NOT NULL CHECK (timestamp > 0),

    -- Raw Data
    rlp_bytes VARCHAR NOT NULL,

    UNIQUE(block_hash, log_index)
);
CREATE INDEX IF NOT EXISTS l2_contract_events_timestamp ON l2_contract_events(timestamp);
CREATE INDEX IF NOT EXISTS l2_contract_events_block_hash ON l2_contract_events(block_hash);
CREATE INDEX IF NOT EXISTS l2_contract_events_event_signature ON l2_contract_events(event_signature);
CREATE INDEX IF NOT EXISTS l2_contract_events_contract_address ON l2_contract_events(contract_address);

/**
 * BRIDGING DATA
 */

-- OptimismPortal/L2ToL1MessagePasser
CREATE TABLE IF NOT EXISTS l1_transaction_deposits (
    source_hash             VARCHAR PRIMARY KEY,
    l2_transaction_hash     VARCHAR NOT NULL UNIQUE,
    initiated_l1_event_guid VARCHAR NOT NULL UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,

    -- transaction data. NOTE: `to_address` is the recipient of funds transferred in value field of the
    -- L2 deposit transaction and not the amount minted on L1 from the source address. Hence the `amount`
    -- column in this table does NOT indicate the amount transferred to the recipient but instead funds
    -- bridged from L1 by the `from_address`.
    from_address VARCHAR NOT NULL,
    to_address   VARCHAR NOT NULL,

    -- This refers to the amount MINTED on L2 (msg.value of the L1 transaction). Important distinction from
    -- the `value` field of the deposit transaction which simply is the value transferred to specified recipient.
    amount       UINT256 NOT NULL,

    gas_limit    UINT256 NOT NULL,
    data         VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l1_transaction_deposits_timestamp ON l1_transaction_deposits(timestamp);
CREATE INDEX IF NOT EXISTS l1_transaction_deposits_initiated_l1_event_guid ON l1_transaction_deposits(initiated_l1_event_guid);
CREATE INDEX IF NOT EXISTS l1_transaction_deposits_from_address ON l1_transaction_deposits(from_address);

CREATE TABLE IF NOT EXISTS l2_transaction_withdrawals (
    withdrawal_hash         VARCHAR PRIMARY KEY,
    nonce                   UINT256 NOT NULL UNIQUE,
    initiated_l2_event_guid VARCHAR NOT NULL UNIQUE REFERENCES l2_contract_events(guid) ON DELETE CASCADE,

    -- Multistep (bedrock) process of a withdrawal
    proven_l1_event_guid    VARCHAR UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,
    finalized_l1_event_guid VARCHAR UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,
    succeeded               BOOLEAN,

    -- transaction data
    from_address VARCHAR NOT NULL,
    to_address   VARCHAR NOT NULL,
    amount       UINT256 NOT NULL,
    gas_limit    UINT256 NOT NULL,
    data         VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l2_transaction_withdrawals_timestamp ON l2_transaction_withdrawals(timestamp);
CREATE INDEX IF NOT EXISTS l2_transaction_withdrawals_initiated_l2_event_guid ON l2_transaction_withdrawals(initiated_l2_event_guid);
CREATE INDEX IF NOT EXISTS l2_transaction_withdrawals_proven_l1_event_guid ON l2_transaction_withdrawals(proven_l1_event_guid);
CREATE INDEX IF NOT EXISTS l2_transaction_withdrawals_finalized_l1_event_guid ON l2_transaction_withdrawals(finalized_l1_event_guid);
CREATE INDEX IF NOT EXISTS l2_transaction_withdrawals_from_address ON l2_transaction_withdrawals(from_address);

-- CrossDomainMessenger
CREATE TABLE IF NOT EXISTS l1_bridge_messages(
    message_hash            VARCHAR PRIMARY KEY,
    nonce                   UINT256 NOT NULL UNIQUE,
    transaction_source_hash VARCHAR NOT NULL UNIQUE REFERENCES l1_transaction_deposits(source_hash) ON DELETE CASCADE,

    sent_message_event_guid    VARCHAR NOT NULL UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,
    relayed_message_event_guid VARCHAR UNIQUE REFERENCES l2_contract_events(guid) ON DELETE CASCADE,

    -- sent message
    from_address VARCHAR NOT NULL,
    to_address   VARCHAR NOT NULL,
    amount       UINT256 NOT NULL,
    gas_limit    UINT256 NOT NULL,
    data         VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l1_bridge_messages_timestamp ON l1_bridge_messages(timestamp);
CREATE INDEX IF NOT EXISTS l1_bridge_messages_transaction_source_hash ON l1_bridge_messages(transaction_source_hash);
CREATE INDEX IF NOT EXISTS l1_bridge_messages_transaction_sent_message_event_guid ON l1_bridge_messages(sent_message_event_guid);
CREATE INDEX IF NOT EXISTS l1_bridge_messages_transaction_relayed_message_event_guid ON l1_bridge_messages(relayed_message_event_guid);
CREATE INDEX IF NOT EXISTS l1_bridge_messages_from_address ON l1_bridge_messages(from_address);

CREATE TABLE IF NOT EXISTS l2_bridge_messages(
    message_hash                VARCHAR PRIMARY KEY,
    nonce                       UINT256 NOT NULL UNIQUE,
    transaction_withdrawal_hash VARCHAR NOT NULL UNIQUE REFERENCES l2_transaction_withdrawals(withdrawal_hash) ON DELETE CASCADE,

    sent_message_event_guid    VARCHAR NOT NULL UNIQUE REFERENCES l2_contract_events(guid) ON DELETE CASCADE,
    relayed_message_event_guid VARCHAR UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,

    -- sent message
    from_address VARCHAR NOT NULL,
    to_address   VARCHAR NOT NULL,
    amount       UINT256 NOT NULL,
    gas_limit    UINT256 NOT NULL,
    data         VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l2_bridge_messages_timestamp ON l2_bridge_messages(timestamp);
CREATE INDEX IF NOT EXISTS l2_bridge_messages_transaction_withdrawal_hash ON l2_bridge_messages(transaction_withdrawal_hash);
CREATE INDEX IF NOT EXISTS l2_bridge_messages_transaction_sent_message_event_guid ON l2_bridge_messages(sent_message_event_guid);
CREATE INDEX IF NOT EXISTS l2_bridge_messages_transaction_relayed_message_event_guid ON l2_bridge_messages(relayed_message_event_guid);
CREATE INDEX IF NOT EXISTS l2_bridge_messages_from_address ON l2_bridge_messages(from_address);

/**
 * Since the CDM uses the latest versioned message hash when emitting the `RelayedMessage` event, we need
 * to keep track of all of the future versions of message hashes such that legacy messages can be queried
 * queried for when relayed on L1 
 *
 * As new the CDM is updated with new versions, we need to ensure that there's a better way to correlate message between
 * chains (adding the message nonce to the RelayedMessage event) or continue to add columns to this table and migrate
 * unrelayed messages such that finalization logic can handle switching between the varying versioned message hashes
 */
CREATE TABLE IF NOT EXISTS l2_bridge_message_versioned_message_hashes(
    message_hash     VARCHAR PRIMARY KEY NOT NULL UNIQUE REFERENCES l2_bridge_messages(message_hash),

    -- only filled in if `message_hash` is for a v0 message
    v1_message_hash  VARCHAR UNIQUE
);

-- StandardBridge
CREATE TABLE IF NOT EXISTS l1_bridge_deposits (
    transaction_source_hash   VARCHAR PRIMARY KEY REFERENCES l1_transaction_deposits(source_hash) ON DELETE CASCADE,
    cross_domain_message_hash VARCHAR NOT NULL UNIQUE REFERENCES l1_bridge_messages(message_hash) ON DELETE CASCADE,

    -- Deposit information
    from_address         VARCHAR NOT NULL,
    to_address           VARCHAR NOT NULL,
    local_token_address  VARCHAR NOT NULL,
    remote_token_address VARCHAR NOT NULL,
    amount               UINT256 NOT NULL,
    data                 VARCHAR NOT NULL,
    timestamp            INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l1_bridge_deposits_timestamp ON l1_bridge_deposits(timestamp);
CREATE INDEX IF NOT EXISTS l1_bridge_deposits_cross_domain_message_hash ON l1_bridge_deposits(cross_domain_message_hash);
CREATE INDEX IF NOT EXISTS l1_bridge_deposits_from_address ON l1_bridge_deposits(from_address);

CREATE TABLE IF NOT EXISTS l2_bridge_withdrawals (
    transaction_withdrawal_hash VARCHAR PRIMARY KEY REFERENCES l2_transaction_withdrawals(withdrawal_hash) ON DELETE CASCADE,
    cross_domain_message_hash   VARCHAR NOT NULL UNIQUE REFERENCES l2_bridge_messages(message_hash) ON DELETE CASCADE,

    -- Withdrawal information
    from_address         VARCHAR NOT NULL,
    to_address           VARCHAR NOT NULL,
    local_token_address  VARCHAR NOT NULL,
    remote_token_address VARCHAR NOT NULL,
    amount               UINT256 NOT NULL,
    data                 VARCHAR NOT NULL,
    timestamp            INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE INDEX IF NOT EXISTS l2_bridge_withdrawals_timestamp ON l2_bridge_withdrawals(timestamp);
CREATE INDEX IF NOT EXISTS l2_bridge_withdrawals_cross_domain_message_hash ON l2_bridge_withdrawals(cross_domain_message_hash);
CREATE INDEX IF NOT EXISTS l2_bridge_withdrawals_from_address ON l2_bridge_withdrawals(from_address);
