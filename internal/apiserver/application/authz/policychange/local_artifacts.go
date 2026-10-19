package policychange

import (
	"bytes"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/message"
	"github.com/FangcunMount/reliable-messaging/transaction"
)

const localPlanContentType = "application/vnd.iam.authz-plan.v1+json"

// LocalArtifacts are frozen before half messages. They remain private SDK plan
// payload, while Broker wire and its original business event remain unchanged.
// No new identity or audit clock may be allocated inside the business root.
type LocalArtifacts struct {
	AuditAt time.Time
	IDs     map[string]meta.ID
}
type artifactEnvelope struct {
	Version int               `json:"version"`
	Event   json.RawMessage   `json:"event"`
	AuditAt string            `json:"audit_at"`
	IDs     map[string]string `json:"ids"`
}

func (a LocalArtifacts) clone() LocalArtifacts {
	ids := make(map[string]meta.ID, len(a.IDs))
	for k, v := range a.IDs {
		ids[k] = v
	}
	return LocalArtifacts{AuditAt: a.AuditAt, IDs: ids}
}
func (a LocalArtifacts) valid() bool {
	if a.AuditAt.IsZero() {
		return false
	}
	seen := map[meta.ID]bool{}
	for k, v := range a.IDs {
		if k == "" || len(k) > 1024 || !utf8.ValidString(k) || v <= 0 || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}

// ArtifactKey frames semantic identity without treating delimiters as identity.
func ArtifactKey(kind string, parts ...string) string {
	values := append([]string{kind}, parts...)
	raw, _ := json.Marshal(values)
	return string(raw)
}
func FreezeVersionPlanWithArtifacts(command CommandIntent, expected int64, operationID, eventID string, at time.Time, logical, physical string, artifacts LocalArtifacts) (transaction.FrozenPlan, error) {
	if !artifacts.valid() {
		return transaction.FrozenPlan{}, ErrInvalidChangePlan
	}
	p, err := FreezeVersionPlan(command, expected, operationID, eventID, at, logical, physical)
	if err != nil {
		return transaction.FrozenPlan{}, err
	}
	in := p.Input()
	m := in.Branches[0].Message.Input()
	envelope := artifactEnvelope{Version: 1, Event: m.Payload, AuditAt: artifacts.AuditAt.Format(time.RFC3339Nano), IDs: map[string]string{}}
	for k, v := range artifacts.IDs {
		envelope.IDs[k] = v.String()
	}
	m.Payload, err = json.Marshal(envelope)
	if err != nil || len(m.Payload) > 256*1024 {
		return transaction.FrozenPlan{}, ErrInvalidChangePlan
	}
	m.ContentType = localPlanContentType
	in.Branches[0].Message, err = message.New(m)
	if err != nil {
		return transaction.FrozenPlan{}, err
	}
	return transaction.NewPlan(in)
}
func decodeLocalArtifacts(in message.Input) ([]byte, *LocalArtifacts, error) {
	if in.ContentType == "application/json" {
		return bytes.Clone(in.Payload), nil, nil
	}
	if in.ContentType != localPlanContentType || len(in.Payload) > 256*1024 {
		return nil, nil, ErrInvalidChangePlan
	}
	var envelope artifactEnvelope
	d := json.NewDecoder(bytes.NewReader(in.Payload))
	d.DisallowUnknownFields()
	if d.Decode(&envelope) != nil || envelope.Version != 1 || envelope.IDs == nil {
		return nil, nil, ErrInvalidChangePlan
	}
	canonical, err := json.Marshal(envelope)
	if err != nil || !bytes.Equal(canonical, in.Payload) {
		return nil, nil, ErrInvalidChangePlan
	}
	at, err := time.Parse(time.RFC3339Nano, envelope.AuditAt)
	if err != nil || at.Format(time.RFC3339Nano) != envelope.AuditAt {
		return nil, nil, ErrInvalidChangePlan
	}
	a := LocalArtifacts{AuditAt: at.Truncate(time.Second), IDs: map[string]meta.ID{}}
	for k, s := range envelope.IDs {
		v, err := meta.ParseID(s)
		if err != nil || v.String() != s {
			return nil, nil, ErrInvalidChangePlan
		}
		a.IDs[k] = v
	}
	if !a.valid() {
		return nil, nil, ErrInvalidChangePlan
	}
	return bytes.Clone(envelope.Event), &a, nil
}
func (p VersionPlan) Artifacts() (LocalArtifacts, bool) {
	if p.artifacts == nil {
		return LocalArtifacts{}, false
	}
	return p.artifacts.clone(), true
}
func (p VersionPlan) EventPayload() []byte { return bytes.Clone(p.eventPayload) }

// ArtifactBuilder is used only after read-only planning has ended and before
// reservation/half preparation. Business roots can only read frozen IDs.
type ArtifactBuilder struct{ artifacts LocalArtifacts }

func NewArtifactBuilder(at time.Time) *ArtifactBuilder {
	return &ArtifactBuilder{LocalArtifacts{AuditAt: at, IDs: map[string]meta.ID{}}}
}
func (b *ArtifactBuilder) ID(key string) (meta.ID, error) {
	if b == nil || b.artifacts.AuditAt.IsZero() || key == "" || len(key) > 1024 || !utf8.ValidString(key) {
		return 0, ErrInvalidChangePlan
	}
	if id, ok := b.artifacts.IDs[key]; ok {
		return id, nil
	}
	id := meta.New()
	b.artifacts.IDs[key] = id
	return id, nil
}
func (b *ArtifactBuilder) Freeze() (LocalArtifacts, error) {
	if b == nil || !b.artifacts.valid() {
		return LocalArtifacts{}, ErrInvalidChangePlan
	}
	return b.artifacts.clone(), nil
}
func (p VersionPlan) CreatedID(key string) (meta.ID, bool) {
	if p.artifacts == nil {
		return 0, false
	}
	id, ok := p.artifacts.IDs[key]
	return id, ok && id > 0
}
func (p VersionPlan) AuditTime() (time.Time, bool) {
	if p.artifacts == nil {
		return time.Time{}, false
	}
	return p.artifacts.AuditAt, !p.artifacts.AuditAt.IsZero()
}

func AllocateKeys(keys ...string) func(*ArtifactBuilder) error {
	keys = append([]string(nil), keys...)
	return func(builder *ArtifactBuilder) error {
		for _, key := range keys {
			if _, err := builder.ID(key); err != nil {
				return err
			}
		}
		return nil
	}
}
