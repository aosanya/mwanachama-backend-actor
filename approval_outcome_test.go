package mwanachamaactor_test

import (
	"context"
	"fmt"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

type outcomeSeat struct {
	mayApproveAbove bool
	limit           *int
	ended           bool
}

func approverWith(t *testing.T, um actor.UserManager, seats ...outcomeSeat) string {
	t.Helper()
	ctx := context.Background()
	holder, err := um.CreateActor(ctx, actor.Actor{DisplayName: "Kovács Anna"})
	if err != nil {
		t.Fatalf("CreateActor: %v", err)
	}
	for at, s := range seats {
		group, err := um.CreateGroup(ctx, actor.Group{Name: fmt.Sprintf("Városi tanács %d", at)})
		if err != nil {
			t.Fatalf("CreateGroup: %v", err)
		}
		kind, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: fmt.Sprintf("Pénzügyi vezető %d", at), ApprovesAboveLimit: s.mayApproveAbove})
		if err != nil {
			t.Fatalf("CreateRoleKind: %v", err)
		}
		granted, err := um.GrantRole(ctx, actor.ActorRoleAssignment{ActorID: holder.ID, GroupID: group.ID, KindID: kind.ID, ApprovalLimit: s.limit})
		if err != nil {
			t.Fatalf("GrantRole: %v", err)
		}
		if s.ended {
			if err := um.RevokeRole(ctx, granted.ID); err != nil {
				t.Fatalf("RevokeRole: %v", err)
			}
		}
	}
	return holder.ID
}

func TestAnApprovalAboveTheLimitIsAllowedAndFlaggedOnlyWhereTheRoleMayApproveAboveIt(t *testing.T) {
	cases := []struct {
		name   string
		seats  []outcomeSeat
		amount int
		want   actor.ApprovalOutcome
	}{
		{"within the limit carries no flag", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000)}}, 400_000, actor.ApprovalOutcome{Allowed: true}},
		{"at the limit carries no flag", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000)}}, 500_000, actor.ApprovalOutcome{Allowed: true}},
		{"above the limit of a role that may is flagged with that limit", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000)}}, 900_000, actor.ApprovalOutcome{Allowed: true, OverLimit: true, Limit: 500_000}},
		{"above the limit of a role that may not is refused", []outcomeSeat{{limit: limitOf(500_000)}}, 900_000, actor.ApprovalOutcome{Limit: 500_000}},
		{"a seat with no limit is never flagged", []outcomeSeat{{mayApproveAbove: true}}, 900_000, actor.ApprovalOutcome{Allowed: true}},
		{"a seat of a role that may not, and covers the amount, carries no flag", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000)}, {limit: limitOf(900_000)}}, 700_000, actor.ApprovalOutcome{Allowed: true}},
		{"the seat that may is the one flagged when another seat falls short", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000)}, {limit: limitOf(900_000)}}, 950_000, actor.ApprovalOutcome{Allowed: true, OverLimit: true, Limit: 500_000}},
		{"refused reports the largest limit among the seats", []outcomeSeat{{limit: limitOf(200_000)}, {limit: limitOf(350_000)}}, 950_000, actor.ApprovalOutcome{Limit: 350_000}},
		{"an ended seat of a role that may lends nothing", []outcomeSeat{{mayApproveAbove: true, limit: limitOf(500_000), ended: true}}, 900_000, actor.ApprovalOutcome{}},
		{"no seat is refused", nil, 1, actor.ApprovalOutcome{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			um := newManager(t)
			approver := approverWith(t, um, c.seats...)
			got, err := actor.ApprovalOutcomeFor(context.Background(), um, approver, c.amount)
			if err != nil {
				t.Fatalf("ApprovalOutcomeFor: %v", err)
			}
			if got != c.want {
				t.Fatalf("ApprovalOutcomeFor = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestARoleKindStoresWhetherItMayApproveAboveItsLimit(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	kind, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Polgármester", ApprovesAboveLimit: true})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	read, err := um.GetRoleKind(ctx, kind.ID)
	if err != nil || !read.ApprovesAboveLimit {
		t.Fatalf("stored kind = %+v %v, want it to approve above its limit", read, err)
	}
	plain, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Jegyző"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if read, _ := um.GetRoleKind(ctx, plain.ID); read.ApprovesAboveLimit {
		t.Fatalf("a kind nobody set approves above its limit")
	}
}

func TestSyncingTheDefinitionCarriesTheSettingToTheKind(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	kind, err := um.CreateRoleKind(ctx, actor.RoleKind{Name: "Polgármester", DefinitionRoleID: "R-1"})
	if err != nil {
		t.Fatalf("CreateRoleKind: %v", err)
	}
	if _, err := um.SyncDefinitionRoles(ctx, []actor.DefinitionRole{{ID: "R-1", Name: "Polgármester", ApprovesAboveLimit: true}}, "owner"); err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	if read, _ := um.GetRoleKind(ctx, kind.ID); !read.ApprovesAboveLimit {
		t.Fatalf("the published setting did not reach the kind: %+v", read)
	}
	if _, err := um.SyncDefinitionRoles(ctx, []actor.DefinitionRole{{ID: "R-1", Name: "Polgármester"}}, "owner"); err != nil {
		t.Fatalf("SyncDefinitionRoles: %v", err)
	}
	if read, _ := um.GetRoleKind(ctx, kind.ID); read.ApprovesAboveLimit {
		t.Fatalf("withdrawing the setting did not reach the kind: %+v", read)
	}
}
