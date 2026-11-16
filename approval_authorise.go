package mwanachamaactor

import (
	"context"
	"fmt"
)

func (m *userManager) SetApprovalLimit(ctx context.Context, assignmentID string, limit *int) (ActorRoleAssignment, error) {
	if limit != nil && *limit < 0 {
		return ActorRoleAssignment{}, fmt.Errorf("SetApprovalLimit: %w", ErrApprovalLimitNegative)
	}
	seat, err := m.GetRoleAssignment(ctx, assignmentID)
	if err != nil {
		return ActorRoleAssignment{}, err
	}
	if !seat.Active {
		return ActorRoleAssignment{}, fmt.Errorf("SetApprovalLimit: %w", ErrSeatEnded)
	}
	res := m.store.Query(ctx, roleActorRoleAssignment).
		Where("id = ? AND active", assignmentID).
		Updates(map[string]any{"approval_limit": limit})
	if res.Error != nil {
		return ActorRoleAssignment{}, fmt.Errorf("SetApprovalLimit: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ActorRoleAssignment{}, fmt.Errorf("SetApprovalLimit: %w", ErrSeatEnded)
	}
	seat.ApprovalLimit = limit
	return seat, nil
}

func (m *userManager) AuthoriseApproval(ctx context.Context, assignmentID string, amount int) error {
	seat, err := m.GetRoleAssignment(ctx, assignmentID)
	if err != nil {
		return err
	}
	if err := CheckLimit(seat, amount); err != nil {
		return fmt.Errorf("AuthoriseApproval: %w", err)
	}
	return nil
}
