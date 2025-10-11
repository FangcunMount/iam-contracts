package business_object

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	domainbo "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/business-object"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/user"
	"github.com/fangcun-mount/iam-contracts/internal/apiserver/infra/mysql"
	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	pkgerrors "github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// TesteeRepository 被测者 MySQL 仓储
type TesteeRepository struct {
	mysql.BaseRepository[*TesteePO]
	mapper *TesteeMapper
}

// NewTesteeRepository 创建被测者仓储
func NewTesteeRepository(db *gorm.DB) *TesteeRepository {
	return &TesteeRepository{
		BaseRepository: mysql.NewBaseRepository[*TesteePO](db),
		mapper:         NewTesteeMapper(),
	}
}

// Save 保存或更新被测者信息（基于 user_id 唯一约束）
func (r *TesteeRepository) Save(ctx context.Context, testee *domainbo.Testee) error {
	if testee == nil {
		return pkgerrors.WithCode(code.ErrValidation, "testee is nil")
	}

	po := r.mapper.ToPO(testee)
	updates := map[string]interface{}{
		"name":       po.Name,
		"sex":        po.Sex,
		"updated_at": gorm.Expr("NOW()"),
	}
	if po.Birthday != nil {
		updates["birthday"] = po.Birthday
	} else {
		updates["birthday"] = nil
	}

	result := r.DB().WithContext(ctx).
		Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(updates),
		}).
		Create(po)

	if result.Error != nil {
		return pkgerrors.WithCode(code.ErrDatabase, "failed to save testee: %v", result.Error)
	}

	// 回填最新数据
	if po.Birthday != nil {
		testee.Birthday = *po.Birthday
	} else {
		testee.Birthday = time.Time{}
	}
	testee.Sex = po.Sex
	testee.Name = po.Name

	return nil
}

// FindByUserID 根据用户 ID 获取被测者信息
func (r *TesteeRepository) FindByUserID(ctx context.Context, userID user.UserID) (*domainbo.Testee, error) {
	var po TesteePO
	err := r.DB().WithContext(ctx).Where("user_id = ?", userID.Value()).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkgerrors.WithCode(code.ErrDatabase, "failed to find testee: %v", err)
	}
	return r.mapper.ToDomain(&po), nil
}

// ListByUserIDs 批量获取被测者信息
func (r *TesteeRepository) ListByUserIDs(ctx context.Context, userIDs []user.UserID) ([]*domainbo.Testee, error) {
	if len(userIDs) == 0 {
		return []*domainbo.Testee{}, nil
	}
	rawIDs := make([]uint64, 0, len(userIDs))
	for _, id := range userIDs {
		rawIDs = append(rawIDs, id.Value())
	}

	var pos []*TesteePO
	if err := r.DB().WithContext(ctx).Where("user_id IN ?", rawIDs).Find(&pos).Error; err != nil {
		return nil, pkgerrors.WithCode(code.ErrDatabase, "failed to list testees: %v", err)
	}

	result := make([]*domainbo.Testee, 0, len(pos))
	for _, po := range pos {
		if domainObj := r.mapper.ToDomain(po); domainObj != nil {
			result = append(result, domainObj)
		}
	}

	return result, nil
}

// DeleteByUserID 根据用户 ID 删除被测者信息
func (r *TesteeRepository) DeleteByUserID(ctx context.Context, userID user.UserID) error {
	result := r.DB().WithContext(ctx).Where("user_id = ?", userID.Value()).Delete(&TesteePO{})
	if result.Error != nil {
		return pkgerrors.WithCode(code.ErrDatabase, "failed to delete testee: %v", result.Error)
	}
	return nil
}

// AuditorRepository 审核员 MySQL 仓储
type AuditorRepository struct {
	mysql.BaseRepository[*AuditorPO]
	mapper *AuditorMapper
}

// NewAuditorRepository 创建审核员仓储
func NewAuditorRepository(db *gorm.DB) *AuditorRepository {
	return &AuditorRepository{
		BaseRepository: mysql.NewBaseRepository[*AuditorPO](db),
		mapper:         NewAuditorMapper(),
	}
}

// Save 保存或更新审核员信息
func (r *AuditorRepository) Save(ctx context.Context, auditor *domainbo.Auditor) error {
	if auditor == nil {
		return pkgerrors.WithCode(code.ErrValidation, "auditor is nil")
	}

	po := r.mapper.ToPO(auditor)
	updates := map[string]interface{}{
		"name":        po.Name,
		"employee_id": po.EmployeeID,
		"department":  po.Department,
		"position":    po.Position,
		"status":      po.Status,
		"updated_at":  gorm.Expr("NOW()"),
	}
	if po.HiredAt != nil {
		updates["hired_at"] = po.HiredAt
	}

	result := r.DB().WithContext(ctx).
		Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(updates),
		}).
		Create(po)

	if result.Error != nil {
		return pkgerrors.WithCode(code.ErrDatabase, "failed to save auditor: %v", result.Error)
	}

	// 更新领域对象
	auditor.Name = po.Name
	auditor.EmployeeID = po.EmployeeID
	auditor.Department = po.Department
	auditor.Position = po.Position
	auditor.Status = domainbo.Status(po.Status)
	if po.HiredAt != nil {
		auditor.HiredAt = *po.HiredAt
	}

	return nil
}

// FindByUserID 根据用户 ID 查询审核员
func (r *AuditorRepository) FindByUserID(ctx context.Context, userID user.UserID) (*domainbo.Auditor, error) {
	var po AuditorPO
	err := r.DB().WithContext(ctx).Where("user_id = ?", userID.Value()).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkgerrors.WithCode(code.ErrDatabase, "failed to find auditor: %v", err)
	}
	return r.mapper.ToDomain(&po), nil
}

// FindByEmployeeID 根据员工编号查询审核员
func (r *AuditorRepository) FindByEmployeeID(ctx context.Context, employeeID string) (*domainbo.Auditor, error) {
	var po AuditorPO
	err := r.DB().WithContext(ctx).Where("employee_id = ?", employeeID).First(&po).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, pkgerrors.WithCode(code.ErrDatabase, "failed to find auditor by employee id: %v", err)
	}
	return r.mapper.ToDomain(&po), nil
}

// ListByDepartment 根据部门查询审核员
func (r *AuditorRepository) ListByDepartment(ctx context.Context, department string) ([]*domainbo.Auditor, error) {
	var pos []*AuditorPO
	err := r.DB().WithContext(ctx).Where("department = ?", department).Find(&pos).Error
	if err != nil {
		return nil, pkgerrors.WithCode(code.ErrDatabase, "failed to list auditors: %v", err)
	}

	result := make([]*domainbo.Auditor, 0, len(pos))
	for _, po := range pos {
		if domainObj := r.mapper.ToDomain(po); domainObj != nil {
			result = append(result, domainObj)
		}
	}

	return result, nil
}

// DeleteByUserID 根据用户 ID 删除审核员
func (r *AuditorRepository) DeleteByUserID(ctx context.Context, userID user.UserID) error {
	result := r.DB().WithContext(ctx).Where("user_id = ?", userID.Value()).Delete(&AuditorPO{})
	if result.Error != nil {
		return pkgerrors.WithCode(code.ErrDatabase, "failed to delete auditor: %v", result.Error)
	}
	return nil
}
