package messagefailure

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/FangcunMount/reliable-messaging/transport"
	driver "github.com/go-sql-driver/mysql"
)

// RecordRocketMQ commits a provider-aware failure fact. Historical NSQ records
// remain in their original table and are never relabelled or silently replayed.
func (a *Audit) RecordRocketMQ(ctx context.Context, f transport.Failure) error {
	m := f.Message
	if a == nil || a.db == nil || f.Provider != "rocketmq" || m.Provider != "rocketmq" || m.Topic == "" || m.Channel == "" || m.TransportID == "" || m.PhysicalTopic != transport.PhysicalTopic(m.Topic) || len(m.Wire) == 0 || m.Attempts < 1 || f.Cause == "" {
		return errors.New("invalid RocketMQ terminal failure")
	}
	metadata, err := json.Marshal(m.Metadata)
	if err != nil {
		return err
	}
	// Malformed input can lack an application ID. Retain that absence and use
	// the physical ID only to distinguish the audit fact, never as business ID.
	identityText := f.Provider + "\x00" + m.Topic + "\x00" + m.Channel + "\x00" + m.ID
	if m.ID == "" {
		identityText += "\x00" + m.TransportID
	}
	identity := sha256.Sum256([]byte(identityText))
	payload := sha256.Sum256(m.Payload)
	meta := sha256.Sum256(metadata)
	wire := sha256.Sum256(m.Wire)
	now := time.Now().UnixMilli()
	_, err = a.db.ExecContext(ctx, `INSERT INTO iam_message_failure_audit(identity_hash,provider,physical_topic,original_wire,wire_hash,topic,channel_name,application_id,first_transport_id,last_transport_id,metadata_json,metadata_hash,payload,payload_hash,first_cause,last_cause,attempts,source_timestamp_ns,first_seen_unix_ms,last_seen_unix_ms) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, identity[:], f.Provider, m.PhysicalTopic, m.Wire, wire[:], m.Topic, m.Channel, m.ID, m.TransportID, m.TransportID, metadata, meta[:], m.Payload, payload[:], f.Cause, f.Cause, m.Attempts, m.Timestamp, now, now)
	if err == nil {
		return nil
	}
	var duplicate *driver.MySQLError
	if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
		return err
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var id uint64
	var provider, physical, topic, channel, application string
	var oldPayload, oldMeta, oldWire []byte
	err = tx.QueryRowContext(ctx, `SELECT id,provider,physical_topic,topic,channel_name,application_id,payload_hash,metadata_hash,wire_hash FROM iam_message_failure_audit WHERE identity_hash=? FOR UPDATE`, identity[:]).Scan(&id, &provider, &physical, &topic, &channel, &application, &oldPayload, &oldMeta, &oldWire)
	if err != nil {
		return err
	}
	if provider != f.Provider || physical != m.PhysicalTopic || topic != m.Topic || channel != m.Channel || application != m.ID || !bytes.Equal(oldPayload, payload[:]) || !bytes.Equal(oldMeta, meta[:]) || !bytes.Equal(oldWire, wire[:]) {
		return errors.New("RocketMQ failure identity reused with different immutable content")
	}
	_, err = tx.ExecContext(ctx, `UPDATE iam_message_failure_audit SET last_transport_id=?,last_cause=?,attempts=?,source_timestamp_ns=?,seen_count=seen_count+1,last_seen_unix_ms=? WHERE id=?`, m.TransportID, f.Cause, m.Attempts, m.Timestamp, now, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// CheckRocketMQAudit is read-only. Schema installation stays in IAM db-ops.
func CheckRocketMQAudit(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT provider,physical_topic,original_wire,wire_hash FROM iam_message_failure_audit LIMIT 0")
	if err != nil {
		return err
	}
	return rows.Close()
}
