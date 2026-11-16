package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func twoActors(t *testing.T, um actor.UserManager) (string, string) {
	t.Helper()
	ctx := context.Background()
	a, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Nagy Béla"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	b, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Szabó Éva"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	return a.ID, b.ID
}

func TestDelegationRefusesAGrantThatNamesNobodyOrTheSamePersonTwice(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, subject := twoActors(t, um)
	cases := map[string]struct {
		by    string
		grant models.DelegationGrant
	}{
		"no grantor":        {"", models.DelegationGrant{DelegateID: delegate, SubjectID: subject}},
		"no delegate":       {"owner", models.DelegationGrant{SubjectID: subject}},
		"no subject":        {"owner", models.DelegationGrant{DelegateID: delegate}},
		"same person":       {"owner", models.DelegationGrant{DelegateID: delegate, SubjectID: delegate}},
		"an unreadable end": {"owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject, EndsAt: "next week"}},
	}
	for name, c := range cases {
		if _, err := um.CreateDelegation(ctx, c.by, c.grant); !errors.Is(err, actor.ErrInvalidDelegation) {
			t.Errorf("%s: err = %v, want ErrInvalidDelegation", name, err)
		}
	}
}

func TestDelegationNeedsBothPeopleToExist(t *testing.T) {
	um := newManager(t)
	delegate, _ := twoActors(t, um)
	_, err := um.CreateDelegation(context.Background(), "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: "nobody"})
	if !errors.Is(err, actor.ErrActorNotFound) {
		t.Fatalf("err = %v, want ErrActorNotFound", err)
	}
}

func TestDelegateActsForTheSubjectUntilRevoked(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, subject := twoActors(t, um)
	d, err := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject})
	if err != nil {
		t.Fatalf("CreateDelegation: %v", err)
	}
	if d.GrantedBy != "owner" || d.GrantedAt == "" || d.EndsAt != "" {
		t.Fatalf("delegation = %+v", d)
	}
	if ok, _ := um.ActsFor(ctx, delegate, subject); !ok {
		t.Fatal("the delegate does not act for the subject after the grant")
	}
	if ok, _ := um.ActsFor(ctx, subject, delegate); ok {
		t.Fatal("a delegation works both ways")
	}
	revoked, err := um.RevokeDelegation(ctx, d.ID, "owner")
	if err != nil {
		t.Fatalf("RevokeDelegation: %v", err)
	}
	if revoked.RevokedBy != "owner" || revoked.RevokedAt == "" {
		t.Fatalf("revoked = %+v", revoked)
	}
	if ok, _ := um.ActsFor(ctx, delegate, subject); ok {
		t.Fatal("the delegate still acts after the revoke")
	}
}

func TestActsForIsFalseForBlankOrSelfPairs(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, _ := twoActors(t, um)
	for _, pair := range [][2]string{{"", delegate}, {delegate, ""}, {delegate, delegate}} {
		ok, err := um.ActsFor(ctx, pair[0], pair[1])
		if err != nil || ok {
			t.Errorf("ActsFor(%q, %q) = %v, %v; want false", pair[0], pair[1], ok, err)
		}
	}
}

func TestADelegationThatHasEndedNoLongerActs(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, subject := twoActors(t, um)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	if _, err := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject, EndsAt: past}); err != nil {
		t.Fatalf("CreateDelegation: %v", err)
	}
	if ok, _ := um.ActsFor(ctx, delegate, subject); ok {
		t.Fatal("an ended delegation still acts")
	}
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
	if _, err := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject, EndsAt: future}); err != nil {
		t.Fatalf("CreateDelegation: %v", err)
	}
	if ok, _ := um.ActsFor(ctx, delegate, subject); !ok {
		t.Fatal("a delegation that ends later does not act now")
	}
}

func TestRevokeRefusesAnUnknownBlankOrAlreadyRevokedDelegation(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, subject := twoActors(t, um)
	d, _ := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject})
	if _, err := um.RevokeDelegation(ctx, d.ID, " "); !errors.Is(err, actor.ErrInvalidDelegation) {
		t.Errorf("blank revoker: err = %v", err)
	}
	if _, err := um.RevokeDelegation(ctx, "missing", "owner"); !errors.Is(err, actor.ErrDelegationNotFound) {
		t.Errorf("unknown id: err = %v", err)
	}
	if _, err := um.RevokeDelegation(ctx, d.ID, "owner"); err != nil {
		t.Fatalf("first revoke: %v", err)
	}
	if _, err := um.RevokeDelegation(ctx, d.ID, "owner"); !errors.Is(err, actor.ErrDelegationRevoked) {
		t.Errorf("second revoke: err = %v, want ErrDelegationRevoked", err)
	}
}

func TestListDelegationsShowsTheOnesAPersonGivesOrReceivesOldestFirst(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	delegate, subject := twoActors(t, um)
	other, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Tóth Ilona"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	first, _ := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: delegate, SubjectID: subject})
	second, _ := um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: subject, SubjectID: delegate})
	um.CreateDelegation(ctx, "owner", models.DelegationGrant{DelegateID: other.ID, SubjectID: subject})
	got, err := um.ListDelegations(ctx, delegate)
	if err != nil {
		t.Fatalf("ListDelegations: %v", err)
	}
	if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
		t.Fatalf("delegations of the delegate = %+v, want the two it is in, oldest first", got)
	}
}
