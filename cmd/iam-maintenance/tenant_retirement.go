package main

import (
	"errors"
	"io"
)

// The tenant schema cutover predates the independent role model. Use its
// historical release for historical schemas; never revive graph writes here.
func runTenantRetirement(_ []string, _ io.Writer) error {
	return errors.New("tenant-retirement 已退役；历史租户结构请使用 v5.0.2 维护工具，当前角色迁移请使用 role-model-migrate")
}
