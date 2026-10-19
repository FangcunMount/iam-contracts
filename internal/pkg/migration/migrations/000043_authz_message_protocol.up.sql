-- Explicit host protocol selection; no SDK evidence table/codec is rewritten.
-- Installation preserves the old writer. The reviewed offline cutover selects
-- transaction only after writer exclusion and legacy publication drain.
CREATE TABLE iam_authz_message_protocol (
 owner VARBINARY(128) NOT NULL PRIMARY KEY,
 protocol_type VARBINARY(32) NOT NULL,
 store_id VARBINARY(128) NOT NULL DEFAULT '',
 writer_epoch VARBINARY(128) NOT NULL DEFAULT '',
 route_revision VARBINARY(128) NOT NULL DEFAULT '',
 generation BIGINT NOT NULL DEFAULT 1,
 schema_version INT NOT NULL DEFAULT 1,
 CONSTRAINT chk_iam_message_protocol_type CHECK (protocol_type IN ('outbox','transaction')),
 CONSTRAINT chk_iam_message_protocol_generation CHECK (generation BETWEEN 1 AND 9223372036854775807),
 CONSTRAINT chk_iam_message_protocol_binding CHECK (protocol_type <> 'transaction' OR (OCTET_LENGTH(store_id)>0 AND OCTET_LENGTH(writer_epoch)>0 AND route_revision='m3-1'))
) ENGINE=InnoDB;
INSERT INTO iam_authz_message_protocol(owner,protocol_type) VALUES ('iam.mysql','outbox');
