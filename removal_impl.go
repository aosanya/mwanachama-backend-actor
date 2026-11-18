package mwanachamaactor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-actor/models"
)

func (m *userManager) RemoveFromGroup(ctx context.Context, actorID, groupID, removedBy string) (models.GroupRemoval, error) {
	if strings.TrimSpace(removedBy) == "" {
		return models.GroupRemoval{}, ErrRemoverRequired
	}
	removal := models.GroupRemoval{
		ID:        newSpecID(),
		ActorID:   actorID,
		GroupID:   groupID,
		RemovedBy: removedBy,
		RemovedAt: NowRFC3339(),
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		registration := m.from(tx, roleActorGroupAssignment).
			Where("actor_id = ? AND group_id = ?", actorID, groupID).
			Delete(map[string]any{})
		if registration.Error != nil {
			return registration.Error
		}
		seats := m.from(tx, roleActorRoleAssignment).
			Where("actor_id = ? AND group_id = ? AND active", actorID, groupID).
			Updates(map[string]any{"active": false, "ended_at": removal.RemovedAt, "ended_reason": string(EndedByEviction)})
		if seats.Error != nil {
			return seats.Error
		}
		if registration.RowsAffected == 0 && seats.RowsAffected == 0 {
			return ErrNotInGroup
		}
		row, err := m.store.Row(roleGroupRemoval, removal)
		if err != nil {
			return err
		}
		return m.from(tx, roleGroupRemoval).Create(row).Error
	})
	if err != nil {
		return models.GroupRemoval{}, fmt.Errorf("RemoveFromGroup: %w", err)
	}
	return removal, nil
}

func (m *userManager) RemoveFromAllGroups(ctx context.Context, actorID, removedBy string) ([]models.GroupRemoval, error) {
	if strings.TrimSpace(removedBy) == "" {
		return nil, ErrRemoverRequired
	}
	held, err := m.ListGroupsForActor(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("RemoveFromAllGroups: %w", err)
	}
	out := make([]models.GroupRemoval, 0, len(held))
	for _, a := range held {
		removal, err := m.RemoveFromGroup(ctx, actorID, a.GroupID, removedBy)
		if err != nil {
			return out, fmt.Errorf("RemoveFromAllGroups: %w", err)
		}
		out = append(out, removal)
	}
	if err := m.endAllRoleAssignments(ctx, actorID, EndedByEviction); err != nil {
		return out, fmt.Errorf("RemoveFromAllGroups: %w", err)
	}
	if err := m.endPendingJoinRequests(ctx, actorID, removedBy); err != nil {
		return out, fmt.Errorf("RemoveFromAllGroups: %w", err)
	}
	return out, nil
}

func (m *userManager) endAllRoleAssignments(ctx context.Context, actorID string, reason models.EndReason) error {
	return m.store.Query(ctx, roleActorRoleAssignment).
		Where("actor_id = ? AND active", actorID).
		Updates(map[string]any{"active": false, "ended_at": NowRFC3339(), "ended_reason": string(reason)}).Error
}

func (m *userManager) endPendingJoinRequests(ctx context.Context, actorID, endedBy string) error {
	asked, err := m.ListJoinRequestsForActor(ctx, actorID)
	if err != nil {
		return err
	}
	for _, req := range asked {
		if !req.Pending() {
			continue
		}
		_, err := m.decide(ctx, req.ID, endedBy, models.JoinRequestDeclined, true, "", vouchedByHost)
		if err != nil && !errors.Is(err, ErrJoinRequestDecided) {
			return err
		}
	}
	return nil
}

func (m *userManager) ListRemovals(ctx context.Context, actorID string) ([]models.GroupRemoval, error) {
	q := m.store.Query(ctx, roleGroupRemoval).Where("actor_id = ?", actorID)
	out, err := specstore.List[models.GroupRemoval](m.store, q, roleGroupRemoval)
	if err != nil {
		return nil, fmt.Errorf("ListRemovals: %w", err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].RemovedAt < out[j].RemovedAt })
	return out, nil
}
