\connect rollup;


CREATE TABLE l2_tx_batch (
  id BIGSERIAL NOT NULL,
  batch_number BIGINT NOT NULL,
  status TEXT NOT NULL DEFAULT 'BATCHED',
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  UNIQUE (batch_number)
);
CREATE INDEX l2_tx_batch_status_idx ON l2_tx_batch USING btree (status);

CREATE TABLE l2_tx (
  id BIGSERIAL NOT NULL,
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
  batch_number BIGINT DEFAULT NULL,
  created TIMESTAMP NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  FOREIGN KEY (batch_number) REFERENCES l2_tx_batch(batch_number),
  UNIQUE (tx_hash)
);

CREATE INDEX l2_tx_block_idx ON l2_tx USING btree (block_number);
CREATE INDEX l2_tx_block_timestamp_idx ON l2_tx USING btree (block_timestamp);
CREATE INDEX l2_tx_state_root_idx ON l2_tx USING btree (state_root);

/** ROLLBACK SCRIPT
  DROP TABLE l2_tx;
  DROP TABLE l2_tx_batch;
 */