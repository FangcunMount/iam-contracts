package operation

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

type ResultCodec struct {
	schema  string
	capture func() (json.RawMessage, error)
	restore func(json.RawMessage) error
}
type resultEnvelope struct {
	Schema string          `json:"schema"`
	Data   json.RawMessage `json:"data"`
}

// ProjectResult requires an explicit public projection, never a domain object
// with private value fields or a persistence object containing credentials.
// Capture/restore perform only deterministic serialization, not DB or MQ I/O.
func ProjectResult[T any](schema string, capture func() (T, error), restore func(T) error) *ResultCodec {
	codec := &ResultCodec{schema: schema}
	if capture != nil {
		codec.capture = func() (json.RawMessage, error) {
			value, err := capture()
			if err != nil {
				return nil, err
			}
			raw, err := json.Marshal(value)
			return json.RawMessage(raw), err
		}
	}
	if restore != nil {
		codec.restore = func(raw json.RawMessage) error {
			var value T
			if err := decodeResult(raw, &value); err != nil {
				return err
			}
			return restore(value)
		}
	}
	return codec
}
func (c *ResultCodec) Valid() bool {
	return c != nil && c.capture != nil && c.restore != nil && c.schema != "" && len(c.schema) <= 128 && utf8.ValidString(c.schema) && strings.TrimSpace(c.schema) == c.schema
}
func (c *ResultCodec) Capture() ([]byte, error) {
	if !c.Valid() {
		return nil, ErrInvalid
	}
	data, err := c.capture()
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) || !json.Valid(data) {
		return nil, ErrInvalid
	}
	raw, err := json.Marshal(resultEnvelope{Schema: c.schema, Data: data})
	if err != nil || len(raw) > MaxResultBytes {
		return nil, ErrInvalid
	}
	return raw, nil
}
func (c *ResultCodec) Restore(raw []byte) error {
	if !c.Valid() || len(raw) > MaxResultBytes || !utf8.Valid(raw) {
		return ErrInvalid
	}
	var envelope resultEnvelope
	if err := decodeResult(raw, &envelope); err != nil || envelope.Schema != c.schema || len(envelope.Data) == 0 {
		return ErrInvalid
	}
	return c.restore(envelope.Data)
}
func decodeResult[T any](raw []byte, value *T) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil {
		return ErrInvalid
	}
	canonical, err := json.Marshal(value)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ErrInvalid
	}
	return nil
}
func EmptyResultCodec() *ResultCodec {
	return ProjectResult("iam.empty-result/v1", func() (struct{}, error) { return struct{}{}, nil }, func(struct{}) error { return nil })
}
