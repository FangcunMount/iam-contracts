package business_object

import (
	"time"

	domainbo "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/business-object"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
)

// TesteeMapper 负责业务被测者对象与持久化对象的转换
type TesteeMapper struct{}

// NewTesteeMapper 创建 TesteeMapper
func NewTesteeMapper() *TesteeMapper {
	return &TesteeMapper{}
}

// ToPO 将领域对象转换为持久化对象
func (m *TesteeMapper) ToPO(testee *domainbo.Testee) *TesteePO {
	if testee == nil {
		return nil
	}

	var birthday *time.Time
	if !testee.Birthday.IsZero() {
		b := testee.Birthday
		birthday = &b
	}

	return &TesteePO{
		UserID:   testee.UserID.Value(),
		Name:     testee.Name,
		Sex:      testee.Sex,
		Birthday: birthday,
	}
}

// ToDomain 将持久化对象转换为领域对象
func (m *TesteeMapper) ToDomain(po *TesteePO) *domainbo.Testee {
	if po == nil {
		return nil
	}

	testee := &domainbo.Testee{
		UserID: user.NewUserID(po.UserID),
		Name:   po.Name,
		Sex:    po.Sex,
	}

	if po.Birthday != nil {
		testee.Birthday = *po.Birthday
	}

	return testee
}

// AuditorMapper 负责审核员对象与持久化对象的转换
type AuditorMapper struct{}

// NewAuditorMapper 创建 AuditorMapper
func NewAuditorMapper() *AuditorMapper {
	return &AuditorMapper{}
}

// ToPO 将领域对象转换为持久化对象
func (m *AuditorMapper) ToPO(auditor *domainbo.Auditor) *AuditorPO {
	if auditor == nil {
		return nil
	}

	var hiredAt *time.Time
	if !auditor.HiredAt.IsZero() {
		h := auditor.HiredAt
		hiredAt = &h
	}

	return &AuditorPO{
		UserID:     auditor.UserID.Value(),
		Name:       auditor.Name,
		EmployeeID: auditor.EmployeeID,
		Department: auditor.Department,
		Position:   auditor.Position,
		Status:     auditor.Status.Value(),
		HiredAt:    hiredAt,
	}
}

// ToDomain 将持久化对象转换为领域对象
func (m *AuditorMapper) ToDomain(po *AuditorPO) *domainbo.Auditor {
	if po == nil {
		return nil
	}

	auditor := &domainbo.Auditor{
		UserID:     user.NewUserID(po.UserID),
		Name:       po.Name,
		EmployeeID: po.EmployeeID,
		Department: po.Department,
		Position:   po.Position,
		Status:     domainbo.Status(po.Status),
	}

	if po.HiredAt != nil {
		auditor.HiredAt = *po.HiredAt
	}

	return auditor
}
