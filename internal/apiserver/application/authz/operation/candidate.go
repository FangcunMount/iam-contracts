// Package operation owns IAM business-operation identity, result and query policy.
// It never sends messages or authorizes a business replay. SDK transaction state
// remains separate from the host's idempotency/result journal.
package operation

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	change "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/policychange"
	domain "github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/authorization"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/authz/subject"
	"github.com/FangcunMount/iam/v5/internal/pkg/meta"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/google/uuid"
)

var (
	ErrInvalid  = errors.New("invalid frozen IAM business operation")
	ErrConflict = errors.New("IAM operation key already binds different business input")
	ErrNotFound = errors.New("IAM operation not found")
	ErrStale    = errors.New("IAM business operation no longer grants local execution")
)

const maxCandidateBytes = 2 * 1024 * 1024

// Permission records the original permission checked by read planning. The
// status-query service rechecks it against the current authenticated principal.
type Permission = change.Permission

type candidateData struct {
	Schema      int                 `json:"schema"`
	ID          string              `json:"operation_id"`
	Identity    string              `json:"identity"`
	InputHash   [32]byte            `json:"input_hash"`
	Binding     transaction.Binding `json:"binding"`
	Expected    int64               `json:"expected_policy_version"`
	Permissions []Permission        `json:"permissions"`
	Plan        []byte              `json:"sdk_plan,omitempty"`
	NoopResult  []byte              `json:"no_op_result,omitempty"`
}

// Candidate freezes the complete first plan, including no-op intent, before
// competing for the business key. A loser reads the winner even if its own IDs,
// timestamps or wire differ. Its private bytes are never a public API DTO.
type Candidate struct {
	data candidateData
	raw  []byte
}

func NewCandidate(command change.CommandIntent, id string, binding transaction.Binding, expected int64, permissions []Permission, plan *transaction.FrozenPlan, noOpResult ...[]byte) (Candidate, error) {
	data := candidateData{Schema: 1, ID: id, Identity: command.Key(), InputHash: command.InputHash(), Binding: binding, Expected: expected, Permissions: slices.Clone(permissions)}
	slices.SortFunc(data.Permissions, func(a, b Permission) int {
		if n := strings.Compare(a.Resource, b.Resource); n != 0 {
			return n
		}
		return strings.Compare(a.Action, b.Action)
	})
	data.Permissions = slices.Compact(data.Permissions)
	if data.Permissions == nil {
		data.Permissions = []Permission{}
	}
	if plan != nil {
		raw, err := transaction.EncodePlan(*plan)
		if err != nil {
			return Candidate{}, ErrInvalid
		}
		data.Plan = raw
	}
	if len(noOpResult) > 1 {
		return Candidate{}, ErrInvalid
	}
	if len(noOpResult) == 1 {
		data.NoopResult = bytes.Clone(noOpResult[0])
	}
	return encode(data)
}
func encode(data candidateData) (Candidate, error) {
	if err := validate(data); err != nil {
		return Candidate{}, err
	}
	raw, err := json.Marshal(data)
	if err != nil || len(raw) > maxCandidateBytes {
		return Candidate{}, ErrInvalid
	}
	return Candidate{data: data, raw: raw}, nil
}
func DecodeCandidate(raw []byte) (Candidate, error) {
	if len(raw) == 0 || len(raw) > maxCandidateBytes || !utf8.Valid(raw) {
		return Candidate{}, ErrInvalid
	}
	var data candidateData
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&data) != nil {
		return Candidate{}, ErrInvalid
	}
	candidate, err := encode(data)
	if err != nil || !bytes.Equal(candidate.raw, raw) {
		return Candidate{}, ErrInvalid
	}
	return candidate, nil
}
func identity(raw string) ([]string, error) {
	var tuple []string
	if len(raw) > 1024 || json.Unmarshal([]byte(raw), &tuple) != nil || len(tuple) != 5 || tuple[0] != "iam-authz-operation/v1" {
		return nil, ErrInvalid
	}
	for _, part := range tuple {
		if part == "" || len(part) > 255 || !utf8.ValidString(part) || strings.TrimSpace(part) != part {
			return nil, ErrInvalid
		}
	}
	canonical, _ := json.Marshal(tuple)
	if string(canonical) != raw {
		return nil, ErrInvalid
	}
	return tuple, nil
}
func validate(data candidateData) error {
	id, err := uuid.Parse(data.ID)
	if err != nil || id == uuid.Nil || id.String() != data.ID || data.Schema != 1 || data.Expected < 1 || data.InputHash == ([32]byte{}) || data.Binding.Owner != "iam.mysql" || !data.Binding.Valid() {
		return ErrInvalid
	}
	for _, value := range []string{data.Binding.StoreID, data.Binding.WriterEpoch} {
		if len(value) > 128 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
			return ErrInvalid
		}
	}
	tuple, err := identity(data.Identity)
	if err != nil {
		return err
	}
	if data.Permissions == nil || len(data.Permissions) > 64 {
		return ErrInvalid
	}
	actor, _ := subject.NewUserRef(meta.ID(1))
	for i, p := range data.Permissions {
		if _, err = domain.NewRequest(actor, p.Resource, p.Action); err != nil {
			return ErrInvalid
		}
		if i > 0 {
			previous := data.Permissions[i-1]
			if previous.Resource > p.Resource || (previous.Resource == p.Resource && previous.Action >= p.Action) {
				return ErrInvalid
			}
		}
	}
	if strings.HasPrefix(tuple[1], "user:") && len(data.Permissions) == 0 {
		return ErrInvalid
	}
	if len(data.Plan) == 0 {
		if len(data.NoopResult) == 0 || len(data.NoopResult) > MaxResultBytes || !utf8.Valid(data.NoopResult) || !json.Valid(data.NoopResult) {
			return ErrInvalid
		}
	} else if len(data.NoopResult) != 0 {
		return ErrInvalid
	}
	if len(data.Plan) > 0 {
		plan, err := transaction.DecodePlan(data.Plan)
		if err != nil {
			return ErrInvalid
		}
		if plan.Input().OperationID != data.ID {
			return ErrInvalid
		}
		if err = plan.MatchRequest("iam.mysql", change.PolicyScope, data.Identity, data.InputHash); err != nil {
			return ErrInvalid
		}
		version, err := change.DecodeVersionPlan(plan)
		if err != nil || version.Expected != data.Expected {
			return ErrInvalid
		}
	}
	return nil
}
func (c Candidate) Valid() bool           { return len(c.raw) > 0 && validate(c.data) == nil }
func (c Candidate) Encode() []byte        { return bytes.Clone(c.raw) }
func (c Candidate) Fingerprint() [32]byte { return sha256.Sum256(c.raw) }
func (c Candidate) RequestKey() [32]byte {
	return sha256.Sum256([]byte("iam-business-operation/v1\x00" + c.data.Identity))
}
func (c Candidate) ID() string                   { return c.data.ID }
func (c Candidate) Identity() string             { return c.data.Identity }
func (c Candidate) InputHash() [32]byte          { return c.data.InputHash }
func (c Candidate) Binding() transaction.Binding { return c.data.Binding }
func (c Candidate) ExpectedVersion() int64       { return c.data.Expected }
func (c Candidate) Changed() bool                { return len(c.data.Plan) > 0 }
func (c Candidate) Permissions() []Permission    { return slices.Clone(c.data.Permissions) }
func (c Candidate) Principal() string {
	tuple, err := identity(c.data.Identity)
	if err != nil {
		return ""
	}
	return tuple[1]
}
func (c Candidate) Scope() string {
	tuple, err := identity(c.data.Identity)
	if err != nil {
		return ""
	}
	return tuple[2]
}
func (c Candidate) Name() string {
	tuple, err := identity(c.data.Identity)
	if err != nil {
		return ""
	}
	return tuple[3]
}
func (c Candidate) Plan() (transaction.FrozenPlan, error) {
	if !c.Changed() {
		return transaction.FrozenPlan{}, ErrInvalid
	}
	return transaction.DecodePlan(c.data.Plan)
}

// SameRequest compares stable business input only. Candidate fingerprints include
// generated first-plan artifacts, so comparing them would incorrectly reject a
// concurrent loser instead of returning the complete winner.
func (c Candidate) SameRequest(other Candidate) bool {
	return c.Valid() && other.Valid() && c.data.Identity == other.data.Identity && c.data.InputHash == other.data.InputHash
}

// RequestKeyFor provides the host journal's identity hash without generating any
// operation ID or new plan during the early idempotency lookup.
func RequestKeyFor(command change.CommandIntent) ([32]byte, error) {
	if _, err := identity(command.Key()); err != nil || len(command.Input()) == 0 || command.InputHash() == ([32]byte{}) {
		return [32]byte{}, ErrInvalid
	}
	return sha256.Sum256([]byte("iam-business-operation/v1\x00" + command.Key())), nil
}

func ValidID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

func ValidBinding(binding transaction.Binding) bool {
	if binding.Owner != "iam.mysql" || !binding.Valid() {
		return false
	}
	for _, value := range []string{binding.StoreID, binding.WriterEpoch} {
		if len(value) > 128 || !utf8.ValidString(value) || strings.TrimSpace(value) != value {
			return false
		}
	}
	return true
}

func (c Candidate) NoopResult() []byte { return bytes.Clone(c.data.NoopResult) }
