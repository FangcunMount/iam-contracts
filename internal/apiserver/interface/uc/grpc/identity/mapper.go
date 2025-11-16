package identity

import (
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/FangcunMount/component-base/pkg/errors"
	childApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/child"
	guardianshipApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/guardianship"
	userApp "github.com/FangcunMount/iam-contracts/internal/apiserver/application/uc/user"
	childDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/child"
	guardianshipDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/guardianship"
	userDomain "github.com/FangcunMount/iam-contracts/internal/apiserver/domain/uc/user"
	identityv1 "github.com/FangcunMount/iam-contracts/internal/apiserver/interface/uc/grpc/pb/iam/identity/v1"
	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

// ============= UserResult 转换 =============

// userResultToProto 将应用层 UserResult 转换为 proto User
func userResultToProto(result *userApp.UserResult) *identityv1.User {
	if result == nil {
		return nil
	}

	contacts := make([]*identityv1.VerifiedContact, 0)
	if result.Phone != "" {
		contacts = append(contacts, &identityv1.VerifiedContact{
			Type:  identityv1.ContactType_CONTACT_TYPE_PHONE,
			Value: result.Phone,
		})
	}
	if result.Email != "" {
		contacts = append(contacts, &identityv1.VerifiedContact{
			Type:  identityv1.ContactType_CONTACT_TYPE_EMAIL,
			Value: result.Email,
		})
	}

	return &identityv1.User{
		Id:                 result.ID,
		Status:             userStatusToProto(result.Status),
		Nickname:           result.Name,
		AvatarUrl:          "",
		Contacts:           contacts,
		ExternalIdentities: []*identityv1.ExternalIdentity{},
		CreatedAt:          nil,
		UpdatedAt:          nil,
	}
}

// userStatusToProto 将领域层 UserStatus 转换为 proto 枚举
func userStatusToProto(status userDomain.UserStatus) identityv1.UserStatus {
	switch status {
	case userDomain.UserActive:
		return identityv1.UserStatus_USER_STATUS_ACTIVE
	case userDomain.UserInactive:
		return identityv1.UserStatus_USER_STATUS_INACTIVE
	case userDomain.UserBlocked:
		return identityv1.UserStatus_USER_STATUS_BLOCKED
	default:
		return identityv1.UserStatus_USER_STATUS_UNSPECIFIED
	}
}

// userDomainToProto 将领域层 User 转换为 proto User
func userDomainToProto(u *userDomain.User) *identityv1.User {
	if u == nil {
		return nil
	}

	contacts := make([]*identityv1.VerifiedContact, 0)
	if !u.Phone.IsEmpty() {
		contacts = append(contacts, &identityv1.VerifiedContact{
			Type:  identityv1.ContactType_CONTACT_TYPE_PHONE,
			Value: u.Phone.String(),
		})
	}
	if !u.Email.IsEmpty() {
		contacts = append(contacts, &identityv1.VerifiedContact{
			Type:  identityv1.ContactType_CONTACT_TYPE_EMAIL,
			Value: u.Email.String(),
		})
	}

	return &identityv1.User{
		Id:                 u.ID.String(),
		Status:             userStatusToProto(u.Status),
		Nickname:           u.Name,
		AvatarUrl:          "",
		Contacts:           contacts,
		ExternalIdentities: []*identityv1.ExternalIdentity{},
		CreatedAt:          nil,
		UpdatedAt:          nil,
	}
}

// ============= ChildResult 转换 =============

// childResultToProto 将应用层 ChildResult 转换为 proto Child
func childResultToProto(result *childApp.ChildResult) *identityv1.Child {
	if result == nil {
		return nil
	}

	return &identityv1.Child{
		Id:        result.ID,
		LegalName: result.Name,
		Gender:    genderStringToProto(result.Gender),
		Dob:       result.Birthday,
		Identity: &identityv1.IdentityDocument{
			Type:         "id_card",
			MaskedNumber: result.IDCard,
		},
		Stats: &identityv1.PhysicalStats{
			HeightCm: int32(result.Height),
			WeightKg: formatWeight(result.Weight),
		},
		CreatedAt: nil,
		UpdatedAt: nil,
	}
}

// childResultToProtoFromGuardianship 从监护关系结果中提取儿童信息
func childResultToProtoFromGuardianship(result *guardianshipApp.GuardianshipResult) *identityv1.Child {
	if result == nil {
		return nil
	}

	return &identityv1.Child{
		Id:        result.ChildID,
		LegalName: result.ChildName,
		Gender:    genderStringToProto(result.ChildGender),
		Dob:       result.ChildBirthday,
		Identity:  nil,
		Stats:     nil,
		CreatedAt: nil,
		UpdatedAt: nil,
	}
}

// childDomainToProto 将领域层 Child 转换为 proto Child
func childDomainToProto(c *childDomain.Child) *identityv1.Child {
	if c == nil {
		return nil
	}

	return &identityv1.Child{
		Id:        c.ID.String(),
		LegalName: c.Name,
		Gender:    genderMetaToProto(c.Gender),
		Dob:       c.Birthday.String(),
		Identity: &identityv1.IdentityDocument{
			Type:         "id_card",
			MaskedNumber: c.IDCard.String(),
		},
		Stats: &identityv1.PhysicalStats{
			HeightCm: int32(c.Height.Tenths() / 10),
			WeightKg: c.Weight.String(),
		},
		CreatedAt: nil,
		UpdatedAt: nil,
	}
}

// genderMetaToProto 将 meta.Gender 转换为 proto 枚举
func genderMetaToProto(gender meta.Gender) identityv1.Gender {
	switch gender {
	case meta.GenderMale:
		return identityv1.Gender_GENDER_MALE
	case meta.GenderFemale:
		return identityv1.Gender_GENDER_FEMALE
	default:
		return identityv1.Gender_GENDER_UNSPECIFIED
	}
}

// genderStringToProto 将字符串性别转换为 proto 枚举
func genderStringToProto(gender string) identityv1.Gender {
	switch gender {
	case "male":
		return identityv1.Gender_GENDER_MALE
	case "female":
		return identityv1.Gender_GENDER_FEMALE
	default:
		return identityv1.Gender_GENDER_UNSPECIFIED
	}
}

// formatWeight 格式化体重（克转千克）
func formatWeight(weightGrams uint32) string {
	if weightGrams == 0 {
		return ""
	}
	kg := float64(weightGrams) / 1000.0
	return strconv.FormatFloat(kg, 'f', 2, 64)
}

// ============= GuardianshipResult 转换 =============

// guardianshipResultToProto 将应用层 GuardianshipResult 转换为 proto Guardianship
func guardianshipResultToProto(result *guardianshipApp.GuardianshipResult) *identityv1.Guardianship {
	if result == nil {
		return nil
	}

	return &identityv1.Guardianship{
		Id:        strconv.FormatUint(result.ID, 10),
		UserId:    result.UserID,
		ChildId:   result.ChildID,
		Relation:  stringToProtoRelation(result.Relation),
		Since:     nil, // 需要解析时间字符串
		RevokedAt: nil,
	}
}

// guardianshipDomainToProto 将领域层 Guardianship 转换为 proto Guardianship
func guardianshipDomainToProto(g *guardianshipDomain.Guardianship) *identityv1.Guardianship {
	if g == nil {
		return nil
	}

	guardianship := &identityv1.Guardianship{
		Id:       g.ID.String(),
		UserId:   g.User.String(),
		ChildId:  g.Child.String(),
		Relation: relationToProto(g.Rel),
		Since:    timestamppb.New(g.EstablishedAt),
	}

	if g.RevokedAt != nil && !g.RevokedAt.IsZero() {
		guardianship.RevokedAt = timestamppb.New(*g.RevokedAt)
	}

	return guardianship
}

// relationToProto 将领域层 Relation 转换为 proto 枚举
func relationToProto(relation guardianshipDomain.Relation) identityv1.GuardianshipRelation {
	switch relation {
	case guardianshipDomain.RelSelf:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_SELF
	case guardianshipDomain.RelParent:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_PARENT
	case guardianshipDomain.RelGrandparents:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_GRANDPARENT
	case guardianshipDomain.RelOther:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_OTHER
	default:
		return identityv1.GuardianshipRelation_GUARDIANSHIP_RELATION_UNSPECIFIED
	}
}

// ============= 错误转换 =============

// toGRPCError 将应用层错误转换为 gRPC 错误
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// 尝试解析错误码
	if coder := errors.ParseCoder(err); coder != nil {
		switch coder.HTTPStatus() {
		case 400:
			return status.Error(codes.InvalidArgument, coder.String())
		case 401:
			return status.Error(codes.Unauthenticated, coder.String())
		case 403:
			return status.Error(codes.PermissionDenied, coder.String())
		case 404:
			return status.Error(codes.NotFound, coder.String())
		case 409:
			return status.Error(codes.AlreadyExists, coder.String())
		case 429:
			return status.Error(codes.ResourceExhausted, coder.String())
		case 500:
			return status.Error(codes.Internal, coder.String())
		case 503:
			return status.Error(codes.Unavailable, coder.String())
		default:
			return status.Error(codes.Unknown, coder.String())
		}
	}

	// 默认返回内部错误
	return status.Error(codes.Internal, err.Error())
}
