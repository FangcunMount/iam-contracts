package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	jwksrepo "github.com/FangcunMount/iam/v5/internal/apiserver/infra/mysql/jwks"
	"github.com/FangcunMount/iam/v5/internal/apiserver/infra/token/keyset"
)

// runSigningKeyCutover 复用现有密钥生命周期；先激活新密钥，再强制退役全部旧密钥。
// 默认只预览公有标识；不输出、不删除私钥。调用期间必须停止所有签发实例。
func runSigningKeyCutover(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("signing-key-cutover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("keys-dir", "", "persistent PEM key directory")
	apply := flags.Bool("apply", false, "activate a new key and retire old verification keys")
	stopped := flags.Bool("issuers-stopped", false, "all token issuers and rotation schedulers are stopped")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || (*apply && (!*stopped || strings.TrimSpace(*dir) == "")) {
		return errors.New("密钥切换需要持久化目录并停止全部签发实例")
	}
	db, err := authzConvergeDatabaseFromEnvironment()
	if err != nil {
		return err
	}
	pool, err := db.DB()
	if err != nil {
		return err
	}
	defer func() { _ = pool.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	repo := jwksrepo.NewKeyRepository(db)
	var old []*keyset.Key
	for _, state := range []keyset.KeyStatus{keyset.KeyActive, keyset.KeyGrace} {
		keys, err := repo.FindByStatus(ctx, state)
		if err != nil {
			return err
		}
		old = append(old, keys...)
	}
	for _, key := range old {
		if _, err := fmt.Fprintf(output, "previous_key kid=%s status=%s\n", key.Kid, key.Status); err != nil {
			return err
		}
	}
	if !*apply {
		_, err = fmt.Fprintf(output, "signing key preview: retirement_candidates=%d\n", len(old))
		return err
	}
	manager := keyset.NewKeyManager(repo, keyset.NewRSAKeyGenerator(), keyset.NewPEMPrivateKeyStorage(*dir))
	next, err := manager.CreateKey(ctx, "RS256", nil, nil)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "new_key_activated kid=%s\n", next.Kid); err != nil {
		return err
	}
	for _, key := range old {
		if err := manager.ForceRetireKey(ctx, key.Kid); err != nil {
			return err
		}
	}
	published, err := repo.FindPublishable(ctx)
	if err != nil {
		return err
	}
	if len(published) != 1 || published[0].Kid != next.Kid {
		return errors.New("切换后可发布密钥不一致")
	}
	_, err = fmt.Fprintf(output, "signing key cutover verified: active_kid=%s retired=%d\n", next.Kid, len(old))
	return err
}
