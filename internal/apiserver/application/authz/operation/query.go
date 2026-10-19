package operation

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/FangcunMount/reliable-messaging/transaction"
)

const MaxResultBytes = 256 * 1024

type State string

const (
	Prepared  State = "prepared"
	Committed State = "committed"
	Aborted   State = "aborted"
)

var ErrAccessDenied = errors.New("current permission to read IAM operation is unavailable")
var ErrProtocolConflict = errors.New("IAM operation protocol evidence conflicts")

// Record is the private adapter receipt, never a transport response. QueryView
// exposes only the authorized business result and three phase classifications.
type Record struct {
	Candidate     Candidate `json:"-"`
	State         State     `json:"-"`
	PolicyVersion int64     `json:"-"`
	Result        []byte    `json:"-"`
	GuardVersion  int64     `json:"-"`
	LeaseUntil    time.Time `json:"-"`
	CreatedAt     time.Time `json:"-"`
	FinalizedAt   time.Time `json:"-"`
}
type Reader interface {
	Read(context.Context, string) (Record, error)
}
type ProtocolState struct {
	Exists bool
	Half   transaction.HalfOutcome
	Local  transaction.LocalOutcome
	Final  transaction.FinalOutcome
}
type ProtocolReader interface {
	ReadProtocol(context.Context, Candidate) (ProtocolState, error)
}

// AccessPolicy must use verified identity, original scope and current original
// permissions. Request data and ChangedBy are never identity evidence.
type AccessPolicy interface {
	Principal(context.Context) (string, bool)
	Require(context.Context, string, []Permission) error
}
type Phases struct {
	Half  transaction.HalfOutcome  `json:"half"`
	Local transaction.LocalOutcome `json:"local"`
	Final transaction.FinalOutcome `json:"final"`
}
type QueryView struct {
	ID            string          `json:"operation_id"`
	Operation     string          `json:"operation"`
	Changed       bool            `json:"changed"`
	State         State           `json:"state"`
	PolicyVersion *int64          `json:"policy_version,omitempty"`
	Result        json.RawMessage `json:"result,omitempty"`
	Phases        Phases          `json:"phases"`
	Issue         string          `json:"issue,omitempty"`
}
type QueryService struct {
	reader   Reader
	protocol ProtocolReader
	access   AccessPolicy
	config   QueryConfig
	life     *queryLifecycle
}

func NewQueryService(reader Reader, protocol ProtocolReader, access AccessPolicy, configs ...QueryConfig) (*QueryService, error) {
	if reader == nil || protocol == nil || access == nil {
		return nil, ErrInvalid
	}
	config := QueryConfig{Concurrency: 8, Timeout: 2 * time.Second}
	if len(configs) > 1 {
		return nil, ErrInvalid
	}
	if len(configs) == 1 {
		config = configs[0]
	}
	if config.Concurrency < 1 || config.Concurrency > 1024 || config.Timeout <= 0 || config.Timeout > time.Minute {
		return nil, ErrInvalid
	}
	return &QueryService{reader: reader, protocol: protocol, access: access, config: config, life: &queryLifecycle{limit: config.Concurrency, done: make(chan struct{})}}, nil
}
func (s *QueryService) Query(ctx context.Context, id string) (QueryView, error) {
	if ctx == nil || !ValidID(id) {
		return QueryView{}, ErrInvalid
	}
	if err := s.life.enter(); err != nil {
		return QueryView{}, err
	}
	defer s.life.leave()
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()
	actor, ok := s.access.Principal(ctx)
	if !ok || actor == "" {
		return QueryView{}, ErrAccessDenied
	}
	record, err := s.reader.Read(ctx, id)
	if err != nil {
		return QueryView{}, err
	}
	if !record.Candidate.Valid() || record.Candidate.ID() != id {
		return QueryView{}, ErrInvalid
	}
	if actor != record.Candidate.Principal() {
		return QueryView{}, ErrNotFound
	}
	if err = s.access.Require(ctx, record.Candidate.Scope(), record.Candidate.Permissions()); err != nil {
		return QueryView{}, err
	}
	if err = ctx.Err(); err != nil {
		return QueryView{}, err
	}
	view := QueryView{ID: id, Operation: record.Candidate.Name(), Changed: record.Candidate.Changed(), State: record.State, Phases: Phases{Half: transaction.HalfNotAttempted, Local: transaction.LocalUnknown, Final: transaction.FinalNotAttempted}}
	if record.State != Prepared && record.State != Committed && record.State != Aborted {
		return QueryView{}, ErrInvalid
	}
	protocol := ProtocolState{Half: transaction.HalfNotAttempted, Local: transaction.LocalNotStarted, Final: transaction.FinalNotAttempted}
	if view.Changed {
		protocol, err = s.protocol.ReadProtocol(ctx, record.Candidate)
		if err != nil {
			view.Issue = "protocol_status_unavailable"
			if errors.Is(err, ErrProtocolConflict) {
				view.Issue = "evidence_conflict"
			}
			protocol = ProtocolState{Half: transaction.HalfUnknown, Local: transaction.LocalUnknown, Final: transaction.FinalUnknown}
		}
		if err == nil && !protocol.Exists {
			protocol = ProtocolState{Half: transaction.HalfNotAttempted, Local: transaction.LocalNotStarted, Final: transaction.FinalNotAttempted}
		}
		// The journal read may precede a concurrent original-root commit. A later
		// SDK proof is not a contradiction until a fresh journal boundary is read.
		if err == nil && protocol.Local == transaction.LocalCommitted && record.State == Prepared {
			refreshed, readErr := s.reader.Read(ctx, id)
			if readErr != nil {
				view.Issue = "result_unavailable"
			} else if refreshed.Candidate.Fingerprint() != record.Candidate.Fingerprint() {
				return QueryView{}, ErrInvalid
			} else {
				record = refreshed
				view.State = record.State
			}
		}
		view.Phases.Half = protocol.Half
		view.Phases.Final = protocol.Final
	}
	if record.State != Prepared && record.State != Committed && record.State != Aborted {
		return QueryView{}, ErrInvalid
	}
	switch record.State {
	case Committed:
		expected := record.Candidate.ExpectedVersion()
		if view.Changed {
			expected++
		}
		if record.PolicyVersion != expected || len(record.Result) == 0 || len(record.Result) > MaxResultBytes || !utf8.Valid(record.Result) || !json.Valid(record.Result) {
			return QueryView{}, ErrInvalid
		}
		view.Phases.Local = transaction.LocalCommitted
		version := record.PolicyVersion
		view.PolicyVersion = &version
		view.Result = append(json.RawMessage(nil), record.Result...)
		if view.Changed && err == nil && (protocol.Local != transaction.LocalCommitted || protocol.Final == transaction.FinalRollbackConfirmed) {
			view.Issue = "evidence_conflict"
		}
	case Aborted:
		view.Phases.Local = transaction.LocalAborted
		if view.Changed && protocol.Final == transaction.FinalCommitConfirmed {
			view.Issue = "evidence_conflict"
		}
		if view.Changed && protocol.Local == transaction.LocalCommitted {
			view.State = Prepared
			view.Phases.Local = transaction.LocalUnknown
			view.Issue = "evidence_conflict"
		}
	case Prepared:
		if view.Changed && err == nil {
			view.Phases.Local = protocol.Local
			if protocol.Local == transaction.LocalCommitted {
				view.State = Committed
				if view.Issue == "" {
					view.Issue = "result_unavailable"
				}
			}
			if protocol.Local == transaction.LocalAborted {
				view.State = Aborted
			}
		}
	}
	return view, nil
}
