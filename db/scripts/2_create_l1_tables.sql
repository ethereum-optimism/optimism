
\connect rollup;

CREATE TABLE l1_block (
  id BIGSERIAL NOT NULL,
  block_hash CHARACTER(66) NOT NULL,
  parent_hash CHARACTER(66) NOT NULL,
  block_number BIGINT NOT NULL,
  block_timestamp BIGINT NOT NULL,
  gas_limit NUMERIC(78) NOT NULL,
  gas_used NUMERIC(78) NOT NULL,
  processed BOOLEAN NOT NULL DEFAULT FALSE,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (block_hash),
  UNIQUE (block_number)
);


CREATE TABLE l1_tx (
  id BIGSERIAL NOT NULL,
  block_number BIGINT NOT NULL,
  tx_hash CHARACTER(66) NOT NULL,
  from_address CHARACTER(42) NOT NULL,
  to_address CHARACTER(42) NOT NULL,
  nonce NUMERIC(78) NOT NULL,
  gas_limit NUMERIC(78) NOT NULL,
  gas_price NUMERIC(78) NOT NULL,
  calldata TEXT NOT NULL,
  signature CHARACTER(130) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (tx_hash),
  FOREIGN KEY (block_number) REFERENCES l1_block(block_number)
);

CREATE TABLE l1_tx_batch (
  id BIGSERIAL NOT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  batch_number BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'BATCHED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash)
);
CREATE INDEX l1_tx_batch_status_idx ON l1_tx_batch USING btree (status);


CREATE TABLE l1_state_root_batch (
  id BIGSERIAL NOT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  batch_number BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'BATCHED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash)
);
CREATE INDEX l1_state_root_batch_status_idx ON l1_state_root_batch USING btree (status);



CREATE TABLE l1_state_root (
  id BIGSERIAL NOT NULL,
  state_root character(66) NOT NULL,
  batch_number BIGINT NOT NULL,
  batch_index INT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (state_root),
  FOREIGN KEY (batch_number) REFERENCES l1_state_root_batch (batch_number)
);



CREATE TABLE rollup_tx (
  id BIGSERIAL NOT NULL,
  sender CHARACTER(42) DEFAULT NULL,
  l1_message_sender CHARACTER(42) DEFAULT NULL,
  target CHARACTER(42) DEFAULT NULL,
  calldata TEXT DEFAULT NULL,
  queue_origin SMALLINT NOT NULL,
  nonce NUMERIC(78) DEFAULT NULL,
  gas_limit NUMERIC(78) DEFAULT NULL,
  signature NUMERIC(78) DEFAULT NULL,
  batch_number BIGINT DEFAULT NULL,
  batch_index INT DEFAULT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  l1_tx_index INT NOT NULL,
  l1_tx_log_index INT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash)
)

/* ROLLBACK SCRIPT
   DROP TABLE rollup_tx;
   DROP TABLE l1_state_root;
   DROP TABLE l1_state_root_batch;
   DROP TABLE l1_tx_batch;
   DROP TABLE l1_tx;
   DROP TABLE l1_block;

 */
