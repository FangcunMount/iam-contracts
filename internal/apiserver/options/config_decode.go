package options

import "github.com/spf13/viper"

// DecodeConfig preserves dotted owner identifiers. Viper's AllSettings rebuilds
// dots as hierarchy; UnmarshalKey decodes the original map without that rebuild.
func (o *Options) DecodeConfig(reader *viper.Viper) error {
	if err := reader.Unmarshal(o); err != nil {
		return err
	}
	if reader.IsSet("messaging.transactions") {
		var bindings map[string]TransactionOptions
		if err := reader.UnmarshalKey("messaging.transactions", &bindings); err != nil {
			return err
		}
		o.Messaging.Transactions = bindings
	}
	return nil
}
