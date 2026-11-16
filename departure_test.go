package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func TestAnActorWhoLeavesLosesSeatsRosterPlacesAndPendingRequests(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g := mustGroup(t, mgr, "Ward A")
	other := mustGroup(t, mgr, "Ward B")
	a := mustActor(t, mgr, "Chausiku")
	if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: a.ID, GroupID: g.ID}); err != nil {
		t.Fatalf("AssignGroup: %v", err)
	}
	kind, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Coordinator"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	seat, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: a.ID, GroupID: g.ID, KindID: kind.ID})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	if _, err := mgr.SetGroupJoinMode(ctx, other.ID, models.JoinConfirmed); err != nil {
		t.Fatalf("SetGroupJoinMode: %v", err)
	}
	res, err := mgr.JoinGroup(ctx, other.ID, a.ID, "")
	if err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}

	if err := mgr.LeaveAllGroups(ctx, a.ID); err != nil {
		t.Fatalf("LeaveAllGroups: %v", err)
	}

	if registered(t, mgr, a.ID, g.ID) {
		t.Error("a departed actor kept a roster place")
	}
	got, err := mgr.GetRoleAssignment(ctx, seat.ID)
	if err != nil || got.Active {
		t.Errorf("seat = %+v, %v; want ended", got, err)
	}
	if _, err := mgr.OperatorApproveJoinRequest(ctx, res.Request.ID, "operator-1"); !errors.Is(err, mwanachamaactor.ErrJoinRequestDecided) {
		t.Errorf("approval after departure = %v, want ErrJoinRequestDecided", err)
	}
	if err := mgr.LeaveAllGroups(ctx, ""); !errors.Is(err, mwanachamaactor.ErrInvalidActor) {
		t.Errorf("LeaveAllGroups with no actor = %v, want ErrInvalidActor", err)
	}
}
