package platform

import (
	"errors"

	"github.com/FangcunMount/iam/v5/internal/apiserver/options"
	"github.com/FangcunMount/reliable-messaging/transaction"
	"github.com/FangcunMount/reliable-messaging/transport/rocketmq"
)

// NewTransactionalSender borrows the host primary store and performs no DB or
// Broker calls. Host composition must explicitly Start, gate and drain it.
func NewTransactionalSender(o options.MessagingOptions, store transaction.PreparedStore, topics []string, native NativeLimitReader) (*TransactionRuntime, error) {
	const owner = "iam.mysql"
	binding, ok := o.Transactions[owner]
	if !ok || store == nil || store.Binding() != (transaction.Binding{Owner: owner, StoreID: binding.StoreID, WriterEpoch: binding.WriterEpoch}) {
		return nil, errors.New("transaction sender requires the configured host primary binding")
	}
	config, err := o.TransactionConfig(owner, topics)
	if err != nil {
		return nil, err
	}
	sender, err := rocketmq.NewTransactionalSender(config, store)
	if err != nil {
		return nil, err
	}
	runtime := &TransactionRuntime{sender: sender, admission: o.PublishEnabled}
	if err = attachProtocolMonitor(runtime, store, config.Limits, native); err != nil {
		return nil, err
	}
	return runtime, nil
}
