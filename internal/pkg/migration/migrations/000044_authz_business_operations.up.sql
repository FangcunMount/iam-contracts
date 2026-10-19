-- Business idempotency and results, separate from rm_tx protocol evidence.
-- Frozen plans are archival inputs, never a publishing queue or replay permit.
CREATE TABLE iam_authz_operations (
 request_key BINARY(32) NOT NULL PRIMARY KEY,
 operation_id VARBINARY(36) NOT NULL,
 owner VARBINARY(128) NOT NULL,
 store_id VARBINARY(128) NOT NULL,
 writer_epoch VARBINARY(128) NOT NULL,
 identity_key BLOB NOT NULL,
 input_hash BINARY(32) NOT NULL,
 candidate LONGBLOB NOT NULL,
 candidate_hash BINARY(32) NOT NULL,
 execution_token BINARY(32) NOT NULL,
 state VARBINARY(16) NOT NULL,
 lease_until DATETIME(6) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 finalized_at DATETIME(6) NULL,
 guard_version BIGINT NOT NULL DEFAULT 0,
 schema_version INT NOT NULL DEFAULT 1,
 policy_version BIGINT NULL,
 result MEDIUMBLOB NULL,
 result_hash BINARY(32) NULL,
 UNIQUE KEY iam_operation_identity(operation_id),
 INDEX iam_operation_expiry(owner,store_id,state,lease_until,request_key),
 CONSTRAINT chk_iam_operation_state CHECK (state IN ('prepared','committed','aborted')),
 CONSTRAINT chk_iam_operation_finalized CHECK ((state='prepared' AND finalized_at IS NULL) OR (state<>'prepared' AND finalized_at IS NOT NULL)),
 CONSTRAINT chk_iam_operation_guard CHECK (guard_version>=0),
 CONSTRAINT chk_iam_operation_result CHECK ((state='committed' AND policy_version>=1 AND result IS NOT NULL AND result_hash IS NOT NULL) OR (state<>'committed' AND policy_version IS NULL AND result IS NULL AND result_hash IS NULL))
) ENGINE=InnoDB;
