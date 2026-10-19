package authzoperation

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	operation "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/operation"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/stretchr/testify/require"
)

func TestConstructorAndInvalidInputsDoNotAccessHostPool(t *testing.T) {
	// An uninitialized pool would panic on a real DB call. Constructors and
	// rejected inputs must remain inert and must never install schema or ping.
	store, err := New(new(sql.DB), transaction.Binding{Owner: "iam.mysql", StoreID: "primary", WriterEpoch: "m3-1"})
	require.NoError(t, err)
	_, err = store.Read(nil, "invalid")
	require.ErrorIs(t, err, operation.ErrInvalid)
	_, err = store.Read(context.Background(), "invalid")
	require.ErrorIs(t, err, operation.ErrInvalid)
	_, err = store.Reserve(context.Background(), operation.Candidate{}, time.Minute)
	require.ErrorIs(t, err, operation.ErrInvalid)
	_, err = store.AbortExpired(context.Background(), 0)
	require.ErrorIs(t, err, operation.ErrInvalid)
	_, err = store.Bind(context.Background(), nil, nil)
	require.ErrorIs(t, err, operation.ErrStale)
	_, err = New(new(sql.DB), transaction.Binding{Owner: "iam.mysql", StoreID: string([]byte{0xff}), WriterEpoch: "m3-1"})
	require.ErrorIs(t, err, operation.ErrInvalid)
	raw, err := json.Marshal(&Claim{token: [32]byte{1}})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(raw), "execution token escaped as operation status")
}
