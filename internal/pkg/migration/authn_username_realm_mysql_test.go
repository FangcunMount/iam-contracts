package migration

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAuthNUsernameDefaultRealmMigrationMySQL(t *testing.T) {
	db := openMigrationMySQL(t)
	up := migrationSQL(t, "000030_authn_username_default_realm.up.sql")
	reset := func() {
		_, err := db.Exec("DROP TABLE IF EXISTS auth_login_identities")
		require.NoError(t, err)
		_, err = db.Exec(`CREATE TABLE auth_login_identities (
   id BIGINT PRIMARY KEY, user_id BIGINT NOT NULL, provider VARCHAR(32) NOT NULL,
   realm VARCHAR(64) NOT NULL, identifier VARCHAR(128) NOT NULL,
   UNIQUE KEY identity_key(provider,realm,identifier))`)
		require.NoError(t, err)
	}
	t.Cleanup(func() { _, _ = db.Exec("DROP TABLE IF EXISTS auth_login_identities") })
	t.Run("preserves identities and external realms", func(t *testing.T) {
		reset()
		_, err := db.Exec(`INSERT INTO auth_login_identities VALUES
   (1,10,'username','42','alice'),(2,20,'username','default','bob'),(3,30,'wechat_minip','wx-app','openid')`)
		require.NoError(t, err)
		_, err = db.Exec(up)
		require.NoError(t, err)
		var realm string
		var owner int
		require.NoError(t, db.QueryRow("SELECT realm,user_id FROM auth_login_identities WHERE id=1").Scan(&realm, &owner))
		require.Equal(t, "default", realm)
		require.Equal(t, 10, owner)
		require.NoError(t, db.QueryRow("SELECT realm FROM auth_login_identities WHERE id=3").Scan(&realm))
		require.Equal(t, "wx-app", realm)
		_, err = db.Exec("INSERT INTO auth_login_identities VALUES (4,40,'username','43','charlie')")
		require.Error(t, err)
	})
	t.Run("conflict fails before changing identities", func(t *testing.T) {
		reset()
		_, err := db.Exec("INSERT INTO auth_login_identities VALUES (1,10,'username','42','alice'),(2,20,'username','default','alice')")
		require.NoError(t, err)
		_, err = db.Exec(up)
		require.ErrorContains(t, err, "AuthN username realm conflict")
		var realm string
		require.NoError(t, db.QueryRow("SELECT realm FROM auth_login_identities WHERE id=1").Scan(&realm))
		require.Equal(t, "42", realm)
	})
}
