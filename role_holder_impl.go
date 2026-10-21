package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) ListRoleHolders(ctx context.Context, kindID string, activeOnly bool) ([]models.RoleHolder, error) {
	if _, err := m.GetRoleKind(ctx, kindID); err != nil {
		return nil, err
	}
	seats, err := m.listRoleAssignments(ctx, "kind_id = ?", kindID, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("ListRoleHolders: %w", err)
	}
	out := make([]models.RoleHolder, 0, len(seats))
	for _, seat := range seats {
		holder := models.RoleHolder{ActorRoleAssignment: seat}
		actor, err := m.GetActor(ctx, seat.ActorID)
		switch {
		case err == nil:
			holder.DisplayName = actor.DisplayName
		case !errors.Is(err, ErrActorNotFound):
			return nil, fmt.Errorf("ListRoleHolders: %w", err)
		}
		out = append(out, holder)
	}
	return out, nil
}
