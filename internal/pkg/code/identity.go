package code

// Base user & identity module errors.
const (
	// ErrUserNotFound - 404: User not found.
	ErrUserNotFound = 110001

	// ErrUserAlreadyExists - 400: User already exist.
	ErrUserAlreadyExists = 110002

	// ErrUserBasicInfoInvalid - 400: User basic info is invalid.
	ErrUserBasicInfoInvalid = 110003

	// ErrUserStatusInvalid - 400: User status is invalid.
	ErrUserStatusInvalid = 110004

	// ErrUserInvalid - 400: User is invalid.
	ErrUserInvalid = 110005

	// ErrUserBlocked - 403: User is blocked.
	ErrUserBlocked = 110006

	// ErrUserInactive - 403: User is inactive.
	ErrUserInactive = 110007
)

// Identity (child, guardianship) module errors (110101+).
const (
	// ErrIdentityUserBlocked - 403: 用户被封禁
	ErrIdentityUserBlocked = 110101

	// ErrIdentityChildExists - 400: 儿童档案已存在
	ErrIdentityChildExists = 110102

	// ErrIdentityChildNotFound - 404: 儿童不存在
	ErrIdentityChildNotFound = 110103

	// ErrIdentityGuardianshipExists - 400: 监护关系已存在
	ErrIdentityGuardianshipExists = 110104

	// ErrIdentityGuardianshipNotFound - 404: 监护关系不存在
	ErrIdentityGuardianshipNotFound = 110105
)
