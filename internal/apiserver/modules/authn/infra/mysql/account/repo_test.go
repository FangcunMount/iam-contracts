package account

import (
"github.com/FangcunMount/iam-contracts/internal/apiserver/modules/authn/domain/account/port"
"gorm.io/gorm"
)

// 编译时验证接口实现
var (
_ port.AccountRepo    = (*AccountRepository)(nil)
_ port.CredentialRepo = (*CredentialRepository)(nil)
)

// 测试构造函数
func ExampleNewAccountRepository() {
	var db *gorm.DB
	repo := NewAccountRepository(db)
	_ = repo
}

func ExampleNewCredentialRepository() {
	var db *gorm.DB
	repo := NewCredentialRepository(db)
	_ = repo
}
