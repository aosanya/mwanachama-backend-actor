package mwanachamaactor_test

import (
	"context"
	"strings"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func namedRoster(t *testing.T, mgr mwanachamaactor.UserManager, names []string) (mwanachamaactor.Group, map[string]string) {
	t.Helper()
	g := mustGroup(t, mgr, "Ward A")
	byID := map[string]string{}
	for _, n := range names {
		a := mustActor(t, mgr, n)
		if _, err := mgr.AssignGroup(context.Background(), mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
		byID[a.ID] = n
	}
	return g, byID
}

func namesOf(rows []mwanachamaactor.ActorGroupAssignment, byID map[string]string) string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = byID[r.ActorID]
	}
	return strings.Join(out, ",")
}

func TestARosterSearchKeepsTheMembersWhoseNameHoldsTheText(t *testing.T) {
	mgr := newTestManager(t)
	g, byID := namedRoster(t, mgr, []string{"Grace Adhiambo", "Asha Otieno", "Joseph Graceful", "Peter Mwangi"})

	got, err := mgr.ListGroupRoster(context.Background(), g.ID, "", 0, "  gRaCe ")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("search grace = %s, want the two Graces", namesOf(got, byID))
	}
	for _, r := range got {
		if !strings.Contains(strings.ToLower(byID[r.ActorID]), "grace") {
			t.Fatalf("search grace returned %s", byID[r.ActorID])
		}
	}
}

func TestARosterSearchTreatsPercentAndUnderscoreAsPlainText(t *testing.T) {
	mgr := newTestManager(t)
	g, byID := namedRoster(t, mgr, []string{"Grace Adhiambo", "Asha 100% Otieno", "Peter_Mwangi"})

	for text, want := range map[string]string{"%": "Asha 100% Otieno", "_": "Peter_Mwangi", "a_a": ""} {
		got, err := mgr.ListGroupRoster(context.Background(), g.ID, "", 0, text)
		if err != nil {
			t.Fatalf("ListGroupRoster %q: %v", text, err)
		}
		if namesOf(got, byID) != want {
			t.Fatalf("search %q = %q, want %q", text, namesOf(got, byID), want)
		}
	}
}

func TestARosterSearchFindsAMemberBeyondTheFirstPageAndStillPages(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, ids := rosterOf(t, mgr, 120)

	got, err := mgr.ListGroupRoster(ctx, g.ID, "", 0, "member 119")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("search for the last member = %d rows, want 1", len(got))
	}
	found, err := mgr.GetActor(ctx, got[0].ActorID)
	if err != nil || found.DisplayName != "Member 119" {
		t.Fatalf("found %+v, %v, want Member 119", found, err)
	}

	first, err := mgr.ListGroupRoster(ctx, g.ID, "", 30, "member 0")
	if err != nil || len(first) != 30 {
		t.Fatalf("first page of member 0 = %d rows, %v, want 30", len(first), err)
	}
	next, err := mgr.ListGroupRoster(ctx, g.ID, first[29].ActorID, 200, "member 0")
	if err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
	if len(first)+len(next) != 100 {
		t.Fatalf("member 1 matches %d + %d rows, want 100 (000 to 099)", len(first), len(next))
	}
	if len(ids) != 120 {
		t.Fatalf("fixture = %d", len(ids))
	}
	if next[0].ActorID <= first[29].ActorID {
		t.Fatalf("next page starts at %s, not after %s", next[0].ActorID, first[29].ActorID)
	}
}

func TestARosterSearchNeverBringsBackSomeoneFromAnotherGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, _ := namedRoster(t, mgr, []string{"Grace Adhiambo"})
	other := mustGroup(t, mgr, "Ward B")
	stranger := mustActor(t, mgr, "Grace Elsewhere")
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: stranger.ID, GroupID: other.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	got, err := mgr.ListGroupRoster(ctx, g.ID, "", 0, "grace")
	if err != nil || len(got) != 1 || got[0].GroupID != g.ID {
		t.Fatalf("search = %+v, %v, want only Ward A's Grace", got, err)
	}
	if _, err := mgr.ListGroupRoster(ctx, g.ID, "", 0, "nobody"); err != nil {
		t.Fatalf("ListGroupRoster: %v", err)
	}
}
