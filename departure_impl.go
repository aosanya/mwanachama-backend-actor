package mwanachamaactor

import (
	"context"
	"fmt"
	"strings"
)

func (m *userManager) LeaveAllGroups(ctx context.Context, actorID string) error {
	if strings.TrimSpace(actorID) == "" {
		return ErrInvalidActor
	}
	seats, err := m.ListRoleAssignmentsForActor(ctx, actorID, true)
	if err != nil {
		return fmt.Errorf("LeaveAllGroups: %w", err)
	}
	for _, seat := range seats {
		if err := m.EndRoleOnDeparture(ctx, seat.ID); err != nil {
			return fmt.Errorf("LeaveAllGroups: %w", err)
		}
	}
	held, err := m.ListGroupsForActor(ctx, actorID)
	if err != nil {
		return fmt.Errorf("LeaveAllGroups: %w", err)
	}
	for _, registration := range held {
		if _, _, err := m.Deregister(ctx, actorID, registration.GroupID); err != nil {
			return fmt.Errorf("LeaveAllGroups: %w", err)
		}
	}
	if err := m.endPendingJoinRequests(ctx, actorID, actorID); err != nil {
		return fmt.Errorf("LeaveAllGroups: %w", err)
	}
	return nil
}
