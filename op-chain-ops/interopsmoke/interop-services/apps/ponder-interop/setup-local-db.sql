CREATE USER "local-db-user";

CREATE ROLE read_only;
CREATE ROLE read_write;

GRANT read_write TO postgres;
GRANT read_write TO "local-db-user";

CREATE DATABASE "ponder-interop";

\c "ponder-interop"

-- read_only
GRANT CONNECT ON DATABASE "ponder-interop" TO read_only;

GRANT USAGE ON SCHEMA public TO read_only;

GRANT SELECT ON ALL TABLES IN SCHEMA public TO read_only;

ALTER DEFAULT PRIVILEGES GRANT SELECT ON TABLES TO read_only;

-- read_write
GRANT CREATE, CONNECT ON DATABASE "ponder-interop" TO read_write;

GRANT USAGE, CREATE ON SCHEMA public TO read_write;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO read_write;

ALTER DEFAULT PRIVILEGES GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO read_write;

GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO read_write;

ALTER DEFAULT PRIVILEGES GRANT USAGE ON SEQUENCES TO read_write;

ALTER DEFAULT PRIVILEGES FOR ROLE read_write GRANT SELECT ON TABLES TO read_only;