package mwanachamaactor

import (
	"context"
	"fmt"
)

func WithinLimit(a ActorRoleAssignment, amount int) bool {
	if !a.Active || amount < 0 {
		return false
	}
	return a.ApprovalLimit == nil || amount <= *a.ApprovalLimit
}

func CheckLimit(a ActorRoleAssignment, amount int) error {
	if !WithinLimit(a, amount) {
		return ErrOverApprovalLimit
	}
	return nil
}

type ApprovalOutcome struct {
	Allowed   bool
	OverLimit bool
	Limit     int
}

func ApprovalOutcomeFor(ctx context.Context, um UserManager, actorID string, amount int) (ApprovalOutcome, error) {
	seats, err := um.ListRoleAssignmentsForActor(ctx, actorID, true)
	if err != nil {
		return ApprovalOutcome{}, fmt.Errorf("ApprovalOutcomeFor: %w", err)
	}
	largest, largestThatMay := 0, 0
	mayOverrun := false
	for _, seat := range seats {
		if WithinLimit(seat, amount) {
			return ApprovalOutcome{Allowed: true}, nil
		}
		if !seat.Active || seat.ApprovalLimit == nil {
			continue
		}
		largest = max(largest, *seat.ApprovalLimit)
		kind, err := um.GetRoleKind(ctx, seat.KindID)
		if err != nil {
			return ApprovalOutcome{}, fmt.Errorf("ApprovalOutcomeFor: %w", err)
		}
		if kind.ApprovesAboveLimit {
			mayOverrun = true
			largestThatMay = max(largestThatMay, *seat.ApprovalLimit)
		}
	}
	if mayOverrun {
		return ApprovalOutcome{Allowed: true, OverLimit: true, Limit: largestThatMay}, nil
	}
	return ApprovalOutcome{Limit: largest}, nil
}
