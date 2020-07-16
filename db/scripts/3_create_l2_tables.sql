\connect rollup;


CREATE TABLE optimistic_canonical_chain_batch (
  id BIGSERIAL NOT NULL,
  batch_number BIGINT NOT NULL,
  submitted_tx_batch_l1_tx_hash character(66) DEFAULT NULL,
  submitted_root_batch_l1_tx_hash character(66) DEFAULT NULL,
  tx_batch_status TEXT NOT NULL DEFAULT 'QUEUED',
  root_batch_status TEXT NOT NULL DEFAULT 'QUEUED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number)
);
CREATE INDEX optimistic_canonical_chain_batch_tx_status_idx ON optimistic_canonical_chain_batch USING btree (tx_batch_status);
CREATE INDEX optimistic_canonical_chain_batch_root_status_idx ON optimistic_canonical_chain_batch USING btree (root_batch_status);

CREATE TABLE l2_tx_output (
  id BIGSERIAL NOT NULL,
  l1_rollup_tx_id BIGINT DEFAULT NULL,
  occ_batch_number BIGINT DEFAULT NULL,
  occ_batch_index INT DEFAULT NULL,
  block_number BIGINT NOT NULL,
  block_timestamp BIGINT NOT NULL,
  tx_index INT NOT NULL,
  tx_hash CHARACTER(66) NOT NULL,
  sender CHARACTER(42) NOT NULL,
  l1_message_sender CHARACTER(42) NOT NULL,
  target CHARACTER(42) NOT NULL,
  nonce NUMERIC(78) NOT NULL,
  gas_limit NUMERIC(78) NOT NULL,
  gas_price NUMERIC(78) NOT NULL,
  calldata TEXT NOT NULL,
  signature CHARACTER(130) NOT NULL,
  state_root CHARACTER(66) NOT NULL,
  status TEXT NOT NULL DEFAULT 'UNBATCHED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (occ_batch_number) REFERENCES optimistic_canonical_chain_batch(batch_number),
  FOREIGN KEY (l1_rollup_tx_id) REFERENCES l1_rollup_tx(id),
  UNIQUE (tx_hash)
);

CREATE INDEX l2_tx_output_block_idx ON l2_tx_output USING btree (block_number);
CREATE INDEX l2_tx_output_block_timestamp_idx ON l2_tx_output USING btree (block_timestamp);
CREATE INDEX l2_tx_output_state_root_idx ON l2_tx_output USING btree (state_root);

/** ROLLBACK SCRIPT
  DROP TABLE l2_tx_output;
  DROP TABLE optimistic_canonical_chain_batch;
 */