package cache

import "testing"

func TestCatalogContainsAllCurrentFamilies(t *testing.T) {
	families := Families()
	if len(families) != 10 {
		t.Fatalf("Families() count = %d, want %d", len(families), 10)
	}

	expected := map[Family]struct{}{
		FamilyAuthnRefreshToken:        {},
		FamilyAuthnRevokedAccessToken:  {},
		FamilyAuthnSession:             {},
		FamilyAuthnUserSessionIndex:    {},
		FamilyAuthnAccountSessionIndex: {},
		FamilyAuthnLoginOTP:            {},
		FamilyAuthnLoginOTPSendGate:    {},
		FamilyIDPWechatAccessToken:     {},
		FamilyIDPWechatSDK:             {},
		FamilyAuthnJWKSPublishSnapshot: {},
	}

	for _, descriptor := range families {
		if _, ok := expected[descriptor.Family]; !ok {
			t.Fatalf("unexpected family %q", descriptor.Family)
		}
		delete(expected, descriptor.Family)
	}

	if len(expected) != 0 {
		t.Fatalf("missing families: %v", expected)
	}
}

func TestCurrentRedisBackedFamiliesUseExpectedDataTypes(t *testing.T) {
	expected := map[Family]RedisDataType{
		FamilyAuthnRefreshToken:        RedisDataTypeString,
		FamilyAuthnRevokedAccessToken:  RedisDataTypeString,
		FamilyAuthnSession:             RedisDataTypeString,
		FamilyAuthnUserSessionIndex:    RedisDataTypeZSet,
		FamilyAuthnAccountSessionIndex: RedisDataTypeZSet,
		FamilyAuthnLoginOTP:            RedisDataTypeString,
		FamilyAuthnLoginOTPSendGate:    RedisDataTypeString,
		FamilyIDPWechatAccessToken:     RedisDataTypeString,
		FamilyIDPWechatSDK:             RedisDataTypeString,
	}
	for _, descriptor := range Families() {
		wantType, ok := expected[descriptor.Family]
		if !ok || descriptor.Backend != BackendKindRedis {
			continue
		}
		if descriptor.RedisType != wantType {
			t.Fatalf("family %q redis type = %q, want %q", descriptor.Family, descriptor.RedisType, wantType)
		}
	}
}

func TestGetFamily(t *testing.T) {
	descriptor, ok := GetFamily(FamilyIDPWechatAccessToken)
	if !ok {
		t.Fatalf("GetFamily(%q) should exist", FamilyIDPWechatAccessToken)
	}
	if descriptor.Codec != ValueCodecKindJSON {
		t.Fatalf("codec = %q, want %q", descriptor.Codec, ValueCodecKindJSON)
	}
	if !descriptor.Policy.HasInternalRefreshCoordination {
		t.Fatalf("expected wechat access token family to expose refresh coordination")
	}
}
