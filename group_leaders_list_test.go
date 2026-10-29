package mwanachamaactor_test

import (
	"context"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
)

func TestAGroupsLeadersAreNamedAndOnlyThoseWithAnActiveLeadingSeatThere(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	ward := mustGroup(t, mgr, "Ward A")
	beside := mustGroup(t, mgr, "Ward B")
	grace := mustActor(t, mgr, "Grace Adhiambo")
	ended := mustActor(t, mgr, "Joseph Ouma")
	elsewhere := mustActor(t, mgr, "Peter Mwangi")
	plain := mustActor(t, mgr, "Asha Otieno")
	chair := leadingKind(t, mgr)
	treasurer, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	grant := func(kindID, groupID, actorID string) mwanachamaactor.ActorRoleAssignment {
		seat, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kindID})
		if err != nil {
			t.Fatalf("GrantRole: %v", err)
		}
		return seat
	}
	grant(chair.ID, ward.ID, grace.ID)
	grant(chair.ID, ward.ID, ended.ID)
	grant(chair.ID, beside.ID, elsewhere.ID)
	grant(treasurer.ID, ward.ID, plain.ID)
	if err := mgr.RevokeRole(ctx, grantedSeat(t, mgr, ward.ID, ended.ID)); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}

	got, err := mgr.ListGroupLeaders(ctx, ward.ID)
	if err != nil {
		t.Fatalf("ListGroupLeaders: %v", err)
	}
	if len(got) != 1 || got[0].ActorID != grace.ID || got[0].DisplayName != "Grace Adhiambo" {
		t.Fatalf("leaders of Ward A = %+v, want only Grace Adhiambo", got)
	}
	none, err := mgr.ListGroupLeaders(ctx, "no-such-group")
	if err != nil || len(none) != 0 {
		t.Fatalf("leaders of no group = %+v, %v, want none", none, err)
	}
}

func grantedSeat(t *testing.T, mgr mwanachamaactor.UserManager, groupID, actorID string) string {
	t.Helper()
	seats, err := mgr.ListRoleAssignmentsForGroup(context.Background(), groupID, true)
	if err != nil {
		t.Fatalf("ListRoleAssignmentsForGroup: %v", err)
	}
	for _, s := range seats {
		if s.ActorID == actorID {
			return s.ID
		}
	}
	t.Fatalf("no seat for %s in %s", actorID, groupID)
	return ""
}
