\connect rollup;

CREATE OR REPLACE VIEW unbatched_rollup_tx
AS

SELECT block.block_number, r.*
FROM l1_rollup_tx r
  INNER JOIN l1_tx tx ON r.l1_tx_hash = tx.tx_hash
  INNER JOIN l1_block block ON block.block_number = tx.block_number
WHERE
  r.geth_submission_queue_index IS NULL
  AND block.processed = TRUE
ORDER BY
  block.block_number ASC,
  r.l1_tx_index ASC,
  r.l1_tx_log_index ASC
;




/* TODO: CREATE THE FOLLOWING VIEWS
next_l2_submission_batch

next_l1_verification_batch

next_l2_verification_batch
*/