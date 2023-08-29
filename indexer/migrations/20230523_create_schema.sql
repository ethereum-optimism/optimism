
CREATE DOMAIN UINT256 AS NUMERIC
    CHECK (VALUE >= 0 AND VALUE < 2^256 and SCALE(VALUE) = 0);

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

CREATE TABLE IF NOT EXISTS l2_block_headers (
    -- Searchable fields
    hash        VARCHAR PRIMARY KEY,
    parent_hash VARCHAR NOT NULL UNIQUE,
    number      UINT256 NOT NULL UNIQUE,
    timestamp   INTEGER NOT NULL UNIQUE CHECK (timestamp > 0),

    -- Raw Data
    rlp_bytes VARCHAR NOT NULL
);

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
    rlp_bytes VARCHAR NOT NULL
);

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
    rlp_bytes VARCHAR NOT NULL
);

-- Tables that index finalization markers for L2 blocks.

CREATE TABLE IF NOT EXISTS legacy_state_batches (
    index      INTEGER PRIMARY KEY,
    root       VARCHAR NOT NULL UNIQUE,
    size       INTEGER NOT NULL,
    prev_total INTEGER NOT NULL,

    state_batch_appended_guid VARCHAR NOT NULL UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS output_proposals (
    output_root     VARCHAR PRIMARY KEY,
    l2_output_index UINT256 NOT NULL UNIQUE,
    l2_block_number UINT256 NOT NULL UNIQUE,

    output_proposed_guid VARCHAR NOT NULL UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE
);

/**
 * BRIDGING DATA
 */

-- Bridged L1/L2 Tokens
CREATE TABLE IF NOT EXISTS l1_bridged_tokens (
    address        VARCHAR PRIMARY KEY,
    bridge_address VARCHAR NOT NULL,

    name     VARCHAR NOT NULL,
    symbol   VARCHAR NOT NULL,
    decimals INTEGER NOT NULL CHECK (decimals >= 0 AND decimals <= 18)
);
CREATE TABLE IF NOT EXISTS l2_bridged_tokens (
    address        VARCHAR PRIMARY KEY,
    bridge_address VARCHAR NOT NULL,

    -- L1-L2 relationship is 1 to many so this is not necessarily unique
    l1_token_address VARCHAR REFERENCES l1_bridged_tokens(address) ON DELETE CASCADE,

    name     VARCHAR NOT NULL,
    symbol   VARCHAR NOT NULL,
    decimals INTEGER NOT NULL CHECK (decimals >= 0 AND decimals <= 18)
);

-- OptimismPortal/L2ToL1MessagePasser
CREATE TABLE IF NOT EXISTS l1_transaction_deposits (
    source_hash             VARCHAR PRIMARY KEY,
    l2_transaction_hash     VARCHAR NOT NULL UNIQUE,
    initiated_l1_event_guid VARCHAR NOT NULL UNIQUE REFERENCES l1_contract_events(guid) ON DELETE CASCADE,

    -- transaction data
    from_address VARCHAR NOT NULL,
    to_address   VARCHAR NOT NULL,
    amount       UINT256 NOT NULL,
    gas_limit    UINT256 NOT NULL,
    data         VARCHAR NOT NULL,
    timestamp    INTEGER NOT NULL CHECK (timestamp > 0)
);
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

-- StandardBridge
CREATE TABLE IF NOT EXISTS l1_bridge_deposits (
    transaction_source_hash   VARCHAR PRIMARY KEY REFERENCES l1_transaction_deposits(source_hash) ON DELETE CASCADE,
    cross_domain_message_hash VARCHAR NOT NULL UNIQUE REFERENCES l1_bridge_messages(message_hash) ON DELETE CASCADE,

    -- Deposit information
    from_address         VARCHAR NOT NULL,
    to_address           VARCHAR NOT NULL,
    local_token_address  VARCHAR NOT NULL, -- REFERENCES l1_bridged_tokens(address), uncomment me in future pr
    remote_token_address VARCHAR NOT NULL, -- REFERENCES l2_bridged_tokens(address), uncomment me in future pr
    amount               UINT256 NOT NULL,
    data                 VARCHAR NOT NULL,
    timestamp            INTEGER NOT NULL CHECK (timestamp > 0)
);
CREATE TABLE IF NOT EXISTS l2_bridge_withdrawals (
    transaction_withdrawal_hash VARCHAR PRIMARY KEY REFERENCES l2_transaction_withdrawals(withdrawal_hash) ON DELETE CASCADE,
    cross_domain_message_hash   VARCHAR NOT NULL UNIQUE REFERENCES l2_bridge_messages(message_hash) ON DELETE CASCADE,

    -- Withdrawal information
    from_address         VARCHAR NOT NULL,
    to_address           VARCHAR NOT NULL,
    local_token_address  VARCHAR NOT NULL, -- REFERENCES l2_bridged_tokens(address), uncomment me in future pr
    remote_token_address VARCHAR NOT NULL, -- REFERENCES l1_bridged_tokens(address), uncomment me in future pr
    amount               UINT256 NOT NULL,
    data                 VARCHAR NOT NULL,
    timestamp            INTEGER NOT NULL CHECK (timestamp > 0)
);
