package mwanachamaactor_test

import (
	"context"
	"errors"
	"testing"

	actor "github.com/aosanya/mwanachama-backend-actor"
)

func grantWithLimit(t *testing.T, um actor.UserManager, limit *int) actor.ActorRoleAssignment {
	t.Helper()
	actorID, groupID, kindID := seatFixture(t, um)
	seat, err := um.GrantRole(context.Background(), actor.ActorRoleAssignment{ActorID: actorID, GroupID: groupID, KindID: kindID, ApprovalLimit: limit})
	if err != nil {
		t.Fatalf("GrantRole: %v", err)
	}
	return seat
}

func TestAuthoriseApprovalRefusesAnAmountOverTheLimit(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	seat := grantWithLimit(t, um, limitOf(100))

	if err := um.AuthoriseApproval(ctx, seat.ID, 101); !errors.Is(err, actor.ErrOverApprovalLimit) {
		t.Fatalf("AuthoriseApproval over = %v, want ErrOverApprovalLimit", err)
	}
	if err := um.AuthoriseApproval(ctx, seat.ID, 100); err != nil {
		t.Fatalf("AuthoriseApproval at the limit = %v", err)
	}
}

func TestAuthoriseApprovalWithoutALimitApprovesAnything(t *testing.T) {
	um := newManager(t)
	seat := grantWithLimit(t, um, nil)
	if err := um.AuthoriseApproval(context.Background(), seat.ID, 2_000_000_000); err != nil {
		t.Fatalf("AuthoriseApproval = %v", err)
	}
}

func TestAuthoriseApprovalRefusesAnEndedSeatAndAMissingOne(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	seat := grantWithLimit(t, um, nil)
	if err := um.RevokeRole(ctx, seat.ID); err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	if err := um.AuthoriseApproval(ctx, seat.ID, 1); !errors.Is(err, actor.ErrOverApprovalLimit) {
		t.Fatalf("ended seat = %v, want ErrOverApprovalLimit", err)
	}
	if err := um.AuthoriseApproval(ctx, "missing", 1); !errors.Is(err, actor.ErrAssignmentNotFound) {
		t.Fatalf("missing seat = %v, want ErrAssignmentNotFound", err)
	}
}

func TestSetApprovalLimitChangesALiveSeatAndTheNextApproval(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	seat := grantWithLimit(t, um, limitOf(100))

	if err := um.AuthoriseApproval(ctx, seat.ID, 500); !errors.Is(err, actor.ErrOverApprovalLimit) {
		t.Fatalf("before raising = %v, want ErrOverApprovalLimit", err)
	}
	updated, err := um.SetApprovalLimit(ctx, seat.ID, limitOf(1000))
	if err != nil {
		t.Fatalf("SetApprovalLimit: %v", err)
	}
	if updated.ApprovalLimit == nil || *updated.ApprovalLimit != 1000 {
		t.Fatalf("returned limit = %v, want 1000", updated.ApprovalLimit)
	}
	read, err := um.GetRoleAssignment(ctx, seat.ID)
	if err != nil || read.ApprovalLimit == nil || *read.ApprovalLimit != 1000 {
		t.Fatalf("stored limit = %v (%v), want 1000", read.ApprovalLimit, err)
	}
	if err := um.AuthoriseApproval(ctx, seat.ID, 500); err != nil {
		t.Fatalf("after raising = %v", err)
	}

	if _, err := um.SetApprovalLimit(ctx, seat.ID, limitOf(10)); err != nil {
		t.Fatalf("lower: %v", err)
	}
	if err := um.AuthoriseApproval(ctx, seat.ID, 500); !errors.Is(err, actor.ErrOverApprovalLimit) {
		t.Fatalf("after lowering = %v, want ErrOverApprovalLimit", err)
	}

	cleared, err := um.SetApprovalLimit(ctx, seat.ID, nil)
	if err != nil || cleared.ApprovalLimit != nil {
		t.Fatalf("clear = %v, %v", cleared.ApprovalLimit, err)
	}
	read, _ = um.GetRoleAssignment(ctx, seat.ID)
	if read.ApprovalLimit != nil {
		t.Fatalf("stored limit after clear = %d, want none", *read.ApprovalLimit)
	}
}

func TestSetApprovalLimitRefusesNegativeEndedAndMissing(t *testing.T) {
	um := newManager(t)
	ctx := context.Background()
	seat := grantWithLimit(t, um, limitOf(5))

	if _, err := um.SetApprovalLimit(ctx, seat.ID, limitOf(-1)); !errors.Is(err, actor.ErrApprovalLimitNegative) {
		t.Fatalf("negative = %v, want ErrApprovalLimitNegative", err)
	}
	if _, err := um.SetApprovalLimit(ctx, "missing", limitOf(1)); !errors.Is(err, actor.ErrAssignmentNotFound) {
		t.Fatalf("missing = %v, want ErrAssignmentNotFound", err)
	}
	if err := um.StepDownRole(ctx, seat.ID); err != nil {
		t.Fatalf("StepDownRole: %v", err)
	}
	if _, err := um.SetApprovalLimit(ctx, seat.ID, limitOf(9)); !errors.Is(err, actor.ErrSeatEnded) {
		t.Fatalf("ended = %v, want ErrSeatEnded", err)
	}
	read, _ := um.GetRoleAssignment(ctx, seat.ID)
	if read.ApprovalLimit == nil || *read.ApprovalLimit != 5 {
		t.Fatalf("ended seat's limit changed to %v", read.ApprovalLimit)
	}
}
