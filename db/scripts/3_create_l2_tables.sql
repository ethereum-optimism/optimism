\connect rollup;


CREATE TABLE canonical_chain_batch (
  id BIGSERIAL NOT NULL,
  batch_number BIGINT NOT NULL,
  submission_tx_hash character(66) DEFAULT NULL,
  status TEXT NOT NULL DEFAULT 'QUEUED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number)
);
CREATE INDEX canonical_chain_batch_status_idx ON canonical_chain_batch USING btree (status);

CREATE TABLE state_commitment_chain_batch (
  id BIGSERIAL NOT NULL,
  batch_number BIGINT NOT NULL,
  submission_tx_hash character(66) DEFAULT NULL,
  status TEXT NOT NULL DEFAULT 'QUEUED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number)
);
CREATE INDEX state_commitment_chain_batch_status_idx ON state_commitment_chain_batch USING btree (status);


CREATE TABLE l2_tx_output (
  id BIGSERIAL NOT NULL,
  block_number BIGINT NOT NULL,
  block_timestamp BIGINT NOT NULL,
  tx_index INT NOT NULL,
  tx_hash CHARACTER(66) NOT NULL,
  sender CHARACTER(42) DEFAULT NULL,
  l1_message_sender CHARACTER(42) DEFAULT NULL,
  target CHARACTER(42) DEFAULT NULL,
  nonce NUMERIC(78) NOT NULL,
  gas_limit NUMERIC(78) NOT NULL,
  gas_price NUMERIC(78) NOT NULL,
  calldata TEXT NOT NULL,
  signature CHARACTER(132) NOT NULL,
  state_root CHARACTER(66) NOT NULL,
  tx_type INT NOT NULL,
  l1_rollup_tx_id BIGINT DEFAULT NULL,
  canonical_chain_batch_number BIGINT DEFAULT NULL,
  canonical_chain_batch_index INT DEFAULT NULL,
  state_commitment_chain_batch_number BIGINT DEFAULT NULL,
  state_commitment_chain_batch_index INT DEFAULT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (canonical_chain_batch_number) REFERENCES canonical_chain_batch(batch_number),
  FOREIGN KEY (state_commitment_chain_batch_number) REFERENCES state_commitment_chain_batch(batch_number),
  FOREIGN KEY (l1_rollup_tx_id) REFERENCES l1_rollup_tx(id),
  UNIQUE (tx_hash)
);

CREATE INDEX l2_tx_output_block_idx ON l2_tx_output USING btree (block_number);
CREATE INDEX l2_tx_output_block_timestamp_idx ON l2_tx_output USING btree (block_timestamp);
CREATE INDEX l2_tx_output_state_root_idx ON l2_tx_output USING btree (state_root);

/** ROLLBACK SCRIPT
  DROP TABLE l2_tx_output;
  DROP TABLE canonical_chain_batch;
  DROP TABLE state_commitment_chain_batch;
 */
