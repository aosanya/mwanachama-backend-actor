package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func delegationPair(t *testing.T) (mwanachamaactor.UserManager, string, string) {
	t.Helper()
	mgr := newTestManager(t)
	ctx := context.Background()
	a, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Delegate"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Subject"})
	if err != nil {
		t.Fatal(err)
	}
	return mgr, a.ID, b.ID
}

func TestALiveDelegationLetsTheDelegateActForTheSubjectOnly(t *testing.T) {
	mgr, a, b := delegationPair(t)
	ctx := context.Background()
	d, err := mgr.CreateDelegation(ctx, "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b})
	if err != nil {
		t.Fatalf("CreateDelegation: %v", err)
	}
	if d.GrantedBy != "granter" || d.GrantedAt == "" || d.RevokedAt != "" {
		t.Errorf("delegation = %+v", d)
	}
	if ok, err := mgr.ActsFor(ctx, a, b); err != nil || !ok {
		t.Errorf("ActsFor(a, b) = %v %v, want true", ok, err)
	}
	if ok, _ := mgr.ActsFor(ctx, b, a); ok {
		t.Error("a delegation runs one way only")
	}
}

func TestARevokedDelegationNoLongerActsAndCannotBeRevokedTwice(t *testing.T) {
	mgr, a, b := delegationPair(t)
	ctx := context.Background()
	d, _ := mgr.CreateDelegation(ctx, "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b})
	r, err := mgr.RevokeDelegation(ctx, d.ID, "revoker")
	if err != nil || r.RevokedBy != "revoker" || r.RevokedAt == "" {
		t.Fatalf("RevokeDelegation = %+v %v", r, err)
	}
	if ok, _ := mgr.ActsFor(ctx, a, b); ok {
		t.Error("a revoked delegation still acts")
	}
	if _, err := mgr.RevokeDelegation(ctx, d.ID, "revoker"); !errors.Is(err, mwanachamaactor.ErrDelegationRevoked) {
		t.Errorf("second revoke = %v, want ErrDelegationRevoked", err)
	}
	if _, err := mgr.RevokeDelegation(ctx, "nope", "revoker"); !errors.Is(err, mwanachamaactor.ErrDelegationNotFound) {
		t.Errorf("unknown revoke = %v, want ErrDelegationNotFound", err)
	}
	held, err := mgr.ListDelegations(ctx, b)
	if err != nil || len(held) != 1 || held[0].RevokedAt == "" {
		t.Errorf("ListDelegations = %+v %v, want the revoked record kept", held, err)
	}
}

func TestALapsedDelegationNoLongerActs(t *testing.T) {
	mgr, a, b := delegationPair(t)
	ctx := context.Background()
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if _, err := mgr.CreateDelegation(ctx, "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b, EndsAt: past}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := mgr.ActsFor(ctx, a, b); ok {
		t.Error("a lapsed delegation still acts")
	}
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	if _, err := mgr.CreateDelegation(ctx, "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b, EndsAt: future}); err != nil {
		t.Fatal(err)
	}
	if ok, _ := mgr.ActsFor(ctx, a, b); !ok {
		t.Error("a delegation ending later acts now")
	}
}

func TestADelegationIsRefusedForItselfAMissingActorOrNoGrantor(t *testing.T) {
	mgr, a, b := delegationPair(t)
	ctx := context.Background()
	cases := []struct {
		name    string
		by      string
		grant   mwanachamaactor.DelegationGrant
		wantErr error
	}{
		{"self", "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: a}, mwanachamaactor.ErrInvalidDelegation},
		{"no grantor", "", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b}, mwanachamaactor.ErrInvalidDelegation},
		{"bad end", "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: b, EndsAt: "soon"}, mwanachamaactor.ErrInvalidDelegation},
		{"missing subject", "granter", mwanachamaactor.DelegationGrant{DelegateID: a, SubjectID: "ghost"}, mwanachamaactor.ErrActorNotFound},
		{"missing delegate", "granter", mwanachamaactor.DelegationGrant{DelegateID: "ghost", SubjectID: b}, mwanachamaactor.ErrActorNotFound},
	}
	for _, c := range cases {
		if _, err := mgr.CreateDelegation(ctx, c.by, c.grant); !errors.Is(err, c.wantErr) {
			t.Errorf("%s: err = %v, want %v", c.name, err, c.wantErr)
		}
	}
	if ok, _ := mgr.ActsFor(ctx, a, a); ok {
		t.Error("an actor acts for themselves through no delegation")
	}
}
