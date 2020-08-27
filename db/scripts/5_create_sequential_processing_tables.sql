\connect rollup;

CREATE TABLE sequential_processing (
   id BIGSERIAL NOT NULL,
   sequence_key TEXT NOT NULL,
   sequence_number BIGSERIAL NOT NULL,
   data_to_process TEXT NOT NULL,
   processed BOOLEAN NOT NULL DEFAULT FALSE,
   created TIMESTAMP NOT NULL DEFAULT NOW(),
   PRIMARY KEY (id),
   UNIQUE (sequence_key, sequence_number)
);

/** Rollback script:
DROP TABLE sequential_processing;
*/

