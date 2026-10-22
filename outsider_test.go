package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	mwanachamaactor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

func seatOutsider(t *testing.T, mgr mwanachamaactor.UserManager, groupID, actorID string) {
	t.Helper()
	ctx := context.Background()
	client, err := mgr.CreateRoleKind(ctx, mwanachamaactor.RoleKind{Name: "Client", Outsider: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if !client.Outsider {
		t.Fatal("an outsider kind read back without its mark")
	}
	if _, err := mgr.GrantRole(ctx, mwanachamaactor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: client.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
}

func TestARoleKindIsNoOutsiderUnlessCreatedSo(t *testing.T) {
	mgr := newTestManager(t)
	k, err := mgr.CreateRoleKind(context.Background(), mwanachamaactor.RoleKind{Name: "Treasurer"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if k.Outsider {
		t.Fatal("a new kind is an outsider kind by default")
	}
}

func TestAnOutsiderCannotJoinAGroup(t *testing.T) {
	mgr := newTestManager(t)
	g, a := seedJoin(t, mgr, models.JoinOpen)
	firm, _ := seedJoin(t, mgr, models.JoinOpen)
	seatOutsider(t, mgr, firm.ID, a.ID)
	if _, err := mgr.JoinGroup(context.Background(), g.ID, a.ID, ""); !errors.Is(err, mwanachamaactor.ErrOutsiderRole) {
		t.Fatalf("JoinGroup by an outsider = %v, want ErrOutsiderRole", err)
	}
	if registered(t, mgr, a.ID, g.ID) {
		t.Fatal("an outsider was registered at a group")
	}
}

func TestAnOutsiderCannotBeInvitedToAGroup(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	leader, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	seatLeader(t, mgr, g.ID, leader.ID)
	seatOutsider(t, mgr, g.ID, a.ID)
	if _, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "", leader.ID, ""); !errors.Is(err, mwanachamaactor.ErrOutsiderRole) {
		t.Fatalf("InviteToGroup of an outsider = %v, want ErrOutsiderRole", err)
	}
}

func TestAnOutsiderIsLeftOutOfMemberListsAndCounts(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, member := seedJoin(t, mgr, models.JoinOpen)
	outsider, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Client"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	for _, id := range []string{member.ID, outsider.ID} {
		if _, err := mgr.AssignGroup(ctx, mwanachamaactor.ActorGroupAssignment{ActorID: id, GroupID: g.ID}); err != nil {
			t.Fatalf("AssignGroup: %v", err)
		}
	}
	seatOutsider(t, mgr, g.ID, outsider.ID)
	members, err := mgr.ListActorsForGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("ListActorsForGroup: %v", err)
	}
	if len(members) != 1 || members[0].ActorID != member.ID {
		t.Fatalf("members = %+v, want only %s", members, member.ID)
	}
	dash, err := mgr.GroupDashboard(ctx, g.ID)
	if err != nil {
		t.Fatalf("GroupDashboard: %v", err)
	}
	if dash.MemberCount != 1 {
		t.Fatalf("MemberCount = %d, want 1", dash.MemberCount)
	}
}

func TestAnInvitationIssuedBeforeTheOutsiderSeatCannotBeAccepted(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()
	g, a := seedJoin(t, mgr, models.JoinConfirmed)
	leader, err := mgr.CreateActor(ctx, mwanachamaactor.Actor{DisplayName: "Baraka"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	seatLeader(t, mgr, g.ID, leader.ID)
	inv, err := mgr.InviteToGroup(ctx, g.ID, a.ID, "", leader.ID, "")
	if err != nil {
		t.Fatalf("InviteToGroup: %v", err)
	}
	firm, _ := seedJoin(t, mgr, models.JoinOpen)
	seatOutsider(t, mgr, firm.ID, a.ID)
	if _, err := mgr.AcceptInvitation(ctx, inv.ID, a.ID); !errors.Is(err, mwanachamaactor.ErrOutsiderRole) {
		t.Fatalf("AcceptInvitation by an outsider = %v, want ErrOutsiderRole", err)
	}
	if _, err := mgr.DeclineInvitation(ctx, inv.ID, a.ID); err != nil {
		t.Fatalf("DeclineInvitation by an outsider: %v", err)
	}
}
