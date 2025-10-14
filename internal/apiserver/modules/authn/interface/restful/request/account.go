package request

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	perrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// OperationCredentialPayload describes credential updates (password/hash).
type OperationCredentialPayload struct {
	Password *string                `json:"password,omitempty"`
	Hash     *string                `json:"hash,omitempty"`
	Algo     *string                `json:"algo,omitempty"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

// CreateOperationAccountReq payload for creating operation account.
type CreateOperationAccountReq struct {
	UserID    string `json:"userId"`
	Username  string `json:"username"`
	MustReset bool   `json:"mustReset"`
	OperationCredentialPayload
}

// Validate basic fields.
func (r *CreateOperationAccountReq) Validate() error {
	if strings.TrimSpace(r.UserID) == "" || strings.TrimSpace(r.Username) == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "userId and username are required")
	}
	if err := r.OperationCredentialPayload.validate(); err != nil {
		return err
	}
	return nil
}

// HashPayload returns hash/algo/params when provided.
func (r *CreateOperationAccountReq) HashPayload() ([]byte, string, []byte, error) {
	return r.OperationCredentialPayload.hashPayload()
}

// UpdateOperationCredentialReq payload.
type UpdateOperationCredentialReq struct {
	NewPassword   *string                `json:"newPassword,omitempty"`
	NewHash       *string                `json:"newHash,omitempty"`
	Algo          *string                `json:"algo,omitempty"`
	Params        map[string]interface{} `json:"params,omitempty"`
	ResetFailures bool                   `json:"resetFailures"`
	UnlockNow     bool                   `json:"unlockNow"`
}

func (r *UpdateOperationCredentialReq) Validate() error {
	if (r.NewPassword == nil || strings.TrimSpace(*r.NewPassword) == "") &&
		(r.NewHash == nil || strings.TrimSpace(*r.NewHash) == "") &&
		!r.ResetFailures && !r.UnlockNow {
		return perrors.WithCode(code.ErrInvalidArgument, "no operation requested")
	}
	if r.NewHash != nil && (r.Algo == nil || strings.TrimSpace(*r.Algo) == "") {
		return perrors.WithCode(code.ErrInvalidArgument, "algo is required when newHash present")
	}
	return nil
}

func (r *UpdateOperationCredentialReq) HashPayload() ([]byte, string, []byte, error) {
	payload := OperationCredentialPayload{
		Password: r.NewPassword,
		Hash:     r.NewHash,
		Algo:     r.Algo,
		Params:   r.Params,
	}
	return payload.hashPayload()
}

// ChangeOperationUsernameReq payload.
type ChangeOperationUsernameReq struct {
	NewUsername string `json:"newUsername"`
}

func (r *ChangeOperationUsernameReq) Validate() error {
	if strings.TrimSpace(r.NewUsername) == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "newUsername cannot be empty")
	}
	return nil
}

// BindWeChatAccountReq payload.
type BindWeChatAccountReq struct {
	UserID   string                 `json:"userId"`
	AppID    string                 `json:"appId"`
	OpenID   string                 `json:"openid"`
	UnionID  *string                `json:"unionid,omitempty"`
	Nickname *string                `json:"nickname,omitempty"`
	Avatar   *string                `json:"avatar,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

func (r *BindWeChatAccountReq) Validate() error {
	if strings.TrimSpace(r.UserID) == "" || strings.TrimSpace(r.AppID) == "" || strings.TrimSpace(r.OpenID) == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "userId, appId and openid are required")
	}
	return nil
}

// UpsertWeChatProfileReq payload.
type UpsertWeChatProfileReq struct {
	Nickname *string                `json:"nickname,omitempty"`
	Avatar   *string                `json:"avatar,omitempty"`
	Meta     map[string]interface{} `json:"meta,omitempty"`
}

func (r *UpsertWeChatProfileReq) Validate() error {
	if (r.Nickname == nil || strings.TrimSpace(*r.Nickname) == "") &&
		(r.Avatar == nil || strings.TrimSpace(*r.Avatar) == "") &&
		len(r.Meta) == 0 {
		return perrors.WithCode(code.ErrInvalidArgument, "at least one field must be provided")
	}
	return nil
}

// SetWeChatUnionIDReq payload.
type SetWeChatUnionIDReq struct {
	UnionID string `json:"unionId"`
}

func (r *SetWeChatUnionIDReq) Validate() error {
	if strings.TrimSpace(r.UnionID) == "" {
		return perrors.WithCode(code.ErrInvalidArgument, "unionId cannot be empty")
	}
	return nil
}

func (p *OperationCredentialPayload) validate() error {
	if p.Password == nil && p.Hash == nil {
		return nil
	}
	if p.Password != nil && p.Hash != nil {
		return perrors.WithCode(code.ErrInvalidArgument, "password and hash are mutually exclusive")
	}
	if p.Hash != nil && (p.Algo == nil || strings.TrimSpace(*p.Algo) == "") {
		return perrors.WithCode(code.ErrInvalidArgument, "algo is required when hash provided")
	}
	return nil
}

func (p *OperationCredentialPayload) hashPayload() ([]byte, string, []byte, error) {
	if p == nil {
		return nil, "", nil, nil
	}
	if p.Password == nil && p.Hash == nil {
		return nil, "", nil, nil
	}

	algo := "plain"
	if p.Algo != nil && strings.TrimSpace(*p.Algo) != "" {
		algo = strings.TrimSpace(*p.Algo)
	}

	var hash []byte
	var err error
	if p.Password != nil {
		password := strings.TrimSpace(*p.Password)
		if password == "" {
			return nil, "", nil, perrors.WithCode(code.ErrInvalidArgument, "password cannot be empty")
		}
		hash = []byte(password)
	} else if p.Hash != nil {
		hash, err = base64.StdEncoding.DecodeString(*p.Hash)
		if err != nil {
			return nil, "", nil, perrors.WithCode(code.ErrInvalidArgument, "hash must be base64 encoded")
		}
	}

	paramsBytes, err := encodeMapToJSON(p.Params)
	if err != nil {
		return nil, "", nil, err
	}

	return hash, algo, paramsBytes, nil
}

func encodeMapToJSON(m map[string]interface{}) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	if len(m) == 0 {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, perrors.WithCode(code.ErrInvalidArgument, "invalid json object")
	}
	return b, nil
}

// CredentialMeta encodes credential params to json bytes.
func (p *OperationCredentialPayload) ParamsJSON() ([]byte, error) {
	return encodeMapToJSON(p.Params)
}

// MetaJSON encodes BindWeChatAccount meta.
func (r *BindWeChatAccountReq) MetaJSON() ([]byte, error) {
	return encodeMapToJSON(r.Meta)
}

// MetaJSON encodes UpsertWeChatProfile meta.
func (r *UpsertWeChatProfileReq) MetaJSON() ([]byte, error) {
	return encodeMapToJSON(r.Meta)
}
