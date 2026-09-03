-- kona-chainview: the consequences of chain facts, maintained incrementally.
--
-- Every table is a plain Z-set: the host inserts (+1) and retracts (-1) rows.
-- "Current status" rows are replaced by retracting the previous row. Only
-- l2_safe_blocks declares LATENESS; l1_blocks is bounded by host pruning and

-- Canonical contiguous L1 window [floor, tip]. Host pushes +1 on append and
-- backfill, (-1, +1) pairs on reorg, -1 when pruning below the floor.
CREATE TABLE l1_blocks (
  number BIGINT NOT NULL,
  hash BINARY(32) NOT NULL,
  parent_hash BINARY(32) NOT NULL,
  ts BIGINT NOT NULL
);

-- One current row per kind: 'head' (tracker tip), 'safe', 'finalized' (pollers),
-- 'current' (derivation origin).
CREATE TABLE l1_status (
  kind VARCHAR NOT NULL,
  number BIGINT NOT NULL,
  hash BINARY(32) NOT NULL,
  parent_hash BINARY(32) NOT NULL,
  ts BIGINT NOT NULL
);

-- One current row per kind from EngineState:
-- 'unsafe', 'local_safe', 'safe', 'finalized'.
CREATE TABLE l2_status (
  kind VARCHAR NOT NULL,
  number BIGINT NOT NULL,
  hash BINARY(32) NOT NULL,
  parent_hash BINARY(32) NOT NULL,
  ts BIGINT NOT NULL,
  l1_origin_number BIGINT NOT NULL,
  l1_origin_hash BINARY(32) NOT NULL,
  seq_num BIGINT NOT NULL
);

-- Engine-confirmed derived L2 block paired with the L1 block it was derived from.
-- LATENESS >= seq_window_size + channel_timeout, the largest backward jump of
-- derived_from after a pipeline reset.
CREATE TABLE l2_safe_blocks (
  seq BIGINT NOT NULL,
  l2_number BIGINT NOT NULL,
  l2_hash BINARY(32) NOT NULL,
  l2_parent_hash BINARY(32) NOT NULL,
  l2_ts BIGINT NOT NULL,
  l1_origin_number BIGINT NOT NULL,
  l1_origin_hash BINARY(32) NOT NULL,
  seq_num BIGINT NOT NULL,
  derived_from_number BIGINT NOT NULL LATENESS 4096,
  derived_from_hash BINARY(32) NOT NULL
);

-- The unsafe-block signer read from the SystemConfig contract at an L1 block: one current
-- row, replaced by the host at every new L1 head.
CREATE TABLE unsafe_block_signer (
  l1_number BIGINT NOT NULL,
  l1_hash BINARY(32) NOT NULL,
  signer BINARY(20) NOT NULL
);

-- Single-row helpers (COALESCE keeps them non-empty).
CREATE LOCAL VIEW finalized_l1 AS
  SELECT COALESCE(MAX(number), -1) AS number FROM l1_status WHERE kind = 'finalized';
CREATE LOCAL VIEW engine_safe_l2 AS
  SELECT COALESCE(MAX(number), -1) AS number FROM l2_status WHERE kind = 'safe';
CREATE LOCAL VIEW engine_finalized_l2 AS
  SELECT COALESCE(MAX(number), -1) AS number FROM l2_status WHERE kind = 'finalized';

-- Reorg-aware filter: a row is dropped only when the tracked L1 hash at its
-- height differs; heights below the floor (<= finalized) are canonical by L1
-- finality.
CREATE VIEW l2_safe_canonical AS
SELECT s.* FROM l2_safe_blocks s
WHERE NOT EXISTS (
  SELECT 1 FROM l1_blocks b
  WHERE b.number = s.derived_from_number AND b.hash <> s.derived_from_hash
);

-- SafeDB parity: per derived-from L1 block, the most recently asserted safe block.
CREATE VIEW safe_head_updates AS
SELECT derived_from_number AS l1_number,
       ARG_MAX(derived_from_hash, seq) AS l1_hash,
       ARG_MAX(l2_number, seq) AS l2_number,
       ARG_MAX(l2_hash, seq) AS l2_hash
FROM l2_safe_canonical
GROUP BY derived_from_number;

-- The single command view: empty when nothing new is finalizable; otherwise the
-- newest canonical derived block that is derived from at or below finalized L1
-- and at or below the engine's safe head, provided it is above the engine's
-- finalized head.
CREATE LOCAL VIEW finalized_candidate AS
SELECT ARG_MAX(s.l2_number, s.seq) AS l2_number,
       ARG_MAX(s.l2_hash, s.seq) AS l2_hash,
       ARG_MAX(s.derived_from_number, s.seq) AS derived_from_number,
       ARG_MAX(s.derived_from_hash, s.seq) AS derived_from_hash
FROM l2_safe_canonical s
CROSS JOIN finalized_l1 f
CROSS JOIN engine_safe_l2 es
WHERE s.derived_from_number <= f.number
  AND s.l2_number <= es.number;

CREATE VIEW finalized_l2 AS
SELECT c.l2_number, c.l2_hash, c.derived_from_number, c.derived_from_hash
FROM finalized_candidate c
CROSS JOIN engine_finalized_l2 ef
WHERE c.l2_number IS NOT NULL
  AND c.l2_number > ef.number;

CREATE VIEW current_signer AS
SELECT signer, l1_number FROM unsafe_block_signer;
