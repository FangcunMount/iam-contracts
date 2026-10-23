package migration

import (
	"crypto/sha256"
	"encoding/hex"
)

// MessagingMigrationDigest binds only the additive M3 source templates. It is
// not a database backup or an assertion that live migration 41 is installed.
func MessagingMigrationDigest() string {
	h := sha256.New()
	for _, name := range []string{"000042_transaction_messaging.up.sql", "000043_authz_message_protocol.up.sql", "000044_authz_business_operations.up.sql", "000045_native_transaction_limit_notices.up.sql"} {
		raw, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			panic("embedded M3 migration missing")
		}
		_, _ = h.Write([]byte(name))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(raw)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
