
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
  tx_index INT NOT NULL,
  tx_hash CHARACTER(66) NOT NULL,
  from_address CHARACTER(42) NOT NULL,
  to_address CHARACTER(42) NOT NULL,
  nonce NUMERIC(78) NOT NULL,
  gas_limit NUMERIC(78) NOT NULL,
  gas_price NUMERIC(78) NOT NULL,
  calldata TEXT NOT NULL,
  signature CHARACTER(132) NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (tx_hash),
  UNIQUE (block_number, tx_index),
  FOREIGN KEY (block_number) REFERENCES l1_block(block_number)
);

CREATE TABLE geth_submission_queue (
  id BIGSERIAL NOT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  queue_index BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'QUEUED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (queue_index),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash)
);
CREATE INDEX geth_submission_queue_status_idx ON geth_submission_queue USING btree (status);


CREATE TABLE l1_rollup_state_root_batch (
  id BIGSERIAL NOT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  batch_number BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'UNVERIFIED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash)
);
CREATE INDEX l1_rollup_state_root_batch_status_idx ON l1_rollup_state_root_batch USING btree (status);



CREATE TABLE l1_rollup_state_root (
  id BIGSERIAL NOT NULL,
  state_root character(66) NOT NULL,
  batch_number BIGINT NOT NULL,
  batch_index INT NOT NULL,
  removed BOOLEAN NOT NULL DEFAULT FALSE,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (batch_number) REFERENCES l1_rollup_state_root_batch (batch_number)
);
CREATE INDEX l1_rollup_state_root_state_root_idx ON l1_rollup_state_root USING btree (state_root);




CREATE TABLE l1_rollup_tx (
  id BIGSERIAL NOT NULL,
  sender CHARACTER(42) DEFAULT NULL,
  l1_message_sender CHARACTER(42) DEFAULT NULL,
  target CHARACTER(42) DEFAULT NULL,
  calldata TEXT DEFAULT NULL,
  queue_origin SMALLINT NOT NULL,
  nonce NUMERIC(78) DEFAULT NULL,
  gas_limit NUMERIC(78) DEFAULT NULL,
  signature CHARACTER(132) DEFAULT NULL,
  geth_submission_queue_index BIGINT DEFAULT NULL,
  index_within_submission INT DEFAULT NULL,
  l1_tx_hash CHARACTER(66) NOT NULL,
  l1_tx_index INT NOT NULL,
  l1_tx_log_index INT NOT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (l1_tx_hash) REFERENCES l1_tx (tx_hash),
  FOREIGN KEY (geth_submission_queue_index) REFERENCES geth_submission_queue(queue_index)
)

/* ROLLBACK SCRIPT
   DROP TABLE l1_rollup_tx;
   DROP TABLE l1_rollup_state_root;
   DROP TABLE l1_rollup_state_root_batch;
   DROP TABLE geth_submission_queue;
   DROP TABLE l1_tx;
   DROP TABLE l1_block;
 */
