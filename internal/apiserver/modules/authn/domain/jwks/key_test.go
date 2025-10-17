package jwks

import (
	"testing"
	"time"

	"github.com/fangcun-mount/iam-contracts/internal/pkg/code"
	"github.com/fangcun-mount/iam-contracts/pkg/errors"
)

// ==================== 工厂方法测试 ====================

func TestNewKey(t *testing.T) {
	kid := "test-key-1"
	jwk := PublicJWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   strPtr("test-n"),
		E:   strPtr("test-e"),
	}

	t.Run("创建默认密钥", func(t *testing.T) {
		key := NewKey(kid, jwk)

		if key.Kid != kid {
			t.Errorf("Kid = %v, want %v", key.Kid, kid)
		}
		if key.Status != KeyActive {
			t.Errorf("Status = %v, want %v", key.Status, KeyActive)
		}
		if key.JWK.Kty != "RSA" {
			t.Errorf("JWK.Kty = %v, want RSA", key.JWK.Kty)
		}
	})

	t.Run("使用选项创建密钥", func(t *testing.T) {
		now := time.Now()
		notBefore := now.Add(-1 * time.Hour)
		notAfter := now.Add(24 * time.Hour)

		key := NewKey(kid, jwk,
			WithNotBefore(notBefore),
			WithNotAfter(notAfter),
			WithStatus(KeyGrace),
		)

		if key.Status != KeyGrace {
			t.Errorf("Status = %v, want %v", key.Status, KeyGrace)
		}
		if key.NotBefore == nil || !key.NotBefore.Equal(notBefore) {
			t.Errorf("NotBefore = %v, want %v", key.NotBefore, notBefore)
		}
		if key.NotAfter == nil || !key.NotAfter.Equal(notAfter) {
			t.Errorf("NotAfter = %v, want %v", key.NotAfter, notAfter)
		}
	})
}

// ==================== 状态查询测试 ====================

func TestKeyStatusQueries(t *testing.T) {
	tests := []struct {
		name       string
		status     KeyStatus
		wantActive bool
		wantGrace  bool
		wantRetire bool
	}{
		{
			name:       "Active状态",
			status:     KeyActive,
			wantActive: true,
			wantGrace:  false,
			wantRetire: false,
		},
		{
			name:       "Grace状态",
			status:     KeyGrace,
			wantActive: false,
			wantGrace:  true,
			wantRetire: false,
		},
		{
			name:       "Retired状态",
			status:     KeyRetired,
			wantActive: false,
			wantGrace:  false,
			wantRetire: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{Status: tt.status}

			if got := key.IsActive(); got != tt.wantActive {
				t.Errorf("IsActive() = %v, want %v", got, tt.wantActive)
			}
			if got := key.IsGrace(); got != tt.wantGrace {
				t.Errorf("IsGrace() = %v, want %v", got, tt.wantGrace)
			}
			if got := key.IsRetired(); got != tt.wantRetire {
				t.Errorf("IsRetired() = %v, want %v", got, tt.wantRetire)
			}
		})
	}
}

// ==================== 能力查询测试 ====================

func TestKeyCanSign(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		status   KeyStatus
		notAfter *time.Time
		wantSign bool
	}{
		{
			name:     "Active状态且未过期",
			status:   KeyActive,
			notAfter: timePtr(now.Add(1 * time.Hour)),
			wantSign: true,
		},
		{
			name:     "Active状态但已过期",
			status:   KeyActive,
			notAfter: timePtr(now.Add(-1 * time.Hour)),
			wantSign: false,
		},
		{
			name:     "Grace状态",
			status:   KeyGrace,
			notAfter: timePtr(now.Add(1 * time.Hour)),
			wantSign: false,
		},
		{
			name:     "Retired状态",
			status:   KeyRetired,
			notAfter: timePtr(now.Add(1 * time.Hour)),
			wantSign: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{
				Status:   tt.status,
				NotAfter: tt.notAfter,
			}

			if got := key.CanSign(); got != tt.wantSign {
				t.Errorf("CanSign() = %v, want %v", got, tt.wantSign)
			}
		})
	}
}

func TestKeyCanVerify(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		status     KeyStatus
		notAfter   *time.Time
		wantVerify bool
	}{
		{
			name:       "Active状态且未过期",
			status:     KeyActive,
			notAfter:   timePtr(now.Add(1 * time.Hour)),
			wantVerify: true,
		},
		{
			name:       "Grace状态且未过期",
			status:     KeyGrace,
			notAfter:   timePtr(now.Add(1 * time.Hour)),
			wantVerify: true,
		},
		{
			name:       "Active状态但已过期",
			status:     KeyActive,
			notAfter:   timePtr(now.Add(-1 * time.Hour)),
			wantVerify: false,
		},
		{
			name:       "Retired状态",
			status:     KeyRetired,
			notAfter:   timePtr(now.Add(1 * time.Hour)),
			wantVerify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{
				Status:   tt.status,
				NotAfter: tt.notAfter,
			}

			if got := key.CanVerify(); got != tt.wantVerify {
				t.Errorf("CanVerify() = %v, want %v", got, tt.wantVerify)
			}
		})
	}
}

func TestKeyShouldPublish(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		status      KeyStatus
		notAfter    *time.Time
		wantPublish bool
	}{
		{
			name:        "Active状态且未过期",
			status:      KeyActive,
			notAfter:    timePtr(now.Add(1 * time.Hour)),
			wantPublish: true,
		},
		{
			name:        "Grace状态且未过期",
			status:      KeyGrace,
			notAfter:    timePtr(now.Add(1 * time.Hour)),
			wantPublish: true,
		},
		{
			name:        "Retired状态",
			status:      KeyRetired,
			notAfter:    timePtr(now.Add(1 * time.Hour)),
			wantPublish: false,
		},
		{
			name:        "Active但已过期",
			status:      KeyActive,
			notAfter:    timePtr(now.Add(-1 * time.Hour)),
			wantPublish: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{
				Status:   tt.status,
				NotAfter: tt.notAfter,
			}

			if got := key.ShouldPublish(); got != tt.wantPublish {
				t.Errorf("ShouldPublish() = %v, want %v", got, tt.wantPublish)
			}
		})
	}
}

// ==================== 有效期检查测试 ====================

func TestKeyIsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		notAfter    *time.Time
		wantExpired bool
	}{
		{
			name:        "未设置过期时间",
			notAfter:    nil,
			wantExpired: false,
		},
		{
			name:        "未过期",
			notAfter:    timePtr(now.Add(1 * time.Hour)),
			wantExpired: false,
		},
		{
			name:        "已过期",
			notAfter:    timePtr(now.Add(-1 * time.Hour)),
			wantExpired: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{NotAfter: tt.notAfter}

			if got := key.IsExpired(now); got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestKeyIsNotYetValid(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name         string
		notBefore    *time.Time
		wantNotValid bool
	}{
		{
			name:         "未设置生效时间",
			notBefore:    nil,
			wantNotValid: false,
		},
		{
			name:         "已生效",
			notBefore:    timePtr(now.Add(-1 * time.Hour)),
			wantNotValid: false,
		},
		{
			name:         "尚未生效",
			notBefore:    timePtr(now.Add(1 * time.Hour)),
			wantNotValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{NotBefore: tt.notBefore}

			if got := key.IsNotYetValid(now); got != tt.wantNotValid {
				t.Errorf("IsNotYetValid() = %v, want %v", got, tt.wantNotValid)
			}
		})
	}
}

func TestKeyIsValidAt(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		notBefore *time.Time
		notAfter  *time.Time
		checkTime time.Time
		wantValid bool
	}{
		{
			name:      "在有效期内",
			notBefore: timePtr(now.Add(-1 * time.Hour)),
			notAfter:  timePtr(now.Add(1 * time.Hour)),
			checkTime: now,
			wantValid: true,
		},
		{
			name:      "尚未生效",
			notBefore: timePtr(now.Add(1 * time.Hour)),
			notAfter:  timePtr(now.Add(2 * time.Hour)),
			checkTime: now,
			wantValid: false,
		},
		{
			name:      "已过期",
			notBefore: timePtr(now.Add(-2 * time.Hour)),
			notAfter:  timePtr(now.Add(-1 * time.Hour)),
			checkTime: now,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{
				NotBefore: tt.notBefore,
				NotAfter:  tt.notAfter,
			}

			if got := key.IsValidAt(tt.checkTime); got != tt.wantValid {
				t.Errorf("IsValidAt() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

// ==================== 状态转换测试 ====================

func TestKeyEnterGrace(t *testing.T) {
	tests := []struct {
		name      string
		status    KeyStatus
		wantErr   bool
		wantState KeyStatus
	}{
		{
			name:      "从Active转到Grace成功",
			status:    KeyActive,
			wantErr:   false,
			wantState: KeyGrace,
		},
		{
			name:      "从Grace转到Grace失败",
			status:    KeyGrace,
			wantErr:   true,
			wantState: KeyGrace,
		},
		{
			name:      "从Retired转到Grace失败",
			status:    KeyRetired,
			wantErr:   true,
			wantState: KeyRetired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{Status: tt.status}

			err := key.EnterGrace()

			if (err != nil) != tt.wantErr {
				t.Errorf("EnterGrace() error = %v, wantErr %v", err, tt.wantErr)
			}
			if key.Status != tt.wantState {
				t.Errorf("Status = %v, want %v", key.Status, tt.wantState)
			}
		})
	}
}

func TestKeyRetire(t *testing.T) {
	tests := []struct {
		name      string
		status    KeyStatus
		wantErr   bool
		wantState KeyStatus
	}{
		{
			name:      "从Grace转到Retired成功",
			status:    KeyGrace,
			wantErr:   false,
			wantState: KeyRetired,
		},
		{
			name:      "从Active转到Retired失败",
			status:    KeyActive,
			wantErr:   true,
			wantState: KeyActive,
		},
		{
			name:      "从Retired转到Retired失败",
			status:    KeyRetired,
			wantErr:   true,
			wantState: KeyRetired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{Status: tt.status}

			err := key.Retire()

			if (err != nil) != tt.wantErr {
				t.Errorf("Retire() error = %v, wantErr %v", err, tt.wantErr)
			}
			if key.Status != tt.wantState {
				t.Errorf("Status = %v, want %v", key.Status, tt.wantState)
			}
		})
	}
}

func TestKeyForceRetire(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus KeyStatus
	}{
		{name: "从Active强制退役", initialStatus: KeyActive},
		{name: "从Grace强制退役", initialStatus: KeyGrace},
		{name: "从Retired强制退役", initialStatus: KeyRetired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &Key{Status: tt.initialStatus}

			key.ForceRetire()

			if key.Status != KeyRetired {
				t.Errorf("Status = %v, want %v", key.Status, KeyRetired)
			}
		})
	}
}

// ==================== 验证方法测试 ====================

func TestKeyValidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		key         *Key
		wantErrCode int
	}{
		{
			name: "合法的RSA密钥",
			key: &Key{
				Kid:    "test-rsa",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "test-rsa",
					N:   strPtr("test-n"),
					E:   strPtr("test-e"),
				},
			},
			wantErrCode: 0,
		},
		{
			name: "合法的EC密钥",
			key: &Key{
				Kid:    "test-ec",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "EC",
					Use: "sig",
					Alg: "ES256",
					Kid: "test-ec",
					Crv: strPtr("P-256"),
					X:   strPtr("test-x"),
					Y:   strPtr("test-y"),
				},
			},
			wantErrCode: 0,
		},
		{
			name: "Kid为空",
			key: &Key{
				Kid:    "",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "",
					N:   strPtr("test-n"),
					E:   strPtr("test-e"),
				},
			},
			wantErrCode: code.ErrInvalidKid,
		},
		{
			name: "JWK.Kty为空",
			key: &Key{
				Kid:    "test",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "",
					Use: "sig",
					Alg: "RS256",
					Kid: "test",
				},
			},
			wantErrCode: code.ErrInvalidJWK,
		},
		{
			name: "JWK.Use不是sig",
			key: &Key{
				Kid:    "test",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "enc",
					Alg: "RS256",
					Kid: "test",
					N:   strPtr("test-n"),
					E:   strPtr("test-e"),
				},
			},
			wantErrCode: code.ErrInvalidJWKUse,
		},
		{
			name: "Kid不匹配",
			key: &Key{
				Kid:    "test-1",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "test-2",
					N:   strPtr("test-n"),
					E:   strPtr("test-e"),
				},
			},
			wantErrCode: code.ErrKidMismatch,
		},
		{
			name: "RSA缺少参数",
			key: &Key{
				Kid:    "test",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "test",
					N:   strPtr("test-n"),
					// 缺少 E
				},
			},
			wantErrCode: code.ErrMissingRSAParams,
		},
		{
			name: "EC缺少参数",
			key: &Key{
				Kid:    "test",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "EC",
					Use: "sig",
					Alg: "ES256",
					Kid: "test",
					Crv: strPtr("P-256"),
					X:   strPtr("test-x"),
					// 缺少 Y
				},
			},
			wantErrCode: code.ErrMissingECParams,
		},
		{
			name: "不支持的密钥类型",
			key: &Key{
				Kid:    "test",
				Status: KeyActive,
				JWK: PublicJWK{
					Kty: "UNKNOWN",
					Use: "sig",
					Alg: "RS256",
					Kid: "test",
				},
			},
			wantErrCode: code.ErrUnsupportedKty,
		},
		{
			name: "有效期时间范围错误",
			key: &Key{
				Kid:       "test",
				Status:    KeyActive,
				NotBefore: timePtr(now.Add(1 * time.Hour)),
				NotAfter:  timePtr(now.Add(-1 * time.Hour)), // NotAfter 在 NotBefore 之前
				JWK: PublicJWK{
					Kty: "RSA",
					Use: "sig",
					Alg: "RS256",
					Kid: "test",
					N:   strPtr("test-n"),
					E:   strPtr("test-e"),
				},
			},
			wantErrCode: code.ErrInvalidTimeRange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.key.Validate()

			if tt.wantErrCode == 0 {
				if err != nil {
					t.Errorf("Validate() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("Validate() error = nil, want error code %d", tt.wantErrCode)
					return
				}
				coder := errors.ParseCoder(err)
				if coder.Code() != tt.wantErrCode {
					t.Errorf("Validate() error code = %d, want %d", coder.Code(), tt.wantErrCode)
				}
			}
		})
	}
}

// ==================== 辅助函数 ====================

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
