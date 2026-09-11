// Package authzcompat validates retired wire/storage fields without evaluating them.
package authzcompat

import (
	"bytes"
	"encoding/json"
	"io"

	perrors "github.com/FangcunMount/component-base/pkg/errors"
	"github.com/FangcunMount/iam/v5/internal/pkg/code"
)

const EmptyConstraints = `{"version":1,"all_of":[]}`
const EmptySchema = `{"version":1,"attributes":[]}`

func ValidateEmpty(data []byte, field string) error {
	if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return invalid()
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return invalid()
		}
		key, ok := token.(string)
		if !ok {
			return invalid()
		}
		if _, exists := fields[key]; exists {
			return invalid()
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return invalid()
		}
		fields[key] = value
	}
	if _, err := decoder.Token(); err != nil {
		return invalid()
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid()
	}
	for key := range fields {
		if key != "version" && key != field {
			return invalid()
		}
	}
	if raw, ok := fields["version"]; ok {
		var v uint32
		if json.Unmarshal(raw, &v) != nil || v != 1 {
			return invalid()
		}
	}
	if raw, ok := fields[field]; ok {
		var list []json.RawMessage
		if json.Unmarshal(raw, &list) != nil || len(list) != 0 {
			return invalid()
		}
	}
	return nil
}
func invalid() error {
	return perrors.WithCode(code.ErrInvalidArgument, "conditional authorization has been retired; only an empty legacy field is accepted")
}

type Constraints struct {
	Version uint32           `json:"version" enums:"1" default:"1"`
	AllOf   []map[string]any `json:"all_of" maxItems:"0"`
}

func (*Constraints) UnmarshalJSON(data []byte) error { return ValidateEmpty(data, "all_of") }
func (Constraints) MarshalJSON() ([]byte, error)     { return []byte(EmptyConstraints), nil }

type Schema struct {
	Version    uint32           `json:"version" enums:"1" default:"1"`
	Attributes []map[string]any `json:"attributes" maxItems:"0"`
}

func (*Schema) UnmarshalJSON(data []byte) error { return ValidateEmpty(data, "attributes") }
func (Schema) MarshalJSON() ([]byte, error)     { return []byte(EmptySchema), nil }
