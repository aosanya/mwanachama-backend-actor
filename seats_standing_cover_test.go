package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
	"github.com/aosanya/mwanachama-backend-actor/models"
)

type seatWorld struct {
	um      actor.UserManager
	group   actor.Group
	holder  actor.Actor
	granter actor.Actor
	kind    actor.RoleKind
}

func newSeatWorld(t *testing.T) seatWorld {
	t.Helper()
	um := newManager(t)
	ctx := context.Background()
	g, _ := um.CreateGroup(ctx, actor.Group{Name: "Városi tanács", JoinMode: models.JoinOpen})
	holder, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Szabó Sára"})
	granter, _ := um.CreateActor(ctx, actor.Actor{DisplayName: "Polgármester Pál"})
	kind, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Jegyző"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	return seatWorld{um: um, group: g, holder: holder, granter: granter, kind: kind}
}

func (w seatWorld) grant(t *testing.T) actor.ActorRoleAssignment {
	t.Helper()
	a, err := w.um.GrantRole(context.Background(), actor.ActorRoleAssignment{
		ActorID: w.holder.ID, GroupID: w.group.ID, KindID: w.kind.ID, GrantedBy: w.granter.ID,
	})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return a
}

func TestAGrantIsRefusedForARetiredKindAnUnknownActorOrGroupAndADoubleSeat(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	w.grant(t)
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: w.kind.ID}); !errors.Is(err, actor.ErrSeatAlreadyHeld) {
		t.Errorf("a second identical seat: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: "nobody", GroupID: w.group.ID, KindID: w.kind.ID}); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: "nowhere", KindID: w.kind.ID}); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: "no-kind"}); !errors.Is(err, actor.ErrRoleKindNotFound) {
		t.Errorf("unknown kind: %v", err)
	}
	retired, _ := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Régi tisztség"})
	if _, err := w.um.RetireRoleKind(ctx, retired.ID, w.granter.ID); err != nil {
		t.Fatalf("RetireRoleKind: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: retired.ID}); !errors.Is(err, actor.ErrKindRetired) {
		t.Errorf("a retired kind: %v", err)
	}
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: w.kind.ID, ApprovalLimit: limitOf(-5)}); !errors.Is(err, actor.ErrApprovalLimitNegative) {
		t.Errorf("a negative limit: %v", err)
	}
}

func TestASeatEndsForTheReasonThatEndedIt(t *testing.T) {
	cases := []struct {
		name   string
		end    func(actor.UserManager, context.Context, string) error
		reason models.EndReason
	}{
		{"revoked", func(um actor.UserManager, ctx context.Context, id string) error { return um.RevokeRole(ctx, id) }, models.EndedByRevocation},
		{"resigned", func(um actor.UserManager, ctx context.Context, id string) error { return um.StepDownRole(ctx, id) }, models.EndedByResignation},
		{"evicted", func(um actor.UserManager, ctx context.Context, id string) error { return um.EndRoleOnEviction(ctx, id) }, models.EndedByEviction},
		{"departed", func(um actor.UserManager, ctx context.Context, id string) error {
			return um.EndRoleOnDeparture(ctx, id)
		}, models.EndedByDeparture},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newSeatWorld(t)
			ctx := context.Background()
			seat := w.grant(t)
			if err := c.end(w.um, ctx, seat.ID); err != nil {
				t.Fatalf("ending the seat: %v", err)
			}
			read, err := w.um.GetRoleAssignment(ctx, seat.ID)
			if err != nil || read.Active || read.EndedAt == "" || read.EndedReason != c.reason {
				t.Fatalf("seat after = %+v %v, want ended as %q", read, err, c.reason)
			}
			if err := c.end(w.um, ctx, seat.ID); err != nil {
				t.Errorf("ending an ended seat again should be quiet: %v", err)
			}
			if err := c.end(w.um, ctx, "no-seat"); !errors.Is(err, actor.ErrAssignmentNotFound) {
				t.Errorf("ending an unknown seat: %v", err)
			}
			again, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: w.kind.ID})
			if err != nil || !again.Active {
				t.Errorf("a person may hold the seat again after it ended: %+v %v", again, err)
			}
		})
	}
}

func TestSeatsListByGroupAndActorWithTheActiveFilter(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	first := w.grant(t)
	other, _ := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Pénztáros"})
	second, _ := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: other.ID})
	if err := w.um.RevokeRole(ctx, first.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	all, err := w.um.ListRoleAssignmentsForGroup(ctx, w.group.ID, false)
	if err != nil || len(all) != 2 {
		t.Fatalf("all for group = %+v %v", all, err)
	}
	live, err := w.um.ListRoleAssignmentsForActor(ctx, w.holder.ID, true)
	if err != nil || len(live) != 1 || live[0].ID != second.ID {
		t.Errorf("live for actor = %+v %v", live, err)
	}
	holders, err := w.um.ListRoleHolders(ctx, other.ID, true)
	if err != nil || len(holders) != 1 || holders[0].ActorID != w.holder.ID {
		t.Errorf("holders = %+v %v", holders, err)
	}
	if _, err := w.um.GetRoleAssignment(ctx, "no-seat"); !errors.Is(err, actor.ErrAssignmentNotFound) {
		t.Errorf("unknown seat: %v", err)
	}
}

func TestAKindWithLiveSeatsCannotBeRetiredUntilTheyEnd(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	seat := w.grant(t)
	if _, err := w.um.RetireRoleKind(ctx, w.kind.ID, w.granter.ID); !errors.Is(err, actor.ErrKindHasLiveAssignments) {
		t.Errorf("retiring a kind in use: %v", err)
	}
	w.um.RevokeRole(ctx, seat.ID)
	retired, err := w.um.RetireRoleKind(ctx, w.kind.ID, w.granter.ID)
	if err != nil || retired.RetiredAt == "" || retired.RetiredBy != w.granter.ID {
		t.Fatalf("RetireRoleKind = %+v %v", retired, err)
	}
	same, err := w.um.RetireRoleKind(ctx, w.kind.ID, "someone-else")
	if err != nil || same.RetiredBy != w.granter.ID {
		t.Errorf("retiring twice must not change who retired it: %+v %v", same, err)
	}
	back, err := w.um.UnretireRoleKind(ctx, w.kind.ID)
	if err != nil || back.RetiredAt != "" || back.RetiredBy != "" {
		t.Errorf("UnretireRoleKind = %+v %v", back, err)
	}
	if _, err := w.um.UnretireRoleKind(ctx, "no-kind"); !errors.Is(err, actor.ErrRoleKindNotFound) {
		t.Errorf("unknown kind: %v", err)
	}
	if _, err := w.um.RetireRoleKind(ctx, "no-kind", "x"); !errors.Is(err, actor.ErrRoleKindNotFound) {
		t.Errorf("retire unknown kind: %v", err)
	}
	if _, err := w.um.GetRoleKind(ctx, w.kind.ID); err != nil {
		t.Errorf("GetRoleKind: %v", err)
	}
}

func TestStandingShowsRegistrationsAndSeatsWithNamesAndWhoGrantedThem(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	w.um.SetRoleKindLeadsGroup(ctx, w.kind.ID, true)
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.holder.ID, ""); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	w.grant(t)
	s, err := w.um.GetActorStanding(ctx, w.holder.ID)
	if err != nil {
		t.Fatalf("GetActorStanding: %v", err)
	}
	if s.DisplayName != "Szabó Sára" || len(s.Registrations) != 1 || s.Registrations[0].GroupName != "Városi tanács" {
		t.Errorf("registrations = %+v", s)
	}
	if len(s.Seats) != 1 {
		t.Fatalf("seats = %+v", s.Seats)
	}
	seat := s.Seats[0]
	if seat.RoleName != "Jegyző" || seat.GroupName != "Városi tanács" || !seat.LeadsGroup || seat.GrantedByName != "Polgármester Pál" || seat.EndedAt != "" {
		t.Errorf("seat = %+v", seat)
	}
	if _, err := w.um.GetActorStanding(ctx, "nobody"); !errors.Is(err, actor.ErrActorNotFound) {
		t.Errorf("unknown actor: %v", err)
	}
}

func TestTheGroupDashboardCountsMembersChildrenAndActiveRoles(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	if _, err := w.um.JoinGroup(ctx, w.group.ID, w.holder.ID, ""); err != nil {
		t.Fatalf("JoinGroup: %v", err)
	}
	w.um.CreateGroup(ctx, actor.Group{Name: "Albizottság", ParentID: w.group.ID})
	w.grant(t)
	d, err := w.um.GroupDashboard(ctx, w.group.ID)
	if err != nil {
		t.Fatalf("GroupDashboard: %v", err)
	}
	if d.MemberCount != 1 || d.ChildCount != 1 || d.ActiveRoles != 1 {
		t.Errorf("dashboard = %+v", d)
	}
	if len(d.Roles) != 1 || d.Roles[0].KindName != "Jegyző" || len(d.Roles[0].Members) != 1 || d.Roles[0].Members[0] != w.holder.ID {
		t.Errorf("roles = %+v", d.Roles)
	}
	if _, err := w.um.GroupDashboard(ctx, "nowhere"); !errors.Is(err, actor.ErrGroupNotFound) {
		t.Errorf("unknown group: %v", err)
	}
}

func TestDefinitionRolesCreateRenameAndRetireTheirKinds(t *testing.T) {
	w := newSeatWorld(t)
	ctx := context.Background()
	if _, err := w.um.RoleKindForDefinitionRole(ctx, ""); !errors.Is(err, actor.ErrDefinitionRoleIDRequired) {
		t.Errorf("blank definition role id: %v", err)
	}
	if _, err := w.um.RoleKindForDefinitionRole(ctx, "r-clerk"); !errors.Is(err, actor.ErrRoleKindNotFound) {
		t.Errorf("no kind yet: %v", err)
	}
	clerk, err := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Clerk", DefinitionRoleID: "r-clerk"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	dropped, _ := w.um.CreateRoleKind(ctx, actor.RoleKind{Name: "Treasurer", DefinitionRoleID: "r-treasurer"})
	if _, err := w.um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: w.holder.ID, GroupID: w.group.ID, KindID: dropped.ID}); err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	found, err := w.um.RoleKindForDefinitionRole(ctx, "r-clerk")
	if err != nil || found.ID != clerk.ID {
		t.Fatalf("RoleKindForDefinitionRole = %+v %v", found, err)
	}
	if _, err := w.um.SyncDefinitionRoles(ctx, []actor.DefinitionRole{{ID: "", Name: "x"}}, "owner"); !errors.Is(err, actor.ErrDefinitionRoleIDRequired) {
		t.Errorf("a role with no id: %v", err)
	}
	sync, err := w.um.SyncDefinitionRoles(ctx, []actor.DefinitionRole{{ID: "r-clerk", Name: "Chief clerk", Staff: true}}, "owner")
	if err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	if len(sync.Renamed) != 1 || sync.Renamed[0].Name != "Chief clerk" {
		t.Errorf("renamed = %+v", sync.Renamed)
	}
	if len(sync.Retired) != 1 || sync.Retired[0].ID != dropped.ID || sync.Retired[0].RetiredBy != "owner" {
		t.Errorf("retired = %+v", sync.Retired)
	}
	renamed, _ := w.um.GetRoleKind(ctx, clerk.ID)
	if renamed.Name != "Chief clerk" || !renamed.Staff {
		t.Errorf("stored kind = %+v", renamed)
	}
	standing, _ := w.um.GetActorStanding(ctx, w.holder.ID)
	if len(standing.Seats) != 1 || standing.Seats[0].EndedReason != models.EndedByRoleRemoved {
		t.Errorf("a seat of a dropped role must end as role removed: %+v", standing.Seats)
	}
	again, err := w.um.SyncDefinitionRoles(ctx, []actor.DefinitionRole{{ID: "r-clerk", Name: "Chief clerk", Staff: true}}, "owner")
	if err != nil || len(again.Renamed) != 0 || len(again.Retired) != 0 {
		t.Errorf("a second identical sync changed things: %+v %v", again, err)
	}
}
