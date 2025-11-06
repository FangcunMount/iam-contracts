package user

import (
	"context"

	"gorm.io/gorm"

	"github.com/FangcunMount/iam-contracts/internal/pkg/meta"
)

type stubUserRepository struct {
	usersByID      map[uint64]*User
	usersByPhone   map[string]*User
	findErr        error
	phoneErr       error
	updateArgs     []*User
	createArgs     []*User
	findIDCalls    int
	findPhoneCalls int
}

func newStubUserRepository() *stubUserRepository {
	return &stubUserRepository{
		usersByID:    make(map[uint64]*User),
		usersByPhone: make(map[string]*User),
	}
}

func (s *stubUserRepository) Create(ctx context.Context, user *User) error {
	s.createArgs = append(s.createArgs, user)
	if s.findErr != nil {
		return s.findErr
	}
	if user != nil {
		s.usersByID[user.ID.ToUint64()] = user
		s.usersByPhone[user.Phone.String()] = user
	}
	return nil
}

func (s *stubUserRepository) FindByID(ctx context.Context, id meta.ID) (*User, error) {
	s.findIDCalls++
	if s.findErr != nil {
		return nil, s.findErr
	}
	if user, ok := s.usersByID[id.ToUint64()]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUserRepository) FindByPhone(ctx context.Context, phone meta.Phone) (*User, error) {
	s.findPhoneCalls++
	if s.phoneErr != nil {
		return nil, s.phoneErr
	}
	if user, ok := s.usersByPhone[phone.String()]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (s *stubUserRepository) Update(ctx context.Context, user *User) error {
	s.updateArgs = append(s.updateArgs, user)
	return s.findErr
}
