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


CREATE OR REPLACE VIEW next_queued_geth_submission
AS

SELECT r.*, b.block_timestamp
FROM l1_rollup_tx r
  INNER JOIN l1_tx t ON r.l1_tx_hash = t.tx_hash
  INNER JOIN l1_block b ON b.block_number = t.block_number
WHERE
  r.geth_submission_queue_index = (SELECT MIN(queue_index)
                                   FROM geth_submission_queue
                                   WHERE status = 'QUEUED')
ORDER BY r.index_within_submission ASC
;


CREATE OR REPLACE VIEW next_verification_batch
AS

SELECT l1.batch_number, l1.batch_index, l1.state_root as l1_root, l2.state_root as geth_root
FROM l1_state_root l1
  LEFT OUTER JOIN l2_tx_output l2
      ON l1.batch_number = l2.occ_batch_number
         AND l1.batch_index = l2.occ_batch_index
WHERE
  l1.batch_number = (SELECT MIN(batch_number)
                     FROM l1_state_root_batch
                     WHERE status = 'UNVERIFIED')
  AND (SELECT COUNT(*)
       FROM l1_state_root_batch
       WHERE status = 'FRAUDULENT' OR status = 'REMOVED'
      ) = 0   -- there is no next_verification_batch if we're in a fraud workflow.
ORDER BY l1.batch_index ASC
;