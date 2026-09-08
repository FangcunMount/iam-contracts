package policy

import (
	"strconv"

	"github.com/FangcunMount/iam/v5/internal/apiserver/eventing"
	"github.com/FangcunMount/iam/v5/pkg/event"
)

type VersionChangedPayload struct {
	Version int64 `json:"version"`
}

type VersionChangedEvent struct {
	event.BaseEvent
	payload VersionChangedPayload
}

func NewVersionChangedEvent(version int64) VersionChangedEvent {
	return VersionChangedEvent{
		BaseEvent: event.NewBaseEvent(
			eventing.AuthzVersionChanged,
			"PolicyVersion",
			strconv.FormatInt(version, 10),
		),
		payload: VersionChangedPayload{

			Version: version,
		},
	}
}

func (e VersionChangedEvent) Payload() any {
	return e.payload
}
