package authorization_test

import (
	"context"
	"errors"
	"testing"

	authorizationapp "github.com/FangcunMount/iam/v5/internal/apiserver/application/authz/authorization"
	appquery "github.com/FangcunMount/iam/v5/internal/apiserver/application/suggest/queryprofile"
	"github.com/FangcunMount/iam/v5/internal/apiserver/domain/suggest/visibility"
	suggestauthz "github.com/FangcunMount/iam/v5/internal/apiserver/infra/suggest/authorization"
)

type stubRouteAuth struct {
	allowAllProfiles  bool
	allowAllMobile    bool
	allowScopedMobile bool
	err               error
}

func (s stubRouteAuth) CheckRoutePermission(_ context.Context, _, _, action string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	switch action {
	case "list_all":
		return s.allowAllProfiles, nil
	case "search_by_mobile_all":
		return s.allowAllMobile, nil
	case appquery.ActionSearchByMobile:
		return s.allowScopedMobile, nil
	}
	return false, nil
}

var _ authorizationapp.RoutePermissionChecker = stubRouteAuth{}

func TestFactsReaderNilReturnsEmpty(t *testing.T) {
	r := suggestauthz.NewFactsReader(nil)
	facts, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{OperatorID: 100})
	if err != nil {
		t.Fatal(err)
	}
	if facts.AllProfilesAllowed || facts.ScopedMobileSearchAllowed {
		t.Fatalf("facts = %#v", facts)
	}
}

func TestFactsReaderAllProfilePermission(t *testing.T) {
	r := suggestauthz.NewFactsReader(stubRouteAuth{
		allowAllProfiles: true,
		allowAllMobile:   true,
	})
	facts, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{
		OperatorID: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !facts.AllProfilesAllowed || !facts.AllProfilesMobileSearchAllowed {
		t.Fatalf("facts = %#v", facts)
	}
}

func TestFactsReaderScopedMobilePermission(t *testing.T) {
	r := suggestauthz.NewFactsReader(stubRouteAuth{allowScopedMobile: true})
	facts, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{
		OperatorID: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if facts.AllProfilesAllowed || !facts.ScopedMobileSearchAllowed {
		t.Fatalf("facts = %#v", facts)
	}
}

func TestFactsReaderPlainUserNoMobile(t *testing.T) {
	r := suggestauthz.NewFactsReader(stubRouteAuth{})
	facts, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{
		OperatorID: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if facts.AllProfilesAllowed || facts.ScopedMobileSearchAllowed {
		t.Fatalf("facts = %#v", facts)
	}
}

func TestFactsReaderCheckerErrorFails(t *testing.T) {
	wantErr := errors.New("authz down")
	r := suggestauthz.NewFactsReader(stubRouteAuth{err: wantErr})
	_, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{
		OperatorID: 100,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestFactsReaderAllMobileSeparateFromScopedMobile(t *testing.T) {
	r := suggestauthz.NewFactsReader(stubRouteAuth{
		allowAllProfiles:  true,
		allowAllMobile:    true,
		allowScopedMobile: false,
	})
	facts, err := r.ReadAuthorizationFacts(context.Background(), visibility.Principal{
		OperatorID: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !facts.AllProfilesAllowed || !facts.AllProfilesMobileSearchAllowed {
		t.Fatalf("facts = %#v", facts)
	}
}
